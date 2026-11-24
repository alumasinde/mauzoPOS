package bindings

import (
	"encoding/json"
	"errors"
	"time"

	"mauzo/desktop/localdb"
	"mauzo/internal/core/security"
	"mauzo/pkg/receiptfmt"
)

// PayLine is the wire shape of one cart line for POST /pos/sales.
type PayLine struct {
	ProductID     string `json:"product_id"`
	VariantID     string `json:"variant_id,omitempty"`
	Qty           string `json:"qty"`
	PriceOverride string `json:"price_override,omitempty"`
	Discount      string `json:"discount,omitempty"`
}

// PayMethod is one tendered payment in the request.
type PayMethod struct {
	Method    string `json:"method"`
	Amount    string `json:"amount"`
	Tendered  string `json:"tendered,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// salePayload is the POST /pos/sales body — also what pending_sales stores so
// the upload pipeline replays the exact same request.
type salePayload struct {
	IdempotencyKey  string      `json:"idempotency_key"`
	CashierID       string      `json:"cashier_id"`
	CustomerID      string      `json:"customer_id,omitempty"`
	SoldAt          string      `json:"sold_at"`
	Lines           []PayLine   `json:"lines"`
	OrderDiscount   string      `json:"order_discount"`
	Payments        []PayMethod `json:"payments"`
	SupervisorGrant string      `json:"supervisor_grant,omitempty"`
	DiscountNote    string      `json:"discount_note,omitempty"`
	// Offline tells the server this sale already happened at the till: an
	// M-Pesa code it cannot match is recorded as unverified, not refused.
	Offline bool `json:"offline,omitempty"`
}

// PayResult is what the UI shows after tendering.
type PayResult struct {
	SaleID    string `json:"sale_id"`
	ReceiptNo uint64 `json:"receipt_no"`
	Total     string `json:"total"`
	ChangeDue string `json:"change_due"`
	Offline   bool   `json:"offline"`
	// Printing outcome (a printer problem never fails the sale).
	Printed    bool   `json:"printed"`
	CanPrint   bool   `json:"can_print"` // PrintLast has a receipt to print
	PrintError string `json:"print_error,omitempty"`
	// ReceiptFile is the saved PDF when the till prints to PDF.
	ReceiptFile string `json:"receipt_file,omitempty"`
}

// saleResponse is the server reply to POST /pos/sales.
type saleResponse struct {
	Sale         saleDetailWire `json:"sale"`
	ReceiptNo    uint64         `json:"receipt_no"`
	Deduplicated bool           `json:"deduplicated"`
	Receipt      *struct {
		ID     string            `json:"id"`
		Output receiptfmt.Output `json:"output"`
	} `json:"receipt"`
}

// buildPayload converts the local cart into the server request. Public ids
// are resolved from the local mirror — never trusted from the UI.
func (a *App) buildPayload(cartJSON string, payments []PayMethod,
	orderDiscount, customerID, grant, note string) ([]byte, string, error) {
	var lines []CartLine
	if err := json.Unmarshal([]byte(cartJSON), &lines); err != nil {
		return nil, "", errors.New("bad cart")
	}
	if len(lines) == 0 {
		return nil, "", errors.New("empty cart")
	}
	ids := make([]uint64, 0, len(lines)*2)
	for _, l := range lines {
		ids = append(ids, l.ProductID)
		if l.VariantID != 0 {
			ids = append(ids, l.VariantID)
		}
	}
	refs, err := a.db.ProductRefs(a.ctx, ids)
	if err != nil {
		return nil, "", err
	}
	key := security.NewPublicID()
	pl := salePayload{
		IdempotencyKey: key, CashierID: a.cashierID(), CustomerID: customerID,
		SoldAt: time.Now().UTC().Format(time.RFC3339), Payments: payments,
		OrderDiscount: orderDiscount, SupervisorGrant: grant, DiscountNote: note,
	}
	for _, l := range lines {
		pub, ok := refs.Products[l.ProductID]
		if !ok {
			return nil, "", errors.New("product missing from local mirror — sync first")
		}
		line := PayLine{ProductID: pub, VariantID: refs.Variants[l.VariantID],
			Qty: l.Qty, PriceOverride: l.PriceOverride, Discount: l.Discount}
		pl.Lines = append(pl.Lines, line)
	}
	raw, err := json.Marshal(pl)
	return raw, key, err
}

// Pay completes a sale: online it posts straight to the API (server recomputes
// pricing, decrements stock, writes the audit row in one transaction);
// offline it queues the same payload in pending_sales for the sync pipeline.
func (a *App) Pay(paymentsJSON, cartJSON, orderDiscount, customerID, grant, note string) (PayResult, error) {
	if a.db == nil {
		return PayResult{}, errors.New("local database not ready")
	}
	var payments []PayMethod
	if err := json.Unmarshal([]byte(paymentsJSON), &payments); err != nil || len(payments) == 0 {
		return PayResult{}, errors.New("at least one payment is required")
	}
	payload, key, err := a.buildPayload(cartJSON, payments, orderDiscount, customerID, grant, note)
	if err != nil {
		return PayResult{}, err
	}
	var out saleResponse
	err = a.request("POST", "/api/v1/pos/sales", json.RawMessage(payload), &out)
	if err == nil {
		res := PayResult{SaleID: out.Sale.ID, ReceiptNo: out.ReceiptNo,
			Total: out.Sale.Total, ChangeDue: out.Sale.ChangeDue}
		var rc receiptfmt.Output
		if out.Receipt != nil {
			rc = out.Receipt.Output // the server's issued snapshot
		} else {
			rc = a.receiptFromDetail(out.Sale)
		}
		a.afterSale(&res, &rc, hasCash(payments))
		return res, nil
	}
	if !errors.Is(err, ErrOffline) {
		return PayResult{}, err // validation errors surface to the cashier
	}
	// Offline: queue the payload, flagged offline; the idempotency key makes
	// replay safe.
	queued, err := markOffline(payload)
	if err != nil {
		return PayResult{}, err
	}
	if err := a.db.RecordPendingSale(a.ctx, key, string(queued), "{}", "", a.cashierID()); err != nil {
		return PayResult{}, err
	}
	res := PayResult{SaleID: key, Offline: true}
	var lines []CartLine
	if json.Unmarshal([]byte(cartJSON), &lines) == nil {
		if rc, perr := a.provisionalReceipt(key, lines, payments, orderDiscount); perr == nil {
			a.afterSale(&res, &rc, hasCash(payments))
		} else {
			res.PrintError = perr.Error()
		}
	}
	return res, nil
}

// SyncPending uploads queued offline sales. Server idempotency keys make a
// mid-batch crash safe — replayed rows deduplicate to the original sale.
func (a *App) SyncPending() (int, error) {
	if a.db == nil {
		return 0, errors.New("local database not ready")
	}
	rows, err := a.db.PendingSales(a.ctx)
	if err != nil {
		return 0, err
	}
	uploaded := 0
	var lastErr error
	for _, row := range rows {
		var res saleResponse
		err := a.request("POST", "/api/v1/pos/sales", json.RawMessage(row.Payload), &res)
		if err != nil {
			if errors.Is(err, ErrOffline) {
				break // keep order: retry the rest next sync
			}
			var apiErr *apiError
			if errors.As(err, &apiErr) && apiErr.refused() {
				// The server refused this sale (it will refuse it every
				// time): park it for a person instead of retrying forever.
				if merr := a.db.MarkSaleFailed(a.ctx, row.ID, apiErr.Error()); merr != nil {
					return uploaded, merr
				}
			}
			lastErr = err
			continue
		}
		if err := a.db.MarkSaleUploaded(a.ctx, row.ID); err != nil {
			return uploaded, err
		}
		uploaded++
	}
	return uploaded, lastErr
}

// SaleDetail fetches one sale for the transaction-lookup screen.
func (a *App) SaleDetail(ref string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/sales/"+urlQuery(ref), nil, &out)
	return out, err
}

// Customers lists synced customer accounts (credit picker).
func (a *App) Customers(search string) (json.RawMessage, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	rows, err := a.db.Customers(a.ctx, search, 50)
	if err != nil {
		return nil, err
	}
	return json.Marshal(rows)
}

// markOffline sets the offline flag on a queued payload.
func markOffline(payload []byte) ([]byte, error) {
	var p salePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	p.Offline = true
	return json.Marshal(p)
}

// FailedSales lists offline sales the server refused (needs attention).
func (a *App) FailedSales() ([]localdb.FailedSale, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	return a.db.FailedSales(a.ctx)
}

// RetryFailedSale queues a refused sale again and tries to upload it now.
func (a *App) RetryFailedSale(id string) (int, error) {
	if a.db == nil {
		return 0, errors.New("local database not ready")
	}
	if err := a.db.RequeueSale(a.ctx, id); err != nil {
		return 0, err
	}
	return a.SyncPending()
}

// PaymentMethods returns the methods this business accepts, in display order
// (the payments.methods setting synced from the back office).
func (a *App) PaymentMethods() []string {
	order := []string{"cash", "mpesa", "card", "bank", "credit"}
	allowed := map[string]bool{"cash": true, "mpesa": true, "card": true}
	if a.db != nil {
		if raw, _ := a.db.Setting(a.ctx, "payments.methods"); raw != "" {
			var list []string
			if json.Unmarshal([]byte(raw), &list) == nil && len(list) > 0 {
				allowed = map[string]bool{}
				for _, m := range list {
					allowed[m] = true
				}
			}
		}
	}
	out := []string{}
	for _, m := range order {
		if allowed[m] {
			out = append(out, m)
		}
	}
	return out
}
