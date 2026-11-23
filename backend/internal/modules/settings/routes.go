package settings

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/settings endpoints.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/settings", func(r chi.Router) {
		r.With(mw.Can(PermBusinessView)).Get("/", h.List)
		r.With(mw.Can(PermBusinessUpdate)).Put("/{key}", h.Set)
	})
}
