package shifts

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves the tenant-facing shift endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) (identity.Actor, bool) {
	a, ok := identity.ActorFrom(r.Context())
	return a, ok && a.TenantID != 0 && a.Type == identity.ActorUser
}

// branchScope limits the actor's list to branches they can see (nil = all).
func branchScope(a identity.Actor) []uint64 {
	if a.Access.AllBranches {
		return nil
	}
	return a.Access.Branches
}

// List handles GET /t/shifts.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	f := ListFilter{BranchIDs: branchScope(a)}
	q := r.URL.Query()
	f.Status = q.Get("status")
	if v := q.Get("branch_id"); v != "" {
		id, err := h.svc.BranchIDByPublic(r.Context(), a.TenantID, v)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		// A branch filter outside the actor's scope yields nothing.
		if f.BranchIDs == nil || containsID(f.BranchIDs, id) {
			f.BranchIDs = []uint64{id}
		} else {
			f.BranchIDs = []uint64{0}
		}
	}
	if v := q.Get("since"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.Since = &t
		}
	}
	if v := q.Get("until"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			u := t.Add(24 * time.Hour)
			f.Until = &u
		}
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PerPage, _ = strconv.Atoi(q.Get("per_page"))
	items, total, err := h.svc.List(r.Context(), a.TenantID, f)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"items": items, "total": total, "page": f.Page, "per_page": f.PerPage})
}

func containsID(ids []uint64, id uint64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Detail handles GET /t/shifts/{id} — the shift plus its report.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	rep, err := h.svc.ReportByPublic(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rep)
}

// Summary handles GET /t/shifts/summary — StatCards rollup.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	out, err := h.svc.Summary(r.Context(), a.TenantID, branchScope(a))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, out)
}

// ForceCloseRequest is POST /t/shifts/{id}/force-close.
type ForceCloseRequest struct {
	Note string `json:"note"`
}

// ForceClose ends a stuck open shift without a blind count
// (shifts.shift.close_others — sensitive).
func (h *Handler) ForceClose(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var req ForceCloseRequest
	_ = httpx.Decode(w, r, &req)
	sh, err := h.svc.ForceClose(r.Context(), a.TenantID, chi.URLParam(r, "id"), a.ID, req.Note)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, sh)
}

// ReviewRequest is POST /t/shifts/{id}/review.
type ReviewRequest struct {
	Status string `json:"status"` // approved | flagged | pending
	Note   string `json:"note"`
}

// Review records the variance verdict (shifts.variance.approve — sensitive).
func (h *Handler) Review(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrUnauthorized)
		return
	}
	var req ReviewRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sh, err := h.svc.Review(r.Context(), a.TenantID, chi.URLParam(r, "id"),
		req.Status, req.Note, a.ID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, sh)
}
