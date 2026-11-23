package auth

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pquerna/otp/totp"

	"mauzo/internal/config"
	"mauzo/internal/core/audit"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/users"
	"mauzo/internal/testutil"
	"mauzo/internal/testutil/rbactest"
)

type sentMail struct{ to, subject, body string }

type fakeMailer struct{ sent []sentMail }

func (f *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	f.sent = append(f.sent, sentMail{to, subject, body})
	return nil
}

type fixture struct {
	svc     *Service
	mailer  *fakeMailer
	conn    *sqlx.DB
	tenants *tenants.Repo
	pUsers  *users.PlatformUserRepo
	users   *users.UserRepo
	params  security.Argon2Params
	rbac    rbactest.Services
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"auth_password_resets", "auth_login_history", "auth_sessions",
		"user_branches", "users", "roles", "platform_users", "tenants", "audit_logs")

	params := security.Argon2Params{MemoryKB: 8192, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	mailer := &fakeMailer{}
	cfg := config.Config{
		AppName:        "MauzoPOS",
		PlatformAppURL: "http://platform.test",
		WebAppURL:      "http://web.test",
		Auth: config.AuthConfig{
			SessionTTL:       time.Hour,
			LoginMaxAttempts: 3,
			LoginLockout:     time.Minute,
		},
	}
	tenantRepo := tenants.NewRepo(conn)
	pUserRepo := users.NewPlatformUserRepo(conn)
	userRepo := users.NewUserRepo(conn)
	auditW := audit.NewWriter(conn, slog.Default())
	rbac := rbactest.Setup(t, conn, nil, userRepo)
	svc := NewService(cfg, ServiceDeps{
		Sessions: NewSessionRepo(conn),
		History:  NewHistoryRepo(conn),
		Resets:   NewResetRepo(conn),
		Platform: NewPlatformAccountStore(pUserRepo),
		Tenant:   NewTenantAccountStore(userRepo),
		Tenants:  NewTenantStore(tenantRepo),
		Mailer:   mailer,
		Audit:    auditW,
		Access:   rbac.Access,
	})
	svc.params = params
	return &fixture{svc, mailer, conn, tenantRepo, pUserRepo, userRepo, params, rbac}
}

func (f *fixture) platformUser(t *testing.T, email, password string) {
	t.Helper()
	hash, err := f.params.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u := users.PlatformUser{
		PublicID: security.NewPublicID(), Name: "Admin", Email: email,
		PasswordHash: hash, Role: "super_admin", Status: "active",
	}
	if err := f.pUsers.Create(context.Background(), &u); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) tenantWithUser(t *testing.T, slug, email, password string) tenants.Tenant {
	t.Helper()
	ctx := context.Background()
	tn := tenants.Tenant{
		PublicID: security.NewPublicID(), Name: "Shop " + slug,
		Slug: slug, Status: tenants.StatusActive,
	}
	if err := f.tenants.Create(ctx, &tn); err != nil {
		t.Fatal(err)
	}
	if err := f.rbac.Roles.ProvisionTenant(ctx, tn.ID); err != nil {
		t.Fatal(err)
	}
	ownerRole, err := f.rbac.Roles.RoleIDByCode(ctx, tn.ID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := f.params.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u := users.User{
		TenantID: tn.ID, PublicID: security.NewPublicID(), Name: "Owner",
		Email: &email, PasswordHash: &hash, RoleID: ownerRole, AllBranches: true, Status: "active",
	}
	if err := f.users.Create(ctx, &u); err != nil {
		t.Fatal(err)
	}
	return tn
}

var meta = RequestMeta{IP: "127.0.0.1", UserAgent: "test"}

func TestPlatformLoginAndAuthenticate(t *testing.T) {
	f := newFixture(t)
	f.platformUser(t, "admin@test.local", "secret1234")
	ctx := context.Background()

	res, err := f.svc.LoginPlatform(ctx, "admin@test.local", "secret1234", meta)
	if err != nil {
		t.Fatal(err)
	}
	if res.MFARequired || res.Token == "" {
		t.Fatalf("expected full session, got %+v", res)
	}
	actor, err := f.svc.Authenticate(ctx, res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Type != identity.ActorPlatformUser || actor.Role != "super_admin" {
		t.Fatalf("bad actor %+v", actor)
	}
	if err := f.svc.Logout(ctx, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(ctx, res.Token); err == nil {
		t.Fatal("expected revoked session to fail")
	}
}

func TestPlatformLoginBadPasswordThenLockout(t *testing.T) {
	f := newFixture(t)
	f.platformUser(t, "admin@test.local", "secret1234")
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := f.svc.LoginPlatform(ctx, "admin@test.local", "wrong", meta); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected invalid credentials, got %v", i, err)
		}
	}
	if _, err := f.svc.LoginPlatform(ctx, "admin@test.local", "secret1234", meta); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("expected lockout, got %v", err)
	}
}

