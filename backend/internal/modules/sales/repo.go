package sales

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"encoding/json"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/httpx"
)

// Repo stores sales, items, payments and returns.
type Repo struct{ conn *sqlx.DB }

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{conn: conn} }

func isDup(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}

// NextCounter atomically increments a tenant counter inside tx and returns
// the new value (receipt/return numbers must never collide).
func (r *Repo) NextCounter(ctx context.Context, tx *sqlx.Tx, tenantID uint64, name string) (uint64, error) {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO tenant_counters (tenant_id, name, value) VALUES (?,?,1)
		 ON DUPLICATE KEY UPDATE value = value + 1`, tenantID, name); err != nil {
		return 0, err
	}
	var v uint64
	if err := tx.GetContext(ctx, &v,
		`SELECT value FROM tenant_counters WHERE tenant_id = ? AND name = ?`, tenantID, name); err != nil {
		return 0, err
	}
	return v, nil
}

// InsertSale writes the sale row inside tx.
func (r *Repo) InsertSale(ctx context.Context, tx *sqlx.Tx, s *Sale) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO sales (tenant_id, branch_id, terminal_id, cashier_id, customer_id,
		 public_id, receipt_no, status, source, idempotency_key, currency,
		 subtotal, discount, tax, total, paid, change_due, discount_note, sold_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.TenantID, s.BranchID, s.TerminalID, s.CashierID, s.CustomerID,
		s.PublicID, s.ReceiptNo, s.Status, s.Source, s.IdemKey, s.Currency,
		s.Subtotal, s.Discount, s.Tax, s.Total, s.Paid, s.ChangeDue, s.DiscountNote, s.SoldAt)
	if err != nil {
		return err
	}
	s.ID, err = lastID(res)
	return err
}

// InsertItem writes one sale line inside tx.
func (r *Repo) InsertItem(ctx context.Context, tx *sqlx.Tx, it *SaleItem) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sale_items (tenant_id, sale_id, line_no, product_id, variant_id,
		 name, sku, qty, unit_price, discount, tax_rate, tax, total, price_override, promotion)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.TenantID, it.SaleID, it.LineNo, it.ProductID, it.VariantID,
		it.Name, it.SKU, it.Qty, it.UnitPrice, it.Discount, it.TaxRate, it.Tax, it.Total,
		it.PriceOverride, it.Promotion)
	return err
}

// InsertPayment writes one payment row inside tx.
func (r *Repo) InsertPayment(ctx context.Context, tx *sqlx.Tx, p *SalePayment) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sale_payments (tenant_id, sale_id, method, amount, tendered, change_due,
		 reference, status, created_by) VALUES (?,?,?,?,?,?,?,?,?)`,
		p.TenantID, p.SaleID, p.Method, p.Amount, p.Tendered, p.ChangeDue,
		p.Reference, p.Status, p.CreatedBy)
	return err
}

