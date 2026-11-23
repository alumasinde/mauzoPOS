package inventory

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Repo persists locations, the movement ledger, derived balances, batches
// and reorder levels.
type Repo struct {
	db *sqlx.DB
}

// NewRepo returns the inventory repository.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const locationCols = `id, tenant_id, branch_id, public_id, name, kind, is_default, created_at`

// CreateLocation inserts a location inside q's transaction; the first
// location in a branch becomes the default.
func (r *Repo) CreateLocation(ctx context.Context, q db.Querier, l *Location) error {
	var n int
	if err := q.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM stock_locations WHERE tenant_id=? AND branch_id=?`,
		l.TenantID, l.BranchID); err != nil {
		return err
	}
	if n == 0 {
		l.IsDefault = true
	}
	res, err := q.ExecContext(ctx,
		`INSERT INTO stock_locations (tenant_id, branch_id, public_id, name, kind, is_default)
		 VALUES (?,?,?,?,?,?)`, l.TenantID, l.BranchID, l.PublicID, l.Name, l.Kind, l.IsDefault)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	l.ID = db.U64(id)
	return q.GetContext(ctx, l, `SELECT `+locationCols+` FROM stock_locations WHERE id=?`, l.ID)
}

// Locations lists a branch's locations.
func (r *Repo) Locations(ctx context.Context, tenantID, branchID uint64) ([]Location, error) {
	var out []Location
	return out, r.db.SelectContext(ctx, &out,
		`SELECT `+locationCols+` FROM stock_locations WHERE tenant_id=? AND branch_id=? ORDER BY name`,
		tenantID, branchID)
}

// LocationByPublicID resolves a location public id (scoped to tenant+branch).
func (r *Repo) LocationByPublicID(ctx context.Context, tenantID, branchID uint64, publicID string) (Location, error) {
	var l Location
	return l, r.db.GetContext(ctx, &l,
		`SELECT `+locationCols+` FROM stock_locations WHERE tenant_id=? AND branch_id=? AND public_id=?`,
		tenantID, branchID, publicID)
}

// DefaultLocation returns the branch's default location, creating "Main
// store" on first use so stock can always be received.
func (r *Repo) DefaultLocation(ctx context.Context, q db.Querier, tenantID, branchID uint64) (Location, error) {
	var l Location
	err := q.GetContext(ctx, &l,
		`SELECT `+locationCols+` FROM stock_locations
		 WHERE tenant_id=? AND branch_id=? ORDER BY is_default DESC, id LIMIT 1`, tenantID, branchID)
	if err == nil {
		return l, nil
	}
	if !db.IsNotFound(err) {
		return l, err
	}
	l = Location{TenantID: tenantID, BranchID: branchID, PublicID: security.NewPublicID(),
		Name: "Main store", Kind: "store", IsDefault: true}
	if err := r.CreateLocation(ctx, q, &l); err != nil {
		if !db.IsDuplicate(err) {
			return l, err
		}
		// Lost a create race — re-read the winner.
		if err := q.GetContext(ctx, &l,
			`SELECT `+locationCols+` FROM stock_locations
			 WHERE tenant_id=? AND branch_id=? ORDER BY is_default DESC, id LIMIT 1`, tenantID, branchID); err != nil {
			return l, err
		}
	}
	return l, nil
}

// DeleteLocation removes an unused location.
func (r *Repo) DeleteLocation(ctx context.Context, tenantID, branchID uint64, publicID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM stock_locations WHERE tenant_id=? AND branch_id=? AND public_id=?`,
		tenantID, branchID, publicID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// LocationUsed reports whether a location has movements or balances.
func (r *Repo) LocationUsed(ctx context.Context, id uint64) (bool, error) {
	var used bool
	return used, r.db.GetContext(ctx, &used,
		`SELECT EXISTS(SELECT 1 FROM stock_movements WHERE location_id=? UNION
		               SELECT 1 FROM stock_balances WHERE location_id=? AND qty<>0)`, id, id)
}

// InsertMovements appends ledger rows inside q's transaction.
func (r *Repo) InsertMovements(ctx context.Context, q db.Querier, movs []Movement) error {
	for i := range movs {
		res, err := q.ExecContext(ctx,
			`INSERT INTO stock_movements
			 (tenant_id, branch_id, location_id, product_id, variant_id, batch_id,
			  qty, unit_cost, kind, ref_type, ref_public_id, reason, note, actor_id)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			movs[i].TenantID, movs[i].BranchID, movs[i].LocationID, movs[i].ProductID,
			movs[i].VariantID, movs[i].BatchID, movs[i].Qty, movs[i].UnitCost, movs[i].Kind,
			movs[i].RefType, movs[i].RefPublicID, movs[i].Reason, movs[i].Note, movs[i].ActorID)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		movs[i].ID = db.U64(id)
	}
	return nil
}

// AddBalance applies a signed delta to the (tenant, location, product,
// variant, batch) balance row inside q's transaction, maintaining the
// weighted-average cost on inbound moves that carry a unit cost.
func (r *Repo) AddBalance(ctx context.Context, q db.Querier, m Movement) error {
	if m.UnitCost != nil && m.Qty.GreaterThan(decimal.Zero) {
		// Merge new inbound cost into the existing weighted average:
		// avg = (qty*avg + in*cost) / (qty + in).
		_, err := q.ExecContext(ctx,
			`INSERT INTO stock_balances
			 (tenant_id, branch_id, location_id, product_id, variant_id, batch_id, qty, avg_cost)
			 VALUES (?,?,?,?,?,?,?,?)
			 ON DUPLICATE KEY UPDATE
			   avg_cost = IF(qty + VALUES(qty) > 0,
			     (qty*avg_cost + VALUES(qty)*VALUES(avg_cost)) / (qty + VALUES(qty)),
			     VALUES(avg_cost)),
			   qty = qty + VALUES(qty)`,
			m.TenantID, m.BranchID, m.LocationID, m.ProductID, m.VariantID, m.BatchID,
			m.Qty, *m.UnitCost)
		return err
	}
	_, err := q.ExecContext(ctx,
		`INSERT INTO stock_balances
		 (tenant_id, branch_id, location_id, product_id, variant_id, batch_id, qty)
		 VALUES (?,?,?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE qty = qty + VALUES(qty)`,
		m.TenantID, m.BranchID, m.LocationID, m.ProductID, m.VariantID, m.BatchID, m.Qty)
	return err
}

// BalanceFilter narrows the balance list.
type BalanceFilter struct {
	BranchID   uint64
	LocationID uint64
	ProductID  uint64
	LowStock   bool // only rows at/below their reorder level
}

const balanceSelect = `SELECT b.branch_id, b.location_id, b.product_id, b.variant_id, b.batch_id,
	b.qty, b.avg_cost, p.name AS product_name, p.sku AS product_sku,
	p.public_id AS product_public_id, COALESCE(v.public_id,'') AS variant_public_id,
	COALESCE(v.name,'') AS variant_name, l.name AS location_name, l.public_id AS location_public_id,
	br.name AS branch_name, br.public_id AS branch_public_id
 FROM stock_balances b
 JOIN products p ON p.id=b.product_id
 LEFT JOIN product_variants v ON v.id=b.variant_id
 JOIN stock_locations l ON l.id=b.location_id
 JOIN branches br ON br.id=b.branch_id`

// Balances lists non-zero balances for a tenant, newest-touched first.
func (r *Repo) Balances(ctx context.Context, tenantID uint64, f BalanceFilter, pg httpx.Page) ([]Balance, int, error) {
	where := `b.tenant_id=? AND b.qty<>0`
	args := []any{tenantID}
	if f.BranchID != 0 {
		where += ` AND b.branch_id=?`
		args = append(args, f.BranchID)
	}
	if f.LocationID != 0 {
		where += ` AND b.location_id=?`
		args = append(args, f.LocationID)
	}
	if f.ProductID != 0 {
		where += ` AND b.product_id=?`
		args = append(args, f.ProductID)
	}
	if f.LowStock {
		where += ` AND EXISTS (SELECT 1 FROM reorder_levels rl
			WHERE rl.tenant_id=b.tenant_id AND rl.branch_id=b.branch_id
			  AND rl.product_id=b.product_id AND rl.variant_id=b.variant_id
			  AND b.qty <= rl.min_qty)`
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM stock_balances b WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Balance
	err := r.db.SelectContext(ctx, &out, balanceSelect+` WHERE `+where+
		` ORDER BY p.name, l.id LIMIT ? OFFSET ?`, append(args, pg.PerPage, pg.Offset())...)
	return out, total, err
}

// MovementFilter narrows the ledger view.
type MovementFilter struct {
	BranchID  uint64
	ProductID uint64
	Kind      string
	RefType   string
	RefID     string
}

// Movements lists ledger rows, newest first.
func (r *Repo) Movements(ctx context.Context, tenantID uint64, f MovementFilter, pg httpx.Page) ([]Movement, int, error) {
	where := `m.tenant_id=?`
	args := []any{tenantID}
	if f.BranchID != 0 {
		where += ` AND m.branch_id=?`
		args = append(args, f.BranchID)
	}
	if f.ProductID != 0 {
		where += ` AND m.product_id=?`
		args = append(args, f.ProductID)
	}
	if f.Kind != "" {
		where += ` AND m.kind=?`
		args = append(args, f.Kind)
	}
	if f.RefType != "" && f.RefID != "" {
		where += ` AND m.ref_type=? AND m.ref_public_id=?`
		args = append(args, f.RefType, f.RefID)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM stock_movements m WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Movement
	err := r.db.SelectContext(ctx, &out,
		`SELECT m.*, p.name AS product_name, p.sku AS product_sku,
			COALESCE(v.name,'') AS variant_name, l.name AS location_name, br.name AS branch_name
		 FROM stock_movements m
		 JOIN products p ON p.id=m.product_id
		 LEFT JOIN product_variants v ON v.id=m.variant_id
		 JOIN stock_locations l ON l.id=m.location_id
		 JOIN branches br ON br.id=m.branch_id
		 WHERE `+where+` ORDER BY m.id DESC LIMIT ? OFFSET ?`,
		append(args, pg.PerPage, pg.Offset())...)
	return out, total, err
}

// BranchBalance is the summed quantity for a product across a branch's
// locations (used by counts and low-stock alerts).
func (r *Repo) BranchBalance(ctx context.Context, q db.Querier, tenantID, branchID, productID, variantID, batchID uint64) (decimal.Decimal, error) {
	var qty decimal.NullDecimal
	if err := q.GetContext(ctx, &qty,
		`SELECT SUM(qty) FROM stock_balances
		 WHERE tenant_id=? AND branch_id=? AND product_id=? AND variant_id=? AND batch_id=?`,
		tenantID, branchID, productID, variantID, batchID); err != nil {
		return decimal.Zero, err
	}
	if !qty.Valid {
		return decimal.Zero, nil
	}
	return qty.Decimal, nil
}

// BatchBalance is one (batch, on-hand) pair used by FEFO allocation.
type BatchBalance struct {
	BatchID uint64
	Qty     decimal.Decimal
}

// BatchBalances returns per-batch on-hand quantities for a product at a
// branch, FEFO ordered (earliest expiry first, unexpired last, then oldest).
func (r *Repo) BatchBalances(ctx context.Context, q db.Querier, tenantID, locationID, productID, variantID uint64) ([]BatchBalance, error) {
	var rows []struct {
		BatchID uint64          `db:"batch_id"`
		Qty     decimal.Decimal `db:"qty"`
	}
	if err := q.SelectContext(ctx, &rows,
		`SELECT b.batch_id, b.qty FROM stock_balances b
		 LEFT JOIN product_batches pb ON pb.id=b.batch_id
		 WHERE b.tenant_id=? AND b.location_id=? AND b.product_id=? AND b.variant_id=?
		   AND b.batch_id<>0 AND b.qty>0
		 ORDER BY pb.expiry_date IS NULL, pb.expiry_date, pb.received_at`,
		tenantID, locationID, productID, variantID); err != nil {
		return nil, err
	}
	out := make([]BatchBalance, len(rows))
	for i, r := range rows {
		out[i] = BatchBalance{BatchID: r.BatchID, Qty: r.Qty}
	}
	return out, nil
}

// AvgCost returns the weighted-average cost of a product at a branch
// (positive balances only); zero when no stock is held.
func (r *Repo) AvgCost(ctx context.Context, q db.Querier, tenantID, branchID, productID, variantID uint64) (decimal.Decimal, error) {
	var v decimal.NullDecimal
	err := q.GetContext(ctx, &v,
		`SELECT SUM(qty*avg_cost)/SUM(qty) FROM stock_balances
		 WHERE tenant_id=? AND branch_id=? AND product_id=? AND variant_id=? AND qty>0`,
		tenantID, branchID, productID, variantID)
	if err != nil {
		return decimal.Zero, err
	}
	if !v.Valid {
		return decimal.Zero, nil
	}
	return v.Decimal, nil
}

// ExpectedRows lists balance rows a count covers (all positive balances in
// the branch, optionally one location).
func (r *Repo) ExpectedRows(ctx context.Context, tenantID, branchID uint64, locationID *uint64) ([]Balance, error) {
	where := `b.tenant_id=? AND b.branch_id=? AND b.qty<>0`
	args := []any{tenantID, branchID}
	if locationID != nil {
		where += ` AND b.location_id=?`
		args = append(args, *locationID)
	}
	var out []Balance
	return out, r.db.SelectContext(ctx, &out, balanceSelect+` WHERE `+where+` ORDER BY p.name`, args...)
}

// UpsertBatch finds or creates a batch for a product/variant/batch_no.
func (r *Repo) UpsertBatch(ctx context.Context, q db.Querier, b *Batch) error {
	err := q.GetContext(ctx, b,
		`SELECT * FROM product_batches
		 WHERE tenant_id=? AND product_id=? AND variant_id=? AND batch_no=?`,
		b.TenantID, b.ProductID, b.VariantID, b.BatchNo)
	if err == nil {
		return nil
	}
	if !db.IsNotFound(err) {
		return err
	}
	res, err := q.ExecContext(ctx,
		`INSERT INTO product_batches (tenant_id, product_id, variant_id, batch_no, expiry_date)
		 VALUES (?,?,?,?,?)`, b.TenantID, b.ProductID, b.VariantID, b.BatchNo, b.ExpiryDate)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	b.ID = db.U64(id)
	return nil
}

// Batches lists a product's batches (expiry first).
func (r *Repo) Batches(ctx context.Context, tenantID, productID uint64) ([]Batch, error) {
	var out []Batch
	return out, r.db.SelectContext(ctx, &out,
		`SELECT * FROM product_batches WHERE tenant_id=? AND product_id=?
		 ORDER BY expiry_date IS NULL, expiry_date`, tenantID, productID)
}

// ExpiringRow is one soon-to-expire batch still in stock.
type ExpiringRow struct {
	Batch
	Qty         decimal.Decimal `db:"qty"`
	BranchName  string          `db:"branch_name"`
	ProductName string          `db:"product_name"`
}

// Expiring lists batches with positive stock expiring within days.
func (r *Repo) Expiring(ctx context.Context, tenantID uint64, days int) ([]ExpiringRow, error) {
	var out []ExpiringRow
	return out, r.db.SelectContext(ctx, &out,
		`SELECT pb.*, b.qty, br.name AS branch_name, p.name AS product_name
		 FROM product_batches pb
		 JOIN stock_balances b ON b.batch_id=pb.id AND b.qty>0
		 JOIN branches br ON br.id=b.branch_id
		 JOIN products p ON p.id=pb.product_id
		 WHERE pb.tenant_id=? AND pb.expiry_date IS NOT NULL
		   AND pb.expiry_date <= DATE_ADD(CURDATE(), INTERVAL ? DAY)
		 ORDER BY pb.expiry_date`, tenantID, days)
}

// SetReorder upserts a reorder level.
func (r *Repo) SetReorder(ctx context.Context, tenantID uint64, rl ReorderLevel) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reorder_levels (tenant_id, branch_id, product_id, variant_id, min_qty, target_qty)
		 VALUES (?,?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE min_qty=VALUES(min_qty), target_qty=VALUES(target_qty)`,
		tenantID, rl.BranchID, rl.ProductID, rl.VariantID, rl.MinQty, rl.TargetQty)
	return err
}

// ReorderRows lists reorder levels with current branch stock.
func (r *Repo) ReorderRows(ctx context.Context, tenantID, branchID uint64) ([]ReorderRow, error) {
	var out []ReorderRow
	return out, r.db.SelectContext(ctx, &out,
		`SELECT rl.branch_id, rl.product_id, rl.variant_id, rl.min_qty, rl.target_qty,
			p.name AS product_name, p.sku AS product_sku, p.public_id AS product_public_id,
			COALESCE(v.public_id,'') AS variant_public_id, COALESCE(v.name,'') AS variant_name,
			COALESCE((SELECT SUM(b.qty) FROM stock_balances b
				WHERE b.tenant_id=rl.tenant_id AND b.branch_id=rl.branch_id
				  AND b.product_id=rl.product_id AND b.variant_id=rl.variant_id),0) AS qty
		 FROM reorder_levels rl
		 JOIN products p ON p.id=rl.product_id
		 LEFT JOIN product_variants v ON v.id=rl.variant_id
		 WHERE rl.tenant_id=? AND rl.branch_id=? ORDER BY p.name`, tenantID, branchID)
}

// ReorderRow is a reorder level plus its current branch stock.
type ReorderRow struct {
	ReorderLevel
	ProductName     string          `db:"product_name"`
	ProductSKU      string          `db:"product_sku"`
	ProductPublicID string          `db:"product_public_id"`
	VariantPublicID string          `db:"variant_public_id"`
	VariantName     string          `db:"variant_name"`
	Qty             decimal.Decimal `db:"qty"`
}

// DeleteReorder removes a reorder level.
func (r *Repo) DeleteReorder(ctx context.Context, tenantID, branchID, productID, variantID uint64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM reorder_levels WHERE tenant_id=? AND branch_id=? AND product_id=? AND variant_id=?`,
		tenantID, branchID, productID, variantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Valuation is the stock-on-hand value for a branch at weighted average cost.
type Valuation struct {
	BranchID   uint64          `db:"branch_id"`
	BranchName string          `db:"branch_name"`
	SkuCount   int             `db:"sku_count"`
	Units      decimal.Decimal `db:"units"`
	Value      decimal.Decimal `db:"value"`
}

// Valuations sums balance values per branch.
func (r *Repo) Valuations(ctx context.Context, tenantID uint64) ([]Valuation, error) {
	var out []Valuation
	return out, r.db.SelectContext(ctx, &out,
		`SELECT b.branch_id, br.name AS branch_name,
			COUNT(DISTINCT b.product_id, b.variant_id) AS sku_count,
			COALESCE(SUM(b.qty),0) AS units, COALESCE(SUM(b.qty*b.avg_cost),0) AS value
		 FROM stock_balances b JOIN branches br ON br.id=b.branch_id
		 WHERE b.tenant_id=? AND b.qty>0
		 GROUP BY b.branch_id, br.name ORDER BY br.name`, tenantID)
}
