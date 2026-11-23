package users

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// PlatformRoutes wires platform user endpoints onto the authenticated
// platform router.
func PlatformRoutes(r chi.Router, h *PlatformUserHandler) {
	r.Route("/users", func(r chi.Router) {
		r.With(mw.Can(PermPlatformUserView)).Get("/", h.List)
		r.With(mw.Can(PermPlatformUserManage)).Post("/", h.Create)
		r.With(mw.Can(PermPlatformUserView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermPlatformUserManage)).Patch("/{id}", h.Update)
		r.With(mw.Can(PermPlatformUserManage)).Post("/{id}/disable", h.Disable)
		r.With(mw.Can(PermPlatformUserManage)).Post("/{id}/enable", h.Enable)
		r.With(mw.Can(PermPlatformUserManage)).Post("/{id}/password", h.SetPassword)
	})
}

// TenantRoutes wires /t/employees.
func TenantRoutes(r chi.Router, h *EmployeeHandler) {
	r.Route("/employees", func(r chi.Router) {
		r.With(mw.Can(PermEmployeeView)).Get("/", h.List)
		r.With(mw.Can(PermEmployeeView)).Get("/assignable-roles", h.AssignableRoles)
		r.With(mw.Can(PermEmployeeView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermEmployeeCreate)).Post("/", h.Create)
		r.With(mw.Can(PermEmployeeUpdate)).Patch("/{id}", h.Update)
		r.With(mw.Can(PermEmployeeDeactivate)).Post("/{id}/disable", h.Disable)
		r.With(mw.Can(PermEmployeeDeactivate)).Post("/{id}/enable", h.Enable)
		r.With(mw.Can(PermEmployeeResetPassword)).Post("/{id}/password", h.SetPassword)
		r.With(mw.Can(PermEmployeeResetPIN)).Post("/{id}/pin", h.SetPIN)
	})
}
