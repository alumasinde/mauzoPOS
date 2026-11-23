package identity

import (
	"testing"

	"mauzo/internal/core/authz"
)

func init() {
	authz.Define(
		authz.Permission{Code: "idtest.thing.view", Module: "idtest_mod"},
		authz.Permission{Code: "idtest.core.view"},
		authz.Permission{Code: "platform.idtest.view"},
	)
}

func user(perms authz.Set, mods ...string) Actor {
	m := map[string]bool{}
	for _, c := range mods {
		m[c] = true
	}
	return Actor{Type: ActorUser, Access: Access{Perms: perms, Modules: m}}
}

func TestCanRequiresModuleForTenantUsers(t *testing.T) {
	a := user(authz.NewSet("idtest.thing.view", "idtest.core.view"))
	if a.Can("idtest.thing.view") {
		t.Fatal("permission must not apply while its module is disabled")
	}
	if !a.Can("idtest.core.view") {
		t.Fatal("core permission should apply")
	}
	if !user(authz.NewSet("idtest.thing.view"), "idtest_mod").Can("idtest.thing.view") {
		t.Fatal("permission should apply once the module is enabled")
	}
}

func TestOwnerNeverGetsPlatformPermissions(t *testing.T) {
	owner := user(authz.AllOf(), "idtest_mod")
	if !owner.Can("idtest.thing.view") {
		t.Fatal("owner should hold tenant permissions")
	}
	if owner.Can("platform.idtest.view") {
		t.Fatal("tenant users must never hold platform permissions")
	}
	if owner.Can("unregistered.perm.x") {
		t.Fatal("unregistered permissions are never granted")
	}
}

func TestBranchAccess(t *testing.T) {
	a := Actor{Type: ActorUser, Access: Access{Branches: []uint64{2, 5}}}
	if !a.CanAccessBranch(5) || a.CanAccessBranch(3) {
		t.Fatal("branch list not respected")
	}
	a.Access.AllBranches = true
	if !a.CanAccessBranch(3) {
		t.Fatal("all-branches users reach every branch")
	}
	if (Actor{Type: ActorPlatformUser}).CanAccessBranch(1) {
		t.Fatal("platform actors have no tenant branch access")
	}
}
