package subscriptions

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// PlatformRoutes wires plan endpoints.
func PlatformRoutes(r chi.Router, h *Handler) {
	r.Route("/plans", func(r chi.Router) {
		r.With(mw.Can(PermSubscriptionView)).Get("/", h.List)
		r.With(mw.Can(PermSubscriptionManage)).Post("/", h.Create)
		r.With(mw.Can(PermSubscriptionManage)).Patch("/{id}", h.Update)
	})
}
