package audit

import (
	"net/http"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler exposes audit-log list endpoints.
type Handler struct {
	repo *Repo
}

// NewHandler builds a Handler.
func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

// ListPlatform handles GET /platform/audit-logs (all tenants + platform events).
func (h *Handler) ListPlatform(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, nil)
}

// ListTenant handles GET /t/audit-logs (own tenant only).
func (h *Handler) ListTenant(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	h.list(w, r, &actor.TenantID)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, tenantID *uint64) {
	p := httpx.ParsePage(r)
	rows, total, err := h.repo.List(r.Context(), tenantID, r.URL.Query().Get("actor_type"), p)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]Response, len(rows))
	for i, l := range rows {
		out[i] = toResponse(l)
	}
	httpx.List(w, out, p.Meta(total))
}
