package onboarding

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/users"
)

// TenantRoutes wires /t/onboarding. Owner/admin only — the wizard writes
// business-wide configuration.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/onboarding", func(r chi.Router) {
		r.Use(mw.RequireUserRole(users.UserRoleOwner, users.UserRoleAdmin))
		r.Get("/", h.Get)
		r.Put("/business", h.PutBusiness)
		r.Put("/profile", h.PutProfile)
		r.Put("/plan", h.PutPlan)
		r.Put("/modules", h.PutModules)
		r.Post("/branch", h.PostBranch)
		r.Put("/admin", h.PutAdmin)
		r.Put("/preferences", h.PutPreferences)
		r.Post("/terminal", h.PostTerminal)
		r.Post("/complete", h.PostComplete)
	})
}
