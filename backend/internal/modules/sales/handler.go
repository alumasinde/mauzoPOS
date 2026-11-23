package sales

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// BranchResolver converts branch public ids (like terminals.BranchResolver).
type BranchResolver interface {
	BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

// Handler serves /t/sales.
type Handler struct {
	svc      *Service
	branches BranchResolver
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, br BranchResolver) *Handler { return &Handler{svc: svc, branches: br} }

func actorOf(r *http.Request) (identity.Actor, bool) {
	a, ok := identity.ActorFrom(r.Context())
	return a, ok && a.TenantID != 0
}

// branchScope returns the branch restriction for list/summary filters: nil =
// unrestricted (all-branches user or view_all_branches).
func branchScope(a identity.Actor) []uint64 {
	if a.Access.AllBranches || a.Can(PermSaleViewAllBranches) {
		return nil
	}
	return a.Access.Branches
}

func parseFilter(r *http.Request, actor identity.Actor, br BranchResolver) (ListFilter, error) {
	f := ListFilter{BranchIDs: branchScope(actor)}
	q := r.URL.Query()
	f.Status, f.Method, f.Search = q.Get("status"), q.Get("method"), q.Get("search")
	if bp := q.Get("branch_id"); bp != "" {
		id, err := br.BranchByPublicID(r.Context(), actor.TenantID, bp)
		if err != nil {
			return f, httpx.ErrValidation.WithDetails(map[string]any{"branch_id": "invalid"})
		}
		if len(f.BranchIDs) > 0 && !actor.CanAccessBranch(id) {
			return f, httpx.ErrForbidden
		}
		f.BranchID = id
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, httpx.ErrValidation.WithDetails(map[string]any{"from": "use YYYY-MM-DD"})
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, httpx.ErrValidation.WithDetails(map[string]any{"to": "use YYYY-MM-DD"})
		}
		end := t.Add(24*time.Hour - time.Nanosecond)
		f.To = &end
	}
	return f, nil
}

// List handles GET /t/sales.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	f, err := parseFilter(r, actor, h.branches)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pg := httpx.ParsePage(r)
	rows, total, err := h.svc.List(r.Context(), actor.TenantID, f, pg)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]SaleResponse, 0, len(rows))
	for _, s := range rows {
		out = append(out, ToResponse(s))
	}
	httpx.List(w, out, pg.Meta(total))
}

// Get handles GET /t/sales/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	d, err := h.svc.Get(r.Context(), actor.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(branchScope(actor)) > 0 && !actor.CanAccessBranch(d.Sale.BranchID) {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	httpx.OK(w, ToDetail(d))
}

// Summary handles GET /t/sales/summary.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	f, err := parseFilter(r, actor, h.branches)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sum, err := h.svc.Summarize(r.Context(), actor.TenantID, f)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, sum)
}

// Daily handles GET /t/sales/daily — per-day totals for charts/KPI trends.
func (h *Handler) Daily(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	f, err := parseFilter(r, actor, h.branches)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := h.svc.Daily(r.Context(), actor.TenantID, f)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rows)
}

// Methods handles GET /t/sales/methods — registry + per-method totals.
func (h *Handler) Methods(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	f, err := parseFilter(r, actor, h.branches)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	split, err := h.svc.MethodTotals(r.Context(), actor.TenantID, f)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"methods": Methods(), "totals": split})
}

// Held handles GET /t/sales/held.
func (h *Handler) Held(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Held(r.Context(), actor.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]HeldResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, ToHeldResponse(c))
	}
	httpx.OK(w, out)
}

// ListReturns handles GET /t/sales/returns.
func (h *Handler) ListReturns(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	pg := httpx.ParsePage(r)
	rows, total, err := h.svc.ListReturns(r.Context(), actor.TenantID, pg)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ReturnResponse, 0, len(rows))
	for _, rt := range rows {
		out = append(out, ToReturnResponse(rt))
	}
	httpx.List(w, out, pg.Meta(total))
}

// Void handles POST /t/sales/{id}/void.
func (h *Handler) Void(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.Reason == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(map[string]any{"reason": "required"}))
		return
	}
	d, err := h.svc.Get(r.Context(), actor.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !actor.CanAccessBranch(d.Sale.BranchID) {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.svc.Void(r.Context(), actor.TenantID, chi.URLParam(r, "id"), req.Reason,
		actor.ID, string(actor.Type)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]string{"status": StatusVoided})
}

// CreateReturn handles POST /t/sales/{id}/returns.
func (h *Handler) CreateReturn(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req struct {
		Reason       string            `json:"reason"`
		RefundMethod string            `json:"refund_method"`
		Lines        []ReturnLineInput `json:"lines"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	d, err := h.svc.Get(r.Context(), actor.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !actor.CanAccessBranch(d.Sale.BranchID) {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rt, err := h.svc.CreateReturn(r.Context(), ReturnInput{
		TenantID: actor.TenantID, SalePublicID: chi.URLParam(r, "id"),
		Reason: req.Reason, RefundMethod: req.RefundMethod, Lines: req.Lines,
		ActorID: actor.ID, ActorType: string(actor.Type),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToReturnResponse(rt))
}
