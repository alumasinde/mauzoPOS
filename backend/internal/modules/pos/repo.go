package pos

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo owns terminal-device and held-sale persistence plus the read queries
// that build the sync bundle.
type Repo struct {
	conn *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{conn: conn} }

type terminalRow struct {
	ID                 uint64     `db:"id"`
	TenantID           uint64     `db:"tenant_id"`
	BranchID           uint64     `db:"branch_id"`
	PublicID           string     `db:"public_id"`
	Name               string     `db:"name"`
	Status             string     `db:"status"`
	BranchName         string     `db:"branch_name"`
	TenantName         string     `db:"tenant_name"`
	TenantSlug         string     `db:"tenant_slug"`
	BusinessType       string     `db:"business_type"`
	ActivationCodeHash *string    `db:"activation_code_hash"`
	DeviceID           *string    `db:"device_id"`
	ActivatedAt        *time.Time `db:"activated_at"`
}

const terminalCols = `t.id, t.tenant_id, t.branch_id, t.public_id, t.name, t.status,
	t.activation_code_hash, t.device_id, t.activated_at,
	b.name AS branch_name, tn.name AS tenant_name, tn.slug AS tenant_slug,
	COALESCE(tn.business_type,'') AS business_type`

func (r *Repo) terminalBy(ctx context.Context, where string, arg any) (terminalRow, error) {
	var row terminalRow
	err := r.conn.GetContext(ctx, &row,
		`SELECT `+terminalCols+` FROM pos_terminals t
		 JOIN branches b ON b.id = t.branch_id
		 JOIN tenants tn ON tn.id = t.tenant_id WHERE `+where, arg)
	return row, err
}

// terminalByTokenHash resolves a device bearer token to its terminal.
func (r *Repo) terminalByTokenHash(ctx context.Context, hash string) (terminalRow, error) {
	return r.terminalBy(ctx, `t.device_token_hash = ?`, hash)
}

// terminalByCodeHash finds a pending terminal by its activation code hash.
func (r *Repo) terminalByCodeHash(ctx context.Context, hash string) (terminalRow, error) {
	return r.terminalBy(ctx, `t.activation_code_hash = ?`, hash)
}

