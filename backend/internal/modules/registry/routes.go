package registry

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/tenants"
)

// TenantRoutes wires /t/modules and /t/navigation.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Get("/navigation", h.Navigation)
	r.Route("/modules", func(r chi.Router) {
		r.Use(mw.Can(PermTenantModuleManage))
		r.Get("/", h.List)
		r.Put("/{code}", h.Toggle)
		r.Put("/{code}/config", h.SetConfig)
	})
}

// PlatformRoutes wires /platform/modules and tenant overrides.
func PlatformRoutes(r chi.Router, h *Handler) {
	r.Route("/modules", func(r chi.Router) {
		r.With(mw.Can(PermPlatformModuleView)).Get("/", h.PlatformList)
		r.With(mw.Can(PermPlatformModuleManage)).Patch("/{code}", h.PlatformSetActive)
	})
	r.Route("/tenants/{id}/modules", func(r chi.Router) {
		r.With(mw.Can(tenants.PermTenantView), mw.Can(PermPlatformModuleView)).Get("/", h.PlatformTenantList)
		r.With(mw.Can(PermPlatformModuleManage)).Put("/{code}", h.PlatformTenantToggle)
	})
}
