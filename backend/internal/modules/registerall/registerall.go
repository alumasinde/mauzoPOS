// Package registerall links every module so their permissions.go init()s run.
// Any binary that syncs the permission registry to the database (the API at
// boot, `migrate seed`) must import it: syncing a partial registry would
// cascade-delete real role grants. Add new modules here.
package registerall

import (
	"mauzo/internal/core/authz"

	_ "mauzo/internal/modules/audit"         // permissions
	_ "mauzo/internal/modules/auth"          // auth context
	_ "mauzo/internal/modules/branches"      // permissions
	_ "mauzo/internal/modules/businesstypes" // permissions
	_ "mauzo/internal/modules/dashboard"     // permissions
	_ "mauzo/internal/modules/onboarding"    // routes
	_ "mauzo/internal/modules/pos"           // permissions
	_ "mauzo/internal/modules/registry"      // permissions
	_ "mauzo/internal/modules/roles"         // permissions
	_ "mauzo/internal/modules/settings"      // permissions + settings schemas
	_ "mauzo/internal/modules/subscriptions" // permissions
	_ "mauzo/internal/modules/tenants"       // permissions
	_ "mauzo/internal/modules/terminals"     // permissions
	_ "mauzo/internal/modules/users"         // permissions
)

func init() { authz.MarkComplete() }
