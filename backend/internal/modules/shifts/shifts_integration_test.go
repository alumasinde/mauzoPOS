package shifts_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	_ "mauzo/internal/modules/registerall"
	"mauzo/internal/modules/shifts"
	"mauzo/internal/testutil"
)

type fixture struct {
	conn       *sqlx.DB
	svc        *shifts.Service
	repo       *shifts.Repo
	tenantID   uint64
	branchID   uint64
	terminalID uint64
	cashierID  uint64
	ctx        context.Context
	receiptNo  int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"cash_movements",
		"sale_payments", "sale_items", "sales", "cash_shifts", "tenant_counters",
		"held_sales", "pos_terminals",
		"user_branches", "users", "role_permissions", "roles",
		"branches", "tenants")
	ctx := context.Background()
	f := &fixture{conn: conn, ctx: ctx}

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status, business_type)
		 VALUES (?, 'Shift Shop', 'shiftshop', 'active', 'retail')`, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if err := conn.GetContext(ctx, &f.tenantID,
		`SELECT id FROM tenants WHERE slug='shiftshop'`); err != nil {
		t.Fatal(err)
	}
	branchSvc := branches.NewService(conn, branches.NewRepo(conn))
	b, err := branchSvc.Create(ctx, f.tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	if f.branchID, err = branchSvc.BranchByPublicID(ctx, f.tenantID, b.PublicID); err != nil {
		t.Fatal(err)
	}

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO roles (tenant_id, public_id, code, name, is_system)
		 VALUES (?, ?, 'cashier', 'Cashier', 0)`, f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var roleID uint64
	if err := conn.GetContext(ctx, &roleID,
		`SELECT id FROM roles WHERE tenant_id=? AND code='cashier'`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO users (tenant_id, role_id, public_id, name, email, pin_hash, password_hash, status)
		 VALUES (?, ?, ?, 'Till Cashier', 'c@s.co', '', '', 'active')`,
		f.tenantID, roleID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if err := conn.GetContext(ctx, &f.cashierID,
		`SELECT id FROM users WHERE tenant_id=? AND name='Till Cashier'`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO pos_terminals (tenant_id, branch_id, public_id, name, status)
		 VALUES (?,?,?,'Till 1','active')`, f.tenantID, f.branchID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if err := conn.GetContext(ctx, &f.terminalID,
		`SELECT id FROM pos_terminals WHERE tenant_id=? AND name='Till 1'`, f.tenantID); err != nil {
		t.Fatal(err)
	}

	f.repo = shifts.NewRepo(conn)
	f.svc = shifts.NewService(conn, f.repo)
	return f
}

// addCashSale inserts a completed sale with a cash payment stamped to a shift.
func (f *fixture) addCashSale(t *testing.T, shiftID uint64, total string, soldAt time.Time) uint64 {
	t.Helper()
	f.receiptNo++
	res, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO sales (tenant_id, branch_id, terminal_id, shift_id, public_id, receipt_no,
			status, source, total, paid, sold_at)
		 VALUES (?,?,?,?,?,?,'completed','pos',?,?,?)`,
		f.tenantID, f.branchID, f.terminalID, shiftID, security.NewPublicID(),
		f.receiptNo, total, total, soldAt)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO sale_payments (tenant_id, sale_id, method, amount, status)
		 VALUES (?,?,'cash',?,'confirmed')`, f.tenantID, id, total); err != nil {
		t.Fatal(err)
	}
	return uint64(id)
}

func (f *fixture) open(t *testing.T, floatAmt string) *shifts.Shift {
	t.Helper()
	sh, err := f.svc.Open(f.ctx, shifts.OpenInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Float: floatAmt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return sh
}

func TestOpenShiftOncePerTill(t *testing.T) {
	f := newFixture(t)
	sh := f.open(t, "500")
	if sh.Status != shifts.StOpen || sh.FloatAmount.String() != "500" {
		t.Fatalf("shift deschis incorrect: %+v", sh)
	}
	cur, err := f.svc.Current(f.ctx, f.tenantID, f.terminalID)
	if err != nil || cur == nil || cur.ID != sh.ID {
		t.Fatalf("Current nu returneaza shiftul deschis")
	}
	_, err = f.svc.Open(f.ctx, shifts.OpenInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Float: "100",
	})
	if !errors.Is(err, shifts.ErrShiftOpen) {
		t.Fatalf("al doilea open trebuia sa esueze cu ErrShiftOpen, am primit %v", err)
	}
	// Float-ul apare ca primul movement cash_in cu motivul 'float'.
	moves, err := f.repo.Movements(f.ctx, sh.ID)
	if err != nil || len(moves) != 1 || moves[0].Kind != shifts.KindIn {
		t.Fatalf("float-ul nu a fost inregistrat ca movement: %+v", moves)
	}
}

