package registry_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"mauzo/internal/core/httpx"
	_ "mauzo/internal/modules/registerall" // full permission registry
	"mauzo/internal/modules/registry"
	"mauzo/internal/testutil"
	"mauzo/internal/testutil/rbactest"
)

type fakePlans struct {
	mods    []string
	hasPlan bool
}

func (f *fakePlans) PlanModules(context.Context, uint64) ([]string, bool, error) {
	return f.mods, f.hasPlan, nil
}

func setup(t *testing.T) (*registry.Service, *fakePlans, uint64) {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, rbactest.Tables...)
	plans := &fakePlans{}
	rbac := rbactest.Setup(t, conn, plans, nil)
	tenantID := rbac.Tenant(t, conn, "reg")
	t.Cleanup(func() {
		_ = rbac.Registry.SetPlatformActive(context.Background(), "mpesa", true)
	})
	return rbac.Registry, plans, tenantID
}

func code(err error) string {
	var e *httpx.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestEnableResolvesDependencies(t *testing.T) {
	svc, _, tid := setup(t)
	ctx := context.Background()

	changed, err := svc.Enable(ctx, tid, "purchasing", registry.SourceTenant)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"catalog", "inventory", "purchasing"}; !slices.Equal(changed, want) {
		t.Fatalf("changed = %v, want %v", changed, want)
	}
	eff, _ := svc.EffectiveModules(ctx, tid)
	for _, c := range []string{"dashboard", "users", "roles", "catalog", "inventory", "purchasing"} {
		if !slices.Contains(eff, c) {
			t.Fatalf("%q should be effective: %v", c, eff)
		}
	}

	if _, err := svc.Disable(ctx, tid, "catalog", false); code(err) != "module_has_dependents" {
		t.Fatalf("expected module_has_dependents, got %v", err)
	}
	off, err := svc.Disable(ctx, tid, "catalog", true)
	if err != nil || len(off) != 3 {
		t.Fatalf("cascade disable = %v, %v", off, err)
	}
	if eff, _ := svc.EffectiveModules(ctx, tid); slices.Contains(eff, "inventory") {
		t.Fatal("cache must be invalidated after disable")
	}
	if _, err := svc.Disable(ctx, tid, "users", false); code(err) != "module_core" {
		t.Fatalf("core modules cannot be disabled: %v", err)
	}
}

func TestPlanGating(t *testing.T) {
	svc, plans, tid := setup(t)
	ctx := context.Background()
	plans.hasPlan, plans.mods = true, []string{"catalog", "payments", "mpesa"}

	if _, err := svc.Enable(ctx, tid, "mpesa", registry.SourceTenant); err != nil {
		t.Fatalf("mpesa (and payments) are in the plan: %v", err)
	}
	_, err := svc.Enable(ctx, tid, "inventory", registry.SourceTenant)
	if code(err) != "module_not_in_plan" {
		t.Fatalf("expected module_not_in_plan, got %v", err)
	}
	// Platform override bypasses the plan.
	if _, err := svc.Enable(ctx, tid, "inventory", registry.SourcePlatform); err != nil {
		t.Fatal(err)
	}
	svc.Invalidate(tid)
	eff, _ := svc.EffectiveModules(ctx, tid)
	if !slices.Contains(eff, "inventory") {
		t.Fatalf("platform override should be effective: %v", eff)
	}

	// A downgrade hides tenant-enabled modules without deleting them.
	plans.mods = []string{"catalog"}
	svc.Invalidate(tid)
	view, _ := svc.TenantView(ctx, tid)
	for _, st := range view {
		if st.Module.Code == "mpesa" && (st.Effective || st.Reason != registry.ReasonNotInPlan) {
			t.Fatalf("mpesa after downgrade: %+v", st)
		}
	}
}

func TestPlatformSwitch(t *testing.T) {
	svc, _, tid := setup(t)
	ctx := context.Background()
	if _, err := svc.Enable(ctx, tid, "mpesa", registry.SourceTenant); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetPlatformActive(ctx, "mpesa", false); err != nil {
		t.Fatal(err)
	}
	eff, _ := svc.EffectiveModules(ctx, tid)
	if slices.Contains(eff, "mpesa") {
		t.Fatal("platform-inactive module must not be effective")
	}
	if _, err := svc.Enable(ctx, tid, "mpesa", registry.SourceTenant); code(err) != "module_unavailable" {
		t.Fatalf("expected module_unavailable, got %v", err)
	}
	if err := svc.SetPlatformActive(ctx, "users", false); code(err) != "module_core" {
		t.Fatalf("core modules cannot be switched off: %v", err)
	}
}

func TestModuleConfig(t *testing.T) {
	svc, _, tid := setup(t)
	ctx := context.Background()
	cfg := json.RawMessage(`{"service_charge_percent": 10, "tables_enabled": true}`)

	if err := svc.SetConfig(ctx, tid, "restaurant", cfg); code(err) != "module_not_enabled" {
		t.Fatalf("config before enable: %v", err)
	}
	if _, err := svc.Enable(ctx, tid, "restaurant", registry.SourceTenant); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetConfig(ctx, tid, "restaurant", cfg); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetConfig(ctx, tid, "restaurant", json.RawMessage(`{"service_charge_percent": 90}`)); code(err) != "validation_failed" {
		t.Fatalf("schema violation: %v", err)
	}
	if err := svc.SetConfig(ctx, tid, "catalog", json.RawMessage(`{}`)); code(err) != "module_no_config" {
		t.Fatalf("module without schema: %v", err)
	}
	got, _ := svc.Config(ctx, tid, "restaurant")
	var m map[string]any
	if json.Unmarshal(got, &m) != nil || m["service_charge_percent"] != float64(10) {
		t.Fatalf("stored config: %s", got)
	}
}
