package businesstypes

import "mauzo/internal/core/authz"

// Business-type permissions (platform). Module management permissions live in
// internal/modules/registry.
const (
	PermBusinessTypeView   = "platform.businesstype.view"
	PermBusinessTypeManage = "platform.businesstype.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermBusinessTypeView, Description: "View business types and profiles"},
		authz.Permission{Code: PermBusinessTypeManage, Description: "Create and edit business profiles"},
	)
}
