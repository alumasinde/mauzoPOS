package subscriptions

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
)

// Handler serves platform plan endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List handles GET /platform/plans.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]PlanResponse, len(rows))
	for i, p := range rows {
		out[i] = planResponse(p)
	}
	httpx.OK(w, out)
}

// Create handles POST /platform/plans.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req PlanRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	p, err := h.svc.Create(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, planResponse(p))
}

// Update handles PATCH /platform/plans/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req PlanRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	p, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, planResponse(p))
}
