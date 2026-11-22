package tenants

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// Routes wires tenant endpoints onto an authenticated platform router.
func Routes(r chi.Router, h *Handler) {
	r.Route("/tenants", func(r chi.Router) {
		r.With(mw.Can(PermTenantView)).Get("/", h.List)
		r.With(mw.Can(PermTenantCreate)).Post("/", h.Create)
		r.With(mw.Can(PermTenantView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermTenantUpdate)).Patch("/{id}", h.Update)
		r.With(mw.Can(PermTenantSuspend)).Post("/{id}/suspend", h.Suspend)
		r.With(mw.Can(PermTenantSuspend)).Post("/{id}/reactivate", h.Reactivate)
	})
}
