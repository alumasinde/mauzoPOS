package audit

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// PlatformRoutes wires the platform audit viewer.
func PlatformRoutes(r chi.Router, h *Handler) {
	r.With(mw.Can(PermPlatformAuditView)).Get("/audit-logs", h.ListPlatform)
}

// TenantRoutes wires the tenant audit viewer.
func TenantRoutes(r chi.Router, h *Handler) {
	r.With(mw.Can(PermTenantAuditView)).Get("/audit-logs", h.ListTenant)
}
