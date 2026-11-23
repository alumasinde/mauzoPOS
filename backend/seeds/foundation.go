package seeds

import (
	"context"
	"os"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/users"
)

// Foundation seeds the first platform super admin and a demo tenant with an
// owner account so a fresh development database is immediately usable.
// Development only — skipped when APP_ENV is anything else.
type Foundation struct{}

// Name implements Seeder.
func (Foundation) Name() string { return "foundation" }

// Seed implements Seeder.
func (Foundation) Seed(ctx context.Context, conn *sqlx.DB) error {
	if os.Getenv("APP_ENV") != "development" {
		return nil
	}
	params := security.DefaultArgon2Params

	var admins int
	if err := conn.GetContext(ctx, &admins, `SELECT COUNT(*) FROM platform_users`); err != nil {
		return err
	}
	if admins == 0 {
		svc := users.NewPlatformUserService(users.NewPlatformUserRepo(conn), params)
		if _, err := svc.Create(ctx, users.CreatePlatformUserRequest{
			Name: "Super Admin", Email: "admin@mauzo.local",
			Password: "ChangeMe123!", Role: authz.RoleSuperAdmin,
		}); err != nil {
			return err
		}
	}

	var n int
	if err := conn.GetContext(ctx, &n, `SELECT COUNT(*) FROM tenants`); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tenant := tenants.Tenant{
		PublicID: security.NewPublicID(),
		Name:     "Demo Shop",
		Slug:     "demo",
		Status:   tenants.StatusActive,
	}
	if err := tenants.NewRepo(conn).Create(ctx, &tenant); err != nil {
		return err
	}
	roleSvc, _ := rbacServices(conn)
	if err := roleSvc.ProvisionTenant(ctx, tenant.ID); err != nil {
		return err
	}
	usvc := users.NewUserService(users.NewUserRepo(conn), params, users.Deps{
		Roles: roleSvc, Branches: branches.NewService(branches.NewRepo(conn)),
	})
	_, err := usvc.CreateTenantUser(ctx, tenant.ID,
		"Owner", "owner@demo.local", "", "ChangeMe123!", authz.RoleOwner)
	return err
}

func init() { Register(Foundation{}) }
