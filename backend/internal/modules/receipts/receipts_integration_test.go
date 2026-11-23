package receipts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/customers"
	"mauzo/internal/modules/inventory"
	"mauzo/internal/modules/pricing"
	"mauzo/internal/modules/receipts"
	"mauzo/internal/modules/sales"
	"mauzo/internal/testutil"
	"mauzo/pkg/receiptfmt"
)

type fixture struct {
	conn      *sqlx.DB
	sales     *sales.Service
	svc       *receipts.Service
	tenantID  uint64
	branchID  uint64
	branchPub string
	cashierID uint64
	productID string
	ctx       context.Context
}

func setup(t *testing.T) fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"issued_receipts", "receipt_assignments", "receipt_template_versions", "receipt_templates",
		"receipt_logos", "audit_logs",
		"sale_return_items", "sale_returns", "sale_payments", "sale_items", "sales",
		"tenant_counters", "customers", "stock_movements", "stock_balances",
		"stock_locations", "user_branches", "users", "role_permissions", "roles",
		"products", "tenant_modules", "tenant_settings", "tenant_subscriptions",
		"branches", "tenants")
	ctx := context.Background()

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status, business_type, kra_pin, vat_registered)
		 VALUES (?, 'Receipt Shop', 'rcptshop', 'active', 'minimart', 'P051111111Z', 1)`,
		security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	must(t, conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug='rcptshop'`))
	branchSvc := branches.NewService(branches.NewRepo(conn))
	b, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main Store"})
	must(t, err)
	branchID, err := branchSvc.BranchByPublicID(ctx, tenantID, b.PublicID)
	must(t, err)

	_, err = conn.ExecContext(ctx, `INSERT INTO roles (tenant_id, public_id, code, name, is_system)
		VALUES (?, ?, 'cashier', 'Cashier', 0)`, tenantID, security.NewPublicID())
	must(t, err)
	var roleID uint64
	must(t, conn.GetContext(ctx, &roleID, `SELECT id FROM roles WHERE tenant_id = ?`, tenantID))
	userPub := security.NewPublicID()
	_, err = conn.ExecContext(ctx, `INSERT INTO users (tenant_id, public_id, name, username, role_id, all_branches, status)
		VALUES (?, ?, 'Mary', 'mary', ?, 1, 'active')`, tenantID, userPub, roleID)
	must(t, err)
	var cashierID uint64
	must(t, conn.GetContext(ctx, &cashierID, `SELECT id FROM users WHERE public_id = ?`, userPub))

	prodPub := security.NewPublicID()
	_, err = conn.ExecContext(ctx, `INSERT INTO products (tenant_id, public_id, sku, name, kind, price, vat_rate, track_stock, status)
		VALUES (?, ?, 'SUG-2', 'Kabras Sugar 2kg', 'standard', 320.00, 16.00, 1, 'active')`, tenantID, prodPub)
	must(t, err)

	taxRepo := catalog.NewTaxonomyRepo(conn)
	catalogSvc := catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, nil)
	pricingSvc := pricing.NewService(conn, pricing.NewRepo(conn), catalogSvc, branchSvc)
	promoSvc := pricing.NewPromoService(conn, pricing.NewRepo(conn), catalogSvc, catalog.NewTaxonomyService(taxRepo))
	invSvc := inventory.NewService(conn, inventory.NewRepo(conn), inventory.NewDocsRepo(conn), catalogSvc, branchSvc)
	salesSvc := sales.NewService(conn, sales.NewRepo(conn), catalogSvc, pricingSvc, promoSvc, invSvc,
		customers.NewService(customers.NewRepo(conn)))
	must(t, db.WithTx(ctx, conn, func(tx *sqlx.Tx) error {
		return invSvc.PostStockInTx(ctx, tx, tenantID, cashierID, branchID, "receive", "test", "seed", "", "",
			[]inventory.LineRequest{{ProductID: prodPub, Qty: "100"}}, +1)
	}))
	return fixture{conn: conn, sales: salesSvc, svc: receipts.NewService(conn, receipts.NewRepo(conn)),
		tenantID: tenantID, branchID: branchID, branchPub: b.PublicID, cashierID: cashierID,
		productID: prodPub, ctx: ctx}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f fixture) sale(t *testing.T) sales.SaleDetail {
	t.Helper()
	tendered := "1000.00"
	d, _, err := f.sales.Complete(f.ctx, sales.CompleteInput{
		TenantID: f.tenantID, BranchID: f.branchID, CashierID: &f.cashierID, ActorID: f.cashierID,
		ActorType: "user", Source: "pos", IdempotencyKey: security.NewPublicID(),
		Lines:    []sales.LineInput{{ProductID: f.productID, Qty: "2"}},
		Payments: []sales.PaymentInput{{Method: "cash", Amount: "640.00", Tendered: &tendered}},
	})
	must(t, err)
	return d
}

