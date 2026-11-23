package businesstypes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
)

// Handler serves the platform business-type/profile endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListTypes handles GET /platform/business-types.
func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListTypes(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]TypeResponse, len(rows))
	for i, t := range rows {
		out[i] = typeResponse(t)
	}
	httpx.OK(w, out)
}

// ListProfiles handles GET /platform/business-profiles?type=.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListProfiles(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ProfileResponse, len(rows))
	for i, p := range rows {
		_ = p.ParseDefinition()
		out[i] = profileResponse(p)
	}
	httpx.OK(w, out)
}

// CreateProfile handles POST /platform/business-profiles.
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	p, err := h.svc.CreateProfile(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, profileResponse(p))
}

// UpdateProfile handles PATCH /platform/business-profiles/{id}.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	p, err := h.svc.UpdateProfile(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, profileResponse(p))
}
