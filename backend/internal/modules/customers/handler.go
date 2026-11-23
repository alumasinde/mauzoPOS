package customers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/customers.
type Handler struct{ svc *Service }

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) (identity.Actor, bool) {
	a, ok := identity.ActorFrom(r.Context())
	return a, ok && a.TenantID != 0
}

// List handles GET /t/customers?search=&all=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.List(r.Context(), actor.TenantID, r.URL.Query().Get("search"), r.URL.Query().Get("all") == "")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]Response, 0, len(rows))
	for _, c := range rows {
		out = append(out, ToResponse(c))
	}
	httpx.OK(w, out)
}

// Get handles GET /t/customers/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	c, err := h.svc.Get(r.Context(), actor.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToResponse(c))
}

// Create handles POST /t/customers.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req Request
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.Create(r.Context(), actor.TenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToResponse(c))
}

// Update handles PATCH /t/customers/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var req struct {
		Request
		Archive *bool `json:"archive"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.Update(r.Context(), actor.TenantID, chi.URLParam(r, "id"), req.Request, req.Archive)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToResponse(c))
}
