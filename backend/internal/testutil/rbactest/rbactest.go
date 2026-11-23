// Package rbactest wires the roles and module-registry services for
// integration tests and syncs reference data into the test database.
package rbactest

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/modules/registry"
	"mauzo/internal/modules/roles"
)

// Services are the RBAC collaborators most tests need.
type Services struct {
	Roles    *roles.Service
	Access   *roles.AccessService
	Registry *registry.Service
}

// Setup syncs the permissions and modules tables of the TEST database and
// returns wired services. plans and branches may be nil.
func Setup(t testing.TB, conn *sqlx.DB, plans registry.PlanSource, branches roles.BranchSource) Services {
	t.Helper()
	// The test binary links only the package under test's permissions; that
	// partial registry is exactly what this test DB should contain.
	authz.MarkComplete()
	reg := registry.NewService(registry.NewRepo(conn), plans)
	repo := roles.NewRepo(conn)
	acc := roles.NewAccessService(repo, reg, branches)
	svc := roles.NewService(repo, acc)
	ctx := context.Background()
	if err := svc.SyncPermissions(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := reg.SyncRegistry(ctx); err != nil {
		t.Fatal(err)
	}
	return Services{Roles: svc, Access: acc, Registry: reg}
}