func setting(t *testing.T, f fixture, key, value string) {
	t.Helper()
	_, err := f.conn.ExecContext(f.ctx, "INSERT INTO tenant_settings (tenant_id, `key`, value) VALUES (?, ?, ?)",
		f.tenantID, key, value)
	must(t, err)
}

// With no template assigned the branch prints the business profile's
// builtin; the snapshot is stored once and re-issuing returns it.
func TestIssueUsesProfileBuiltinAndIsIdempotent(t *testing.T) {
	f := setup(t)
	setting(t, f, "profile.receipt_template", `"minimart"`)
	setting(t, f, "receipt.mpesa_till", `"555111"`)
	sd := f.sale(t)

	rc, err := f.svc.Issue(f.ctx, sd, receipts.IssueInput{Channel: receipts.ChannelPOS, UserID: &f.cashierID})
	must(t, err)
	if rc.Issued.BuiltinKey != "minimart" || rc.Issued.TemplateID != nil || rc.Output.Paper != 58 {
		t.Fatalf("issued %+v paper %d", rc.Issued, rc.Output.Paper)
	}
	txt := rc.Output.Text()
	for _, want := range []string{"Receipt Shop", "2 x Kabras Sugar 2kg", "640.00", "CHANGE", "360.00", "Till No: 555111", "Mary"} {
		if !strings.Contains(txt, want) {
			t.Errorf("missing %q in\n%s", want, txt)
		}
	}
	again, err := f.svc.Issue(f.ctx, sd, receipts.IssueInput{Channel: receipts.ChannelBackoffice})
	must(t, err)
	if again.Issued.PublicID != rc.Issued.PublicID {
		t.Fatal("second Issue created another original")
	}
}

// Templates are versioned; assignment applies per branch; an issued receipt
// keeps the layout version it was printed with after later edits.
func TestTemplateVersionsAssignmentAndSnapshot(t *testing.T) {
	f := setup(t)
	tpl, err := f.svc.CreateTemplate(f.ctx, f.tenantID, f.cashierID, receipts.CreateInput{Name: "Shop", Builtin: "minimart"})
	must(t, err)
	if tpl.Version != 1 || tpl.BasedOn != "minimart" {
		t.Fatalf("created %+v", tpl)
	}

	body := `{"version":1,"paper":80,"blocks":[{"type":"text","text":"LAYOUT TWO {{sale.receipt_no}}"},{"type":"items"},{"type":"totals"}]}`
	v2, err := f.svc.UpdateTemplate(f.ctx, f.tenantID, f.cashierID, tpl.PublicID,
		receipts.UpdateInput{Name: "Shop", Body: json.RawMessage(body), Version: 1})
	must(t, err)
	if v2.Version != 2 || v2.PaperWidth != 80 {
		t.Fatalf("v2 %+v", v2)
	}
	if _, err := f.svc.UpdateTemplate(f.ctx, f.tenantID, f.cashierID, tpl.PublicID,
		receipts.UpdateInput{Name: "Shop", Body: json.RawMessage(body), Version: 1}); !errors.Is(err, receipts.ErrStaleVersion) {
		t.Fatalf("stale save: %v", err)
	}
	// Same body, new name: renamed without a new version.
	renamed, err := f.svc.UpdateTemplate(f.ctx, f.tenantID, f.cashierID, tpl.PublicID,
		receipts.UpdateInput{Name: "Shop 80mm", Body: json.RawMessage(body), Version: 2})
	must(t, err)
	if renamed.Version != 2 || renamed.Name != "Shop 80mm" {
		t.Fatalf("rename %+v", renamed)
	}
	vs, err := f.svc.Versions(f.ctx, f.tenantID, tpl.PublicID)
	must(t, err)
	if len(vs) != 2 {
		t.Fatalf("versions %d", len(vs))
	}

	must(t, f.svc.Assign(f.ctx, f.tenantID, f.cashierID, f.branchPub, tpl.PublicID))
	kit, err := f.svc.Kit(f.ctx, f.tenantID, f.branchID)
	must(t, err)
	if kit.TemplateID != tpl.PublicID || kit.TemplateVersion != 2 || kit.ParsedTemplate().Paper != 80 {
		t.Fatalf("kit %+v", kit)
	}
	if kit.Business.KRAPin != "P051111111Z" || kit.Branch.Name != "Main Store" {
		t.Fatalf("kit header %+v %+v", kit.Business, kit.Branch)
	}

	sd := f.sale(t)
	rc, err := f.svc.Issue(f.ctx, sd, receipts.IssueInput{Channel: receipts.ChannelPOS})
	must(t, err)
	if rc.Issued.TemplateID == nil || *rc.Issued.TemplateVersion != 2 {
		t.Fatalf("issued with %v v%v", rc.Issued.TemplateID, rc.Issued.TemplateVersion)
	}
	if !strings.Contains(rc.Output.Text(), "LAYOUT TWO") {
		t.Fatalf("layout not applied:\n%s", rc.Output.Text())
	}

	// Edit again: the issued receipt must not change.
	_, err = f.svc.UpdateTemplate(f.ctx, f.tenantID, f.cashierID, tpl.PublicID, receipts.UpdateInput{
		Name: "Shop 80mm", Version: 2,
		Body: json.RawMessage(`{"paper":80,"blocks":[{"type":"text","text":"LAYOUT THREE"}]}`)})
	must(t, err)
	view, err := f.svc.ForSale(f.ctx, sd, f.cashierID)
	must(t, err)
	if !strings.Contains(view.Output.Text(), "LAYOUT TWO") || strings.Contains(view.Output.Text(), "LAYOUT THREE") {
		t.Fatalf("snapshot changed:\n%s", view.Output.Text())
	}

	if err := f.svc.ArchiveTemplate(f.ctx, f.tenantID, tpl.PublicID); !errors.Is(err, receipts.ErrTemplateInUse) {
		t.Fatalf("archive assigned: %v", err)
	}
	must(t, f.svc.Assign(f.ctx, f.tenantID, f.cashierID, f.branchPub, ""))
	must(t, f.svc.ArchiveTemplate(f.ctx, f.tenantID, tpl.PublicID))
}

