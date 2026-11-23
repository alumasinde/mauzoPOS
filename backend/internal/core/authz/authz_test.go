package authz

import (
	"slices"
	"testing"
)

func withRegistry(t *testing.T, perms ...Permission) {
	t.Helper()
	mu.Lock()
	saved := registry
	registry = map[string]Permission{}
	mu.Unlock()
	Define(perms...)
	t.Cleanup(func() {
		mu.Lock()
		registry = saved
		mu.Unlock()
	})
}

func TestDefineRejectsDuplicatesAndMalformed(t *testing.T) {
	withRegistry(t, Permission{Code: "sales.sale.view"})
	for _, bad := range []string{"sales.sale.view", "nodot", "sales.*", "a b.c"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Define(%q) should panic", bad)
				}
			}()
			Define(Permission{Code: bad})
		}()
	}
}

func TestScope(t *testing.T) {
	if (Permission{Code: "platform.tenant.view"}).Scope() != ScopePlatform {
		t.Fatal("platform prefix should be platform scope")
	}
	if (Permission{Code: "sales.sale.view"}).Scope() != ScopeTenant {
		t.Fatal("other codes are tenant scope")
	}
}

func TestExpandPatterns(t *testing.T) {
	withRegistry(t,
		Permission{Code: "catalog.product.view"},
		Permission{Code: "catalog.product.create"},
		Permission{Code: "sales.sale.view"},
		Permission{Code: "sales.refund.create"},
		Permission{Code: "platform.tenant.view"},
	)
	got := Expand(ScopeTenant, []string{"catalog.*", "*.view", "missing.perm.x"})
	want := []string{"catalog.product.create", "catalog.product.view", "sales.sale.view"}
	if !slices.Equal(got, want) {
		t.Fatalf("Expand = %v, want %v", got, want)
	}
	if all := Expand(ScopeTenant, []string{"*"}); slices.Contains(all, "platform.tenant.view") {
		t.Fatal("tenant expansion must never include platform permissions")
	}
}

func TestSet(t *testing.T) {
	s := NewSet("a.b.view", "a.b.edit")
	if !s.Has("a.b.view") || s.Has("a.b.delete") {
		t.Fatal("Has wrong")
	}
	if !s.Contains([]string{"a.b.view"}) || s.Contains([]string{"a.b.view", "x.y.z"}) {
		t.Fatal("Contains wrong")
	}
	if !AllOf().Has("anything.at.all") || !AllOf().IsAll() {
		t.Fatal("AllOf should grant everything")
	}
}

func TestTemplatesWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, tpl := range RoleTemplates {
		if seen[tpl.Code] {
			t.Fatalf("duplicate template %q", tpl.Code)
		}
		seen[tpl.Code] = true
		if tpl.Name == "" || len(tpl.Grants) == 0 {
			t.Fatalf("template %q incomplete", tpl.Code)
		}
	}
	owner, ok := Template(RoleOwner)
	if !ok || !owner.Locked || !owner.AllBranches {
		t.Fatal("owner must exist, be locked and see all branches")
	}
}
