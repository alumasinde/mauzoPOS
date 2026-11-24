package shifts

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes mounts the back-office shift endpoints.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/shifts", func(r chi.Router) {
		r.With(mw.Can(PermShiftView)).Get("/", h.List)
		r.With(mw.Can(PermShiftView)).Get("/summary", h.Summary)
		r.With(mw.Can(PermShiftView)).Get("/{id}", h.Detail)
		r.With(mw.Can(PermShiftCloseOther)).Post("/{id}/force-close", h.ForceClose)
		r.With(mw.Can(PermVarianceApprove)).Post("/{id}/review", h.Review)
	})
}