// ByIdempotencyKey finds an existing sale by its device key — the dedupe
// path for retried uploads.
func (r *Repo) ByIdempotencyKey(ctx context.Context, tenantID uint64, key string) (*Sale, error) {
	var s Sale
	err := r.conn.GetContext(ctx, &s,
		`SELECT `+saleCols+` FROM sales s `+saleJoins+
			` WHERE s.tenant_id = ? AND s.idempotency_key = ?`, tenantID, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

const saleCols = `s.id, s.tenant_id, s.branch_id, s.terminal_id, s.cashier_id, s.customer_id,
	s.public_id, s.receipt_no, s.status, s.source, s.idempotency_key, s.currency,
	s.subtotal, s.discount, s.tax, s.total, s.paid, s.change_due, s.discount_note,
	s.voided_at, s.voided_by, s.void_reason, s.sold_at, s.created_at, s.updated_at,
	b.name AS branch_name,
	COALESCE(t.name,'') AS terminal_name,
	COALESCE(u.name,'') AS cashier_name,
	COALESCE(c.name,'') AS customer_name`

const saleJoins = `JOIN branches b ON b.id = s.branch_id
	LEFT JOIN pos_terminals t ON t.id = s.terminal_id
	LEFT JOIN users u ON u.id = s.cashier_id
	LEFT JOIN customers c ON c.id = s.customer_id`

// ListFilter narrows the sales list.
type ListFilter struct {
	BranchIDs  []uint64 // actor's allowed branches; empty = all tenant branches
	BranchID   uint64
	CashierID  uint64
	CustomerID uint64
	Status     string
	Method     string
	Search     string // receipt no / customer / item name
	From       *time.Time
	To         *time.Time
}

// List returns sales newest-first with pagination.
func (r *Repo) List(ctx context.Context, tenantID uint64, f ListFilter, pg httpx.Page) ([]Sale, int, error) {
	where, args := filterWhere(tenantID, f)
	var total int
	if err := r.conn.GetContext(ctx, &total,
		`SELECT COUNT(DISTINCT s.id) FROM sales s `+saleJoins+where, args...); err != nil {
		return nil, 0, err
	}
	out := []Sale{}
	err := r.conn.SelectContext(ctx, &out,
		`SELECT `+saleCols+` FROM sales s `+saleJoins+where+
			` GROUP BY s.id ORDER BY s.sold_at DESC, s.id DESC LIMIT ? OFFSET ?`,
		append(args, pg.Limit(), pg.Offset())...)
	return out, total, err
}

func filterWhere(tenantID uint64, f ListFilter) (string, []any) {
	w := ` WHERE s.tenant_id = ?`
	args := []any{tenantID}
	if len(f.BranchIDs) > 0 {
		w += ` AND s.branch_id IN (` + placeholders(len(f.BranchIDs)) + `)`
		for _, b := range f.BranchIDs {
			args = append(args, b)
		}
	}
	if f.BranchID != 0 {
		w += ` AND s.branch_id = ?`
		args = append(args, f.BranchID)
	}
	if f.CashierID != 0 {
		w += ` AND s.cashier_id = ?`
		args = append(args, f.CashierID)
	}
	if f.CustomerID != 0 {
		w += ` AND s.customer_id = ?`
		args = append(args, f.CustomerID)
	}
	if f.Status != "" {
		w += ` AND s.status = ?`
		args = append(args, f.Status)
	}
	if f.Method != "" {
		w += ` AND EXISTS (SELECT 1 FROM sale_payments p WHERE p.sale_id = s.id AND p.method = ?)`
		args = append(args, f.Method)
	}
	if f.Search != "" {
		w += ` AND (CAST(s.receipt_no AS CHAR) LIKE ? OR c.name LIKE ?
		   OR EXISTS (SELECT 1 FROM sale_items i WHERE i.sale_id = s.id AND i.name LIKE ?))`
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.From != nil {
		w += ` AND s.sold_at >= ?`
		args = append(args, *f.From)
	}
	if f.To != nil {
		w += ` AND s.sold_at <= ?`
		args = append(args, *f.To)
	}
	return w, args
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// ByPublicID loads one sale with display fields.
func (r *Repo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Sale, error) {
	var s Sale
	err := r.conn.GetContext(ctx, &s,
		`SELECT `+saleCols+` FROM sales s `+saleJoins+
			` WHERE s.tenant_id = ? AND s.public_id = ?`, tenantID, publicID)
	return s, err
}

// ByReceipt loads one sale by receipt number within a branch scope (device
// lookup). branchID = 0 matches any branch.
func (r *Repo) ByReceipt(ctx context.Context, tenantID, branchID uint64, ref string) (Sale, error) {
	var s Sale
	w := ` WHERE s.tenant_id = ? AND (s.public_id = ? OR CAST(s.receipt_no AS CHAR) = ?)`
	args := []any{tenantID, ref, ref}
	if branchID != 0 {
		w += ` AND s.branch_id = ?`
		args = append(args, branchID)
	}
	err := r.conn.GetContext(ctx, &s,
		`SELECT `+saleCols+` FROM sales s `+saleJoins+w, args...)
	return s, err
}

// Items returns a sale's lines with product refs joined for restock/returns.
func (r *Repo) Items(ctx context.Context, saleID uint64) ([]SaleItem, error) {
	out := []SaleItem{}
	return out, r.conn.SelectContext(ctx, &out,
		`SELECT i.*, p.public_id AS product_public_id, p.track_stock,
		        COALESCE(v.public_id,'') AS variant_public_id
		 FROM sale_items i
		 JOIN products p ON p.id = i.product_id
		 LEFT JOIN product_variants v ON v.id = i.variant_id
		 WHERE i.sale_id = ? ORDER BY i.line_no`, saleID)
}

// Payments returns a sale's payment rows.
func (r *Repo) Payments(ctx context.Context, saleID uint64) ([]SalePayment, error) {
	out := []SalePayment{}
	return out, r.conn.SelectContext(ctx, &out,
		`SELECT * FROM sale_payments WHERE sale_id = ? ORDER BY id`, saleID)
}

// Returns returns refunds recorded against a sale.
func (r *Repo) Returns(ctx context.Context, saleID uint64) ([]SaleReturn, error) {
	out := []SaleReturn{}
	return out, r.conn.SelectContext(ctx, &out,
		`SELECT r.*, s.receipt_no, COALESCE(u.name,'') AS cashier_name
		 FROM sale_returns r JOIN sales s ON s.id = r.sale_id
		 LEFT JOIN users u ON u.id = r.created_by
		 WHERE r.sale_id = ? ORDER BY r.id`, saleID)
}

// ReturnItems returns the lines of a return.
func (r *Repo) ReturnItems(ctx context.Context, returnID uint64) ([]SaleReturnItem, error) {
	out := []SaleReturnItem{}
	return out, r.conn.SelectContext(ctx, &out,
		`SELECT * FROM sale_return_items WHERE return_id = ?`, returnID)
}

// ListReturns pages the returns register for the tenant.
func (r *Repo) ListReturns(ctx context.Context, tenantID uint64, pg httpx.Page) ([]SaleReturn, int, error) {
	var total int
	if err := r.conn.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM sale_returns WHERE tenant_id = ?`, tenantID); err != nil {
		return nil, 0, err
	}
	out := []SaleReturn{}
	err := r.conn.SelectContext(ctx, &out,
		`SELECT r.*, s.receipt_no, COALESCE(u.name,'') AS cashier_name
		 FROM sale_returns r JOIN sales s ON s.id = r.sale_id
		 LEFT JOIN users u ON u.id = r.created_by
		 WHERE r.tenant_id = ? ORDER BY r.id DESC LIMIT ? OFFSET ?`,
		tenantID, pg.Limit(), pg.Offset())
	return out, total, err
}

// MarkVoided stamps the void inside tx.
func (r *Repo) MarkVoided(ctx context.Context, tx *sqlx.Tx, saleID, actorID uint64, reason string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE sales SET status = 'voided', voided_at = NOW(3), voided_by = ?, void_reason = ?
		 WHERE id = ? AND status = 'completed'`, actorID, reason, saleID)
	return err
}

// VoidPayments marks a sale's payment rows voided inside tx.
func (r *Repo) VoidPayments(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE sale_payments SET status = 'voided' WHERE sale_id = ? AND status = 'confirmed'`, saleID)
	return err
}

// InsertReturn writes the return header inside tx.
func (r *Repo) InsertReturn(ctx context.Context, tx *sqlx.Tx, rt *SaleReturn) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO sale_returns (tenant_id, sale_id, public_id, return_no, reason,
		 refund_method, refund_amount, status, supervisor_id, created_by)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		rt.TenantID, rt.SaleID, rt.PublicID, rt.ReturnNo, rt.Reason,
		rt.RefundMethod, rt.RefundAmount, rt.Status, rt.SupervisorID, rt.CreatedBy)
	if err != nil {
		return err
	}
	rt.ID, err = lastID(res)
	return err
}

// InsertReturnItem writes one returned line inside tx.
func (r *Repo) InsertReturnItem(ctx context.Context, tx *sqlx.Tx, it *SaleReturnItem) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sale_return_items (tenant_id, return_id, sale_item_id, qty, restock, amount)
		 VALUES (?,?,?,?,?,?)`, it.TenantID, it.ReturnID, it.SaleItemID, it.Qty, it.Restock, it.Amount)
	return err
}

// AddReturnedQty increments returned_qty on a sale line inside tx, guarding
// against over-return.
func (r *Repo) AddReturnedQty(ctx context.Context, tx *sqlx.Tx, itemID uint64, qty string) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE sale_items SET returned_qty = returned_qty + ?
		 WHERE id = ? AND returned_qty + ? <= qty`, qty, itemID, qty)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrOverReturn
	}
	return nil
}

// RefreshReturnStatus recomputes the sale's return status inside tx.
func (r *Repo) RefreshReturnStatus(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE sales SET status = CASE
		   WHEN NOT EXISTS (SELECT 1 FROM sale_items i WHERE i.sale_id = sales.id AND i.returned_qty < i.qty)
		     THEN 'returned'
		   ELSE 'partially_returned' END
		 WHERE id = ? AND status IN ('completed','partially_returned')`, saleID)
	return err
}

