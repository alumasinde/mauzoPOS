package authz

import (
	"slices"
	"strings"
)

// Platform roles are a fixed set (docs/permissions.md §1), unlike tenant roles
// which are DB rows. Keep the role→permission mapping here so middleware can
// check it without a database round-trip.
const (
	RoleSuperAdmin = "super_admin"
	RoleSupport    = "support"
	RoleBilling    = "billing"
	RoleViewer     = "viewer"
)

// PlatformRoles lists every valid platform role.
var PlatformRoles = []string{RoleSuperAdmin, RoleSupport, RoleBilling, RoleViewer}

// platformGrants maps non-super-admin roles to their permission set.
var platformGrants = map[string][]string{
	RoleSupport: {
		"platform.dashboard.view",
		"platform.tenant.view",
		"platform.tenant.impersonate",
		"platform.user.view",
		"platform.monitoring.view",
		"platform.audit.view",
	},
	RoleBilling: {
		"platform.dashboard.view",
		"platform.tenant.view",
		"platform.subscription.view",
		"platform.subscription.manage",
	},
	RoleViewer: {}, // every platform.*.view permission; see PlatformRoleAllows
}

// PlatformRoleAllows reports whether role holds perm. Super admins hold every
// registered platform permission; viewers hold every "*.view" platform
// permission. Unknown roles hold nothing.
func PlatformRoleAllows(role, perm string) bool {
	switch role {
	case RoleSuperAdmin:
		return strings.HasPrefix(perm, "platform.")
	case RoleViewer:
		return strings.HasPrefix(perm, "platform.") && strings.HasSuffix(perm, ".view")
	}
	return slices.Contains(platformGrants[role], perm)
}
