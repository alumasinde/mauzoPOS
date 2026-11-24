// Package shifts owns cashier shift sessions: open with a float, manual cash
// movements, blind-count close, X/Z report data and variance review.
package shifts

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Service is the shifts surface for tenant + till endpoints.
type Service struct {
	conn *sqlx.DB
	repo *Repo
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo) *Service {
	return &Service{conn: conn, repo: repo}
}

// OpenInput is POST /pos/shift/open.
type OpenInput struct {
	TenantID   uint64
	BranchID   uint64
	TerminalID uint64
	CashierID  uint64
	Float      string // decimal
}

// Open starts a drawer session. One per terminal — a concurrent open fails
// on the open-marker unique key.
func (s *Service) Open(ctx context.Context, in OpenInput) (*Shift, error) {
	if cur, err := s.repo.Current(ctx, in.TenantID, in.TerminalID); err != nil {
		return nil, err
	} else if cur != nil {
		return nil, ErrShiftOpen
	}
	floatAmt, err := decimal.NewFromString(in.Float)
	if err != nil || floatAmt.IsNegative() {
		return nil, ErrBadFloat
	}
	sh := &Shift{
		TenantID: in.TenantID, BranchID: in.BranchID, TerminalID: in.TerminalID,
		PublicID: security.NewPublicID(), OpenedBy: in.CashierID,
		Status: StOpen, FloatAmount: floatAmt.Round(2), OpenedAt: time.Now(),
		VarianceStatus: VarPending,
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if err := s.repo.Insert(ctx, tx, sh); err != nil {
			if isDup(err) {
				return ErrShiftOpen
			}
			return err
		}
		// The float is the first cash_in — the drawer statement balances.
		m := &Movement{
			TenantID: in.TenantID, ShiftID: &sh.ID, BranchID: in.BranchID,
			TerminalID: in.TerminalID, PublicID: security.NewPublicID(),
			Kind: KindIn, Amount: sh.FloatAmount, Reason: "float",
			CreatedBy: in.CashierID,
		}
		return s.repo.InsertMovement(ctx, tx, m)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Current(ctx, in.TenantID, in.TerminalID)
}

// Current returns the till's open shift (nil when none).
func (s *Service) Current(ctx context.Context, tenantID, terminalID uint64) (*Shift, error) {
	return s.repo.Current(ctx, tenantID, terminalID)
}

// ShiftForSale resolves which shift a completed sale belongs to — the
// terminal's open shift, or the closed window covering a late offline sale.
func (s *Service) ShiftForSale(ctx context.Context, tenantID, branchID, terminalID uint64,
	soldAt time.Time) (uint64, error) {
	return s.repo.ForSale(ctx, tenantID, branchID, terminalID, soldAt)
}

// MovementInput is a manual cash in / cash out.
type MovementInput struct {
	TenantID   uint64
	BranchID   uint64
	TerminalID uint64
	CashierID  uint64
	Direction  string // in | out
	Amount     string
	Reason     string
	MaxOut     *decimal.Decimal // role limit; nil = unlimited
}

// AddMovement records cash in/out on the till's open shift.
func (s *Service) AddMovement(ctx context.Context, in MovementInput) (*Movement, error) {
	amount, err := decimal.NewFromString(in.Amount)
	if err != nil || !amount.IsPositive() {
		return nil, ErrBadAmount
	}
	var kind string
	switch in.Direction {
	case "in":
		kind = KindIn
	case "out":
		kind = KindOut
		if in.MaxOut != nil && amount.GreaterThan(*in.MaxOut) {
			return nil, ErrCashOutLimit
		}
	default:
		return nil, ErrBadDirection
	}
	cur, err := s.repo.Current(ctx, in.TenantID, in.TerminalID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNoShift
	}
	m := &Movement{
		TenantID: in.TenantID, ShiftID: &cur.ID, BranchID: in.BranchID,
		TerminalID: in.TerminalID, PublicID: security.NewPublicID(),
		Kind: kind, Amount: amount.Round(2), Reason: in.Reason,
		CreatedBy: in.CashierID,
	}
	if err := s.repo.InsertMovement(ctx, s.conn, m); err != nil {
		return nil, err
	}
	m.CreatedByName = cur.OpenedByName
	return m, nil
}

// DrawerOpen audits a no-sale drawer kick. Recorded under the open shift
// when there is one; standalone otherwise.
func (s *Service) DrawerOpen(ctx context.Context, tenantID, branchID, terminalID,
	cashierID uint64) error {
	var shiftID *uint64
	if cur, err := s.repo.Current(ctx, tenantID, terminalID); err != nil {
		return err
	} else if cur != nil {
		shiftID = &cur.ID
	}
	return s.repo.InsertMovement(ctx, s.conn, &Movement{
		TenantID: tenantID, ShiftID: shiftID, BranchID: branchID,
		TerminalID: terminalID, PublicID: security.NewPublicID(),
		Kind: KindDrawer, Amount: decimal.Zero, CreatedBy: cashierID,
	})
}

// expected computes what the drawer should hold for a shift.
func (s *Service) expected(ctx context.Context, sh Shift) (cash, refunds, cin, cout decimal.Decimal, err error) {
	totals, err := s.repo.MethodTotals(ctx, sh.ID)
	if err != nil {
		return cash, refunds, cin, cout, err
	}
	for _, t := range totals {
		if t.Method == "cash" {
			cash = t.Total
		}
	}
	if refunds, err = s.repo.CashRefunded(ctx, sh); err != nil {
		return cash, refunds, cin, cout, err
	}
	if cin, cout, err = s.repo.MovementSums(ctx, sh.ID); err != nil {
		return cash, refunds, cin, cout, err
	}
	return cash, refunds, cin, cout, nil
}

// Report builds the X-report for an open shift or Z-report for a closed one.
func (s *Service) Report(ctx context.Context, sh Shift) (*Report, error) {
	count, salesTotal, err := s.repo.SalesCount(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	methods, err := s.repo.MethodTotals(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	cash, refunds, cin, cout, err := s.expected(ctx, sh)
	if err != nil {
		return nil, err
	}
	moves, err := s.repo.Movements(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	// Expected = numerarul aplicat + mișcările nete − refundurile cash. Floatul
	// este deja înregistrat ca primul cash_in — în raport îl afișăm pe linia lui
	// și-l scădem din „Cash in" ca să nu apară de două ori.
	extraIn := cin.Sub(sh.FloatAmount)
	if extraIn.IsNegative() {
		extraIn = decimal.Zero
	}
	return &Report{
		Shift: sh, SalesCount: count, SalesTotal: salesTotal, Methods: methods,
		CashSales: cash, CashRefund: refunds, CashIn: extraIn, CashOut: cout,
		Expected:  cash.Add(cin).Sub(cout).Sub(refunds),
		Movements: moves,
	}, nil
}

// ReportByPublic resolves a shift then reports on it.
func (s *Service) ReportByPublic(ctx context.Context, tenantID uint64, publicID string) (*Report, error) {
	sh, err := s.repo.ByPublic(ctx, tenantID, publicID)
	if err != nil {
		return nil, err
	}
	return s.Report(ctx, sh)
}

// CloseInput is POST /pos/shift/close — a blind count.
type CloseInput struct {
	TenantID   uint64
	BranchID   uint64
	TerminalID uint64
	CashierID  uint64 // internal id
	CashierPub string // public id (ownership check)
	Declared   string
	Note       string
	Force      bool // back-office force-close, no blind count
}

// Close ends the till's open shift with a blind count and returns the
// Z-report. Force skips the count (variance left unset).
func (s *Service) Close(ctx context.Context, in CloseInput) (*Report, error) {
	cur, err := s.repo.Current(ctx, in.TenantID, in.TerminalID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNoShift
	}
	declared := decimal.Zero
	if !in.Force {
		declared, err = decimal.NewFromString(in.Declared)
		if err != nil || declared.IsNegative() {
			return nil, ErrBadFloat
		}
	}
	cash, refunds, cin, cout, err := s.expected(ctx, *cur)
	if err != nil {
		return nil, err
	}
	expected := cash.Add(cin).Sub(cout).Sub(refunds)
	now := time.Now()
	cur.Status = StClosed
	cur.ClosedAt = &now
	cur.ClosedBy = &in.CashierID
	cur.Expected = &expected
	cur.CloseNote = in.Note
	if !in.Force {
		cur.Declared = &declared
		v := declared.Sub(expected)
		cur.Variance = &v
	}
	if err := db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		return s.repo.Close(ctx, tx, *cur)
	}); err != nil {
		return nil, err
	}
	return s.Report(ctx, *cur)
}

// ForceClose ends a shift from the back office without a blind count.
func (s *Service) ForceClose(ctx context.Context, tenantID uint64, publicID string,
	byUserID uint64, note string) (*Shift, error) {
	sh, err := s.repo.ByPublic(ctx, tenantID, publicID)
	if err != nil {
		return nil, err
	}
	if sh.Status == StClosed {
		return nil, ErrShiftClosed
	}
	rep, err := s.Close(ctx, CloseInput{
		TenantID: tenantID, BranchID: sh.BranchID, TerminalID: sh.TerminalID,
		CashierID: byUserID, Note: note, Force: true,
	})
	if err != nil {
		return nil, err
	}
	return &rep.Shift, nil
}

// Review approves or flags a closed shift's variance.
func (s *Service) Review(ctx context.Context, tenantID uint64, publicID,
	status, note string, byUserID uint64) (*Shift, error) {
	if status != VarApproved && status != VarFlagged && status != VarPending {
		return nil, httpx.ErrValidation.WithDetails(map[string]any{
			"status": "approved | flagged | pending"})
	}
	sh, err := s.repo.ByPublic(ctx, tenantID, publicID)
	if err != nil {
		return nil, err
	}
	if sh.Status != StClosed {
		return nil, ErrNoShift
	}
	if err := s.repo.ReviewVariance(ctx, tenantID, sh.ID, status, note, byUserID); err != nil {
		return nil, err
	}
	out, err := s.repo.ByPublic(ctx, tenantID, publicID)
	return &out, err
}

// BranchIDByPublic resolves a public branch id for list filters.
func (s *Service) BranchIDByPublic(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	return s.repo.BranchIDByPublic(ctx, tenantID, publicID)
}

// List returns the back-office page of shifts.
func (s *Service) List(ctx context.Context, tenantID uint64, f ListFilter) ([]Shift, int, error) {
	return s.repo.List(ctx, tenantID, f)
}

// Summary returns the dashboard rollup.
func (s *Service) Summary(ctx context.Context, tenantID uint64, branchIDs []uint64) (Summary, error) {
	return s.repo.Summary(ctx, tenantID, branchIDs)
}
