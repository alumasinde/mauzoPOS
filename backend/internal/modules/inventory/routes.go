package inventory

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/inventory. Stock changes only ever happen through
// these endpoints → service → ledger; there is no "set balance" API.
func TenantRoutes(r chi.Router, h *Handler, d *DocsHandler) {
	r.Route("/inventory", func(r chi.Router) {
		r.With(mw.Can(PermStockView)).Get("/locations", h.Locations)
		r.With(mw.Can(PermAdjustmentCreate)).Post("/locations", h.CreateLocation)
		r.With(mw.Can(PermAdjustmentCreate)).Delete("/locations/{id}", h.DeleteLocation)

		r.With(mw.Can(PermStockView)).Get("/balances", h.Balances)
		r.With(mw.Can(PermStockView)).Get("/movements", h.Movements)
		r.With(mw.Can(PermStockView)).Get("/batches", h.Batches)
		r.With(mw.Can(PermStockView)).Get("/expiring", h.Expiring)
		r.With(mw.Can(PermStockView)).Get("/valuation", h.Valuations)
		r.With(mw.Can(PermStockView)).Get("/reorders", h.Reorders)
		r.With(mw.Can(PermAdjustmentCreate)).Put("/reorders", h.SetReorder)
		r.With(mw.Can(PermAdjustmentCreate)).Delete("/reorders", h.DeleteReorder)

		r.With(mw.Can(PermAdjustmentCreate)).Post("/receive", h.Receive)
		r.With(mw.Can(PermWastageCreate)).Post("/wastage", h.Wastage)

		r.With(mw.Can(PermStockView)).Get("/adjustments", d.ListAdjustments)
		r.With(mw.Can(PermAdjustmentCreate)).Post("/adjustments", d.CreateAdjustment)
		r.With(mw.Can(PermStockView)).Get("/adjustments/{id}", d.GetAdjustment)
		r.With(mw.Can(PermAdjustmentApprove)).Post("/adjustments/{id}/approve", d.DecideAdjustment(true))
		r.With(mw.Can(PermAdjustmentApprove)).Post("/adjustments/{id}/reject", d.DecideAdjustment(false))
		r.With(mw.Can(PermAdjustmentApprove)).Post("/adjustments/{id}/post", d.PostAdjustment)

		r.With(mw.Can(PermStockView)).Get("/transfers", d.ListTransfers)
		r.With(mw.Can(PermTransferCreate)).Post("/transfers", d.CreateTransfer)
		r.With(mw.Can(PermStockView)).Get("/transfers/{id}", d.GetTransfer)
		r.With(mw.Can(PermTransferCreate)).Post("/transfers/{id}/send", d.SendTransfer)
		r.With(mw.Can(PermTransferReceive)).Post("/transfers/{id}/receive", d.ReceiveTransfer)
		r.With(mw.Can(PermTransferCreate)).Post("/transfers/{id}/cancel", d.CancelTransfer)

		r.With(mw.Can(PermStockView)).Get("/counts", d.ListCounts)
		r.With(mw.Can(PermStocktakeManage)).Post("/counts", d.CreateCount)
		r.With(mw.Can(PermStockView)).Get("/counts/{id}", d.GetCount)
		r.With(mw.Can(PermStocktakeManage)).Put("/counts/{id}/items", d.SetCounted)
		r.With(mw.Can(PermStocktakeManage)).Post("/counts/{id}/post", d.PostCount)
		r.With(mw.Can(PermStocktakeManage)).Post("/counts/{id}/cancel", d.CancelCount)
	})
}
