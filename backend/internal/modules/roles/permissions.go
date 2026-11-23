package roles

import "mauzo/internal/core/authz"

// Role permissions.
const (
	PermRoleView   = "roles.role.view"
	PermRoleManage = "roles.role.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermRoleView, Module: "roles", Description: "View roles and permissions"},
		authz.Permission{Code: PermRoleManage, Module: "roles", Description: "Create, edit and delete roles", Sensitive: true},
	)
}
