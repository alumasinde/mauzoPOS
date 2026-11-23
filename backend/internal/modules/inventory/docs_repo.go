package inventory

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// DocsRepo persists stock documents: adjustments, transfers and counts.
type DocsRepo struct {
	db *sqlx.DB
}

// NewDocsRepo returns the documents repository.
func NewDocsRepo(conn *sqlx.DB) *DocsRepo { return &DocsRepo{db: conn} }

const adjFrom = ` FROM stock_adjustments d JOIN branches br ON br.id=d.branch_id`
const adjCols = `d.id, d.tenant_id, d.branch_id, d.public_id, d.status, d.reason, d.note,
	d.created_by, d.approved_by, d.posted_at, d.created_at,
	br.public_id AS branch_public_id, br.name AS branch_name`

// CreateAdjustment inserts a pending adjustment with its items.
func (r *DocsRepo) CreateAdjustment(ctx context.Context, q db.Querier, a *Adjustment) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO stock_adjustments (tenant_id, branch_id, public_id, status, reason, note, created_by)
		 VALUES (?,?,?,?,?,?,?)`, a.TenantID, a.BranchID, a.PublicID, a.Status, a.Reason, a.Note, a.CreatedBy)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	a.ID = db.U64(id)
	for i := range a.Items {
		a.Items[i].AdjustmentID = a.ID
		it := &a.Items[i]
		if _, err := q.ExecContext(ctx,
			`INSERT INTO stock_adjustment_items (adjustment_id, location_id, product_id, variant_id, batch_id, qty)
			 VALUES (?,?,?,?,?,?)`, it.AdjustmentID, it.LocationID, it.ProductID, it.VariantID, it.BatchID, it.Qty); err != nil {
			return err
		}
	}
	return nil
}

// Adjustment loads one adjustment by public id.
func (r *DocsRepo) Adjustment(ctx context.Context, q db.Querier, tenantID uint64, publicID string) (Adjustment, error) {
	var a Adjustment
	if err := q.GetContext(ctx, &a, `SELECT `+adjCols+adjFrom+` WHERE d.tenant_id=? AND d.public_id=?`,
		tenantID, publicID); err != nil {
		return a, err
	}
	return a, q.SelectContext(ctx, &a.Items,
		`SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name,
			l.public_id AS location_public_id, l.name AS location_name
		 FROM stock_adjustment_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 JOIN stock_locations l ON l.id=i.location_id
		 WHERE i.adjustment_id=?`, a.ID)
}

// Adjustments lists adjustments for a tenant, newest first.
func (r *DocsRepo) Adjustments(ctx context.Context, tenantID, branchID uint64, status string, pg httpx.Page) ([]Adjustment, int, error) {
	where := `d.tenant_id=?`
	args := []any{tenantID}
	if branchID != 0 {
		where += ` AND d.branch_id=?`
		args = append(args, branchID)
	}
	if status != "" {
		where += ` AND d.status=?`
		args = append(args, status)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM stock_adjustments d WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Adjustment
	if err := r.db.SelectContext(ctx, &out,
		`SELECT `+adjCols+adjFrom+` WHERE `+where+` ORDER BY d.id DESC LIMIT ? OFFSET ?`,
		append(args, pg.PerPage, pg.Offset())...); err != nil {
		return nil, 0, err
	}
	for i := range out {
		if err := r.db.SelectContext(ctx, &out[i].Items,
			`SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name,
			l.public_id AS location_public_id, l.name AS location_name
		 FROM stock_adjustment_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 JOIN stock_locations l ON l.id=i.location_id
		 WHERE i.adjustment_id=?`, out[i].ID); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// SetAdjustmentStatus moves an adjustment to a new status.
func (r *DocsRepo) SetAdjustmentStatus(ctx context.Context, q db.Querier, id uint64, status string, approver *uint64) error {
	var res sql.Result
	var err error
	switch status {
	case StatusApproved, StatusRejected:
		res, err = q.ExecContext(ctx,
			`UPDATE stock_adjustments SET status=?, approved_by=? WHERE id=? AND status='pending'`,
			status, *approver, id)
	case StatusPosted:
		res, err = q.ExecContext(ctx,
			`UPDATE stock_adjustments SET status=?, posted_at=? WHERE id=? AND status='approved'`,
			status, time.Now(), id)
	default:
		res, err = q.ExecContext(ctx, `UPDATE stock_adjustments SET status=? WHERE id=?`, status, id)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

const trFrom = ` FROM stock_transfers d
	JOIN branches fb ON fb.id=d.from_branch_id
	JOIN branches tb ON tb.id=d.to_branch_id`
const trCols = `d.id, d.tenant_id, d.public_id, d.from_branch_id, d.to_branch_id, d.status, d.note,
	d.created_by, d.sent_at, d.received_at, d.created_at,
	fb.public_id AS from_public_id, fb.name AS from_name,
	tb.public_id AS to_public_id, tb.name AS to_name`

// CreateTransfer inserts a draft transfer with its items.
func (r *DocsRepo) CreateTransfer(ctx context.Context, q db.Querier, t *Transfer) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO stock_transfers (tenant_id, public_id, from_branch_id, to_branch_id, status, note, created_by)
		 VALUES (?,?,?,?,?,?,?)`, t.TenantID, t.PublicID, t.FromBranchID, t.ToBranchID, t.Status, t.Note, t.CreatedBy)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	t.ID = db.U64(id)
	for i := range t.Items {
		t.Items[i].TransferID = t.ID
		it := &t.Items[i]
		if _, err := q.ExecContext(ctx,
			`INSERT INTO stock_transfer_items (public_id, transfer_id, product_id, variant_id, qty_sent)
			 VALUES (?,?,?,?,?)`, it.PublicID, it.TransferID, it.ProductID, it.VariantID, it.QtySent); err != nil {
			return err
		}
	}
	return nil
}

