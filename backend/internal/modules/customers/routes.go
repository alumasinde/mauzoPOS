package customers

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/customers.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/customers", func(r chi.Router) {
		r.With(mw.Can(PermCustomerView)).Get("/", h.List)
		r.With(mw.Can(PermCustomerView)).Get("/{id}", h.Get)
		r.With(mw.Can(PermCustomerManage)).Post("/", h.Create)
		r.With(mw.Can(PermCustomerManage)).Patch("/{id}", h.Update)
	})
}
