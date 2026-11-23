package registry

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/modules"
)

// TenantResolver maps a tenant public id to its id (platform overrides).
type TenantResolver interface {
	TenantIDByPublicID(ctx context.Context, publicID string) (uint64, error)
}

// Handler serves /t/modules, /t/navigation and /platform/modules.
type Handler struct {
	svc     *Service
	tenants TenantResolver
	audit   *audit.Writer
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, tenants TenantResolver, aw *audit.Writer) *Handler {
	return &Handler{svc: svc, tenants: tenants, audit: aw}
}

func (h *Handler) writeTenantView(w http.ResponseWriter, r *http.Request, tenantID uint64, changed []string) {
	states, err := h.svc.TenantView(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	mods := make([]ModuleResponse, len(states))
	for i, st := range states {
		mods[i] = stateResponse(st)
	}
	if changed == nil {
		httpx.OK(w, mods)
		return
	}
	httpx.OK(w, ToggleResponse{Changed: changed, Modules: mods})
}

// List handles GET /t/modules.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, _ := identity.ActorFrom(r.Context())
	h.writeTenantView(w, r, a.TenantID, nil)
}

// Toggle handles PUT /t/modules/{code}.
func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	a, _ := identity.ActorFrom(r.Context())
	h.toggle(w, r, a.TenantID, SourceTenant)
}

func (h *Handler) toggle(w http.ResponseWriter, r *http.Request, tenantID uint64, source string) {
	var req ToggleRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	code := chi.URLParam(r, "code")
	var changed []string
	var err error
	if req.Enabled {
		changed, err = h.svc.Enable(r.Context(), tenantID, code, source)
	} else {
		changed, err = h.svc.Disable(r.Context(), tenantID, code, req.Cascade || source == SourcePlatform)
	}
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	action := "settings.module.disable"
	if req.Enabled {
		action = "settings.module.enable"
	}
	e := audit.FromRequest(r, audit.Entry{
		Action: action, EntityType: "module", EntityID: code,
		Data: map[string]any{"changed": changed, "source": source},
	})
	e.TenantID = &tenantID
	h.audit.Log(r.Context(), e)
	if changed == nil {
		changed = []string{}
	}
	h.writeTenantView(w, r, tenantID, changed)
}

// SetConfig handles PUT /t/modules/{code}/config.
func (h *Handler) SetConfig(w http.ResponseWriter, r *http.Request) {
	a, _ := identity.ActorFrom(r.Context())
	var req struct {
		Config json.RawMessage `json:"config"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	code := chi.URLParam(r, "code")
	if err := h.svc.SetConfig(r.Context(), a.TenantID, code, req.Config); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "settings.module.configure", EntityType: "module", EntityID: code,
		Data: map[string]any{"config": json.RawMessage(req.Config)},
	}))
	h.writeTenantView(w, r, a.TenantID, nil)
}

// Navigation handles GET /t/navigation.
func (h *Handler) Navigation(w http.ResponseWriter, r *http.Request) {
	a, _ := identity.ActorFrom(r.Context())
	items := Navigation(a)
	if items == nil {
		items = []modules.NavItem{}
	}
	httpx.OK(w, NavResponse{Items: items})
}

// PlatformList handles GET /platform/modules.
func (h *Handler) PlatformList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.PlatformView(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ModuleResponse, len(rows))
	for i, st := range rows {
		out[i] = platformResponse(st)
	}
	httpx.OK(w, out)
}

// PlatformSetActive handles PATCH /platform/modules/{code}.
func (h *Handler) PlatformSetActive(w http.ResponseWriter, r *http.Request) {
	var req ActiveRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	code := chi.URLParam(r, "code")
	if err := h.svc.SetPlatformActive(r.Context(), code, req.Active); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.module.set_active", EntityType: "module", EntityID: code,
		Data: map[string]any{"active": req.Active},
	}))
	h.PlatformList(w, r)
}

func (h *Handler) platformTenant(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := h.tenants.TenantIDByPublicID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return 0, false
	}
	return id, true
}

// PlatformTenantList handles GET /platform/tenants/{id}/modules.
func (h *Handler) PlatformTenantList(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.platformTenant(w, r); ok {
		h.writeTenantView(w, r, id, nil)
	}
}

// PlatformTenantToggle handles PUT /platform/tenants/{id}/modules/{code}: an
// override that bypasses plan gating (disabling cascades).
func (h *Handler) PlatformTenantToggle(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.platformTenant(w, r); ok {
		h.toggle(w, r, id, SourcePlatform)
	}
}
