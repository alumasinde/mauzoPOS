package tenants

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
)

// Handler exposes tenant endpoints in the platform context.
type Handler struct {
	svc   *Service
	audit *audit.Writer
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, audit *audit.Writer) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// List handles GET /platform/tenants.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePage(r)
	ts, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]TenantResponse, len(ts))
	for i, t := range ts {
		out[i] = toResponse(t)
	}
	httpx.List(w, out, p.Meta(total))
}

// Create handles POST /platform/tenants.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action:     "platform.tenant.create",
		EntityType: "tenant",
		EntityID:   t.PublicID,
		Data:       map[string]any{"name": t.Name, "slug": t.Slug},
	}))
	httpx.Created(w, toResponse(t))
}

// Get handles GET /platform/tenants/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toResponse(t))
}

// Update handles PATCH /platform/tenants/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateTenantRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.tenant.update", EntityType: "tenant", EntityID: t.PublicID,
	}))
	httpx.OK(w, toResponse(t))
}

// Suspend handles POST /platform/tenants/{id}/suspend.
func (h *Handler) Suspend(w http.ResponseWriter, r *http.Request) {
	var req SuspendRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Suspend(r.Context(), chi.URLParam(r, "id"), req.Reason)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action:     "platform.tenant.suspend",
		EntityType: "tenant",
		EntityID:   t.PublicID,
		Data:       map[string]any{"reason": req.Reason},
	}))
	httpx.OK(w, toResponse(t))
}

// Reactivate handles POST /platform/tenants/{id}/reactivate.
func (h *Handler) Reactivate(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Reactivate(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.tenant.reactivate", EntityType: "tenant", EntityID: t.PublicID,
	}))
	httpx.OK(w, toResponse(t))
}
