package settings

import "mauzo/internal/core/authz"

// Tenant settings permissions.
const (
	PermBusinessView   = "settings.business.view"
	PermBusinessUpdate = "settings.business.update"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermBusinessView, Module: "settings", Description: "View business settings"},
		authz.Permission{Code: PermBusinessUpdate, Module: "settings", Description: "Edit business settings and run setup"},
	)
}
