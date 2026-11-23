package roles

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/roles and /t/permissions.
func TenantRoutes(r chi.Router, h *Handler) {
	r.With(mw.Can(PermRoleView)).Get("/permissions", h.Permissions)
	r.Route("/roles", func(r chi.Router) {
		r.With(mw.Can(PermRoleView)).Get("/", h.List)
		r.With(mw.Can(PermRoleView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermRoleManage)).Post("/", h.Create)
		r.With(mw.Can(PermRoleManage)).Patch("/{id}", h.Update)
		r.With(mw.Can(PermRoleManage)).Delete("/{id}", h.Delete)
		r.With(mw.Can(PermRoleManage)).Post("/{id}/reset", h.Reset)
	})
}
