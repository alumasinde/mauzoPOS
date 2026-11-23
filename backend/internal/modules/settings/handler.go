// Package settings exposes the tenant-facing settings API: list and upsert
// key/value configuration validated against registered schemas.
package settings

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	coresettings "mauzo/internal/core/settings"
)

// Handler serves /t/settings endpoints.
type Handler struct {
	store *coresettings.Store
}

// NewHandler builds a Handler.
func NewHandler(store *coresettings.Store) *Handler { return &Handler{store: store} }

// EntryResponse is the API view of one setting.
type EntryResponse struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SetRequest is PUT /t/settings/{key}; the value is arbitrary JSON validated
// against the key's schema.
type SetRequest struct {
	Value json.RawMessage `json:"value"`
}

// List handles GET /t/settings.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	rows, err := h.store.List(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]EntryResponse, len(rows))
	for i, e := range rows {
		out[i] = EntryResponse{Key: e.Key, Value: e.Value, UpdatedAt: e.UpdatedAt}
	}
	httpx.OK(w, out)
}

// Set handles PUT /t/settings/{key}.
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	var req SetRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		return
	}
	key := chi.URLParam(r, "key")
	if err := h.store.Set(r.Context(), tenantID, key, req.Value); err != nil {
		switch {
		case errors.Is(err, coresettings.ErrUnknownKey):
			httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
				map[string]any{"key": "unknown setting key"}))
		default:
			httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(
				map[string]any{"value": err.Error()}))
		}
		return
	}
	e, err := h.store.Get(r.Context(), tenantID, key)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, EntryResponse{Key: e.Key, Value: e.Value, UpdatedAt: e.UpdatedAt})
}

func tenantCtx(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return 0, false
	}
	return actor.TenantID, true
}
