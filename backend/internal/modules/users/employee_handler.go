package users

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// EmployeeHandler serves /t/employees.
type EmployeeHandler struct {
	svc   *UserService
	audit *audit.Writer
}

// NewEmployeeHandler builds an EmployeeHandler.
func NewEmployeeHandler(svc *UserService, aw *audit.Writer) *EmployeeHandler {
	return &EmployeeHandler{svc: svc, audit: aw}
}

func actorOf(r *http.Request) identity.Actor {
	a, _ := identity.ActorFrom(r.Context())
	return a
}

func (h *EmployeeHandler) respond(w http.ResponseWriter, r *http.Request, status int, u User) {
	idx, err := h.svc.branchIndex(r.Context(), u.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, status, httpx.Envelope{Data: employeeResponse(u, idx)})
}

func (h *EmployeeHandler) log(r *http.Request, action string, u User, data map[string]any) {
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: action, EntityType: "user", EntityID: u.PublicID, Data: data,
	}))
}

// List handles GET /t/employees?q=&status=&role=.
func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	a := actorOf(r)
	page := httpx.ParsePage(r)
	f := ListFilter{Query: page.Query, Status: r.URL.Query().Get("status")}
	if pid := r.URL.Query().Get("role"); pid != "" {
		role, _, err := h.svc.deps.Roles.ByPublicIDForAssign(r.Context(), a.TenantID, pid)
		if err != nil {
			httpx.WriteError(w, r, ErrInvalidRole)
			return
		}
		f.RoleID = role.ID
	}
	rows, total, err := h.svc.ListEmployees(r.Context(), a.TenantID, f, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	idx, err := h.svc.branchIndex(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]EmployeeResponse, len(rows))
	for i, u := range rows {
		out[i] = employeeResponse(u, idx)
	}
	httpx.List(w, out, page.Meta(total))
}

// Get handles GET /t/employees/{id}.
func (h *EmployeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.GetEmployee(r.Context(), actorOf(r).TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.respond(w, r, http.StatusOK, u)
}

func decodeEmployee(w http.ResponseWriter, r *http.Request) (EmployeeRequest, bool) {
	var req EmployeeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return req, false
	}
	return req, true
}

// Create handles POST /t/employees.
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeEmployee(w, r)
	if !ok {
		return
	}
	u, err := h.svc.CreateEmployee(r.Context(), actorOf(r), req.input())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "users.employee.create", u, map[string]any{"role": u.Role, "all_branches": u.AllBranches})
	h.respond(w, r, http.StatusCreated, u)
}

// Update handles PATCH /t/employees/{id}. Password/PIN changes are refused
// here; they have dedicated, separately-permissioned endpoints.
func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeEmployee(w, r)
	if !ok {
		return
	}
	if req.Password != nil || req.PIN != nil {
		//nolint:gosec // these are field names in a validation message, not credentials
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(map[string]any{
			"password": "use POST /employees/{id}/password", "pin": "use POST /employees/{id}/pin",
		}))
		return
	}
	u, err := h.svc.UpdateEmployee(r.Context(), actorOf(r), chi.URLParam(r, "id"), req.input())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "users.employee.update", u, map[string]any{"role": u.Role, "all_branches": u.AllBranches})
	h.respond(w, r, http.StatusOK, u)
}

func (h *EmployeeHandler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	u, err := h.svc.SetEmployeeStatus(r.Context(), actorOf(r), chi.URLParam(r, "id"), status)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, "users.employee."+map[string]string{StatusActive: "enable", StatusDisabled: "disable"}[status], u, nil)
	h.respond(w, r, http.StatusOK, u)
}

// Disable handles POST /t/employees/{id}/disable.
func (h *EmployeeHandler) Disable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, StatusDisabled)
}

// Enable handles POST /t/employees/{id}/enable.
func (h *EmployeeHandler) Enable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, StatusActive)
}

func (h *EmployeeHandler) setSecret(w http.ResponseWriter, r *http.Request, pin bool) {
	var req SecretRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in := EmployeeInput{Password: &req.Value}
	action := "users.employee.reset_password"
	if pin {
		in = EmployeeInput{PIN: &req.Value}
		action = "users.employee.reset_pin"
	}
	u, err := h.svc.UpdateEmployee(r.Context(), actorOf(r), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.log(r, action, u, nil)
	h.respond(w, r, http.StatusOK, u)
}

// SetPassword handles POST /t/employees/{id}/password (signs the user out).
func (h *EmployeeHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	h.setSecret(w, r, false)
}

// SetPIN handles POST /t/employees/{id}/pin.
func (h *EmployeeHandler) SetPIN(w http.ResponseWriter, r *http.Request) { h.setSecret(w, r, true) }

// AssignableRoles handles GET /t/employees/assignable-roles.
func (h *EmployeeHandler) AssignableRoles(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.AssignableRoles(r.Context(), actorOf(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, out)
}
