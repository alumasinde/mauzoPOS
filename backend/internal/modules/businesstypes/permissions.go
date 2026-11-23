package businesstypes

import "mauzo/internal/core/authz"

// Business type & profile template permissions (platform context).
const (
	PermBusinessTypeView   = "platform.businesstype.view"
	PermBusinessTypeManage = "platform.businesstype.manage"
	PermModuleView         = "platform.module.view"
	PermModuleManage       = "platform.module.manage"
)

func init() {
	authz.Register(PermBusinessTypeView, PermBusinessTypeManage,
		PermModuleView, PermModuleManage)
}
