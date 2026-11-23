package onboarding_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	coresettings "mauzo/internal/core/settings"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/businesstypes"
	"mauzo/internal/modules/onboarding"
	_ "mauzo/internal/modules/settings" // registers settings schemas
	"mauzo/internal/modules/subscriptions"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/terminals"
	"mauzo/internal/modules/users"
	"mauzo/internal/testutil"
	"mauzo/internal/testutil/rbactest"
)

type fakeRevoker struct{}

func (fakeRevoker) RevokeAllForTenant(context.Context, uint64) error { return nil }

func setup(t *testing.T) (*onboarding.Service, *sqlx.DB, uint64) {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"pos_terminals", "branches", "tenant_modules", "tenant_settings",
		"tenant_subscriptions", "plans", "business_profiles", "business_types",
		"user_branches", "users", "roles", "tenants")
	ctx := context.Background()

	// Reference data normally seeded by seeds/catalogue.go (INSERT IGNORE — the
	// row may persist from a prior suite run on the shared test database).
	if _, err := conn.ExecContext(ctx,
		`INSERT IGNORE INTO business_types (code, name, sort) VALUES ('butchery','Butchery',1)`); err != nil {
		t.Fatal(err)
	}
	var typeID uint64
	if err := conn.GetContext(ctx, &typeID, `SELECT id FROM business_types WHERE code='butchery'`); err != nil {
		t.Fatal(err)
	}
	def := `{"modules":["dashboard","settings","sales","payments","weighing_scales"],
		"categories":["Beef","Goat"],
		"settings":{"tax.prices_include_vat":false,"receipt.paper_width":80},
		"receipt_template":"","pos_layout":{"weighted_items":true}}`
	if _, err := conn.ExecContext(ctx,
		`INSERT IGNORE INTO business_profiles (public_id, business_type_id, name, is_default, definition)
		 VALUES (?, ?, 'Standard', 1, ?)`, security.NewPublicID(), typeID, def); err != nil {
		t.Fatal(err)
	}
	mods, _ := json.Marshal([]string{"dashboard", "settings", "catalog", "sales", "payments", "weighing_scales", "reports"})
	if _, err := conn.ExecContext(ctx,
		`INSERT IGNORE INTO plans (public_id, name, price_kes, billing_interval, modules, is_active, sort)
		 VALUES (?, 'Starter', 1500, 'monthly', ?, 1, 1)`, security.NewPublicID(), mods); err != nil {
		t.Fatal(err)
	}

	tRepo := tenants.NewRepo(conn)
	tenantSvc := tenants.NewService(tRepo, fakeRevoker{})
	settingsStore := coresettings.NewStore(conn)
	planSvc := subscriptions.NewService(subscriptions.NewRepo(conn), 14)
	userRepo := users.NewUserRepo(conn)
	rbac := rbactest.Setup(t, conn, planSvc, userRepo)
	planSvc.OnPlanChange(rbac.Registry.Invalidate)
	tenantSvc.AddProvisioner(rbac.Roles)
	btSvc := businesstypes.NewService(businesstypes.NewRepo(conn), settingsStore, tenantSvc, rbac.Registry)
	branchSvc := branches.NewService(branches.NewRepo(conn))
	termSvc := terminals.NewService(terminals.NewRepo(conn), branchSvc)
	userSvc := users.NewUserService(userRepo, security.DefaultArgon2Params, users.Deps{
		Roles: rbac.Roles, Branches: branchSvc, Seats: planSvc,
	})
	svc := onboarding.NewService(onboarding.Deps{
		Tenants: tenantSvc, Types: btSvc, Profile: btSvc, Plans: planSvc,
		Subs: planSvc, Mods: rbac.Registry, Branch: branchSvc, Admins: userSvc,
		Prefs: settingsStore, Term: termSvc,
	})

	tn, err := tenantSvc.Create(ctx, tenants.CreateTenantRequest{Name: "Test Shop", Slug: "test-shop"})
	if err != nil {
		t.Fatal(err)
	}
	return svc, conn, tn.ID
}

