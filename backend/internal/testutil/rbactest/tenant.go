package rbactest

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

// Tables lists, in delete order, every table these fixtures write. Pass it
// to testutil.Truncate.
var Tables = []string{
	"audit_logs", "auth_sessions", "user_branches", "users", "roles", "tenant_modules",
	"tenant_subscriptions", "plans", "pos_terminals", "branches", "tenant_settings", "tenants",
}

// Tenant inserts a tenant with its system roles and returns its id.
func (s Services) Tenant(t testing.TB, conn *sqlx.DB, slug string) uint64 {
	t.Helper()
	ctx := context.Background()
	res, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status) VALUES (?, ?, ?, 'active')`,
		security.NewPublicID(), "Shop "+slug, slug)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if err := s.Roles.ProvisionTenant(ctx, uint64(id)); err != nil {
		t.Fatal(err)
	}
	return uint64(id)
}

// Branch inserts a branch and returns its id and public id.
func Branch(t testing.TB, conn *sqlx.DB, tenantID uint64, code string) (uint64, string) {
	t.Helper()
	pid := security.NewPublicID()
	res, err := conn.ExecContext(context.Background(),
		`INSERT INTO branches (tenant_id, public_id, code, name) VALUES (?, ?, ?, ?)`,
		tenantID, pid, code, "Branch "+code)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), pid
}

// Actor builds a signed-in tenant user holding the role with template code
// (resolving its real permissions and the tenant's modules), like the auth
// middleware would.
func (s Services) Actor(t testing.TB, tenantID, userID uint64, roleCode string, allBranches bool, branches ...uint64) identity.Actor {
	t.Helper()
	ctx := context.Background()
	role, set, err := s.Roles.ByCodeForAssign(ctx, tenantID, roleCode)
	if err != nil {
		t.Fatal(err)
	}
	mods, err := s.Registry.EffectiveModules(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]bool{}
	for _, c := range mods {
		m[c] = true
	}
	return identity.Actor{
		Type: identity.ActorUser, ID: userID, TenantID: tenantID, Role: role.Code, RoleID: role.ID,
		Access: identity.Access{Perms: set, Modules: m, AllBranches: allBranches || set.IsAll(), Branches: branches},
	}
}

// Set is a convenience for building permission sets in tests.
func Set(codes ...string) authz.Set { return authz.NewSet(codes...) }
