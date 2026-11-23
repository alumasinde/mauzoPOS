package onboarding

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/registry"
	"mauzo/internal/modules/settings"
	"mauzo/internal/modules/users"
)

// TenantRoutes wires /t/onboarding. The wizard writes business-wide
// configuration, so every step needs settings.business.update; the module and
// employee steps also need their own permissions.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/onboarding", func(r chi.Router) {
		r.Use(mw.Can(settings.PermBusinessUpdate))
		r.Get("/", h.Get)
		r.Put("/business", h.PutBusiness)
		r.Put("/profile", h.PutProfile)
		r.Put("/plan", h.PutPlan)
		r.With(mw.Can(registry.PermTenantModuleManage)).Put("/modules", h.PutModules)
		r.Post("/branch", h.PostBranch)
		r.With(mw.Can(users.PermEmployeeCreate)).Put("/admin", h.PutAdmin)
		r.Put("/preferences", h.PutPreferences)
		r.Post("/terminal", h.PostTerminal)
		r.Post("/complete", h.PostComplete)
	})
}
