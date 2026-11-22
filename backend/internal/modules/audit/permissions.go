package audit

import "mauzo/internal/core/authz"

// Audit viewer permissions.
const (
	PermPlatformAuditView = "platform.audit.view"
	PermTenantAuditView   = "audit.log.view" // enforced via roles in Phase 3
)

func init() {
	authz.Register(PermPlatformAuditView, PermTenantAuditView)
}
