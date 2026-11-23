package registry

import "mauzo/internal/core/authz"

// Module management permissions.
const (
	PermPlatformModuleView   = "platform.module.view"
	PermPlatformModuleManage = "platform.module.manage"
	PermTenantModuleManage   = "settings.module.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermPlatformModuleView, Description: "View the module registry"},
		authz.Permission{Code: PermPlatformModuleManage, Description: "Switch modules platform-wide and override tenant modules", Sensitive: true},
		authz.Permission{Code: PermTenantModuleManage, Module: "settings", Description: "Enable, disable and configure modules", Sensitive: true},
	)
}