// SaleStatus loads just the status for guards.
func (r *Repo) SaleStatus(ctx context.Context, tenantID uint64, saleID uint64) (string, error) {
	var st string
	err := r.conn.GetContext(ctx, &st,
		`SELECT status FROM sales WHERE tenant_id = ? AND id = ?`, tenantID, saleID)
	return st, err
}

// Held lists parked carts for the back-office.
func (r *Repo) Held(ctx context.Context, tenantID uint64) ([]HeldCart, error) {
	out := []HeldCart{}
	return out, r.conn.SelectContext(ctx, &out,
		`SELECT h.public_id, h.label, COALESCE(u.name,'') AS held_by,
		        t.name AS terminal_name, b.name AS branch_name, h.cart, h.status, h.created_at
		 FROM held_sales h
		 JOIN branches b ON b.id = h.branch_id
		 JOIN pos_terminals t ON t.id = h.terminal_id
		 LEFT JOIN users u ON u.id = h.user_id
		 WHERE h.tenant_id = ? AND h.status = 'held' ORDER BY h.created_at DESC`, tenantID)
}

// Summary aggregates sales for dashboard widgets and the list header.
type Summary struct {
	Total   string `db:"total" json:"total"`
	Count   int    `db:"cnt" json:"cnt"`
	Average string `db:"avg" json:"avg"`
}