func profileID(t *testing.T, conn *sqlx.DB) string {
	t.Helper()
	var id string
	if err := conn.GetContext(context.Background(), &id,
		`SELECT p.public_id FROM business_profiles p
		 JOIN business_types t ON t.id = p.business_type_id
		 WHERE t.code = 'butchery' ORDER BY p.id LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	return id
}

func planID(t *testing.T, conn *sqlx.DB) string {
	t.Helper()
	var id string
	if err := conn.GetContext(context.Background(), &id,
		`SELECT public_id FROM plans LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestOnboardingFlow(t *testing.T) {
	svc, conn, tenantID := setup(t)
	ctx := context.Background()

	if err := svc.SaveBusiness(ctx, tenantID, onboarding.BusinessRequest{
		Name: "Test Shop", KRAPin: "A001", VATRegistered: true,
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.SaveProfile(ctx, tenantID, onboarding.ProfileRequest{ProfileID: profileID(t, conn)}); err != nil {
		t.Fatal(err)
	}

	st, err := svc.Current(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if st.BusinessType == nil || *st.BusinessType != "butchery" {
		t.Fatalf("business_type not applied: %+v", st)
	}
	// Profile modules (+ their dependency catalog) are effective, plus core.
	for _, want := range []string{"sales", "payments", "weighing_scales", "catalog", "users", "roles"} {
		if !slices.Contains(st.Modules, want) {
			t.Fatalf("expected %q after applying the profile, got %v", want, st.Modules)
		}
	}

	// Profile defaults land in tenant_settings without overwriting later edits.
	var vatVal []byte
	if err := conn.GetContext(ctx, &vatVal,
		`SELECT value FROM tenant_settings WHERE tenant_id = ? AND `+"`key`='tax.prices_include_vat'",
		tenantID); err != nil {
		t.Fatal(err)
	}
	if string(vatVal) != "false" {
		t.Fatalf("default setting = %s", vatVal)
	}

	// Applying again is idempotent (no dup modules, no error).
	if err := svc.SaveProfile(ctx, tenantID, onboarding.ProfileRequest{ProfileID: profileID(t, conn)}); err != nil {
		t.Fatalf("re-apply: %v", err)
	}

	if err := svc.SavePlan(ctx, tenantID, planID(t, conn)); err != nil {
		t.Fatal(err)
	}
	st, _ = svc.Current(ctx, tenantID)
	if st.Subscription == nil || st.Subscription.Status != "trialing" || st.Subscription.TrialEndsAt == nil {
		t.Fatalf("subscription: %+v", st.Subscription)
	}

	// Re-assigning the same plan is a no-op (still one open subscription).
	if err := svc.SavePlan(ctx, tenantID, planID(t, conn)); err != nil {
		t.Fatal(err)
	}
	var open int
	if err := conn.GetContext(ctx, &open,
		`SELECT COUNT(*) FROM tenant_subscriptions WHERE tenant_id = ? AND status <> 'cancelled'`,
		tenantID); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("expected 1 open subscription, got %d", open)
	}

	if err := svc.SaveModules(ctx, tenantID,
		onboarding.ModulesRequest{Modules: []string{"dashboard", "sales", "payments"}}); err != nil {
		t.Fatal(err)
	}
	st, _ = svc.Current(ctx, tenantID)
	// Effective set = core modules + chosen + dependencies (catalog for sales);
	// weighing_scales (enabled by the profile) must now be off.
	for _, want := range []string{"dashboard", "settings", "users", "roles", "sales", "payments", "catalog"} {
		if !slices.Contains(st.Modules, want) {
			t.Fatalf("expected %q enabled: %v", want, st.Modules)
		}
	}
	if slices.Contains(st.Modules, "weighing_scales") {
		t.Fatalf("SetModules should disable others: %v", st.Modules)
	}

	// Modules outside the plan are refused.
	if err := svc.SaveModules(ctx, tenantID,
		onboarding.ModulesRequest{Modules: []string{"restaurant"}}); err == nil {
		t.Fatal("expected module_not_in_plan")
	}

	// Unknown module rejected.
	if err := svc.SaveModules(ctx, tenantID,
		onboarding.ModulesRequest{Modules: []string{"nope"}}); err == nil {
		t.Fatal("expected unknown-module validation error")
	}

	br, err := svc.SaveBranch(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	if !br.IsMain {
		t.Fatal("first branch should be main")
	}

	owner := ownerActor(t, conn, tenantID)
	if err := svc.SaveAdmin(ctx, owner, onboarding.AdminRequest{
		Name: "Admin", Email: "a@test.local", Password: "password123",
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.SavePreferences(ctx, tenantID, onboarding.PreferencesRequest{
		Values: map[string]json.RawMessage{
			"receipt.footer": json.RawMessage(`"Karibu"`),
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Unknown key rejected.
	if err := svc.SavePreferences(ctx, tenantID, onboarding.PreferencesRequest{
		Values: map[string]json.RawMessage{"bogus.key": json.RawMessage(`1`)},
	}); err == nil {
		t.Fatal("expected unknown-key validation error")
	}

	term, err := svc.SaveTerminal(ctx, tenantID, terminals.CreateRequest{BranchID: br.PublicID, Name: "Till 1"})
	if err != nil {
		t.Fatal(err)
	}
	if term.Status != terminals.StatusPending {
		t.Fatalf("terminal: %+v", term)
	}

	if err := svc.Complete(ctx, tenantID); err != nil {
		t.Fatal(err)
	}
	st, _ = svc.Current(ctx, tenantID)
	if !st.Done || st.Step < onboarding.LastStep {
		t.Fatalf("completion: %+v", st)
	}
}

func TestOnboardingCompleteGuards(t *testing.T) {
	svc, _, tenantID := setup(t)
	ctx := context.Background()

	if err := svc.Complete(ctx, tenantID); err == nil {
		t.Fatal("expected incomplete error without branch")
	}
	if _, err := svc.SaveBranch(ctx, tenantID,
		branches.CreateRequest{Code: "B1", Name: "B1"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(ctx, tenantID); err == nil {
		t.Fatal("expected incomplete error without terminal")
	}
}

func TestPlanValidation(t *testing.T) {
	svc := subscriptions.NewService(subscriptions.NewRepo(testutil.DB(t)), 14)
	ctx := context.Background()

	if _, err := svc.Create(ctx, subscriptions.PlanRequest{
		Name: "Bad", PriceKES: "0", BillingInterval: "weekly",
	}); err == nil {
		t.Fatal("expected interval validation")
	}
	if _, err := svc.Create(ctx, subscriptions.PlanRequest{
		Name: "Bad", PriceKES: "abc", BillingInterval: "monthly",
	}); err == nil {
		t.Fatal("expected price validation")
	}
	if _, err := svc.Create(ctx, subscriptions.PlanRequest{
		Name: "Bad", PriceKES: "0", BillingInterval: "monthly", Modules: []string{"ghost"},
	}); err == nil {
		t.Fatal("expected module validation")
	}
	if _, err := svc.Assign(ctx, 999, "01ZZZZZZZZZZZZZZZZZZZZZZZZ"); !errors.Is(err, subscriptions.ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestSettingsValidation(t *testing.T) {
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "tenant_settings")
	settingsStore := coresettings.NewStore(conn)
	ctx := context.Background()
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID,
		`SELECT id FROM tenants LIMIT 1`); err != nil {
		// no tenant — create a stub row so the FK is satisfied
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (public_id, name, slug, status) VALUES (?, 'S', 's1', 'active')`,
			security.NewPublicID()); err != nil {
			t.Fatal(err)
		}
		if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants LIMIT 1`); err != nil {
			t.Fatal(err)
		}
	}

	if err := settingsStore.Set(ctx, tenantID, "no.such.key", json.RawMessage(`1`)); !errors.Is(err, coresettings.ErrUnknownKey) {
		t.Fatalf("expected ErrUnknownKey, got %v", err)
	}
	if err := settingsStore.Set(ctx, tenantID, "receipt.paper_width", json.RawMessage(`81`)); err == nil {
		t.Fatal("expected schema violation for paper_width=81")
	}
	if err := settingsStore.Set(ctx, tenantID, "receipt.paper_width", json.RawMessage(`80`)); err != nil {
		t.Fatal(err)
	}
	if err := settingsStore.Set(ctx, tenantID, "tax.default_rate", json.RawMessage(`16`)); err != nil {
		t.Fatal(err)
	}
	// SetDefaults never overwrites an explicit set.
	if err := settingsStore.SetDefaults(ctx, tenantID, map[string]json.RawMessage{
		"receipt.paper_width": json.RawMessage(`58`),
	}); err != nil {
		t.Fatal(err)
	}
	e, err := settingsStore.Get(ctx, tenantID, "receipt.paper_width")
	if err != nil {
		t.Fatal(err)
	}
	if string(e.Value) != "80" {
		t.Fatalf("SetDefaults overwrote explicit value: %s", e.Value)
	}
}

// ownerActor builds a signed-in owner for steps that check the actor.
func ownerActor(t *testing.T, _ *sqlx.DB, tenantID uint64) identity.Actor {
	t.Helper()
	return identity.Actor{
		Type: identity.ActorUser, TenantID: tenantID, Role: "owner",
		Access: identity.Access{Perms: authz.AllOf(), AllBranches: true},
	}
}
