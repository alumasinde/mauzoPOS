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