// MethodSplit is one payment-method row of the split.
type MethodSplit struct {
	Method string `db:"method" json:"method"`
	Total  string `db:"total" json:"total"`
	Count  int    `db:"cnt" json:"cnt"`
}

// Summarize totals completed/returned sales in a window.
func (r *Repo) Summarize(ctx context.Context, tenantID uint64, f ListFilter) (Summary, error) {
	where, args := filterWhere(tenantID, f)
	where += ` AND s.status IN ('completed','partially_returned','returned')`
	var s Summary
	err := r.conn.GetContext(ctx, &s,
		`SELECT COALESCE(SUM(s.total),0) AS total, COUNT(DISTINCT s.id) AS cnt,
		        COALESCE(ROUND(AVG(s.total),2),0) AS avg
		 FROM sales s `+saleJoins+where, args...)
	return s, err
}

// MethodTotals splits confirmed payments by method in a window.
func (r *Repo) MethodTotals(ctx context.Context, tenantID uint64, f ListFilter) ([]MethodSplit, error) {
	where, args := filterWhere(tenantID, f)
	where += ` AND s.status <> 'voided'`
	out := []MethodSplit{}
	err := r.conn.SelectContext(ctx, &out,
		`SELECT p.method, COALESCE(SUM(CASE WHEN p.status='refunded' THEN -p.amount ELSE p.amount END),0) AS total,
		        COUNT(*) AS cnt
		 FROM sale_payments p JOIN sales s ON s.id = p.sale_id `+saleJoins+
			where+` AND p.status IN ('confirmed','refunded') GROUP BY p.method ORDER BY total DESC`, args...)
	return out, err
}

