package registerall_test

import (
	"testing"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/modules"
	_ "mauzo/internal/modules/registerall"
)

// A nav entry or widget guarded by an unregistered permission would be hidden
// from everyone, including owners — catch typos at test time.
func TestNavAndWidgetPermissionsAreRegistered(t *testing.T) {
	for _, m := range modules.All() {
		for _, n := range m.Nav {
			checkPerm(t, m.Code, "nav "+n.Key, n.Permission)
		}
		for _, w := range m.Widgets {
			checkPerm(t, m.Code, "widget "+w.Key, w.Permission)
		}
	}
}

func checkPerm(t *testing.T, module, what, perm string) {
	t.Helper()
	if perm == "" {
		return
	}
	p, ok := authz.Lookup(perm)
	if !ok {
		t.Errorf("%s (%s): permission %q is not defined", what, module, perm)
		return
	}
	// The permission must be switched on together with the item, or the item
	// would show while its permission is inert.
	if p.Module != "" && p.Module != module {
		if dep, _ := modules.Get(module); !contains(dep.Depends, p.Module) {
			t.Errorf("%s (%s): permission %q belongs to unrelated module %q", what, module, perm, p.Module)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// Every module that owns permissions must be linked here, or `migrate seed`
// would sync a partial registry.
func TestEveryPermissionModuleIsKnown(t *testing.T) {
	for _, p := range authz.All() {
		if p.Module != "" && !modules.Known(p.Module) {
			t.Errorf("permission %q names unknown module %q", p.Code, p.Module)
		}
	}
}
