package sales_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/customers"
	"mauzo/internal/modules/inventory"
	"mauzo/internal/modules/pricing"
	_ "mauzo/internal/modules/registerall" // every permission, for locked-role checks
	"mauzo/internal/modules/sales"
	"mauzo/internal/testutil"
)

type fixture struct {
	conn      *sqlx.DB
	svc       *sales.Service
	invSvc    *inventory.Service
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
		"sale_return_items", "sale_returns", "sale_payments", "sale_items", "sales",
		"tenant_counters", "customers", "stock_movements", "stock_balances",
		"stock_locations", "user_branches", "users", "role_permissions", "roles",
		"price_list_items", "price_lists", "promotion_targets", "promotions",
		"products", "tenant_modules", "tenant_settings", "tenant_subscriptions",
		"branches", "tenants")
	ctx := context.Background()

	pub := security.NewPublicID()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status, business_type)
		 VALUES (?, 'Sale Shop', 'saleshop', 'active', 'retail')`, pub); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug='saleshop'`); err != nil {
		t.Fatal(err)
	}
	branchSvc := branches.NewService(branches.NewRepo(conn))
	b, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	branchID, err := branchSvc.BranchByPublicID(ctx, tenantID, b.PublicID)
	if err != nil {
		t.Fatal(err)
	}

	// Cashier role — pos perms, no discount/refund limits.
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
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO role_permissions (role_id, permission) VALUES
		 (?, 'pos.sale.create'), (?, 'pos.discount.apply')`, roleID, roleID); err != nil {
		t.Fatal(err)
	}
	userPub := security.NewPublicID()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO users (tenant_id, public_id, name, username, role_id, all_branches, status)
		 VALUES (?, ?, 'Jane', 'jane', ?, 1, 'active')`, tenantID, userPub, roleID); err != nil {
		t.Fatal(err)
	}
	var cashierID uint64
	if err := conn.GetContext(ctx, &cashierID,
		`SELECT id FROM users WHERE tenant_id = ? AND public_id = ?`, tenantID, userPub); err != nil {
		t.Fatal(err)
	}

	// One priced, stock-tracked product.
	prodPub := security.NewPublicID()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO products (tenant_id, public_id, sku, name, kind, price, vat_rate, track_stock, status)
		 VALUES (?, ?, 'SKU-1', 'Bread', 'standard', 100.00, 16.00, 1, 'active')`,
		tenantID, prodPub); err != nil {
		t.Fatal(err)
	}

	taxRepo := catalog.NewTaxonomyRepo(conn)
	catalogSvc := catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, nil)
	pricingSvc := pricing.NewService(conn, pricing.NewRepo(conn), catalogSvc, branchSvc)
	promoSvc := pricing.NewPromoService(conn, pricing.NewRepo(conn), catalogSvc, catalog.NewTaxonomyService(taxRepo))
	invSvc := inventory.NewService(conn, inventory.NewRepo(conn),
		inventory.NewDocsRepo(conn), catalogSvc, branchSvc)
	custSvc := customers.NewService(customers.NewRepo(conn))
	svc := sales.NewService(conn, sales.NewRepo(conn), catalogSvc, pricingSvc, promoSvc, invSvc, custSvc)

	f := fixture{conn: conn, svc: svc, invSvc: invSvc, tenantID: tenantID,
		branchID: branchID, branchPub: b.PublicID, cashierID: cashierID,
		productID: prodPub, ctx: ctx}
	f.seedStock(t, "50")
	return f
}

func (f fixture) seedStock(t *testing.T, qty string) {
	t.Helper()
	err := db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		return f.invSvc.PostStockInTx(f.ctx, tx, f.tenantID, f.cashierID, f.branchID,
			"receive", "test", "seed", "", "", []inventory.LineRequest{
				{ProductID: f.productID, Qty: qty},
			}, +1)
	})
	if err != nil {
		t.Fatalf("seed stock: %v", err)
	}
}

func (f fixture) balance(t *testing.T) decimal.Decimal {
	t.Helper()
	var q decimal.Decimal
	if err := f.conn.GetContext(f.ctx, &q,
		`SELECT COALESCE(qty,0) FROM stock_balances sb
		 JOIN products p ON p.id = sb.product_id
		 WHERE sb.tenant_id = ? AND sb.branch_id = ? AND p.public_id = ?`,
		f.tenantID, f.branchID, f.productID); err != nil {
		t.Fatalf("balance: %v", err)
	}
	return q
}

func strptr(s string) *string { return &s }

func baseInput(f fixture) sales.CompleteInput {
	return sales.CompleteInput{
		TenantID: f.tenantID, BranchID: f.branchID, CashierID: &f.cashierID,
		ActorID: f.cashierID, ActorType: "user", Source: "pos",
		IdempotencyKey: security.NewPublicID(),
		Lines: []sales.LineInput{
			{ProductID: f.productID, Qty: "2"},
		},
		Payments: []sales.PaymentInput{
			{Method: "cash", Amount: "200.00", Tendered: strptr("250.00")},
		},
	}
}

// A cash sale: totals recomputed server-side, change computed, stock
// decremented, receipt numbered — all atomically.
func TestCompleteCashSale(t *testing.T) {
	f := setup(t)
	d, dup, err := f.svc.Complete(f.ctx, baseInput(f))
	if err != nil || dup {
		t.Fatalf("complete: %v dup=%v", err, dup)
	}
	if d.Sale.ReceiptNo != 1 || d.Sale.Status != "completed" {
		t.Fatalf("sale %+v", d.Sale)
	}
	if !d.Sale.Total.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("total %s", d.Sale.Total)
	}
	if !d.Sale.ChangeDue.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("change %s", d.Sale.ChangeDue)
	}
	if len(d.Items) != 1 || len(d.Payments) != 1 {
		t.Fatalf("items=%d payments=%d", len(d.Items), len(d.Payments))
	}
	// VAT-inclusive: 200 × 16/116 = 27.59
	wantTax, _ := decimal.NewFromString("27.59")
	if !d.Sale.Tax.Equal(wantTax) {
		t.Fatalf("tax %s want %s", d.Sale.Tax, wantTax)
	}
	if got := f.balance(t); !got.Equal(decimal.NewFromInt(48)) {
		t.Fatalf("stock %s want 48", got)
	}
	var audit int
	if err := f.conn.GetContext(f.ctx, &audit,
		`SELECT COUNT(*) FROM audit_log WHERE entity_id = ?`, d.Sale.PublicID); err == nil && audit == 0 {
		t.Fatal("no audit row")
	}
}

// A retry with the same idempotency key returns the original sale — no
// second row, no second stock decrement.
func TestIdempotentRetry(t *testing.T) {
	f := setup(t)
	in := baseInput(f)
	d1, dup, err := f.svc.Complete(f.ctx, in)
	if err != nil || dup {
		t.Fatalf("first: %v", err)
	}
	d2, dup, err := f.svc.Complete(f.ctx, in)
	if err != nil || !dup {
		t.Fatalf("retry: dup=%v err=%v", dup, err)
	}
	if d1.Sale.PublicID != d2.Sale.PublicID || d1.Sale.ReceiptNo != d2.Sale.ReceiptNo {
		t.Fatal("retry produced a different sale")
	}
	var n int
	if err := f.conn.GetContext(f.ctx, &n,
		`SELECT COUNT(*) FROM sales WHERE tenant_id = ?`, f.tenantID); err != nil || n != 1 {
		t.Fatalf("sales rows %d", n)
	}
	if got := f.balance(t); !got.Equal(decimal.NewFromInt(48)) {
		t.Fatalf("stock %s want 48 — double decrement", got)
	}
}

// Split tender, card reference required, under/over-payment rejected.
func TestPaymentRules(t *testing.T) {
	f := setup(t)

	in := baseInput(f)
	in.Payments = []sales.PaymentInput{
		{Method: "cash", Amount: "150.00"},
		{Method: "card", Amount: "50.00", Reference: "AUTH-991"},
	}
	d, _, err := f.svc.Complete(f.ctx, in)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(d.Payments) != 2 || d.Sale.ChangeDue.Sign() != 0 {
		t.Fatalf("split payments %+v", d.Payments)
	}

	in = baseInput(f)
	in.Payments = []sales.PaymentInput{{Method: "card", Amount: "200.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrNeedReference) {
		t.Fatalf("card without ref: %v", err)
	}

	in = baseInput(f)
	in.Payments = []sales.PaymentInput{{Method: "cash", Amount: "150.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrUnderpaid) {
		t.Fatalf("underpaid: %v", err)
	}

	in = baseInput(f)
	in.Payments = []sales.PaymentInput{{Method: "cash", Amount: "250.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrOverpaid) {
		t.Fatalf("overpaid: %v", err)
	}
}

// Credit sales need a customer account and respect the credit limit.
func TestCreditSaleAndLimit(t *testing.T) {
	f := setup(t)

	in := baseInput(f)
	in.Payments = []sales.PaymentInput{{Method: "credit", Amount: "200.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrNeedCustomer) {
		t.Fatalf("credit without customer: %v", err)
	}

	custPub := security.NewPublicID()
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO customers (tenant_id, public_id, name, credit_limit) VALUES (?, ?, 'Wanjiku', 150.00)`,
		f.tenantID, custPub); err != nil {
		t.Fatal(err)
	}
	in.CustomerID = custPub
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrCreditExceeded) {
		t.Fatalf("over limit: %v", err)
	}
	// Raise the limit — sale goes through, balance tracked.
	if _, err := f.conn.ExecContext(f.ctx,
		`UPDATE customers SET credit_limit = 500.00 WHERE public_id = ?`, custPub); err != nil {
		t.Fatal(err)
	}
	in = baseInput(f)
	in.CustomerID = custPub
	in.Payments = []sales.PaymentInput{{Method: "credit", Amount: "200.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in); err != nil {
		t.Fatalf("credit sale: %v", err)
	}
	var outstanding decimal.Decimal
	if err := f.conn.GetContext(f.ctx, &outstanding,
		`SELECT COALESCE(SUM(amount),0) FROM sale_payments WHERE method='credit'`); err != nil {
		t.Fatal(err)
	}
	if !outstanding.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("outstanding %s", outstanding)
	}
}

