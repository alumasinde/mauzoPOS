package receipts

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// Permissions owned elsewhere that guard receipt views.
const (
	permSaleView = "sales.sale.view"
	permReprint  = "pos.receipt.reprint"
)

// TenantRoutes wires /t/receipts. The designer needs the receipts module;
// viewing and reprinting a sale's receipt follow the sales permissions,
// since every shop prints receipts whether or not it customises them.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/receipts", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(mw.Can(PermTemplateManage))
			r.Get("/meta", h.Meta)
			r.Get("/templates", h.List)
			r.Post("/templates", h.Create)
			r.Get("/templates/{id}", h.Get)
			r.Put("/templates/{id}", h.Update)
			r.Delete("/templates/{id}", h.Archive)
			r.Get("/templates/{id}/versions", h.Versions)
			r.Get("/assignments", h.Assignments)
			r.Put("/assignments", h.Assign)
			r.Post("/preview", h.Preview)
			r.Put("/logo", h.UploadLogo)
			r.Delete("/logo", h.DeleteLogo)
		})
		r.With(mw.Can(permSaleView)).Get("/logo", h.LogoPNG)
		r.With(mw.Can(permSaleView)).Get("/sales/{id}", h.SaleReceipt)
		r.With(mw.Can(permSaleView)).Get("/sales/{id}/invoice", h.Invoice)
		r.With(mw.Can(permReprint)).Post("/sales/{id}/reprint", h.Reprint)
	})
}
