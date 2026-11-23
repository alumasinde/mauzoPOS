package pricing

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/pricing. Promotion endpoints carry the promotions
// module via their permission — they 403 when the module is off even if
// pricing itself is enabled.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/pricing", func(r chi.Router) {
		r.With(mw.Can(PermPriceView)).Get("/lists", h.Lists)
		r.With(mw.Can(PermPriceUpdate)).Post("/lists", h.CreateList)
		r.With(mw.Can(PermPriceView)).Get("/lists/{id}", h.GetList)
		r.With(mw.Can(PermPriceUpdate)).Patch("/lists/{id}", h.UpdateList)
		r.With(mw.Can(PermPriceUpdate)).Delete("/lists/{id}", h.DeleteList)
		r.With(mw.Can(PermPriceView)).Post("/quote", h.Quote)
		r.With(mw.Can(PermPriceView)).Get("/history", h.History)
	})
	r.Route("/promotions", func(r chi.Router) {
		r.With(mw.Can(PermPromoManage)).Get("/", h.Promotions)
		r.With(mw.Can(PermPromoManage)).Post("/", h.CreatePromotion)
		r.With(mw.Can(PermPromoManage)).Get("/{id}", h.GetPromotion)
		r.With(mw.Can(PermPromoManage)).Patch("/{id}", h.UpdatePromotion)
		r.With(mw.Can(PermPromoManage)).Delete("/{id}", h.DeletePromotion)
	})
}