func TestTenantLoginAndSuspend(t *testing.T) {
	f := newFixture(t)
	tn := f.tenantWithUser(t, "demo", "owner@demo.local", "secret1234")
	ctx := context.Background()

	res, err := f.svc.LoginTenant(ctx, "demo", "owner@demo.local", "secret1234", meta)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := f.svc.Authenticate(ctx, res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Type != identity.ActorUser || actor.TenantID != tn.ID {
		t.Fatalf("bad actor %+v", actor)
	}

	// Suspend the tenant: existing sessions die, new logins are rejected.
	if _, err := f.tenants.SetStatus(ctx, tn.ID, tenants.StatusSuspended, nil); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RevokeAllForTenant(ctx, tn.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(ctx, res.Token); err == nil {
		t.Fatal("expected suspended tenant session to fail")
	}
	if _, err := f.svc.LoginTenant(ctx, "demo", "owner@demo.local", "secret1234", meta); err == nil {
		t.Fatal("expected suspended tenant login to fail")
	}
}

func TestMFAFlow(t *testing.T) {
	f := newFixture(t)
	f.platformUser(t, "admin@test.local", "secret1234")
	ctx := context.Background()

	// Enable TOTP directly on the account.
	u, err := f.pUsers.ByEmail(ctx, "admin@test.local")
	if err != nil {
		t.Fatal(err)
	}
	secret, _, err := security.GenerateTOTPSecret("MauzoPOS", u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.pUsers.SetTOTP(ctx, u.ID, &secret, true, nil); err != nil {
		t.Fatal(err)
	}

	res, err := f.svc.LoginPlatform(ctx, "admin@test.local", "secret1234", meta)
	if err != nil {
		t.Fatal(err)
	}
	if !res.MFARequired || res.Ticket == "" || res.Token != "" {
		t.Fatalf("expected MFA ticket, got %+v", res)
	}
	// Pending ticket must not authenticate.
	if _, err := f.svc.Authenticate(ctx, res.Ticket); err == nil {
		t.Fatal("pending MFA ticket authenticated")
	}
	code, err := totpCode(secret)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.svc.VerifyMFA(ctx, res.Ticket, code, meta)
	if err != nil {
		t.Fatal(err)
	}
	if out.Token == "" {
		t.Fatal("expected session token after MFA")
	}
	if _, err := f.svc.Authenticate(ctx, out.Token); err != nil {
		t.Fatalf("verified session rejected: %v", err)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	f := newFixture(t)
	f.platformUser(t, "admin@test.local", "secret1234")
	ctx := context.Background()

	login, err := f.svc.LoginPlatform(ctx, "admin@test.local", "secret1234", meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.ForgotPassword(ctx, identity.ActorPlatformUser, "", "admin@test.local"); err != nil {
		t.Fatal(err)
	}
	if len(f.mailer.sent) != 1 {
		t.Fatalf("expected 1 mail, got %d", len(f.mailer.sent))
	}
	token := extractToken(t, f.mailer.sent[0].body)
	if err := f.svc.ResetPassword(ctx, token, "newpassword9"); err != nil {
		t.Fatal(err)
	}
	// Old session revoked, old password dead, new password works.
	if _, err := f.svc.Authenticate(ctx, login.Token); err == nil {
		t.Fatal("old session still valid after reset")
	}
	if _, err := f.svc.LoginPlatform(ctx, "admin@test.local", "secret1234", meta); err == nil {
		t.Fatal("old password still works")
	}
	if _, err := f.svc.LoginPlatform(ctx, "admin@test.local", "newpassword9", meta); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
	// Token is one-time.
	if err := f.svc.ResetPassword(ctx, token, "another123"); !errors.Is(err, ErrResetTokenInvalid) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

// extractToken pulls the raw reset token out of the emailed link.
func extractToken(t *testing.T, mailBody string) string {
	t.Helper()
	idx := strings.Index(mailBody, "token=")
	if idx < 0 {
		t.Fatalf("no token in mail body: %q", mailBody)
	}
	rest := mailBody[idx+len("token="):]
	if end := strings.IndexAny(rest, "\n\r \t"); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

func totpCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
