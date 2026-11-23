package branches

import (
	"net/http"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/branches endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List handles GET /t/branches.
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
	out := make([]Response, len(rows))
	for i, b := range rows {
		out[i] = ToResponse(b)
	}
	httpx.OK(w, out)
}

// Create handles POST /t/branches.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req CreateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	b, err := h.svc.Create(r.Context(), actor.TenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToResponse(b))
}
