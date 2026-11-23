package onboarding

import (
	"net/http"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/subscriptions"
	"mauzo/internal/modules/terminals"
)

// Handler serves /t/onboarding endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func tenantID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	actor, ok := identity.ActorFrom(r.Context())
	if !ok || actor.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return 0, false
	}
	return actor.TenantID, true
}

func refresh(w http.ResponseWriter, r *http.Request, svc *Service, id uint64) {
	st, err := svc.Current(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, st)
}

// Get handles GET /t/onboarding — current state + option lists.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	refresh(w, r, h.svc, id)
}

// PutBusiness handles PUT /t/onboarding/business.
func (h *Handler) PutBusiness(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req BusinessRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SaveBusiness(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PutProfile handles PUT /t/onboarding/profile.
func (h *Handler) PutProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req ProfileRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SaveProfile(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PutPlan handles PUT /t/onboarding/plan.
func (h *Handler) PutPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req subscriptions.AssignRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SavePlan(r.Context(), id, req.PlanID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PutModules handles PUT /t/onboarding/modules.
func (h *Handler) PutModules(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req ModulesRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SaveModules(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PostBranch handles POST /t/onboarding/branch.
func (h *Handler) PostBranch(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req branches.CreateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := h.svc.SaveBranch(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PutAdmin handles PUT /t/onboarding/admin (empty body = skip).
func (h *Handler) PutAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req AdminRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, _ := identity.ActorFrom(r.Context())
	if err := h.svc.SaveAdmin(r.Context(), actor, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PutPreferences handles PUT /t/onboarding/preferences.
func (h *Handler) PutPreferences(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req PreferencesRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.SavePreferences(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PostTerminal handles POST /t/onboarding/terminal.
func (h *Handler) PostTerminal(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req terminals.CreateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := h.svc.SaveTerminal(r.Context(), id, req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}

// PostComplete handles POST /t/onboarding/complete.
func (h *Handler) PostComplete(w http.ResponseWriter, r *http.Request) {
	id, ok := tenantID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Complete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	refresh(w, r, h.svc, id)
}
