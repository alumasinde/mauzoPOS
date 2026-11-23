package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"mauzo/internal/modules/roles"
	"mauzo/migrations"
)

// SyncReferenceData mirrors code-defined reference data into the database:
// the permission registry, the module registry, every tenant's system roles
// and template-following role grants. Idempotent; run before serving.
func (c *Container) SyncReferenceData(ctx context.Context) error {
	return SyncRBAC(ctx, c.Log, c.RoleSvc, c.RegistrySvc.SyncRegistry, c.DB.DB)
}

// SyncRBAC is shared by the API boot and `migrate seed`.
func SyncRBAC(ctx context.Context, log *slog.Logger, roleSvc *roles.Service,
	syncModules func(context.Context) error, sqlDB *sql.DB) error {

	steps := []struct {
		name string
		fn   func() error
	}{
		{"permissions", func() error { return roleSvc.SyncPermissions(ctx, log) }},
		{"modules", func() error { return syncModules(ctx) }},
		{"system roles", func() error { return migrations.EnsureSystemRolesAll(ctx, sqlDB) }},
		{"role templates", func() error { return roleSvc.SyncTemplates(ctx) }},
	}
	for _, s := range steps {
		if err := s.fn(); err != nil {
			return fmt.Errorf("sync %s: %w", s.name, err)
		}
	}
	log.Info("reference data synced")
	return nil
}
