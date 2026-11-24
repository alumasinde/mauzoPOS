package pos_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/desktop/localdb"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/pos"
	"mauzo/internal/modules/terminals"
	"mauzo/internal/testutil"
)

type fixture struct {
	conn     *sqlx.DB
	svc      *pos.Service
	termSvc  *terminals.Service
	tenantID uint64
	branchID uint64
	ctx      context.Context
}

func setup(t *testing.T) fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"held_sales", "pos_terminals", "user_branches", "users", "role_permissions",
		"roles", "price_list_items", "price_lists", "promotion_targets", "promotions",
		"products", "tenant_modules", "tenant_settings", "tenant_subscriptions",
		"branches", "tenants")
	ctx := context.Background()

	pub := security.NewPublicID()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status, business_type)
		 VALUES (?, 'POS Shop', 'posshop', 'active', 'butchery')`, pub); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug='posshop'`); err != nil {
		t.Fatal(err)
	}

	branchSvc := branches.NewService(conn, branches.NewRepo(conn))
	b, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	branchID, err := branchSvc.BranchByPublicID(ctx, tenantID, b.PublicID)
	if err != nil {
		t.Fatal(err)
	}

	// A role with pos.access + supervisor authorize, and a cashier with a PIN.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO permissions (code, scope, module, description) VALUES
		 ('pos.access','tenant','pos','POS access'),
		 ('pos.supervisor.authorize','tenant','pos','Supervisor auth')
		 ON DUPLICATE KEY UPDATE description=VALUES(description)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO roles (tenant_id, public_id, code, name, is_system) VALUES (?, ?, 'cashier', 'Cashier', 0)`,
		tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var roleID uint64
	if err := conn.GetContext(ctx, &roleID,
		`SELECT id FROM roles WHERE tenant_id = ? AND code='cashier'`, tenantID); err != nil {
		t.Fatal(err)
	}
	testutil.EnsurePermissions(t, conn, "pos.access", "pos.supervisor.authorize")
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO role_permissions (role_id, permission) VALUES (?, 'pos.access'), (?, 'pos.supervisor.authorize')`,
		roleID, roleID); err != nil {
		t.Fatal(err)
	}
	params := security.Argon2Params{MemoryKB: 8192, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	hash, err := params.HashPassword("2468")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO users (tenant_id, public_id, name, username, pin_hash, role_id, all_branches, status)
		 VALUES (?, ?, 'Jane', 'jane', ?, ?, 1, 'active')`,
		tenantID, security.NewPublicID(), hash, roleID); err != nil {
		t.Fatal(err)
	}

	termSvc := terminals.NewService(terminals.NewRepo(conn), branchSvc)
	return fixture{
		conn: conn, svc: pos.NewService(conn, pos.NewRepo(conn)),
		termSvc: termSvc, tenantID: tenantID, branchID: branchID, ctx: ctx,
	}
}

func TestActivationAndAuth(t *testing.T) {
	f := setup(t)

	// Register + issue a code through the tenant-facing service.
	var branchPub string
	if err := f.conn.GetContext(f.ctx, &branchPub,
		`SELECT public_id FROM branches WHERE id = ?`, f.branchID); err != nil {
		t.Fatal(err)
	}
	term, err := f.termSvc.CreateAt(f.ctx, f.tenantID,
		terminals.CreateRequest{BranchID: branchPub, Name: "Till T"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := f.termSvc.IssueCode(f.ctx, f.tenantID, term.PublicID, nil)
	if err != nil {
		t.Fatal(err)
	}
	code := issued.Code
	if issued.Repair || len(code) != 11 || code[5] != '-' || strings.ContainsAny(code, "01OIL") {
		t.Fatalf("issued %+v", issued)
	}

	// Bad code rejected.
	if _, err := f.svc.Activate(f.ctx, pos.ActivateRequest{
		Code: "wrong-code", DeviceName: "x", DeviceID: "dev-1",
	}); !errors.Is(err, pos.ErrBadActivationCode) {
		t.Fatalf("want ErrBadActivationCode, got %v", err)
	}

	// Typed in lower case without the dash still works.
	act, err := f.svc.Activate(f.ctx, pos.ActivateRequest{
		Code: strings.ToLower(strings.ReplaceAll(code, "-", "")), DeviceName: "Till T1", DeviceID: "dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if act.Token == "" || act.Terminal.Branch != "Main" || act.Terminal.BusinessType != "butchery" {
		t.Fatalf("activation %+v", act.Terminal)
	}

	// Device token authenticates.
	d, err := f.svc.AuthenticateDevice(f.ctx, act.Token)
	if err != nil {
		t.Fatal(err)
	}
	if d.TenantID != f.tenantID || d.BranchID != f.branchID {
		t.Fatalf("device %+v", d)
	}
	if _, err := f.svc.AuthenticateDevice(f.ctx, "bogus"); err == nil {
		t.Fatal("bogus token accepted")
	}

	// One-time: replaying the code fails.
	if _, err := f.svc.Activate(f.ctx, pos.ActivateRequest{
		Code: code, DeviceName: "x", DeviceID: "dev-2",
	}); !errors.Is(err, pos.ErrBadActivationCode) {
		t.Fatalf("replayed code: %v", err)
	}

	// Re-pair: a new code moves the terminal to a new PC; the old token dies.
	again, err := f.termSvc.IssueCode(f.ctx, f.tenantID, term.PublicID, nil)
	if err != nil || !again.Repair {
		t.Fatalf("re-pair code %+v %v", again, err)
	}
	act2, err := f.svc.Activate(f.ctx, pos.ActivateRequest{Code: again.Code, DeviceName: "New PC", DeviceID: "dev-2"})
	if err != nil {
		t.Fatalf("re-pair: %v", err)
	}
	if _, err := f.svc.AuthenticateDevice(f.ctx, act.Token); err == nil {
		t.Fatal("old device still authenticated after re-pair")
	}
	if _, err := f.svc.AuthenticateDevice(f.ctx, act2.Token); err != nil {
		t.Fatalf("new device: %v", err)
	}

	// Expired codes are reported as expired.
	exp, _ := f.termSvc.IssueCode(f.ctx, f.tenantID, term.PublicID, nil)
	if _, err := f.conn.ExecContext(f.ctx,
		`UPDATE pos_terminals SET activation_code_expires_at = UTC_TIMESTAMP(3) - INTERVAL 1 MINUTE WHERE public_id = ?`,
		term.PublicID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Activate(f.ctx, pos.ActivateRequest{Code: exp.Code, DeviceID: "dev-3"}); !errors.Is(err, pos.ErrActivationExpired) {
		t.Fatalf("expired: %v", err)
	}

	// Disabling signs the device out and blocks new codes.
	disabled := "disabled"
	if _, err := f.termSvc.Update(f.ctx, f.tenantID, term.PublicID, terminals.UpdateRequest{Status: &disabled}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AuthenticateDevice(f.ctx, act2.Token); err == nil {
		t.Fatal("disabled terminal still authenticated")
	}
	if _, err := f.termSvc.IssueCode(f.ctx, f.tenantID, term.PublicID, nil); !errors.Is(err, terminals.ErrIsDisabled) {
		t.Fatalf("code for disabled terminal: %v", err)
	}
}

func TestSyncSignInAndHeld(t *testing.T) {
	f := setup(t)
	var branchPub string
	if err := f.conn.GetContext(f.ctx, &branchPub,
		`SELECT public_id FROM branches WHERE id = ?`, f.branchID); err != nil {
		t.Fatal(err)
	}
	term, err := f.termSvc.CreateAt(f.ctx, f.tenantID,
		terminals.CreateRequest{BranchID: branchPub, Name: "Till T"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	issued, _ := f.termSvc.IssueCode(f.ctx, f.tenantID, term.PublicID, nil)
	act, err := f.svc.Activate(f.ctx, pos.ActivateRequest{Code: issued.Code, DeviceID: "dev-1"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.AuthenticateDevice(f.ctx, act.Token)
	if err != nil {
		t.Fatal(err)
	}

	// A product plus an out-of-window promo and price list: their child rows
	// must not reach the bundle either — the terminal mirrors them under FK
	// constraints and orphan rows would fail the sync apply.
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO products (tenant_id, public_id, sku, name, price) VALUES (?, ?, 'P1', 'Widget', 100)`,
		f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var productID uint64
	if err := f.conn.GetContext(f.ctx, &productID,
		`SELECT id FROM products WHERE tenant_id = ? AND sku = 'P1'`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO price_lists (tenant_id, public_id, name, is_active, valid_from)
		 VALUES (?, ?, 'Future List', 1, DATE_ADD(NOW(), INTERVAL 30 DAY))`,
		f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO price_list_items (list_id, product_id, min_qty, price)
		 VALUES ((SELECT id FROM price_lists WHERE name = 'Future List'), ?, 1, 90)`, productID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO promotions (tenant_id, public_id, name, kind, config, starts_at, is_active)
		 VALUES (?, ?, 'Future Promo', 'percent', '{"percent":10}', DATE_ADD(NOW(), INTERVAL 30 DAY), 1)`,
		f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO promotion_targets (promotion_id, product_id)
		 VALUES ((SELECT id FROM promotions WHERE name = 'Future Promo'), ?)`, productID); err != nil {
		t.Fatal(err)
	}

	// Sync bundle carries cashiers with permissions.
	bundle, err := f.svc.Sync(f.ctx, d, timeZero(), modLister{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Cashiers) != 1 || bundle.Cashiers[0].Name != "Jane" {
		t.Fatalf("cashiers %+v", bundle.Cashiers)
	}
	if len(bundle.Cashiers[0].Permissions) != 2 {
		t.Fatalf("perms %+v", bundle.Cashiers[0].Permissions)
	}
	if len(bundle.PriceLists) != 0 || len(bundle.PriceItems) != 0 ||
		len(bundle.Promotions) != 0 || len(bundle.PromoTargets) != 0 {
		t.Fatalf("orphan pricing rows: lists=%v items=%v promos=%v targets=%v",
			bundle.PriceLists, bundle.PriceItems, bundle.Promotions, bundle.PromoTargets)
	}

	// Wire contract: the JSON the terminal receives must decode into
	// localdb.Bundle and apply under the local FK constraints.
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var local localdb.Bundle
	if err := json.Unmarshal(raw, &local); err != nil {
		t.Fatal(err)
	}
	ldb, err := localdb.Open(f.ctx, filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ldb.Close() }()
	if err := ldb.ApplySync(f.ctx, local); err != nil {
		t.Fatalf("bundle does not apply to the terminal DB: %v", err)
	}
	cashiers, err := ldb.Cashiers(f.ctx)
	if err != nil || len(cashiers) != 1 || cashiers[0].Name != "Jane" {
		t.Fatalf("local cashiers %+v err=%v", cashiers, err)
	}
	prods, err := ldb.Grid(f.ctx, 0, 50)
	if err != nil || len(prods) != 1 || prods[0].Name != "Widget" {
		t.Fatalf("local products %+v err=%v", prods, err)
	}

	// PIN sign-in.
	var userPub string
	if err := f.conn.GetContext(f.ctx, &userPub,
		`SELECT public_id FROM users WHERE username = 'jane'`); err != nil {
		t.Fatal(err)
	}
	var userID uint64
	if err := f.conn.GetContext(f.ctx, &userID,
		`SELECT id FROM users WHERE public_id = ?`, userPub); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.SignIn(f.ctx, d, userID, "0000"); !errors.Is(err, pos.ErrBadPIN) {
		t.Fatalf("wrong pin: %v", err)
	}
	_, perms, err := f.svc.SignIn(f.ctx, d, userID, "2468")
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 2 {
		t.Fatalf("signin perms %+v", perms)
	}

	// Supervisor authorize: Jane's PIN grants pos.supervisor.authorize here.
	res, err := f.svc.Authorize(f.ctx, d, "pos.supervisor.authorize", "2468")
	if err != nil || res.Name != "Jane" || res.Grant == "" {
		t.Fatalf("authorize %+v %v", res, err)
	}
	if _, err := f.svc.Authorize(f.ctx, d, "pos.supervisor.authorize", "0000"); !errors.Is(err, pos.ErrBadPIN) {
		t.Fatalf("bad supervisor pin: %v", err)
	}

	// Held sale lifecycle.
	h, err := f.svc.Hold(f.ctx, d, pos.HoldRequest{
		UserPublicID: userPub, Label: "table 4",
		Cart: json.RawMessage(`[{"product_id":1,"qty":"2"}]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	held, err := f.svc.Held(f.ctx, d)
	if err != nil || len(held) != 1 || held[0].HeldBy != "Jane" {
		t.Fatalf("held %+v", held)
	}
	recalled, err := f.svc.RecallHeld(f.ctx, d, h.PublicID)
	if err != nil || recalled.Cart == "" {
		t.Fatalf("recall %+v", recalled)
	}
	if _, err := f.svc.RecallHeld(f.ctx, d, h.PublicID); !errors.Is(err, pos.ErrHeldNotFound) {
		t.Fatalf("double recall: %v", err)
	}
}

func timeZero() time.Time { return time.Time{} }

type modLister struct{}

func (modLister) EnabledModules(context.Context, uint64) ([]string, error) {
	return []string{"catalog", "pos"}, nil
}
