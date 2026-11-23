package branches

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/users"
)

// TenantRoutes wires /t/branches (minimal: list + create for onboarding).
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/branches", func(r chi.Router) {
		r.Get("/", h.List)
		r.With(mw.RequireUserRole(users.UserRoleOwner, users.UserRoleAdmin)).Post("/", h.Create)
	})
}
