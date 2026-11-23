package identity

import (
	"slices"

	"mauzo/internal/core/authz"
)

// Access is what a tenant user may do, resolved once per request by the auth
// middleware (cached by the roles/registry modules).
type Access struct {
	Perms       authz.Set
	Modules     map[string]bool // effective enabled modules of the tenant
	AllBranches bool
	Branches    []uint64 // branch ids when AllBranches is false
}

// Can reports whether the actor holds perm. For tenant users a permission
// only takes effect while its module is enabled for the tenant, and platform
// permissions never apply. Platform actors are checked against their role.
func (a Actor) Can(perm string) bool {
	switch a.Type {
	case ActorPlatformUser:
		return authz.PlatformRoleAllows(a.Role, perm)
	case ActorUser:
		p, ok := authz.Lookup(perm)
		if !ok || p.Scope() != authz.ScopeTenant {
			return false
		}
		if p.Module != "" && !a.Access.Modules[p.Module] {
			return false
		}
		return a.Access.Perms.Has(perm)
	}
	return false
}

// HasModule reports whether the actor's tenant has module code enabled.
func (a Actor) HasModule(code string) bool {
	return a.Type == ActorUser && a.Access.Modules[code]
}

// CanAccessBranch reports whether a tenant user may act on branch id.
func (a Actor) CanAccessBranch(id uint64) bool {
	if a.Type != ActorUser {
		return false
	}
	return a.Access.AllBranches || slices.Contains(a.Access.Branches, id)
}