// Transfer loads one transfer by public id.
func (r *DocsRepo) Transfer(ctx context.Context, q db.Querier, tenantID uint64, publicID string) (Transfer, error) {
	var t Transfer
	if err := q.GetContext(ctx, &t, `SELECT `+trCols+trFrom+` WHERE d.tenant_id=? AND d.public_id=?`,
		tenantID, publicID); err != nil {
		return t, err
	}
	return t, q.SelectContext(ctx, &t.Items,
		`SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name
		 FROM stock_transfer_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 WHERE i.transfer_id=?`, t.ID)
}

// Transfers lists transfers involving a branch (or all when branchID=0).
func (r *DocsRepo) Transfers(ctx context.Context, tenantID, branchID uint64, pg httpx.Page) ([]Transfer, int, error) {
	where := `d.tenant_id=?`
	args := []any{tenantID}
	if branchID != 0 {
		where += ` AND (d.from_branch_id=? OR d.to_branch_id=?)`
		args = append(args, branchID, branchID)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM stock_transfers d WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Transfer
	if err := r.db.SelectContext(ctx, &out,
		`SELECT `+trCols+trFrom+` WHERE `+where+` ORDER BY d.id DESC LIMIT ? OFFSET ?`,
		append(args, pg.PerPage, pg.Offset())...); err != nil {
		return nil, 0, err
	}
	for i := range out {
		if err := r.db.SelectContext(ctx, &out[i].Items,
			`SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name
		 FROM stock_transfer_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 WHERE i.transfer_id=?`, out[i].ID); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// SetTransferStatus moves a transfer through draft → sent → received (or
// cancelled from draft/sent).
func (r *DocsRepo) SetTransferStatus(ctx context.Context, q db.Querier, id uint64, status string) error {
	var res sql.Result
	var err error
	switch status {
	case StatusSent:
		res, err = q.ExecContext(ctx,
			`UPDATE stock_transfers SET status='sent', sent_at=? WHERE id=? AND status='draft'`, time.Now(), id)
	case StatusReceived:
		res, err = q.ExecContext(ctx,
			`UPDATE stock_transfers SET status='received', received_at=? WHERE id=? AND status='sent'`, time.Now(), id)
	case StatusCancelled:
		res, err = q.ExecContext(ctx,
			`UPDATE stock_transfers SET status='cancelled' WHERE id=? AND status IN ('draft','sent')`, id)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkReceived records one item's received quantity (by public id, scoped to
// its transfer).
func (r *DocsRepo) MarkReceived(ctx context.Context, q db.Querier, transferID uint64, itemPubID, qty string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE stock_transfer_items SET qty_received=? WHERE public_id=? AND transfer_id=?`,
		qty, itemPubID, transferID)
	return err
}

const cntFrom = ` FROM stock_counts d JOIN branches br ON br.id=d.branch_id
	LEFT JOIN stock_locations l ON l.id=d.location_id`
const cntCols = `d.id, d.tenant_id, d.branch_id, d.public_id, d.status, d.location_id, d.note,
	d.created_by, d.posted_at, d.created_at,
	br.public_id AS branch_public_id, br.name AS branch_name,
	COALESCE(l.public_id,'') AS location_public_id`

// CreateCount inserts an open count and snapshots expected balances.
func (r *DocsRepo) CreateCount(ctx context.Context, q db.Querier, c *Count) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO stock_counts (tenant_id, branch_id, public_id, status, location_id, note, created_by)
		 VALUES (?,?,?,?,?,?,?)`, c.TenantID, c.BranchID, c.PublicID, c.Status, c.LocationID, c.Note, c.CreatedBy)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	c.ID = db.U64(id)
	for i := range c.Items {
		c.Items[i].CountID = c.ID
		it := &c.Items[i]
		if _, err := q.ExecContext(ctx,
			`INSERT INTO stock_count_items (public_id, count_id, location_id, product_id, variant_id, batch_id, expected_qty)
			 VALUES (?,?,?,?,?,?,?)`, it.PublicID, it.CountID, it.LocationID, it.ProductID, it.VariantID, it.BatchID, it.ExpectedQty); err != nil {
			return err
		}
	}
	return nil
}

// Count loads one count with items by public id.
func (r *DocsRepo) Count(ctx context.Context, q db.Querier, tenantID uint64, publicID string) (Count, error) {
	var c Count
	if err := q.GetContext(ctx, &c, `SELECT `+cntCols+cntFrom+` WHERE d.tenant_id=? AND d.public_id=?`,
		tenantID, publicID); err != nil {
		return c, err
	}
	return c, q.SelectContext(ctx, &c.Items, `SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name,
			l.public_id AS location_public_id, l.name AS location_name
		 FROM stock_count_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 JOIN stock_locations l ON l.id=i.location_id
		 WHERE i.count_id=?`, c.ID)
}

// Counts lists counts for a tenant, newest first.
func (r *DocsRepo) Counts(ctx context.Context, tenantID, branchID uint64, pg httpx.Page) ([]Count, int, error) {
	where := `d.tenant_id=?`
	args := []any{tenantID}
	if branchID != 0 {
		where += ` AND branch_id=?`
		args = append(args, branchID)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM stock_counts d WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Count
	if err := r.db.SelectContext(ctx, &out,
		`SELECT `+cntCols+cntFrom+` WHERE `+where+` ORDER BY d.id DESC LIMIT ? OFFSET ?`,
		append(args, pg.PerPage, pg.Offset())...); err != nil {
		return nil, 0, err
	}
	for i := range out {
		if err := r.db.SelectContext(ctx, &out[i].Items,
			`SELECT i.*, p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
			p.name AS product_name, COALESCE(v.name,'') AS variant_name,
			l.public_id AS location_public_id, l.name AS location_name
		 FROM stock_count_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 JOIN stock_locations l ON l.id=i.location_id
		 WHERE i.count_id=?`, out[i].ID); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// SetCounted records a counted quantity on one line (by public id, scoped to
// its count).
func (r *DocsRepo) SetCounted(ctx context.Context, countID uint64, itemPubID, qty string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE stock_count_items SET counted_qty=? WHERE public_id=? AND count_id=?`, qty, itemPubID, countID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CloseCount marks a count posted or cancelled.
func (r *DocsRepo) CloseCount(ctx context.Context, q db.Querier, id uint64, status string) error {
	var res sql.Result
	var err error
	if status == StatusPosted {
		res, err = q.ExecContext(ctx,
			`UPDATE stock_counts SET status='posted', posted_at=? WHERE id=? AND status='open'`, time.Now(), id)
	} else {
		res, err = q.ExecContext(ctx,
			`UPDATE stock_counts SET status='cancelled' WHERE id=? AND status='open'`, id)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
