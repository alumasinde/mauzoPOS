package inventory

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/inventory stock endpoints (locations, balances,
// movements, receive/wastage, batches, reorders, valuation).
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func actor(r *http.Request) (identity.Actor, bool) {
	return identity.ActorFrom(r.Context())
}

// ---- locations ---------------------------------------------------------------

type locationIn struct {
	BranchID string `json:"branch_id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// Locations handles GET /t/inventory/locations?branch_id=.
func (h *Handler) Locations(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Locations(r.Context(), a, r.URL.Query().Get("branch_id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]LocationResponse, len(rows))
	for i, l := range rows {
		out[i] = locResponse(l)
	}
	httpx.OK(w, out)
}

// CreateLocation handles POST /t/inventory/locations.
func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in locationIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	l, err := h.svc.CreateLocation(r.Context(), a, in.BranchID, in.Name, in.Kind)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, locResponse(l))
}

// DeleteLocation handles DELETE /t/inventory/locations/{id}?branch_id=.
func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.svc.DeleteLocation(r.Context(), a,
		r.URL.Query().Get("branch_id"), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ---- stock reads ----------------------------------------------------------------

// Balances handles GET /t/inventory/balances.
func (h *Handler) Balances(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	f := BalanceFilter{LowStock: r.URL.Query().Get("low_stock") == "1"}
	var err error
	if f.BranchID, err = h.branchID(r, a); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if loc := r.URL.Query().Get("location_id"); loc != "" {
		l, err := h.svc.repo.LocationByPublicID(r.Context(), a.TenantID, f.BranchID, loc)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		f.LocationID = l.ID
	}
	if prod := r.URL.Query().Get("product_id"); prod != "" {
		p, err := h.svc.products.Get(r.Context(), a.TenantID, prod)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		f.ProductID = p.ID
	}
	rows, total, err := h.svc.Balances(r.Context(), a, f, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]BalanceResponse, len(rows))
	for i, b := range rows {
		out[i] = balResponse(b)
	}
	httpx.List(w, out, page.Meta(total))
}

// Movements handles GET /t/inventory/movements.
func (h *Handler) Movements(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	f := MovementFilter{Kind: r.URL.Query().Get("kind")}
	var err error
	if f.BranchID, err = h.branchID(r, a); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if prod := r.URL.Query().Get("product_id"); prod != "" {
		p, err := h.svc.products.Get(r.Context(), a.TenantID, prod)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		f.ProductID = p.ID
	}
	rows, total, err := h.svc.Movements(r.Context(), a, f, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]MovementResponse, len(rows))
	for i, m := range rows {
		out[i] = movResponse(m)
	}
	httpx.List(w, out, page.Meta(total))
}

// ---- stock ops -------------------------------------------------------------------

type stockOpIn struct {
	BranchID string        `json:"branch_id"`
	Reason   string        `json:"reason"` // wastage only
	Note     string        `json:"note"`
	Items    []LineRequest `json:"items"`
}

// Receive handles POST /t/inventory/receive (goods in, optional cost/batch).
func (h *Handler) Receive(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in stockOpIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.Receive(r.Context(), a, in.BranchID, in.Items, in.Note); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, map[string]string{"status": "posted"})
}

// Wastage handles POST /t/inventory/wastage.
func (h *Handler) Wastage(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in stockOpIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.Wastage(r.Context(), a, in.BranchID, in.Reason, in.Items, in.Note); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, map[string]string{"status": "posted"})
}

// ---- batches, alerts, valuation ---------------------------------------------------

// Batches handles GET /t/inventory/batches?product_id=.
func (h *Handler) Batches(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	p, err := h.svc.products.Get(r.Context(), a.TenantID, r.URL.Query().Get("product_id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := h.svc.Batches(r.Context(), a.TenantID, p.ID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]BatchResponse, len(rows))
	for i, b := range rows {
		out[i] = BatchResponse{ID: b.ID, BatchNo: b.BatchNo, ExpiryDate: b.ExpiryDate, ReceivedAt: b.ReceivedAt}
	}
	httpx.OK(w, out)
}

// Expiring handles GET /t/inventory/expiring?days=.
func (h *Handler) Expiring(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30
	}
	rows, err := h.svc.Expiring(r.Context(), a.TenantID, days)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ExpiringResponse, len(rows))
	for i, x := range rows {
		out[i] = ExpiringResponse{
			BatchNo: x.BatchNo, ProductName: x.ProductName, Branch: x.BranchName,
			Qty: x.Qty.String(), ExpiryDate: x.ExpiryDate,
		}
	}
	httpx.OK(w, out)
}

// Valuations handles GET /t/inventory/valuation.
func (h *Handler) Valuations(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Valuations(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ValuationResponse, len(rows))
	for i, v := range rows {
		out[i] = ValuationResponse{
			BranchID: v.BranchID, Branch: v.BranchName, SKUs: v.SkuCount,
			Units: v.Units.String(), Value: v.Value.String(),
		}
	}
	httpx.OK(w, out)
}

// ---- reorder levels -----------------------------------------------------------------

type reorderIn struct {
	BranchID  string `json:"branch_id"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	MinQty    string `json:"min_qty"`
	TargetQty string `json:"target_qty"`
}

// Reorders handles GET /t/inventory/reorders?branch_id=.
func (h *Handler) Reorders(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	branchID, err := h.branchID(r, a)
	if err != nil || branchID == 0 {
		httpx.WriteError(w, r, errOrRequired(err, "branch_id"))
		return
	}
	rows, err := h.svc.Reorders(r.Context(), a.TenantID, branchID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ReorderResponse, len(rows))
	for i, ro := range rows {
		out[i] = reorderResponse(ro)
	}
	httpx.OK(w, out)
}

func reorderResponse(ro ReorderRow) ReorderResponse {
	return ReorderResponse{
		ProductID: ro.ProductPublicID, ProductName: ro.ProductName, SKU: ro.ProductSKU,
		VariantID: ro.VariantPublicID, VariantName: ro.VariantName,
		MinQty: ro.MinQty.String(), TargetQty: ro.TargetQty.String(), Qty: ro.Qty.String(),
	}
}

// SetReorder handles PUT /t/inventory/reorders.
func (h *Handler) SetReorder(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in reorderIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	branchID, err := h.svc.branch(r.Context(), a, in.BranchID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.svc.products.Get(r.Context(), a.TenantID, in.ProductID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rl := ReorderLevel{BranchID: branchID, ProductID: p.ID}
	for _, v := range p.Variants {
		if v.PublicID == in.VariantID {
			rl.VariantID = v.ID
		}
	}
	if in.VariantID != "" && rl.VariantID == 0 {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"variant_id": "invalid"}))
		return
	}
	var errV error
	if rl.MinQty, errV = decimal.NewFromString(in.MinQty); errV != nil {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
			map[string]any{"min_qty": "invalid"}))
		return
	}
	rl.TargetQty, _ = decimal.NewFromString(in.TargetQty)
	if err := h.svc.SetReorder(r.Context(), a.TenantID, rl); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// DeleteReorder handles DELETE /t/inventory/reorders?branch_id=&product_id=&variant_id=.
func (h *Handler) DeleteReorder(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	q := r.URL.Query()
	branchID, err := h.svc.branch(r.Context(), a, q.Get("branch_id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.svc.products.Get(r.Context(), a.TenantID, q.Get("product_id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var variantID uint64
	for _, v := range p.Variants {
		if v.PublicID == q.Get("variant_id") {
			variantID = v.ID
		}
	}
	if err := h.svc.DeleteReorder(r.Context(), a.TenantID, branchID, p.ID, variantID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ---- helpers --------------------------------------------------------------------

// branchID resolves the optional ?branch_id= filter (empty = all branches).
func (h *Handler) branchID(r *http.Request, a identity.Actor) (uint64, error) {
	pub := r.URL.Query().Get("branch_id")
	if pub == "" {
		return 0, nil
	}
	return h.svc.branch(r.Context(), a, pub)
}

func errOrRequired(err error, field string) error {
	if err != nil {
		return err
	}
	return httpx.ErrValidation.WithDetails(map[string]any{field: "required"})
}