// CashierAuthz loads a user's role permissions and discount/refund limits —
// the server-side re-check of what the PIN screen allowed.
type CashierAuthz struct {
	UserID      uint64
	Perms       map[string]bool
	MaxDiscount *string
	MaxRefund   *string
	RoleName    string
}

// CashierAuthz loads permissions + limits for one tenant user.
func (r *Repo) CashierAuthz(ctx context.Context, tenantID, userID uint64) (*CashierAuthz, error) {
	var row struct {
		ID          uint64  `db:"id"`
		RoleName    string  `db:"role_name"`
		MaxDiscount *string `db:"max_discount_percent"`
		MaxRefund   *string `db:"max_refund_amount"`
	}
	err := r.conn.GetContext(ctx, &row,
		`SELECT u.id, ro.name AS role_name, ro.max_discount_percent, ro.max_refund_amount
		 FROM users u JOIN roles ro ON ro.id = u.role_id
		 WHERE u.id = ? AND u.tenant_id = ? AND u.status = 'active'`, userID, tenantID)
	if err != nil {
		return nil, err
	}
	perms := []struct {
		Permission string `db:"permission"`
	}{}
	err = r.conn.SelectContext(ctx, &perms,
		`SELECT rp.permission FROM users u JOIN role_permissions rp ON rp.role_id = u.role_id
		 WHERE u.id = ?`, userID)
	if err != nil {
		return nil, err
	}
	out := &CashierAuthz{UserID: row.ID, RoleName: row.RoleName, MaxDiscount: row.MaxDiscount, MaxRefund: row.MaxRefund, Perms: map[string]bool{}}
	for _, p := range perms {
		out.Perms[p.Permission] = true
	}
	return out, nil
}

// InsertAudit writes an audit row inside the sale transaction — completion,
// void and refund are one atomic record with their ledger entries.
func (r *Repo) InsertAudit(ctx context.Context, tx *sqlx.Tx, tenantID, branchID, actorID uint64,
	actorType, action, entityID string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO audit_logs (tenant_id, branch_id, actor_type, actor_id, action,
		 entity_type, entity_id, data) VALUES (?,?,?,?,?,?,?,?)`,
		tenantID, branchID, actorType, actorID, action, "sale", entityID, payload)
	return err
}

func lastID(res sql.Result) (uint64, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id < 0 {
		return 0, nil
	}
	return uint64(id), nil //nolint:gosec // LastInsertId is non-negative on success
}

// DailyPoint is one day's totals for charts and KPI trends.
type DailyPoint struct {
	Day   string `db:"day" json:"day"`
	Total string `db:"total" json:"total"`
	Count int    `db:"cnt" json:"cnt"`
}

// Daily totals by sale date in a window (sold_at, not created_at).
func (r *Repo) Daily(ctx context.Context, tenantID uint64, f ListFilter) ([]DailyPoint, error) {
	where, args := filterWhere(tenantID, f)
	where += ` AND s.status IN ('completed','partially_returned','returned')`
	out := []DailyPoint{}
	err := r.conn.SelectContext(ctx, &out,
		`SELECT DATE_FORMAT(s.sold_at, '%Y-%m-%d') AS day, COALESCE(SUM(s.total),0) AS total, COUNT(*) AS cnt
		 FROM sales s `+saleJoins+where+` GROUP BY DATE(s.sold_at) ORDER BY day`, args...)
	return out, err
}
