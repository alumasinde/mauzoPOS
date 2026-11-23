package settings

import "mauzo/internal/core/authz"

// Tenant settings permissions (interim: RequireUserRole enforces them).
const (
	PermBusinessView   = "settings.business.view"
	PermBusinessUpdate = "settings.business.update"
	PermModuleManage   = "settings.module.manage"
)

func init() {
	authz.Register(PermBusinessView, PermBusinessUpdate, PermModuleManage)
}