// Void: payments voided, stock restored, status + audit — one transaction.
func TestVoidSale(t *testing.T) {
	f := setup(t)
	d, _, err := f.svc.Complete(f.ctx, baseInput(f))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Void(f.ctx, f.tenantID, d.Sale.PublicID, "customer walked out",
		f.cashierID, "user"); err != nil {
		t.Fatalf("void: %v", err)
	}
	got, err := f.svc.GetByReceipt(f.ctx, f.tenantID, 0, d.Sale.PublicID)
	if err != nil || got.Sale.Status != "voided" {
		t.Fatalf("status %v", got.Sale.Status)
	}
	if b := f.balance(t); !b.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("stock %s want 50 — void did not restock", b)
	}
	var voided int
	if err := f.conn.GetContext(f.ctx, &voided,
		`SELECT COUNT(*) FROM sale_payments WHERE status='voided'`); err != nil || voided != 1 {
		t.Fatal("payment not voided")
	}
	if err := f.svc.Void(f.ctx, f.tenantID, d.Sale.PublicID, "again",
		f.cashierID, "user"); !errors.Is(err, sales.ErrNotVoidable) {
		t.Fatalf("double void: %v", err)
	}
}

// Partial then full return: returned_qty, restock, negative refund payment,
// status transitions.
func TestReturns(t *testing.T) {
	f := setup(t)
	d, _, err := f.svc.Complete(f.ctx, baseInput(f))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := f.svc.CreateReturn(f.ctx, sales.ReturnInput{
		TenantID: f.tenantID, SalePublicID: d.Sale.PublicID,
		Reason: "stale", RefundMethod: "cash",
		Lines:   []sales.ReturnLineInput{{LineNo: 1, Qty: "1", Restock: true}},
		ActorID: f.cashierID, ActorType: "user",
	})
	if err != nil {
		t.Fatalf("return: %v", err)
	}
	if !rt.RefundAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("refund %s", rt.RefundAmount)
	}
	got, _ := f.svc.GetByReceipt(f.ctx, f.tenantID, 0, d.Sale.PublicID)
	if got.Sale.Status != "partially_returned" {
		t.Fatalf("status %s", got.Sale.Status)
	}
	if b := f.balance(t); !b.Equal(decimal.NewFromInt(49)) {
		t.Fatalf("stock %s want 49", b)
	}
	// Refund row is a negative payment so method totals net out.
	var neg decimal.Decimal
	if err := f.conn.GetContext(f.ctx, &neg,
		`SELECT COALESCE(SUM(amount),0) FROM sale_payments WHERE status='refunded'`); err != nil {
		t.Fatal(err)
	}
	if !neg.Equal(decimal.NewFromInt(-100)) {
		t.Fatalf("refund payment %s", neg)
	}
	// Returning more than remains fails.
	_, err = f.svc.CreateReturn(f.ctx, sales.ReturnInput{
		TenantID: f.tenantID, SalePublicID: d.Sale.PublicID,
		Reason: "again", RefundMethod: "cash",
		Lines:   []sales.ReturnLineInput{{LineNo: 1, Qty: "5"}},
		ActorID: f.cashierID, ActorType: "user",
	})
	if !errors.Is(err, sales.ErrOverReturn) {
		t.Fatalf("over-return: %v", err)
	}
	// The last unit completes the return.
	if _, err := f.svc.CreateReturn(f.ctx, sales.ReturnInput{
		TenantID: f.tenantID, SalePublicID: d.Sale.PublicID,
		Reason: "rest", RefundMethod: "cash",
		Lines:   []sales.ReturnLineInput{{LineNo: 1, Qty: "1"}},
		ActorID: f.cashierID, ActorType: "user",
	}); err != nil {
		t.Fatalf("second return: %v", err)
	}
	got, _ = f.svc.GetByReceipt(f.ctx, f.tenantID, 0, d.Sale.PublicID)
	if got.Sale.Status != "returned" {
		t.Fatalf("status %s", got.Sale.Status)
	}
}

