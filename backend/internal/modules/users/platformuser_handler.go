package users

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// PlatformUserHandler exposes platform user CRUD endpoints.
type PlatformUserHandler struct {
	svc   *PlatformUserService
	audit *audit.Writer
}

// NewPlatformUserHandler builds a PlatformUserHandler.
func NewPlatformUserHandler(svc *PlatformUserService, audit *audit.Writer) *PlatformUserHandler {
	return &PlatformUserHandler{svc: svc, audit: audit}
}

// List handles GET /platform/users.
func (h *PlatformUserHandler) List(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePage(r)
	us, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]PlatformUserResponse, len(us))
	for i, u := range us {
		out[i] = toPlatformUserResponse(u)
	}
	httpx.List(w, out, p.Meta(total))
}

// Create handles POST /platform/users.
func (h *PlatformUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePlatformUserRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, err := h.svc.Create(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action:     "platform.user.create",
		EntityType: "platform_user",
		EntityID:   u.PublicID,
		Data:       map[string]any{"email": u.Email, "role": u.Role},
	}))
	httpx.Created(w, toPlatformUserResponse(u))
}

// Get handles GET /platform/users/{id}.
func (h *PlatformUserHandler) Get(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toPlatformUserResponse(u))
}

// Update handles PATCH /platform/users/{id}.
func (h *PlatformUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdatePlatformUserRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.user.update", EntityType: "platform_user", EntityID: u.PublicID,
	}))
	httpx.OK(w, toPlatformUserResponse(u))
}

// Disable handles POST /platform/users/{id}/disable.
func (h *PlatformUserHandler) Disable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, StatusDisabled)
}

// Enable handles POST /platform/users/{id}/enable.
func (h *PlatformUserHandler) Enable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, StatusActive)
}

func (h *PlatformUserHandler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	var selfID uint64
	if a, ok := identity.ActorFrom(r.Context()); ok {
		selfID = a.ID
	}
	u, err := h.svc.SetStatus(r.Context(), chi.URLParam(r, "id"), status, selfID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.user." + status, EntityType: "platform_user", EntityID: u.PublicID,
	}))
	httpx.OK(w, toPlatformUserResponse(u))
}

// SetPassword handles POST /platform/users/{id}/password (admin reset).
func (h *PlatformUserHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req SetPasswordRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, err := h.svc.SetPassword(r.Context(), chi.URLParam(r, "id"), req.Password)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "platform.user.reset_password", EntityType: "platform_user", EntityID: u.PublicID,
	}))
	httpx.NoContent(w)
}