// Reprints are numbered copies of the original with a banner, each audited.
func TestReprintIsMarkedAndAudited(t *testing.T) {
	f := setup(t)
	sd := f.sale(t)
	for want := 1; want <= 2; want++ {
		rc, err := f.svc.Reprint(f.ctx, sd, receipts.ReprintInput{Channel: receipts.ChannelBackoffice, UserID: &f.cashierID})
		must(t, err)
		if rc.Issued.CopyNo != want {
			t.Fatalf("copy %d want %d", rc.Issued.CopyNo, want)
		}
		banner := receiptfmt.ReprintBanner(want)
		if !strings.Contains(rc.Output.Lines[0].Text, banner) {
			t.Fatalf("banner %q missing: %q", banner, rc.Output.Lines[0].Text)
		}
	}
	copies, err := f.svc.Copies(f.ctx, f.tenantID, sd.Sale.ID)
	must(t, err)
	if len(copies) != 3 {
		t.Fatalf("copies %d (want original + 2)", len(copies))
	}
	var audits int
	must(t, f.conn.GetContext(f.ctx, &audits, `SELECT COUNT(*) FROM audit_logs
		WHERE tenant_id = ? AND action = 'receipt.reprinted' AND entity_id = ?`, f.tenantID, sd.Sale.PublicID))
	if audits != 2 {
		t.Fatalf("audit rows %d", audits)
	}
}

func TestLogoValidationAndPreview(t *testing.T) {
	f := setup(t)
	if _, err := f.svc.SetLogo(f.ctx, f.tenantID, f.cashierID, []byte("nope")); !errors.Is(err, receipts.ErrBadImage) {
		t.Fatalf("bad logo: %v", err)
	}
	img := image.NewGray(image.Rect(0, 0, 100, 40))
	for x := range 50 {
		for y := range 40 {
			img.SetGray(x, y, color.Gray{})
		}
	}
	var buf bytes.Buffer
	must(t, png.Encode(&buf, img))
	r, err := f.svc.SetLogo(f.ctx, f.tenantID, f.cashierID, buf.Bytes())
	must(t, err)
	if r.Width != 100 || r.Height != 40 {
		t.Fatalf("logo %dx%d", r.Width, r.Height)
	}
	kit, err := f.svc.Kit(f.ctx, f.tenantID, f.branchID)
	must(t, err)
	if kit.Logo == nil || !kit.Base().Flags.HasLogo {
		t.Fatal("kit without logo")
	}

	b, _ := receiptfmt.BuiltinByKey("supermarket")
	out, logo, err := f.svc.Preview(f.ctx, f.tenantID, b.Raw, f.branchID, nil)
	must(t, err)
	if logo == nil || out.Lines[0].Text == "" || !strings.Contains(out.Text(), "SAMPLE - NOT A RECEIPT") ||
		!strings.Contains(out.Text(), "Main Store") {
		t.Fatalf("preview:\n%s", out.Text())
	}
	hasLogoLine := false
	for _, l := range out.Lines {
		hasLogoLine = hasLogoLine || l.Kind == receiptfmt.LineLogo
	}
	if !hasLogoLine {
		t.Fatal("preview has no logo line")
	}

	_, err = f.svc.CreateTemplate(f.ctx, f.tenantID, f.cashierID, receipts.CreateInput{Name: "Bad",
		Body: json.RawMessage(`{"paper":80,"blocks":[{"type":"text","text":"{{sale.nonsense}}"}]}`)})
	var he *httpx.Error
	if !errors.As(err, &he) || he.Code != "validation_failed" {
		t.Fatalf("invalid body: %v", err)
	}
}
