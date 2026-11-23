package audit

import "mauzo/internal/core/authz"

// Audit permissions.
const (
	PermPlatformAuditView = "platform.audit.view"
	PermTenantAuditView   = "audit.log.view"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermPlatformAuditView, Description: "View the platform audit log"},
		authz.Permission{Code: PermTenantAuditView, Description: "View the business audit log"},
	)
}
