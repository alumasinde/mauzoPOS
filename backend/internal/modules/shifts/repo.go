package shifts

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

func isDup(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}

// Repo is the shifts store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const shiftCols = `s.id, s.tenant_id, s.branch_id, s.terminal_id, s.public_id,
	s.opened_by, s.closed_by, s.status, s.float_amount, s.opened_at, s.closed_at,
	s.expected_amount, s.declared_amount, s.variance, s.close_note,
	s.variance_status, s.variance_note, s.reviewed_by, s.reviewed_at,
	uo.name AS opened_by_name, COALESCE(uc.name, '') AS closed_by_name,
	t.name AS terminal_name, b.name AS branch_name`

const shiftJoins = `FROM cash_shifts s
	JOIN users uo ON uo.id = s.opened_by
	LEFT JOIN users uc ON uc.id = s.closed_by
	JOIN pos_terminals t ON t.id = s.terminal_id
	JOIN branches b ON b.id = s.branch_id`

// Insert creates a shift row inside tx (open_marker enforces one open per
// terminal; a duplicate key means a race lost to another open).
func (r *Repo) Insert(ctx context.Context, tx *sqlx.Tx, s *Shift) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO cash_shifts (tenant_id, branch_id, terminal_id, public_id,
			opened_by, status, float_amount, opened_at, open_marker)
		 VALUES (?,?,?,?,?,?,?,?,1)`,
		s.TenantID, s.BranchID, s.TerminalID, s.PublicID,
		s.OpenedBy, StOpen, s.FloatAmount, s.OpenedAt)
	if err != nil {
		return err
	}
	id, ierr := res.LastInsertId()
	if ierr != nil {
		return ierr
	}
	s.ID = uint64(id) //nolint:gosec // LastInsertId is non-negative on success
	return err
}

