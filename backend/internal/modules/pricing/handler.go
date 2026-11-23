package pricing

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/pricing (lists, resolve, history) and /t/promotions.
type Handler struct {
	svc      *Service
	promos   *PromoService
	branches Branches
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, promos *PromoService, branches Branches) *Handler {
	return &Handler{svc: svc, promos: promos, branches: branches}
}

func actorOf(r *http.Request) (identity.Actor, bool) {
	return identity.ActorFrom(r.Context())
}

// ---- price lists ----------------------------------------------------------------

// Lists handles GET /t/pricing/lists.
func (h *Handler) Lists(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Lists(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ListResponse, len(rows))
	for i, l := range rows {
		out[i] = listResponse(l)
	}
	httpx.OK(w, out)
}

// CreateList handles POST /t/pricing/lists.
func (h *Handler) CreateList(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in ListRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	l, err := h.svc.CreateList(r.Context(), a, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, listResponse(l))
}

// GetList handles GET /t/pricing/lists/{id}.
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	l, err := h.svc.GetList(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, listResponse(l))
}

// UpdateList handles PATCH /t/pricing/lists/{id}.
func (h *Handler) UpdateList(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in ListRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	l, err := h.svc.UpdateList(r.Context(), a, chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, listResponse(l))
}

// DeleteList handles DELETE /t/pricing/lists/{id}.
func (h *Handler) DeleteList(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.svc.DeleteList(r.Context(), a.TenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ---- quote + history -------------------------------------------------------------

// Quote handles POST /t/pricing/quote — resolves list prices and applies
// live promotions to a basket.
func (h *Handler) Quote(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in QuoteRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := Quote(r.Context(), a, h.svc, h.promos, h.branches, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, out)
}

// History handles GET /t/pricing/history?product_id=.
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	rows, total, err := h.svc.History(r.Context(), a.TenantID,
		r.URL.Query().Get("product_id"), page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]HistoryResponse, len(rows))
	for i, row := range rows {
		out[i] = historyResponse(row)
	}
	httpx.List(w, out, page.Meta(total))
}

// ---- promotions -----------------------------------------------------------------

// Promotions handles GET /t/pricing/promotions.
func (h *Handler) Promotions(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.promos.Promotions(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]PromotionResponse, len(rows))
	for i, p := range rows {
		out[i] = promoResponse(p)
	}
	httpx.OK(w, out)
}

// CreatePromotion handles POST /t/pricing/promotions.
func (h *Handler) CreatePromotion(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in PromotionRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.promos.CreatePromotion(r.Context(), a, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, promoResponse(p))
}

// GetPromotion handles GET /t/pricing/promotions/{id}.
func (h *Handler) GetPromotion(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	p, err := h.promos.GetPromotion(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, promoResponse(p))
}

// UpdatePromotion handles PATCH /t/pricing/promotions/{id}.
func (h *Handler) UpdatePromotion(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in PromotionRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.promos.UpdatePromotion(r.Context(), a, chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, promoResponse(p))
}

// DeletePromotion handles DELETE /t/pricing/promotions/{id}.
func (h *Handler) DeletePromotion(w http.ResponseWriter, r *http.Request) {
	a, ok := actorOf(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.promos.DeletePromotion(r.Context(), a.TenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}
