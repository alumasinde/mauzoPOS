package terminals

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/terminals endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List handles GET /t/terminals.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.List(r.Context(), actor.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]Response, 0, len(rows))
	for _, t := range rows {
		if actor.CanAccessBranch(t.BranchID) {
			out = append(out, ToResponse(t))
		}
	}
	httpx.OK(w, out)
}

// Create handles POST /t/terminals.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req CreateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.CreateAt(r.Context(), actor.TenantID, req, actor.CanAccessBranch)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToResponse(t))
}

// IssueCode handles POST /t/terminals/{id}/code — mints an activation code.
func (h *Handler) IssueCode(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	code, err := h.svc.IssueCode(r.Context(), actor.TenantID, chi.URLParam(r, "id"), actor.CanAccessBranch)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, code)
}

// Update handles PATCH /t/terminals/{id} — rename, disable, enable.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req UpdateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Update(r.Context(), actor.TenantID, chi.URLParam(r, "id"), req, actor.CanAccessBranch)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToResponse(t))
}