// Atomicity: a line referencing a product that does not exist leaves no
// partial sale, items, payments, or stock movements behind.
func TestAtomicRollback(t *testing.T) {
	f := setup(t)
	in := baseInput(f)
	in.Lines = append(in.Lines, sales.LineInput{ProductID: "zzzzzzzzzzzzzzzzzzzzzzzzzz", Qty: "1"})
	_, _, err := f.svc.Complete(f.ctx, in)
	if err == nil {
		t.Fatal("expected validation failure")
	}
	for _, tbl := range []string{"sales", "sale_items", "sale_payments"} {
		var n int
		if err := f.conn.GetContext(f.ctx, &n,
			`SELECT COUNT(*) FROM `+tbl+` WHERE tenant_id = ?`, f.tenantID); err != nil || n != 0 {
			t.Fatalf("%s rows %d — partial write leaked", tbl, n)
		}
	}
	var saleMoves int
	if err := f.conn.GetContext(f.ctx, &saleMoves,
		`SELECT COUNT(*) FROM stock_movements WHERE tenant_id = ? AND kind = 'sale'`,
		f.tenantID); err != nil || saleMoves != 0 {
		t.Fatalf("sale movements %d — partial write leaked", saleMoves)
	}
	if b := f.balance(t); !b.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("stock %s", b)
	}
}

