package tenants

import "mauzo/internal/core/authz"

// Tenant management permissions (platform).
const (
	PermTenantView    = "platform.tenant.view"
	PermTenantCreate  = "platform.tenant.create"
	PermTenantUpdate  = "platform.tenant.update"
	PermTenantSuspend = "platform.tenant.suspend"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermTenantView, Description: "View tenants"},
		authz.Permission{Code: PermTenantCreate, Description: "Create tenants"},
		authz.Permission{Code: PermTenantUpdate, Description: "Edit tenants"},
		authz.Permission{Code: PermTenantSuspend, Description: "Suspend and reactivate tenants", Sensitive: true},
	)
}
