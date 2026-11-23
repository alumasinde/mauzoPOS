package sales

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/sales.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/sales", func(r chi.Router) {
		r.With(mw.Can(PermSaleView)).Get("/", h.List)
		r.With(mw.Can(PermSaleView)).Get("/summary", h.Summary)
		r.With(mw.Can(PermSaleView)).Get("/daily", h.Daily)
		r.With(mw.Can(PermSaleView)).Get("/methods", h.Methods)
		r.With(mw.Can(PermSaleView)).Get("/held", h.Held)
		r.With(mw.Can(PermSaleView)).Get("/returns", h.ListReturns)
		r.With(mw.Can(PermSaleView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermSaleVoid)).Post("/{id}/void", h.Void)
		r.With(mw.Can(PermRefundCreate)).Post("/{id}/returns", h.CreateReturn)
	})
}
