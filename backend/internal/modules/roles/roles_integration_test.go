package roles_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	_ "mauzo/internal/modules/registerall" // full permission registry
	"mauzo/internal/modules/roles"
	"mauzo/internal/testutil"
	"mauzo/internal/testutil/rbactest"
)

func errCode(err error) string {
	var e *httpx.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func setup(t *testing.T) (rbactest.Services, uint64) {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, rbactest.Tables...)
	rbac := rbactest.Setup(t, conn, nil, nil)
	return rbac, rbac.Tenant(t, conn, "roles")
}

func TestProvisionCreatesTemplateRoles(t *testing.T) {
	rbac, tid := setup(t)
	ctx := context.Background()
	list, err := rbac.Roles.List(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != len(authz.RoleTemplates) {
		t.Fatalf("roles = %d, want %d", len(list), len(authz.RoleTemplates))
	}
	byCode := map[string]roles.Role{}
	for _, r := range list {
		byCode[r.Code] = r
	}
	owner := byCode[authz.RoleOwner]
	if !owner.IsLocked || !slices.Contains(owner.Permissions, "roles.role.manage") {
		t.Fatalf("owner must be locked and hold everything: %+v", owner)
	}
	admin := byCode[authz.RoleAdmin]
	if !slices.Contains(admin.Permissions, "roles.role.manage") {
		t.Fatalf("admin template should grant roles.role.manage: %v", admin.Permissions)
	}
	if slices.Contains(byCode[authz.RoleCashier].Permissions, "roles.role.view") {
		t.Fatal("cashier must not see roles")
	}
	// Re-provisioning is idempotent.
	if err := rbac.Roles.ProvisionTenant(ctx, tid); err != nil {
		t.Fatal(err)
	}
	if again, _ := rbac.Roles.List(ctx, tid); len(again) != len(list) {
		t.Fatal("provision should not duplicate roles")
	}
}

func TestCustomRoleLifecycleAndEscalation(t *testing.T) {
	rbac, tid := setup(t)
	ctx := context.Background()
	admin := rbac.Actor(t, tid, 1, authz.RoleAdmin, true)
	cashier := rbac.Actor(t, tid, 2, authz.RoleCashier, false)

	in := roles.Input{Name: "Stock clerk", Permissions: []string{"dashboard.view", "users.employee.view"}}
	role, err := rbac.Roles.Create(ctx, admin, in)
	if err != nil {
		t.Fatal(err)
	}
	if role.Code != "custom_stock_clerk" || role.IsSystem {
		t.Fatalf("custom role: %+v", role)
	}
	if _, err := rbac.Roles.Create(ctx, admin, in); errCode(err) != "role_name_taken" {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := rbac.Roles.Create(ctx, cashier, roles.Input{Name: "Boss", Permissions: []string{"roles.role.manage"}}); errCode(err) != "permission_escalation" {
		t.Fatalf("cashier escalating: %v", err)
	}
	if _, err := rbac.Roles.Create(ctx, admin, roles.Input{Name: "X", Permissions: []string{"platform.tenant.view"}}); errCode(err) != "permission_unknown" {
		t.Fatalf("platform permission in tenant role: %v", err)
	}

	in.Permissions = []string{"dashboard.view"}
	updated, err := rbac.Roles.Update(ctx, admin, role.PublicID, in)
	if err != nil || len(updated.Permissions) != 1 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if err := rbac.Roles.Delete(ctx, admin, role.PublicID); err != nil {
		t.Fatal(err)
	}
}

func TestSystemRoleCustomiseAndReset(t *testing.T) {
	rbac, tid := setup(t)
	ctx := context.Background()
	admin := rbac.Actor(t, tid, 1, authz.RoleAdmin, true)
	owner := rbac.Actor(t, tid, 2, authz.RoleOwner, true)

	ownerRole, _, _ := rbac.Roles.ByCodeForAssign(ctx, tid, authz.RoleOwner)
	if _, err := rbac.Roles.Update(ctx, owner, ownerRole.PublicID, roles.Input{Name: "Owner"}); errCode(err) != "role_locked" {
		t.Fatalf("owner role must be locked: %v", err)
	}
	if err := rbac.Roles.Delete(ctx, owner, ownerRole.PublicID); errCode(err) != "role_system" {
		t.Fatalf("system roles cannot be deleted: %v", err)
	}

	cashier, _, _ := rbac.Roles.ByCodeForAssign(ctx, tid, authz.RoleCashier)
	edited, err := rbac.Roles.Update(ctx, admin, cashier.PublicID, roles.Input{
		Name: "Cashier", Permissions: []string{"pos.access", "pos.sale.create"},
	})
	if err != nil || !edited.Customized {
		t.Fatalf("editing a system role must mark it customized: %+v %v", edited, err)
	}
	// Template sync leaves customized roles alone.
	if err := rbac.Roles.SyncTemplates(ctx); err != nil {
		t.Fatal(err)
	}
	got, _ := rbac.Roles.Get(ctx, tid, cashier.PublicID)
	if len(got.Permissions) != 2 {
		t.Fatalf("sync overwrote a customized role: %v", got.Permissions)
	}
	reset, err := rbac.Roles.Reset(ctx, admin, cashier.PublicID)
	if err != nil || reset.Customized || len(reset.Permissions) <= 2 {
		t.Fatalf("reset: %+v %v", reset, err)
	}
}
