package pos

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/mpesa"
)

// STKRequest is POST /pos/mpesa/stk — send a customer PIN prompt.
type STKRequest struct {
	CashierID  string `json:"cashier_id"` // user public id — must hold pos.sale.create
	Phone      string `json:"phone"`
	Amount     string `json:"amount"`
	AccountRef string `json:"account_ref"` // shown on the customer's prompt (≤12 chars)
}

// STK handles POST /pos/mpesa/stk.
func (h *Handler) STK(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req STKRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.mpesa == nil {
		httpx.WriteError(w, r, mpesa.ErrNotConfigured)
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, "pos.sale.create")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		httpx.WriteError(w, r, mpesa.ErrInvalidAmount)
		return
	}
	t, err := h.svc.mpesa.InitiateSTK(r.Context(), d.TenantID, d.BranchID, cashierID,
		req.Phone, req.AccountRef, "POS sale", amount)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, t)
}

// STKStatus handles GET /pos/mpesa/{id}/status — the till polls this while the
// customer types their PIN; the service asks Daraja when the callback is late.
func (h *Handler) STKStatus(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	if h.svc.mpesa == nil {
		httpx.WriteError(w, r, mpesa.ErrNotConfigured)
		return
	}
	t, err := h.svc.mpesa.STKStatus(r.Context(), d.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if t.BranchID != nil && *t.BranchID != d.BranchID {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	httpx.OK(w, t)
}

// MpesaFeed handles GET /pos/mpesa/feed — a long-lived SSE stream of confirmed
// collections (and prompt results) for this till's branch.
func (h *Handler) MpesaFeed(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	if h.svc.mpesa == nil {
		httpx.WriteError(w, r, mpesa.ErrNotConfigured)
		return
	}
	h.svc.mpesa.ServeFeed(w, r, d.TenantID, d.BranchID)
}

// MpesaRecent handles GET /pos/mpesa/recent — this branch's unclaimed
// collections from the last few hours (feed hydration after a reconnect).
func (h *Handler) MpesaRecent(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	if h.svc.mpesa == nil {
		httpx.WriteError(w, r, mpesa.ErrNotConfigured)
		return
	}
	rows, err := h.svc.mpesa.FeedForTill(r.Context(), d.TenantID, d.BranchID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"items": rows})
}

// MpesaAllocateRequest is POST /pos/mpesa/{id}/allocate.
type MpesaAllocateRequest struct {
	CashierID string `json:"cashier_id"`
	SaleRef   string `json:"sale_ref"` // receipt number — resolves within this till's branch
}

// MpesaAllocate handles POST /pos/mpesa/{id}/allocate — links a feed
// collection to a completed sale at this branch.
func (h *Handler) MpesaAllocate(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req MpesaAllocateRequest
	if err := httpx.Decode(w, r, &req); err != nil || req.SaleRef == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"sale_ref": "required"}))
		return
	}
	if h.svc.mpesa == nil {
		httpx.WriteError(w, r, mpesa.ErrNotConfigured)
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, mpesa.PermTransactionAllocate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sale, err := h.svc.sales.GetByReceipt(r.Context(), d.TenantID, d.BranchID, req.SaleRef)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.mpesa.Allocate(r.Context(), d.TenantID, cashierID,
		chi.URLParam(r, "id"), sale.Sale.PublicID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"allocated": true})
}

// cashierCan resolves a cashier public id and enforces one permission.
func (h *Handler) cashierCan(r *http.Request, tenantID uint64, cashierPub, perm string) (uint64, error) {
	cashierID, err := h.svc.userByPublicID(r.Context(), tenantID, cashierPub)
	if err != nil {
		return 0, httpx.NewError(422, "bad_cashier", "Unknown cashier.")
	}
	authz, err := h.svc.sales.Authz(r.Context(), tenantID, cashierID)
	if err != nil {
		return 0, httpx.NewError(422, "bad_cashier", "Unknown cashier.")
	}
	if !authz.Perms[perm] {
		return 0, httpx.ErrForbidden
	}
	return cashierID, nil
}
