package seeds

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/app"
	_ "mauzo/internal/modules/registerall" // complete permission registry
	modreg "mauzo/internal/modules/registry"
	"mauzo/internal/modules/roles"
)

// RBAC mirrors the permission and module registries into the database and
// brings every tenant's system roles up to date (same as API boot).
type RBAC struct{}

// Name implements Seeder.
func (RBAC) Name() string { return "rbac" }

// Order runs RBAC before anything that creates tenants or users.
func (RBAC) Order() int { return -10 }

// Seed implements Seeder.
func (RBAC) Seed(ctx context.Context, conn *sqlx.DB) error {
	roleSvc, regSvc := rbacServices(conn)
	return app.SyncRBAC(ctx, slog.Default(), roleSvc, regSvc.SyncRegistry, conn.DB)
}

// rbacServices builds the role and module services seeds need (no plan
// gating, no branch source: seeds never resolve per-request access).
func rbacServices(conn *sqlx.DB) (*roles.Service, *modreg.Service) {
	regSvc := modreg.NewService(modreg.NewRepo(conn), nil)
	repo := roles.NewRepo(conn)
	return roles.NewService(repo, roles.NewAccessService(repo, regSvc, nil)), regSvc
}

func init() { Register(RBAC{}) }