func TestExpectedCashAndCloseVariance(t *testing.T) {
	f := newFixture(t)
	sh := f.open(t, "500")
	f.addCashSale(t, sh.ID, "800", time.Now())
	if _, err := f.svc.AddMovement(f.ctx, shifts.MovementInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Direction: "in", Amount: "100", Reason: "monede",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AddMovement(f.ctx, shifts.MovementInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Direction: "out", Amount: "50", Reason: "platit curier",
	}); err != nil {
		t.Fatal(err)
	}
	// Expected = 500 float + 800 cash + 100 in - 50 out = 1350.
	rep, err := f.svc.Report(f.ctx, *sh)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Expected.String() != "1350" {
		t.Fatalf("expected=1350, am primit %s", rep.Expected)
	}
	if rep.SalesCount != 1 || rep.SalesTotal.String() != "800" {
		t.Fatalf("totals gresite: %+v", rep)
	}

	z, err := f.svc.Close(f.ctx, shifts.CloseInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Declared: "1300",
	})
	if err != nil {
		t.Fatal(err)
	}
	if z.Shift.Status != shifts.StClosed {
		t.Fatal("shiftul nu s-a inchis")
	}
	if z.Shift.Declared == nil || z.Shift.Declared.String() != "1300" ||
		z.Shift.Variance == nil || z.Shift.Variance.String() != "-50" {
		t.Fatalf("variance trebuia -50: %+v", z.Shift)
	}
	if cur, _ := f.svc.Current(f.ctx, f.tenantID, f.terminalID); cur != nil {
		t.Fatal("Current inca returneaza un shift dupa close")
	}
}

func TestShiftForSaleWindow(t *testing.T) {
	f := newFixture(t)
	sh := f.open(t, "0")
	if _, err := f.svc.Close(f.ctx, shifts.CloseInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Declared: "0",
	}); err != nil {
		t.Fatal(err)
	}
	// O vanzare intampinata in fereastra shiftului inchis e atribuita lui.
	closed, err := f.repo.ByPublic(f.ctx, f.tenantID, sh.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.repo.ForSale(f.ctx, f.tenantID, f.branchID, f.terminalID, closed.OpenedAt)
	if err != nil || got != sh.ID {
		t.Fatalf("ForSale in fereastra: got %d, want %d (err %v)", got, sh.ID, err)
	}
	before := sh.OpenedAt.Add(-time.Hour)
	if got, _ := f.repo.ForSale(f.ctx, f.tenantID, f.branchID, f.terminalID, before); got != 0 {
		t.Fatalf("ForSale inainte de shift: got %d, want 0", got)
	}
}

func TestDrawerOpenWithoutShift(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.DrawerOpen(f.ctx, f.tenantID, f.branchID, f.terminalID, f.cashierID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.conn.GetContext(f.ctx, &n,
		`SELECT COUNT(*) FROM cash_movements WHERE tenant_id=? AND kind='drawer_open' AND shift_id IS NULL`,
		f.tenantID); err != nil || n != 1 {
		t.Fatalf("drawer_open fara shift nu a fost auditat: n=%d err=%v", n, err)
	}
	// Cash in/out fara shift deschis este respins.
	if _, err := f.svc.AddMovement(f.ctx, shifts.MovementInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Direction: "in", Amount: "10",
	}); !errors.Is(err, shifts.ErrNoShift) {
		t.Fatalf("cash in fara shift trebuia ErrNoShift, am primit %v", err)
	}
}

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCashOutRoleLimit(t *testing.T) {
	f := newFixture(t)
	f.open(t, "0")
	lim := mustDec(t, "100")
	if _, err := f.svc.AddMovement(f.ctx, shifts.MovementInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Direction: "out", Amount: "150", MaxOut: &lim,
	}); !errors.Is(err, shifts.ErrCashOutLimit) {
		t.Fatalf("cash out peste limita trebuia ErrCashOutLimit, am primit %v", err)
	}
	if _, err := f.svc.AddMovement(f.ctx, shifts.MovementInput{
		TenantID: f.tenantID, BranchID: f.branchID, TerminalID: f.terminalID,
		CashierID: f.cashierID, Direction: "out", Amount: "90", MaxOut: &lim,
	}); err != nil {
		t.Fatalf("cash out sub limita a esuat: %v", err)
	}
}

func TestForceCloseAndReview(t *testing.T) {
	f := newFixture(t)
	sh := f.open(t, "200")
	closed, err := f.svc.ForceClose(f.ctx, f.tenantID, sh.PublicID, f.cashierID, "uitat deschis")
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != shifts.StClosed || closed.Declared != nil || closed.Variance != nil {
		t.Fatalf("force-close trebuia fara blind count: %+v", closed)
	}
	reviewed, err := f.svc.Review(f.ctx, f.tenantID, sh.PublicID,
		shifts.VarFlagged, "verificare de mana", f.cashierID)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.VarianceStatus != shifts.VarFlagged || reviewed.ReviewedBy == nil {
		t.Fatalf("review nu a fost salvat: %+v", reviewed)
	}
}

func TestListAndSummary(t *testing.T) {
	f := newFixture(t)
	sh := f.open(t, "300")
	f.addCashSale(t, sh.ID, "100", time.Now())
	sum, err := f.svc.Summary(f.ctx, f.tenantID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sum.OpenCount != 1 || sum.ExpectedOpen.String() != "400" {
		t.Fatalf("summary gresit: %+v", sum)
	}
	items, total, err := f.svc.List(f.ctx, f.tenantID, shifts.ListFilter{Status: "open"})
	if err != nil || total != 1 || items[0].Terminal != "Till 1" {
		t.Fatalf("list open: total=%d err=%v", total, err)
	}
	if _, total, _ := f.svc.List(f.ctx, f.tenantID, shifts.ListFilter{Status: "closed"}); total != 0 {
		t.Fatal("list closed trebuia 0")
	}
}
