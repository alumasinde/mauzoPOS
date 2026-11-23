package businesstypes

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// PlatformRoutes wires business-type and profile-template endpoints.
func PlatformRoutes(r chi.Router, h *Handler) {
	r.With(mw.Can(PermBusinessTypeView)).Get("/business-types", h.ListTypes)
	r.Route("/business-profiles", func(r chi.Router) {
		r.With(mw.Can(PermBusinessTypeView)).Get("/", h.ListProfiles)
		r.With(mw.Can(PermBusinessTypeManage)).Post("/", h.CreateProfile)
		r.With(mw.Can(PermBusinessTypeManage)).Patch("/{id}", h.UpdateProfile)
	})
}
