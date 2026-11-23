package pos

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/sales"
)

// SaleRequest is POST /pos/sales — a completed checkout from a terminal.
type SaleRequest struct {
	IdempotencyKey  string               `json:"idempotency_key"`
	CashierID       string               `json:"cashier_id"` // user public id
	CustomerID      string               `json:"customer_id"`
	SoldAt          string               `json:"sold_at"`
	Lines           []sales.LineInput    `json:"lines"`
	OrderDiscount   string               `json:"order_discount"`
	Payments        []sales.PaymentInput `json:"payments"`
	SupervisorGrant string               `json:"supervisor_grant"`
	DiscountNote    string               `json:"discount_note"`
}

// Sale handles POST /pos/sales. Idempotent on idempotency_key — a retried
// upload after a network drop returns the original receipt, never a second
// sale.
func (h *Handler) Sale(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req SaleRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.sales == nil {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	// The cashier must hold pos.sale.create — the PIN session is not the
	// security boundary for uploads.
	cashierID, err := h.svc.userByPublicID(r.Context(), d.TenantID, req.CashierID)
	if err != nil {
		httpx.WriteError(w, r, sales.ErrCashierNotFound)
		return
	}
	authz, err := h.svc.sales.Authz(r.Context(), d.TenantID, cashierID)
	if err != nil {
		httpx.WriteError(w, r, sales.ErrCashierNotFound)
		return
	}
	if !authz.Perms["pos.sale.create"] {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	soldAt, err := time.Parse(time.RFC3339, req.SoldAt)
	if err != nil {
		soldAt = time.Now()
	}
	// A supervisor grant binds the sale to the authorizing user; discounts or
	// overrides that exceed the cashier's role require it.
	supervisorID, err := h.svc.ConsumeGrant(d, req.SupervisorGrant, "pos.discount.override")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	termID := d.TerminalID
	detail, deduped, err := h.svc.sales.Complete(r.Context(), sales.CompleteInput{
		TenantID: d.TenantID, BranchID: d.BranchID, TerminalID: &termID,
		CashierID: &cashierID, CustomerID: req.CustomerID,
		IdempotencyKey: req.IdempotencyKey, Source: sales.SourcePOS, SoldAt: soldAt,
		Lines: req.Lines, OrderDiscount: req.OrderDiscount, Payments: req.Payments,
		ActorID: cashierID, ActorType: "user", SupervisorID: supervisorID,
		DiscountNote: req.DiscountNote,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{
		"sale":         sales.ToDetail(detail),
		"receipt_no":   detail.Sale.ReceiptNo,
		"deduplicated": deduped,
		"receipt":      h.svc.issueReceipt(r.Context(), d, detail, cashierID),
	})
}

// SaleLookup handles GET /pos/sales/{ref} — transaction lookup by receipt
// number or public id, scoped to the terminal's branch.
func (h *Handler) SaleLookup(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	detail, err := h.svc.sales.GetByReceipt(r.Context(), d.TenantID, 0, chi.URLParam(r, "ref"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if detail.Sale.BranchID != d.BranchID {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	httpx.OK(w, sales.ToDetail(detail))
}

// Heartbeat handles POST /pos/heartbeat — the device-auth middleware already
// refreshed last_seen; this returns server time for the status bar.
func (h *Handler) Heartbeat(w http.ResponseWriter, _ *http.Request) {
	httpx.OK(w, map[string]any{"server_time": time.Now().UTC().Format(time.RFC3339)})
}
