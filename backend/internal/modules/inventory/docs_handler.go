package inventory

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
)

// DocsHandler serves inventory document endpoints (adjustments, transfers,
// stocktakes).
type DocsHandler struct {
	svc *Service
}

// NewDocsHandler builds a DocsHandler.
func NewDocsHandler(svc *Service) *DocsHandler {
	return &DocsHandler{svc: svc}
}

// ---- adjustments ----------------------------------------------------------------

// ListAdjustments handles GET /t/inventory/adjustments?branch_id=&status=.
func (h *DocsHandler) ListAdjustments(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	rows, total, err := h.svc.Adjustments(r.Context(), a,
		r.URL.Query().Get("branch_id"), r.URL.Query().Get("status"), page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]AdjustmentResponse, len(rows))
	for i, row := range rows {
		out[i] = adjResponse(row)
	}
	httpx.List(w, out, page.Meta(total))
}

// CreateAdjustment handles POST /t/inventory/adjustments.
func (h *DocsHandler) CreateAdjustment(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in AdjustmentRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := h.svc.CreateAdjustment(r.Context(), a, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, adjResponse(row))
}

// GetAdjustment handles GET /t/inventory/adjustments/{id}.
func (h *DocsHandler) GetAdjustment(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.GetAdjustment(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, adjResponse(row))
}

// DecideAdjustment handles POST /t/inventory/adjustments/{id}/approve and
// .../reject — the approval gate before posting.
func (h *DocsHandler) DecideAdjustment(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, ok := actor(r)
		if !ok {
			httpx.WriteError(w, r, httpx.ErrForbidden)
			return
		}
		row, err := h.svc.ApproveAdjustment(r.Context(), a, chi.URLParam(r, "id"), approve)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.OK(w, adjResponse(row))
	}
}

// PostAdjustment handles POST /t/inventory/adjustments/{id}/post — writes the
// ledger movements and balances for an approved adjustment.
func (h *DocsHandler) PostAdjustment(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.PostAdjustment(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, adjResponse(row))
}

// ---- transfers -----------------------------------------------------------------

// ListTransfers handles GET /t/inventory/transfers?branch_id=.
func (h *DocsHandler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	rows, total, err := h.svc.Transfers(r.Context(), a, r.URL.Query().Get("branch_id"), page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]TransferResponse, len(rows))
	for i, row := range rows {
		out[i] = trResponse(row)
	}
	httpx.List(w, out, page.Meta(total))
}

// CreateTransfer handles POST /t/inventory/transfers.
func (h *DocsHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in TransferRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := h.svc.CreateTransfer(r.Context(), a, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, trResponse(row))
}

// GetTransfer handles GET /t/inventory/transfers/{id}.
func (h *DocsHandler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.GetTransfer(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, trResponse(row))
}

// SendTransfer handles POST /t/inventory/transfers/{id}/send — deducts stock
// at the source branch.
func (h *DocsHandler) SendTransfer(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.SendTransfer(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, trResponse(row))
}

// ReceiveTransfer handles POST /t/inventory/transfers/{id}/receive.
func (h *DocsHandler) ReceiveTransfer(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in ReceiveRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := h.svc.ReceiveTransfer(r.Context(), a, chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, trResponse(row))
}

// CancelTransfer handles POST /t/inventory/transfers/{id}/cancel.
func (h *DocsHandler) CancelTransfer(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.CancelTransfer(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, trResponse(row))
}

// ---- stocktakes -----------------------------------------------------------------

type countIn struct {
	BranchID   string `json:"branch_id"`
	LocationID string `json:"location_id"`
	Note       string `json:"note"`
}

// ListCounts handles GET /t/inventory/counts?branch_id=.
func (h *DocsHandler) ListCounts(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	rows, total, err := h.svc.Counts(r.Context(), a, r.URL.Query().Get("branch_id"), page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]CountResponse, len(rows))
	for i, row := range rows {
		out[i] = cntResponse(row)
	}
	httpx.List(w, out, page.Meta(total))
}

// CreateCount handles POST /t/inventory/counts — snapshots expected stock.
func (h *DocsHandler) CreateCount(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in countIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := h.svc.CreateCount(r.Context(), a, in.BranchID, in.LocationID, in.Note)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, cntResponse(row))
}

// GetCount handles GET /t/inventory/counts/{id}.
func (h *DocsHandler) GetCount(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.GetCount(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, cntResponse(row))
}

// SetCounted handles PUT /t/inventory/counts/{id}/items — records physical
// counts keyed by item public id.
func (h *DocsHandler) SetCounted(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in struct {
		Counts map[string]string `json:"counts"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := h.svc.SetCounted(r.Context(), a, chi.URLParam(r, "id"), in.Counts)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, cntResponse(row))
}

// PostCount handles POST /t/inventory/counts/{id}/post — writes variance
// adjustments to the ledger.
func (h *DocsHandler) PostCount(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.PostCount(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, cntResponse(row))
}

// CancelCount handles POST /t/inventory/counts/{id}/cancel.
func (h *DocsHandler) CancelCount(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	row, err := h.svc.CancelCount(r.Context(), a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, cntResponse(row))
}