// Current returns the open shift on a terminal, or nil.
func (r *Repo) Current(ctx context.Context, tenantID, terminalID uint64) (*Shift, error) {
	var s Shift
	err := r.db.GetContext(ctx, &s,
		`SELECT `+shiftCols+` `+shiftJoins+
			` WHERE s.tenant_id = ? AND s.terminal_id = ? AND s.status = 'open'`,
		tenantID, terminalID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ForSale resolves the shift a sale belongs to: the terminal's open shift,
// else the closed shift whose [opened_at, closed_at) window covered sold_at
// on that terminal (delayed offline upload). Zero means unattributed.
func (r *Repo) ForSale(ctx context.Context, tenantID, branchID, terminalID uint64, soldAt time.Time) (uint64, error) {
	var id uint64
	err := r.db.GetContext(ctx, &id,
		`SELECT s.id FROM cash_shifts s
		 WHERE s.tenant_id = ? AND s.branch_id = ? AND s.terminal_id = ?
		   AND s.opened_at <= ? AND (s.closed_at IS NULL OR s.closed_at > ?)
		 ORDER BY s.status = 'open' DESC, s.opened_at DESC
		 LIMIT 1`, tenantID, branchID, terminalID, soldAt, soldAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

// BranchIDByPublic resolves a branch public id to the internal id within the
// tenant (used by the list filter).
func (r *Repo) BranchIDByPublic(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	var id uint64
	err := r.db.GetContext(ctx, &id,
		`SELECT id FROM branches WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// ByPublic fetches one shift by public id (joined view).
func (r *Repo) ByPublic(ctx context.Context, tenantID uint64, publicID string) (Shift, error) {
	var s Shift
	err := r.db.GetContext(ctx, &s,
		`SELECT `+shiftCols+` `+shiftJoins+` WHERE s.tenant_id = ? AND s.public_id = ?`,
		tenantID, publicID)
	return s, err
}

// ListFilter narrows the back-office list.
type ListFilter struct {
	BranchIDs []uint64 // nil = all the actor can see
	Status    string   // "" | open | closed
	CashierID *uint64
	Since     *time.Time
	Until     *time.Time
	Page      int
	PerPage   int
}

// List returns shifts newest-first plus the total for pagination.
func (r *Repo) List(ctx context.Context, tenantID uint64, f ListFilter) ([]Shift, int, error) {
	where := ` WHERE s.tenant_id = ?`
	args := []any{tenantID}
	if f.Status != "" {
		where += ` AND s.status = ?`
		args = append(args, f.Status)
	}
	if f.CashierID != nil {
		where += ` AND s.opened_by = ?`
		args = append(args, *f.CashierID)
	}
	if f.Since != nil {
		where += ` AND s.opened_at >= ?`
		args = append(args, *f.Since)
	}
	if f.Until != nil {
		where += ` AND s.opened_at < ?`
		args = append(args, *f.Until)
	}
	if len(f.BranchIDs) > 0 {
		where += ` AND s.branch_id IN (?)`
		args = append(args, f.BranchIDs)
	}
	var total int
	countQ, countArgs, err := sqlx.In(`SELECT COUNT(*) `+shiftJoins+where, args...)
	if err != nil {
		return nil, 0, err
	}
	if err := r.db.GetContext(ctx, &total, countQ, countArgs...); err != nil {
		return nil, 0, err
	}
	page := max(f.Page, 1)
	per := f.PerPage
	if per < 1 || per > 200 {
		per = 50
	}
	listQ, listArgs, err := sqlx.In(
		`SELECT `+shiftCols+` `+shiftJoins+where+` ORDER BY s.opened_at DESC LIMIT ? OFFSET ?`,
		append(args, per, (page-1)*per)...)
	if err != nil {
		return nil, 0, err
	}
	items := []Shift{}
	if err := r.db.SelectContext(ctx, &items, listQ, listArgs...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// InsertMovement records a drawer movement. Accepts either the pool or an
// open transaction — inside Open the float movement must share the shift's
// tx, otherwise the FK check deadlocks on the uncommitted parent row.
func (r *Repo) InsertMovement(ctx context.Context, ex sqlx.ExtContext, m *Movement) error {
	res, err := ex.ExecContext(ctx,
		`INSERT INTO cash_movements (tenant_id, shift_id, branch_id, terminal_id,
			public_id, kind, amount, reason, created_by)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		m.TenantID, m.ShiftID, m.BranchID, m.TerminalID,
		m.PublicID, m.Kind, m.Amount, m.Reason, m.CreatedBy)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = uint64(id) //nolint:gosec // LastInsertId is non-negative on success
	return nil
}

// Movements lists a shift's manual movements in order.
func (r *Repo) Movements(ctx context.Context, shiftID uint64) ([]Movement, error) {
	items := []Movement{}
	err := r.db.SelectContext(ctx, &items,
		`SELECT m.*, u.name AS created_by_name
		 FROM cash_movements m JOIN users u ON u.id = m.created_by
		 WHERE m.shift_id = ? ORDER BY m.created_at, m.id`, shiftID)
	return items, err
}

// MethodTotals aggregates confirmed payments by method for a shift's sales.
func (r *Repo) MethodTotals(ctx context.Context, shiftID uint64) ([]MethodTotal, error) {
	rows := []struct {
		Method string          `db:"method"`
		Count  int             `db:"cnt"`
		Total  decimal.Decimal `db:"total"`
	}{}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT p.method, COUNT(*) AS cnt, COALESCE(SUM(p.amount),0) AS total
		 FROM sale_payments p JOIN sales s ON s.id = p.sale_id
		 WHERE s.shift_id = ? AND p.status = 'confirmed' AND s.status <> 'voided'
		 GROUP BY p.method ORDER BY p.method`, shiftID)
	out := make([]MethodTotal, 0, len(rows))
	for _, r := range rows {
		out = append(out, MethodTotal{Method: r.Method, Count: r.Count, Total: r.Total})
	}
	return out, err
}

// SalesCount returns how many non-voided sales the shift carries.
func (r *Repo) SalesCount(ctx context.Context, shiftID uint64) (int, decimal.Decimal, error) {
	var row struct {
		Count int             `db:"cnt"`
		Total decimal.Decimal `db:"total"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT COUNT(*) AS cnt, COALESCE(SUM(s.total),0) AS total
		 FROM sales s WHERE s.shift_id = ? AND s.status <> 'voided'`, shiftID)
	return row.Count, row.Total, err
}

// CashRefunded returns cash refunds handed out at the branch inside the
// shift window (sale_returns carry no terminal, so branch + time is the
// attribution key).
func (r *Repo) CashRefunded(ctx context.Context, s Shift) (decimal.Decimal, error) {
	var sum decimal.Decimal
	until := time.Now()
	if s.ClosedAt != nil {
		until = *s.ClosedAt
	}
	err := r.db.GetContext(ctx, &sum,
		`SELECT COALESCE(SUM(r.refund_amount),0)
		 FROM sale_returns r JOIN sales s2 ON s2.id = r.sale_id
		 WHERE r.tenant_id = ? AND s2.branch_id = ? AND r.refund_method = 'cash'
		   AND r.status = 'completed' AND r.created_at >= ? AND r.created_at < ?`,
		s.TenantID, s.BranchID, s.OpenedAt, until)
	return sum, err
}

// MovementSums returns cash_in and cash_out totals for a shift.
func (r *Repo) MovementSums(ctx context.Context, shiftID uint64) (in, out decimal.Decimal, err error) {
	var row struct {
		In  decimal.Decimal `db:"cin"`
		Out decimal.Decimal `db:"cout"`
	}
	err = r.db.GetContext(ctx, &row,
		`SELECT COALESCE(SUM(CASE WHEN kind='cash_in' THEN amount END),0) AS cin,
		        COALESCE(SUM(CASE WHEN kind='cash_out' THEN amount END),0) AS cout
		 FROM cash_movements WHERE shift_id = ?`, shiftID)
	return row.In, row.Out, err
}

// Close stamps the close fields and clears the open marker.
func (r *Repo) Close(ctx context.Context, tx *sqlx.Tx, s Shift) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE cash_shifts SET status='closed', closed_by=?, closed_at=?,
			expected_amount=?, declared_amount=?, variance=?, close_note=?,
			open_marker=NULL
		 WHERE id=? AND status='open'`,
		s.ClosedBy, s.ClosedAt, s.Expected, s.Declared, s.Variance,
		s.CloseNote, s.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrShiftClosed
	}
	return nil
}

// ReviewVariance records the back-office verdict on a closed shift.
func (r *Repo) ReviewVariance(ctx context.Context, tenantID, shiftID uint64,
	status, note string, by uint64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE cash_shifts SET variance_status=?, variance_note=?,
			reviewed_by=?, reviewed_at=NOW(3)
		 WHERE id=? AND tenant_id=? AND status='closed'`, status, note, by, shiftID, tenantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrShiftNotFound
	}
	return nil
}

// Summary is the head-office rollup behind the StatCards: open shifts, the
// cash expected across them, today's net variance and today's closes.
type Summary struct {
	OpenCount     int             `db:"open_count" json:"open_count"`
	ExpectedOpen  decimal.Decimal `db:"expected_open" json:"expected_open"`
	VarianceToday decimal.Decimal `db:"variance_today" json:"variance_today"`
	ClosedToday   int             `db:"closed_today" json:"closed_today"`
}

// Summary returns the rollup, optionally scoped to branches.
func (r *Repo) Summary(ctx context.Context, tenantID uint64, branchIDs []uint64) (Summary, error) {
	var s Summary
	branchClause := func(alias string) string {
		if len(branchIDs) == 0 {
			return ""
		}
		return " AND " + alias + ".branch_id IN (?)"
	}
	// Expected cash across open shifts = floats + cash sales − refunds +
	// movements, computed per shift and summed.
	q := `SELECT COUNT(*) AS open_count,
		COALESCE((SELECT SUM(p.amount) FROM sale_payments p
			JOIN sales s2 ON s2.id = p.sale_id
			WHERE s2.shift_id IN (SELECT id FROM cash_shifts o WHERE o.tenant_id=? AND o.status='open'` + branchClause("o") + `)
			  AND p.method='cash' AND p.status='confirmed' AND s2.status <> 'voided'),0)
		+ COALESCE((SELECT SUM(CASE WHEN m.kind='cash_in' THEN m.amount ELSE -m.amount END)
			FROM cash_movements m
			WHERE m.shift_id IN (SELECT id FROM cash_shifts o2 WHERE o2.tenant_id=? AND o2.status='open'` + branchClause("o2") + `)
			  AND m.kind <> 'drawer_open'),0)
		AS expected_open,
		(SELECT COALESCE(SUM(v.variance),0) FROM cash_shifts v
			WHERE v.tenant_id=? AND v.status='closed' AND DATE(v.closed_at)=CURDATE()` + branchClause("v") + `) AS variance_today,
		(SELECT COUNT(*) FROM cash_shifts c
			WHERE c.tenant_id=? AND c.status='closed' AND DATE(c.closed_at)=CURDATE()` + branchClause("c") + `) AS closed_today
		FROM cash_shifts s WHERE s.tenant_id=? AND s.status='open'` + branchClause("s")
	args := []any{tenantID, tenantID, tenantID, tenantID, tenantID}
	if len(branchIDs) > 0 {
		args = []any{tenantID, branchIDs, tenantID, branchIDs, tenantID, branchIDs, tenantID, branchIDs, tenantID, branchIDs}
	}
	final, finalArgs, err := sqlx.In(q, args...)
	if err != nil {
		return s, err
	}
	err = r.db.GetContext(ctx, &s, final, finalArgs...)
	return s, err
}
