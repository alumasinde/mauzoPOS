package audit

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
	"mauzo/internal/modules/users"
)

// PlatformRoutes wires the platform audit viewer.
func PlatformRoutes(r chi.Router, h *Handler) {
	r.With(mw.Can(PermPlatformAuditView)).Get("/audit-logs", h.ListPlatform)
}

// TenantRoutes wires the tenant audit viewer. Interim: only owners/admins may
// view until role-based permissions land in Phase 3.
func TenantRoutes(r chi.Router, h *Handler) {
	r.With(mw.RequireUserRole(users.UserRoleOwner, users.UserRoleAdmin)).
		Get("/audit-logs", h.ListTenant)
}
