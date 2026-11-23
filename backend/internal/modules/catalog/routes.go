package catalog

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/catalog. Every route requires the catalog module via
// its permission (mw.Can rejects with module_disabled when it's off).
func TenantRoutes(r chi.Router, h *Handler, th *TaxonomyHandler) {
	r.Route("/catalog", func(r chi.Router) {
		r.With(mw.Can(PermProductView)).Get("/products", h.List)
		r.With(mw.Can(PermProductCreate)).Post("/products", h.Create)
		r.With(mw.Can(PermProductImport)).Post("/products/import", h.Import)
		r.With(mw.Can(PermProductExport)).Get("/products/export", h.Export)
		r.With(mw.Can(PermProductView)).Get("/products/{id}", h.Get)
		r.With(mw.Can(PermProductUpdate)).Patch("/products/{id}", h.Update)
		r.With(mw.Can(PermProductDelete)).Delete("/products/{id}", h.Archive)
		r.With(mw.Can(PermProductUpdate)).Post("/products/{id}/images", h.UploadImage)
		r.With(mw.Can(PermProductView)).Get("/images/{id}", h.ServeImage)
		r.With(mw.Can(PermProductUpdate)).Delete("/images/{id}", h.DeleteImage)
		r.With(mw.Can(PermProductView)).Get("/lookup", h.Lookup)
		r.With(mw.Can(PermProductView)).Get("/labels", h.Labels)

		r.With(mw.Can(PermProductView)).Get("/categories", th.ListCategories)
		r.With(mw.Can(PermTaxonomy)).Post("/categories", th.CreateCategory)
		r.With(mw.Can(PermTaxonomy)).Patch("/categories/{id}", th.UpdateCategory)
		r.With(mw.Can(PermTaxonomy)).Delete("/categories/{id}", th.DeleteCategory)
		r.With(mw.Can(PermProductView)).Get("/brands", th.ListBrands)
		r.With(mw.Can(PermTaxonomy)).Post("/brands", th.CreateBrand)
		r.With(mw.Can(PermTaxonomy)).Patch("/brands/{id}", th.UpdateBrand)
		r.With(mw.Can(PermTaxonomy)).Delete("/brands/{id}", th.DeleteBrand)
		r.With(mw.Can(PermProductView)).Get("/units", th.ListUnits)
		r.With(mw.Can(PermTaxonomy)).Post("/units", th.CreateUnit)
	})
}
