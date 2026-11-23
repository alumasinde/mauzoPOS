package bindings

import (
	"encoding/json"
	"errors"
	"time"

	"mauzo/internal/core/security"
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
}

// PayResult is what the UI shows after tendering.
type PayResult struct {
	SaleID    string `json:"sale_id"`
	ReceiptNo uint64 `json:"receipt_no"`
	Total     string `json:"total"`
	ChangeDue string `json:"change_due"`
	Offline   bool   `json:"offline"`
}

// saleResponse is the server reply to POST /pos/sales.
type saleResponse struct {
	Sale struct {
		ID        string `json:"id"`
		Total     string `json:"total"`
		ChangeDue string `json:"change_due"`
	} `json:"sale"`
	ReceiptNo    uint64 `json:"receipt_no"`
	Deduplicated bool   `json:"deduplicated"`
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
		return PayResult{SaleID: out.Sale.ID, ReceiptNo: out.ReceiptNo,
			Total: out.Sale.Total, ChangeDue: out.Sale.ChangeDue}, nil
	}
	if !errors.Is(err, ErrOffline) {
		return PayResult{}, err // validation errors surface to the cashier
	}
	// Offline: queue the exact payload; the idempotency key makes replay safe.
	if err := a.db.RecordPendingSale(a.ctx, key, string(payload), "{}", "", a.cashierID()); err != nil {
		return PayResult{}, err
	}
	return PayResult{SaleID: key, Offline: true}, nil
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
