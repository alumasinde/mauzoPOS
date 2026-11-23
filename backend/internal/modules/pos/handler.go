package pos

import (
	"net/http"
	"time"

	"mauzo/internal/core/httpx"
)

// Handler serves /pos endpoints.
type Handler struct {
	svc     *Service
	modules ModuleLister
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, modules ModuleLister) *Handler {
	return &Handler{svc: svc, modules: modules}
}

// Activate handles POST /pos/activate (public, code-guarded).
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	var req ActivateRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := h.svc.Activate(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, out)
}

// Whoami handles GET /pos/whoami — also the heartbeat.
func (h *Handler) Whoami(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	out, err := h.svc.Whoami(r.Context(), d)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, out)
}

// Sync handles GET /pos/sync?since=RFC3339.
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var since time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(map[string]any{"since": "RFC3339 timestamp"}))
			return
		}
		since = t
	}
	out, err := h.svc.Sync(r.Context(), d, since, h.modules)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, out)
}

// SignInRequest is POST /pos/signin.
type SignInRequest struct {
	UserID string `json:"user_id"` // user public id
	PIN    string `json:"pin"`
}

// SignIn handles POST /pos/signin — online PIN verification.
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req SignInRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	uid, err := h.svc.userByPublicID(r.Context(), d.TenantID, req.UserID)
	if err != nil {
		httpx.WriteError(w, r, ErrBadPIN)
		return
	}
	_, perms, err := h.svc.SignIn(r.Context(), d, uid, req.PIN)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"ok": true, "permissions": perms})
}

// AuthorizeRequest is POST /pos/authorize — supervisor PIN check.
type AuthorizeRequest struct {
	Permission string `json:"permission"`
	PIN        string `json:"pin"`
}

// Authorize handles POST /pos/authorize.
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req AuthorizeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.Permission == "" {
		req.Permission = PermSupervisorAuthorize
	}
	name, err := h.svc.Authorize(r.Context(), d, req.Permission, req.PIN)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"ok": true, "authorized_by": name})
}

// Held handles GET /pos/held (list) and POST /pos/held (park).
func (h *Handler) Held(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	switch r.Method {
	case http.MethodGet:
		rows, err := h.svc.Held(r.Context(), d)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		out := make([]HeldResponse, 0, len(rows))
		for _, x := range rows {
			out = append(out, toHeldResponse(x))
		}
		httpx.OK(w, out)
	case http.MethodPost:
		var req HoldRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		h2, err := h.svc.Hold(r.Context(), d, req)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Created(w, toHeldResponse(h2))
	default:
		httpx.WriteError(w, r, httpx.ErrNotFound)
	}
}

// RecallHeld handles POST /pos/held/{id}/recall.
func (h *Handler) RecallHeld(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	h2, err := h.svc.RecallHeld(r.Context(), d, publicIDParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toHeldResponse(h2))
}

// CancelHeld handles DELETE /pos/held/{id}.
func (h *Handler) CancelHeld(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	if err := h.svc.CancelHeld(r.Context(), d, publicIDParam(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}
