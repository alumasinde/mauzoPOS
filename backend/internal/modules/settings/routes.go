package settings

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/users"
)

// TenantRoutes wires /t/settings endpoints; owner/admin until Phase 3 roles.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/settings", func(r chi.Router) {
		r.With(mw.RequireUserRole(users.UserRoleOwner, users.UserRoleAdmin)).Get("/", h.List)
		r.With(mw.RequireUserRole(users.UserRoleOwner, users.UserRoleAdmin)).Put("/{key}", h.Set)
	})
}
