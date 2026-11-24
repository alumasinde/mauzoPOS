package mpesa

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/pkg/daraja"
)

// Handler serves the tenant-facing M-Pesa endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) (identity.Actor, bool) {
	a, ok := identity.ActorFrom(r.Context())
	return a, ok && a.TenantID != 0 && a.Type == identity.ActorUser
}

// --- configuration ----------------------------------------------------------

// Config handles GET /t/mpesa/config — masked credentials view.
func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	v, err := h.svc.Config(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, v)
}

// SaveConfig handles PUT /t/mpesa/config.
func (h *Handler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var in ConfigInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SaveConfig(r.Context(), a.TenantID, in, a.ID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"saved": true})
}

// RegisterC2B handles POST /t/mpesa/config/register — pushes the webhook URLs
// to Daraja.
func (h *Handler) RegisterC2B(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	if err := h.svc.RegisterC2B(r.Context(), a.TenantID, a.ID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"registered": true})
}

// testRequest is POST /t/mpesa/test — a 1-KES prompt proving the credentials
// work end to end.
type testRequest struct {
	Phone string `json:"phone"`
}

// TestPush handles POST /t/mpesa/test.
func (h *Handler) TestPush(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var in testRequest
	if err := httpx.Decode(w, r, &in); err != nil || in.Phone == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"phone": "required"}))
		return
	}
	t, err := h.svc.InitiateSTK(r.Context(), a.TenantID, 0, a.ID, in.Phone,
		"TEST", "config check", decimal.NewFromInt(1))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"sent": true, "txn": t.PublicID})
}

// --- transactions ------------------------------------------------------------

// List handles GET /t/mpesa/transactions.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	pg := httpx.ParsePage(r)
	f := ListFilter{
		Status: r.URL.Query().Get("status"), Kind: r.URL.Query().Get("kind"),
		Allocated: r.URL.Query().Get("allocated"), Query: r.URL.Query().Get("q"),
		From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"),
	}
	if !a.Access.AllBranches {
		// Branch-scoped users see only their branches' till activity; C2B rows
		// (branch NULL) stay visible — they still need allocating.
		if len(a.Access.Branches) > 0 {
			f.BranchID = a.Access.Branches[0]
		}
	}
	rows, total, err := h.svc.repo.List(r.Context(), a.TenantID, f, pg.Page, pg.PerPage)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if rows == nil {
		rows = []Transaction{}
	}
	httpx.OK(w, map[string]any{"items": rows, "total": total, "page": pg.Page, "per_page": pg.PerPage})
}

// Get handles GET /t/mpesa/transactions/{id} — detail plus the event timeline.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	t, err := h.svc.repo.ByPublic(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if t == nil {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	events, err := h.svc.repo.Events(r.Context(), a.TenantID, t.ID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var sale any
	if t.AllocatedSaleID != nil {
		if b, ok := h.svc.sales.LookupBrief(r.Context(), a.TenantID, *t.AllocatedSaleID); ok {
			sale = b
		}
	}
	httpx.OK(w, map[string]any{"txn": t, "events": events, "sale": sale})
}

// Summary handles GET /t/mpesa/summary — stat cards and the dashboard widget.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	s, err := h.svc.repo.Summary(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, s)
}

// Suggestions handles GET /t/mpesa/suggestions — candidate allocations.
func (h *Handler) Suggestions(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	out, err := h.svc.Suggestions(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"items": out})
}

// allocateRequest is POST …/allocate|reallocate.
type allocateRequest struct {
	SaleID string `json:"sale_id"` // sale public id
}

// Allocate handles POST /t/mpesa/transactions/{id}/allocate.
func (h *Handler) Allocate(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var in allocateRequest
	if err := httpx.Decode(w, r, &in); err != nil || in.SaleID == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"sale_id": "required"}))
		return
	}
	if err := h.svc.Allocate(r.Context(), a.TenantID, a.ID, chi.URLParam(r, "id"), in.SaleID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"allocated": true})
}

