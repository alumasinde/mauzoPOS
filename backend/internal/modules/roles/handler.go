package roles

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/roles and /t/permissions.
type Handler struct {
	svc   *Service
	audit *audit.Writer
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, aw *audit.Writer) *Handler { return &Handler{svc: svc, audit: aw} }

func actorOf(r *http.Request) identity.Actor {
	a, _ := identity.ActorFrom(r.Context())
	return a
}

// List handles GET /t/roles.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.List(r.Context(), actorOf(r).TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]RoleResponse, len(rows))
	for i, role := range rows {
		out[i] = toResponse(role)
	}
	httpx.OK(w, out)
}

// Get handles GET /t/roles/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	role, err := h.svc.Get(r.Context(), actorOf(r).TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toResponse(role))
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request) (Input, bool) {
	var req RoleRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return Input{}, false
	}
	in, err := req.input()
	if err != nil {
		httpx.WriteError(w, r, err)
		return Input{}, false
	}
	return in, true
}

func (h *Handler) log(r *http.Request, action string, role Role) {
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: action, EntityType: "role", EntityID: role.PublicID,
		Data: map[string]any{"name": role.Name, "permissions": role.Permissions},
	}))
}

// Create handles POST /t/roles.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := h.decode(w, r)
	if !ok {
		return
	}
	role, err := h.svc.Create(r.Context(), actorOf(r), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "roles.role.create", role)
	httpx.Created(w, toResponse(role))
}

// Update handles PATCH /t/roles/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	in, ok := h.decode(w, r)
	if !ok {
		return
	}
	role, err := h.svc.Update(r.Context(), actorOf(r), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "roles.role.update", role)
	httpx.OK(w, toResponse(role))
}

// Delete handles DELETE /t/roles/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), actorOf(r), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "roles.role.delete", Role{PublicID: id})
	httpx.NoContent(w)
}

// Reset handles POST /t/roles/{id}/reset.
func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	role, err := h.svc.Reset(r.Context(), actorOf(r), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "roles.role.reset", role)
	httpx.OK(w, toResponse(role))
}

// Permissions handles GET /t/permissions: the catalogue grouped by module,
// flagging groups whose module is disabled for the tenant.
func (h *Handler) Permissions(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, catalogue(actorOf(r).Access.Modules))
}
