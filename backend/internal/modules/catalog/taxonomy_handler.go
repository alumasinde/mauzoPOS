package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// TaxonomyHandler serves /t/catalog/{categories,brands,units}.
type TaxonomyHandler struct {
	svc *TaxonomyService
}

// NewTaxonomyHandler builds a TaxonomyHandler.
func NewTaxonomyHandler(svc *TaxonomyService) *TaxonomyHandler {
	return &TaxonomyHandler{svc: svc}
}

func tenantOf(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	a, ok := identity.ActorFrom(r.Context())
	if !ok || a.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return 0, false
	}
	return a.TenantID, true
}

// ListCategories handles GET /t/catalog/categories. Public ids on parents are
// resolved into parent_id so the UI can build the tree.
func (h *TaxonomyHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.ListCategories(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pubs := make(map[uint64]string, len(rows))
	for _, c := range rows {
		pubs[c.ID] = c.PublicID
	}
	out := make([]CategoryResponse, len(rows))
	for i, c := range rows {
		out[i] = ToCategoryResponse(c, pubs)
	}
	httpx.OK(w, out)
}

// CreateCategory handles POST /t/catalog/categories.
func (h *TaxonomyHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req CategoryRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), tenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pubs := map[uint64]string{c.ID: c.PublicID}
	if c.ParentID != nil && req.ParentID != nil {
		pubs[*c.ParentID] = *req.ParentID
	}
	httpx.Created(w, ToCategoryResponse(c, pubs))
}

// UpdateCategory handles PATCH /t/catalog/categories/{id}.
func (h *TaxonomyHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req CategoryRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.UpdateCategory(r.Context(), tenantID, chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pubs := map[uint64]string{c.ID: c.PublicID}
	if c.ParentID != nil {
		if pub, err := h.svc.CategoryPublicID(r.Context(), tenantID, *c.ParentID); err == nil {
			pubs[*c.ParentID] = pub
		}
	}
	httpx.OK(w, ToCategoryResponse(c, pubs))
}

// DeleteCategory handles DELETE /t/catalog/categories/{id}.
func (h *TaxonomyHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteCategory(r.Context(), tenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ListBrands handles GET /t/catalog/brands.
func (h *TaxonomyHandler) ListBrands(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.ListBrands(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]BrandResponse, len(rows))
	for i, b := range rows {
		out[i] = ToBrandResponse(b)
	}
	httpx.OK(w, out)
}

// CreateBrand handles POST /t/catalog/brands.
func (h *TaxonomyHandler) CreateBrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req BrandRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	b, err := h.svc.CreateBrand(r.Context(), tenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToBrandResponse(b))
}

// UpdateBrand handles PATCH /t/catalog/brands/{id}.
func (h *TaxonomyHandler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req BrandRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	b, err := h.svc.UpdateBrand(r.Context(), tenantID, chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToBrandResponse(b))
}

// DeleteBrand handles DELETE /t/catalog/brands/{id}.
func (h *TaxonomyHandler) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteBrand(r.Context(), tenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ListUnits handles GET /t/catalog/units.
func (h *TaxonomyHandler) ListUnits(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.ListUnits(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bases := make(map[uint64]string, len(rows))
	for _, u := range rows {
		bases[u.ID] = u.Code
	}
	out := make([]UnitResponse, len(rows))
	for i, u := range rows {
		out[i] = ToUnitResponse(u, bases)
	}
	httpx.OK(w, out)
}

// CreateUnit handles POST /t/catalog/units.
func (h *TaxonomyHandler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req UnitRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, err := h.svc.CreateUnit(r.Context(), tenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToUnitResponse(u, map[uint64]string{}))
}
