package users

import "mauzo/internal/core/authz"

// Platform user management permissions.
const (
	PermPlatformUserView   = "platform.user.view"
	PermPlatformUserManage = "platform.user.manage"
)

func init() {
	authz.Register(PermPlatformUserView, PermPlatformUserManage)
}