// SetActivationCode stores (or rotates) the one-time activation code hash.
func (r *Repo) SetActivationCode(ctx context.Context, q db.Querier, terminalID uint64, hash *string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE pos_terminals SET activation_code_hash = ? WHERE id = ?`, hash, terminalID)
	return err
}

// Activate marks the terminal active and binds the device.
func (r *Repo) Activate(ctx context.Context, q db.Querier, terminalID uint64,
	deviceID, deviceName, tokenHash string) error {
	res, err := q.ExecContext(ctx,
		`UPDATE pos_terminals SET status = 'active', device_id = ?, device_name = ?,
			device_token_hash = ?, activation_code_hash = NULL, activated_at = NOW(3), last_seen_at = NOW(3)
		 WHERE id = ? AND status <> 'disabled'`,
		deviceID, deviceName, tokenHash, terminalID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}

// Touch records a heartbeat.
func (r *Repo) Touch(ctx context.Context, terminalID uint64) {
	_, _ = r.conn.ExecContext(ctx,
		`UPDATE pos_terminals SET last_seen_at = NOW(3) WHERE id = ?`, terminalID)
}

// ---- sync bundle ------------------------------------------------------------

// Categories returns every category.
func (r *Repo) Categories(ctx context.Context, tenantID uint64) ([]SyncCategory, error) {
	var out []SyncCategory
	err := r.conn.SelectContext(ctx, &out,
		`SELECT id, public_id, COALESCE(parent_id,0) AS parent_id, name, sort
		 FROM categories WHERE tenant_id = ? ORDER BY sort, name`, tenantID)
	return out, err
}

// Units returns tenant + built-in units.
func (r *Repo) Units(ctx context.Context, tenantID uint64) ([]SyncUnit, error) {
	var out []SyncUnit
	err := r.conn.SelectContext(ctx, &out,
		`SELECT id, code, name, kind, COALESCE(factor,1) AS factor
		 FROM units WHERE tenant_id = ? OR tenant_id IS NULL`, tenantID)
	return out, err
}

// Products returns active sellable products changed since the cursor.
func (r *Repo) Products(ctx context.Context, tenantID uint64, since time.Time) ([]SyncProduct, error) {
	var out []SyncProduct
	err := r.conn.SelectContext(ctx, &out,
		`SELECT p.id, p.public_id, p.sku, p.name, p.kind,
			COALESCE(p.category_id,0) AS category_id,
			COALESCE(u.code,'pc') AS unit_code,
			p.sell_by, p.price, p.vat_rate, p.track_stock, p.updated_at
		 FROM products p LEFT JOIN units u ON u.id = p.unit_id
		 WHERE p.tenant_id = ? AND p.status = 'active' AND p.updated_at > ?
		 ORDER BY p.name`, tenantID, since)
	return out, err
}

// Variants returns variants for the tenant's active products.
func (r *Repo) Variants(ctx context.Context, tenantID uint64) ([]SyncVariant, error) {
	var out []SyncVariant
	err := r.conn.SelectContext(ctx, &out,
		`SELECT v.id, v.product_id, v.public_id, COALESCE(v.sku,'') AS sku, v.name, v.price
		 FROM product_variants v JOIN products p ON p.id = v.product_id
		 WHERE v.tenant_id = ? AND p.status = 'active'`, tenantID)
	return out, err
}

// Barcodes returns every scannable code for active products.
func (r *Repo) Barcodes(ctx context.Context, tenantID uint64) ([]SyncBarcode, error) {
	var out []SyncBarcode
	err := r.conn.SelectContext(ctx, &out,
		`SELECT b.product_id, COALESCE(b.variant_id,0) AS variant_id, b.barcode, b.pack_qty
		 FROM product_barcodes b JOIN products p ON p.id = b.product_id
		 WHERE b.tenant_id = ? AND p.status = 'active'`, tenantID)
	return out, err
}

// PriceLists returns active lists usable in this branch right now.
func (r *Repo) PriceLists(ctx context.Context, tenantID, branchID uint64, now time.Time) ([]SyncPriceList, error) {
	var out []SyncPriceList
	err := r.conn.SelectContext(ctx, &out,
		`SELECT id, COALESCE(branch_id,0) AS branch_id, COALESCE(customer_group,'') AS customer_group,
			priority, valid_from, valid_to
		 FROM price_lists
		 WHERE tenant_id = ? AND is_active = 1
		   AND (branch_id IS NULL OR branch_id = ?)
		   AND (valid_from IS NULL OR valid_from <= ?) AND (valid_to IS NULL OR valid_to >= ?)`,
		tenantID, branchID, now, now)
	return out, err
}

// PriceItems returns tier rows for the lists PriceLists would return (same
// branch + validity filters) so the terminal never gets an orphan row.
func (r *Repo) PriceItems(ctx context.Context, tenantID, branchID uint64, now time.Time) ([]SyncPriceItem, error) {
	var out []SyncPriceItem
	err := r.conn.SelectContext(ctx, &out,
		`SELECT i.list_id, i.product_id, i.variant_id, i.min_qty, i.price
		 FROM price_list_items i JOIN price_lists l ON l.id = i.list_id
		 WHERE l.tenant_id = ? AND l.is_active = 1
		   AND (l.branch_id IS NULL OR l.branch_id = ?)
		   AND (l.valid_from IS NULL OR l.valid_from <= ?) AND (l.valid_to IS NULL OR l.valid_to >= ?)`,
		tenantID, branchID, now, now)
	return out, err
}

// Promotions returns active promotions inside their date window.
func (r *Repo) Promotions(ctx context.Context, tenantID uint64, now time.Time) ([]SyncPromotion, error) {
	var out []SyncPromotion
	err := r.conn.SelectContext(ctx, &out,
		`SELECT id, name, kind, config, starts_at, ends_at, days_mask, priority
		 FROM promotions
		 WHERE tenant_id = ? AND is_active = 1
		   AND (starts_at IS NULL OR starts_at <= ?) AND (ends_at IS NULL OR ends_at >= ?)`,
		tenantID, now, now)
	return out, err
}

// PromoTargets returns targeting rows for the promotions Promotions would
// return (same validity window) so the terminal never gets an orphan row.
func (r *Repo) PromoTargets(ctx context.Context, tenantID uint64, now time.Time) ([]SyncPromoTarget, error) {
	var out []SyncPromoTarget
	err := r.conn.SelectContext(ctx, &out,
		`SELECT t.promotion_id, COALESCE(t.product_id,0) AS product_id,
			COALESCE(t.category_id,0) AS category_id
		 FROM promotion_targets t JOIN promotions p ON p.id = t.promotion_id
		 WHERE p.tenant_id = ? AND p.is_active = 1
		   AND (p.starts_at IS NULL OR p.starts_at <= ?) AND (p.ends_at IS NULL OR p.ends_at >= ?)`,
		tenantID, now, now)
	return out, err
}

// Cashiers returns active users who can sign in with a PIN.
func (r *Repo) Cashiers(ctx context.Context, tenantID uint64) ([]SyncCashier, error) {
	var out []SyncCashier
	err := r.conn.SelectContext(ctx, &out,
		`SELECT u.id, u.public_id, u.name, r.name AS role_name, u.pin_hash, u.all_branches
		 FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE u.tenant_id = ? AND u.status = 'active' AND u.pin_hash IS NOT NULL
		 ORDER BY u.name`, tenantID)
	return out, err
}

// CashierBranches returns explicit branch grants (all_branches=0 users).
func (r *Repo) CashierBranches(ctx context.Context, tenantID uint64) ([]SyncCashierBranch, error) {
	var out []SyncCashierBranch
	err := r.conn.SelectContext(ctx, &out,
		`SELECT ub.user_id, ub.branch_id FROM user_branches ub
		 JOIN users u ON u.id = ub.user_id
		 WHERE u.tenant_id = ? AND u.status = 'active'`, tenantID)
	return out, err
}

// CashierPermissions returns role → permission grants for tenant roles.
func (r *Repo) CashierPermissions(ctx context.Context, tenantID uint64) (map[uint64][]string, error) {
	rows := []struct {
		UserID     uint64 `db:"user_id"`
		Permission string `db:"permission"`
	}{}
	err := r.conn.SelectContext(ctx, &rows,
		`SELECT u.id AS user_id, rp.permission
		 FROM users u JOIN role_permissions rp ON rp.role_id = u.role_id
		 WHERE u.tenant_id = ? AND u.status = 'active'`, tenantID)
	if err != nil {
		return nil, err
	}
	out := map[uint64][]string{}
	for _, row := range rows {
		out[row.UserID] = append(out[row.UserID], row.Permission)
	}
	return out, nil
}

// TenantSettings returns POS-relevant tenant settings.
func (r *Repo) TenantSettings(ctx context.Context, tenantID uint64) ([]SyncSetting, error) {
	var out []SyncSetting
	err := r.conn.SelectContext(ctx, &out,
		`SELECT `+"`key`"+` AS `+"`key`"+`, `+"`value`"+` AS `+"`value`"+`
		 FROM tenant_settings WHERE tenant_id = ?`, tenantID)
	return out, err
}

// UserPIN returns (pin_hash, role_id, status) for PIN verification.
func (r *Repo) UserPIN(ctx context.Context, tenantID, userID uint64) (string, uint64, string, error) {
	var row struct {
		PINHash *string `db:"pin_hash"`
		RoleID  uint64  `db:"role_id"`
		Status  string  `db:"status"`
	}
	err := r.conn.GetContext(ctx, &row,
		`SELECT pin_hash, role_id, status FROM users WHERE tenant_id = ? AND id = ?`,
		tenantID, userID)
	if err != nil {
		return "", 0, "", err
	}
	if row.PINHash == nil {
		return "", 0, "", sql.ErrNoRows
	}
	return *row.PINHash, row.RoleID, row.Status, nil
}

// UsersWithPermission returns active users whose role grants a permission.
func (r *Repo) UsersWithPermission(ctx context.Context, tenantID uint64, perm string) ([]SyncCashier, error) {
	var out []SyncCashier
	err := r.conn.SelectContext(ctx, &out,
		`SELECT u.id, u.public_id, u.name, ro.name AS role_name, u.pin_hash, u.all_branches
		 FROM users u JOIN roles ro ON ro.id = u.role_id
		 JOIN role_permissions rp ON rp.role_id = ro.id AND rp.permission = ?
		 WHERE u.tenant_id = ? AND u.status = 'active' AND u.pin_hash IS NOT NULL`,
		perm, tenantID)
	return out, err
}

// ---- held sales --------------------------------------------------------------

// HoldCreate inserts a held sale.
func (r *Repo) HoldCreate(ctx context.Context, h HeldSale) (uint64, error) {
	res, err := r.conn.ExecContext(ctx,
		`INSERT INTO held_sales (tenant_id, branch_id, terminal_id, user_id, public_id, label, cart)
		 VALUES (?,?,?,?,?,?,?)`,
		h.TenantID, h.BranchID, h.TerminalID, h.UserID, h.PublicID, h.Label, h.Cart)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // AUTO_INCREMENT ids are non-negative
}

// HeldList returns parked carts for the branch.
func (r *Repo) HeldList(ctx context.Context, tenantID, branchID uint64) ([]HeldSale, error) {
	var out []HeldSale
	err := r.conn.SelectContext(ctx, &out,
		`SELECT h.*, u.name AS held_by, t.name AS terminal_name
		 FROM held_sales h JOIN users u ON u.id = h.user_id
		 JOIN pos_terminals t ON t.id = h.terminal_id
		 WHERE h.tenant_id = ? AND h.branch_id = ? AND h.status = 'held'
		 ORDER BY h.created_at DESC`, tenantID, branchID)
	return out, err
}

// HeldByPublicID loads one held sale.
func (r *Repo) HeldByPublicID(ctx context.Context, tenantID, branchID uint64, publicID string) (HeldSale, error) {
	var h HeldSale
	err := r.conn.GetContext(ctx, &h,
		`SELECT h.*, u.name AS held_by, t.name AS terminal_name
		 FROM held_sales h JOIN users u ON u.id = h.user_id
		 JOIN pos_terminals t ON t.id = h.terminal_id
		 WHERE h.tenant_id = ? AND h.branch_id = ? AND h.public_id = ?`,
		tenantID, branchID, publicID)
	return h, err
}

// HeldSetStatus transitions a held sale (recalled | cancelled).
func (r *Repo) HeldSetStatus(ctx context.Context, tenantID, id uint64, status string) error {
	res, err := r.conn.ExecContext(ctx,
		`UPDATE held_sales SET status = ? WHERE tenant_id = ? AND id = ? AND status = 'held'`,
		status, tenantID, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
