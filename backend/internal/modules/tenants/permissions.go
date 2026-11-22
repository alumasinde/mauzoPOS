package tenants

import "mauzo/internal/core/authz"

// Tenant management permissions (platform context).
const (
	PermTenantView    = "platform.tenant.view"
	PermTenantCreate  = "platform.tenant.create"
	PermTenantUpdate  = "platform.tenant.update"
	PermTenantSuspend = "platform.tenant.suspend"
)

func init() {
	authz.Register(PermTenantView, PermTenantCreate, PermTenantUpdate, PermTenantSuspend)
}