// Reallocate handles POST /t/mpesa/transactions/{id}/reallocate.
func (h *Handler) Reallocate(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var in allocateRequest
	if err := httpx.Decode(w, r, &in); err != nil || in.SaleID == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"sale_id": "required"}))
		return
	}
	if err := h.svc.Reallocate(r.Context(), a.TenantID, a.ID, chi.URLParam(r, "id"), in.SaleID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"reallocated": true})
}

// reverseRequest is POST …/reverse.
type reverseRequest struct {
	Reason string `json:"reason"`
}

// Reverse handles POST /t/mpesa/transactions/{id}/reverse.
func (h *Handler) Reverse(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var in reverseRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.repo.ByPublic(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil || t == nil {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	if err := h.svc.RequestReversal(r.Context(), a.TenantID, t.ID, a.ID, in.Reason); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"reversal_requested": true})
}

// --- public callbacks ---------------------------------------------------------
// These run unauthenticated; the {token} path segment is the shared secret and
// every handler stores the raw body before processing.

func (h *Handler) hookTenant(w http.ResponseWriter, r *http.Request) (uint64, []byte, bool) {
	token := chi.URLParam(r, "token")
	tenantID, err := h.svc.repo.TenantByToken(r.Context(), token)
	if err != nil || tenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrNotFound) // don't confirm token validity
		return 0, nil, false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.WriteError(w, r, httpx.ErrBadRequest)
		return 0, nil, false
	}
	return tenantID, body, true
}

// STKHook handles POST /hooks/mpesa/{token}/stk.
func (h *Handler) STKHook(w http.ResponseWriter, r *http.Request) {
	tenantID, body, ok := h.hookTenant(w, r)
	if !ok {
		return
	}
	h.svc.HandleSTKCallback(r.Context(), tenantID, body)
	httpx.JSON(w, http.StatusOK, daraja.Ack)
}

// C2BConfirmationHook handles POST /hooks/mpesa/{token}/c2b/confirmation.
func (h *Handler) C2BConfirmationHook(w http.ResponseWriter, r *http.Request) {
	tenantID, body, ok := h.hookTenant(w, r)
	if !ok {
		return
	}
	h.svc.HandleC2BConfirmation(r.Context(), tenantID, body)
	httpx.JSON(w, http.StatusOK, daraja.Ack)
}

// C2BValidationHook handles POST /hooks/mpesa/{token}/c2b/validation.
func (h *Handler) C2BValidationHook(w http.ResponseWriter, r *http.Request) {
	tenantID, body, ok := h.hookTenant(w, r)
	if !ok {
		return
	}
	h.svc.HandleC2BValidation(r.Context(), tenantID, body)
	httpx.JSON(w, http.StatusOK, daraja.Ack)
}

// ReversalHook handles POST /hooks/mpesa/{token}/reversal.
func (h *Handler) ReversalHook(w http.ResponseWriter, r *http.Request) {
	tenantID, body, ok := h.hookTenant(w, r)
	if !ok {
		return
	}
	h.svc.HandleReversalResult(r.Context(), tenantID, body)
	httpx.JSON(w, http.StatusOK, daraja.Ack)
}

// BranchTills handles GET /t/mpesa/branch-tills — till / store number → branch.
func (h *Handler) BranchTills(w http.ResponseWriter, r *http.Request) {
	a, ok := identity.ActorFrom(r.Context())
	if !ok || a.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.BranchTills(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rows)
}

// SetBranchTills handles PUT /t/mpesa/branch-tills — replaces the mapping.
func (h *Handler) SetBranchTills(w http.ResponseWriter, r *http.Request) {
	a, ok := identity.ActorFrom(r.Context())
	if !ok || a.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in struct {
		Tills []BranchTillInput `json:"tills"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := h.svc.SetBranchTills(r.Context(), a.TenantID, in.Tills)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rows)
}
