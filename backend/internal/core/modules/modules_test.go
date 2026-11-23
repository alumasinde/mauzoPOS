package modules

import (
	"errors"
	"slices"
	"testing"
)

func TestCatalogueKeepsExistingCodes(t *testing.T) {
	// Plans and profile templates in the DB reference these codes.
	for _, c := range []string{"dashboard", "settings", "users", "roles", "branches", "terminals",
		"catalog", "pricing", "promotions", "inventory", "purchasing", "sales", "customers",
		"payments", "mpesa", "tax", "etims", "receipts", "shifts", "reports",
		"weighing_scales", "restaurant", "loyalty", "credit", "delivery"} {
		if !Known(c) {
			t.Errorf("module %q missing from registry", c)
		}
	}
}

func TestClosureAddsTransitiveDeps(t *testing.T) {
	got, err := Closure([]string{"purchasing"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"catalog", "inventory", "purchasing"}; !slices.Equal(got, want) {
		t.Fatalf("Closure = %v, want %v", got, want)
	}
	if _, err := Closure([]string{"nope"}); !errors.As(err, &ErrUnknown{}) {
		t.Fatalf("expected ErrUnknown, got %v", err)
	}
}

func TestDependentsAreTransitive(t *testing.T) {
	enabled := []string{"catalog", "inventory", "purchasing", "pricing", "reports"}
	got := Dependents("catalog", enabled)
	if want := []string{"pricing", "inventory", "purchasing"}; !slices.Equal(got, want) {
		t.Fatalf("Dependents = %v, want %v", got, want)
	}
	if len(Dependents("reports", enabled)) != 0 {
		t.Fatal("nothing depends on reports")
	}
}

func TestMissingDeps(t *testing.T) {
	got, err := MissingDeps("mpesa", []string{"catalog"})
	if err != nil || !slices.Equal(got, []string{"payments"}) {
		t.Fatalf("MissingDeps = %v, %v", got, err)
	}
}

func TestCoreModulesHaveNoDeps(t *testing.T) {
	for _, c := range CoreCodes() {
		m, _ := Get(c)
		for _, d := range m.Depends {
			if mod, _ := Get(d); !mod.Core {
				t.Errorf("core module %q depends on non-core %q", c, d)
			}
		}
	}
}

func TestCycleDetection(t *testing.T) {
	savedBy, savedOrder := byCode, order
	t.Cleanup(func() { byCode, order = savedBy, savedOrder })
	byCode, order = map[string]Module{}, nil
	register(Module{Code: "a", Depends: []string{"b"}}, Module{Code: "b", Depends: []string{"a"}})
	defer func() {
		if recover() == nil {
			t.Fatal("cycle not detected")
		}
	}()
	validateGraph()
}