// Above-limit discount without the permission is refused; a supervisor
// grant or the override permission allows it.
func TestDiscountGate(t *testing.T) {
	f := setup(t)
	in := baseInput(f)
	in.Lines[0].Discount = "50.00" // cashier holds pos.discount.apply
	in.Payments = []sales.PaymentInput{{Method: "cash", Amount: "150.00"}}
	d, _, err := f.svc.Complete(f.ctx, in)
	if err != nil {
		t.Fatalf("permitted discount: %v", err)
	}
	if !d.Sale.Total.Equal(decimal.NewFromInt(150)) || !d.Sale.Discount.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("total %s discount %s", d.Sale.Total, d.Sale.Discount)
	}
	// A line discount on a role *without* pos.discount.apply is refused.
	if _, err := f.conn.ExecContext(f.ctx,
		`DELETE FROM role_permissions WHERE permission = 'pos.discount.apply'`); err != nil {
		t.Fatal(err)
	}
	in2 := baseInput(f)
	in2.Lines[0].Discount = "50.00"
	in2.Payments = []sales.PaymentInput{{Method: "cash", Amount: "150.00"}}
	if _, _, err := f.svc.Complete(f.ctx, in2); !errors.Is(err, sales.ErrDiscountOver) {
		t.Fatalf("discount without perm: %v", err)
	}
}

// Price override without pos.price.override or a supervisor grant fails.
func TestPriceOverrideGate(t *testing.T) {
	f := setup(t)
	in := baseInput(f)
	in.Lines[0].PriceOverride = strptr("80.00")
	if _, _, err := f.svc.Complete(f.ctx, in); !errors.Is(err, sales.ErrPriceOverride) {
		t.Fatalf("override without perm: %v", err)
	}
	sup := f.cashierID
	in.SupervisorID = &sup
	in.IdempotencyKey = security.NewPublicID()
	in.Payments = []sales.PaymentInput{{Method: "cash", Amount: "160.00"}}
	d, _, err := f.svc.Complete(f.ctx, in)
	if err != nil {
		t.Fatalf("supervised override: %v", err)
	}
	if !d.Sale.Total.Equal(decimal.NewFromInt(160)) {
		t.Fatalf("total %s", d.Sale.Total)
	}
}

var _ = httpx.ErrValidation // keep the import if unused paths change

// The owner's role is locked: it holds every permission implicitly and has no
// role_permissions rows. Till-side checks must still see its permissions.
func TestLockedOwnerHoldsEveryPermission(t *testing.T) {
	f := setup(t)
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO roles (tenant_id, public_id, code, name, is_system, is_locked) VALUES (?, ?, 'owner', 'Owner', 1, 1)`,
		f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	pub := security.NewPublicID()
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO users (tenant_id, public_id, name, username, role_id, all_branches, status)
		 SELECT ?, ?, 'Owner', 'owner1', id, 1, 'active' FROM roles WHERE tenant_id = ? AND code = 'owner'`,
		f.tenantID, pub, f.tenantID); err != nil {
		t.Fatal(err)
	}
	var ownerID uint64
	if err := f.conn.GetContext(f.ctx, &ownerID, `SELECT id FROM users WHERE public_id = ?`, pub); err != nil {
		t.Fatal(err)
	}
	a, err := f.svc.Authz(f.ctx, f.tenantID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"pos.sale.create", "pos.receipt.reprint", "sales.refund.create"} {
		if !a.Perms[p] {
			t.Errorf("owner lacks %s", p)
		}
	}
}
