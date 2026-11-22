package users

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/testutil"
)

var testParams = security.Argon2Params{
	MemoryKB: 8192, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
}

func newDB(t *testing.T) *sqlx.DB {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "auth_sessions", "users", "platform_users", "tenants")
	return conn
}

func TestPlatformUserLifecycle(t *testing.T) {
	conn := newDB(t)
	ctx := context.Background()
	svc := NewPlatformUserService(NewPlatformUserRepo(conn), testParams)
	repo := NewPlatformUserRepo(conn)

	u, err := svc.Create(ctx, CreatePlatformUserRequest{
		Name: "Support One", Email: "s1@test.local", Password: "password123", Role: authz.RoleSupport,
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.PublicID == "" || u.Status != StatusActive || u.CreatedAt.IsZero() {
		t.Fatalf("bad user %+v", u)
	}

	rows, total, err := svc.List(ctx, httpx.Page{Page: 1, PerPage: 10, Query: "s1@"})
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("list: %v total=%d", err, total)
	}

	name, role := "Support Uno", authz.RoleBilling
	upd, err := svc.Update(ctx, u.PublicID, UpdatePlatformUserRequest{Name: &name, Role: &role})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != name || upd.Role != role {
		t.Fatalf("update: %+v", upd)
	}

	if _, err := svc.SetPassword(ctx, u.PublicID, "newpass999"); err != nil {
		t.Fatal(err)
	}
	fresh, err := repo.ByPublicID(ctx, u.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if err := security.VerifyPassword("newpass999", fresh.PasswordHash); err != nil {
		t.Fatal("new password does not verify")
	}
	if err := security.VerifyPassword("password123", fresh.PasswordHash); err == nil {
		t.Fatal("old password still verifies")
	}

	// Disable → login-blocking status, then re-enable.
	dis, err := svc.SetStatus(ctx, u.PublicID, StatusDisabled, 0)
	if err != nil {
		t.Fatal(err)
	}
	if dis.Status != StatusDisabled {
		t.Fatalf("disable: %+v", dis)
	}
	if _, err := svc.SetStatus(ctx, u.PublicID, StatusDisabled, 0); !errors.Is(err, ErrAlreadyDisabled) {
		t.Fatalf("expected ErrAlreadyDisabled, got %v", err)
	}
	en, err := svc.SetStatus(ctx, u.PublicID, StatusActive, 0)
	if err != nil || en.Status != StatusActive {
		t.Fatalf("enable: %v %+v", err, en)
	}
}

func TestPlatformUserGuards(t *testing.T) {
	conn := newDB(t)
	ctx := context.Background()
	svc := NewPlatformUserService(NewPlatformUserRepo(conn), testParams)

	admin, err := svc.Create(ctx, CreatePlatformUserRequest{
		Name: "Root", Email: "root@test.local", Password: "password123", Role: authz.RoleSuperAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SetStatus(ctx, admin.PublicID, StatusDisabled, admin.ID); err == nil {
		t.Fatal("self-disable should be rejected")
	}
	if _, err := svc.SetStatus(ctx, admin.PublicID, StatusDisabled, 0); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("expected ErrLastSuperAdmin, got %v", err)
	}
	if _, err := svc.Create(ctx, CreatePlatformUserRequest{
		Name: "Dup", Email: "root@test.local", Password: "password123", Role: authz.RoleViewer,
	}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
	if _, err := svc.Create(ctx, CreatePlatformUserRequest{
		Name: "Weak", Email: "weak@test.local", Password: "short", Role: authz.RoleViewer,
	}); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
	if _, err := svc.Create(ctx, CreatePlatformUserRequest{
		Name: "Bad", Email: "bad@test.local", Password: "password123", Role: "wizard",
	}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestTenantUserCreate(t *testing.T) {
	conn := newDB(t)
	ctx := context.Background()

	tn := tenants.Tenant{PublicID: security.NewPublicID(), Name: "T", Slug: "tt", Status: tenants.StatusActive}
	if err := tenants.NewRepo(conn).Create(ctx, &tn); err != nil {
		t.Fatal(err)
	}

	svc := NewUserService(NewUserRepo(conn), testParams)
	u, err := svc.CreateTenantUser(ctx, tn.ID, "Cashier", "C@T.Local", "c1", "password123", UserRoleCashier)
	if err != nil {
		t.Fatal(err)
	}
	if u.PublicID == "" || u.Email == nil || *u.Email != "c@t.local" {
		t.Fatalf("bad user %+v", u)
	}

	repo := NewUserRepo(conn)
	got, err := repo.ByIdentity(ctx, tn.ID, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID || got.PasswordHash == nil {
		t.Fatalf("by identity: %+v", got)
	}
	if err := security.VerifyPassword("password123", *got.PasswordHash); err != nil {
		t.Fatal("password does not verify")
	}
}
