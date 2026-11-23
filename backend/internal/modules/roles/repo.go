package roles

import (
	"context"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo stores roles and role_permissions. Every query is tenant-scoped except
// the boot-time template sync. q is the pool or, inside InTx, a transaction.
type Repo struct {
	q    db.Querier
	conn *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{q: conn, conn: conn} }

// InTx runs fn with a repo bound to one transaction.
func (r *Repo) InTx(ctx context.Context, fn func(txr *Repo) error) error {
	return db.WithTx(ctx, r.conn, func(tx *sqlx.Tx) error { return fn(&Repo{q: tx, conn: r.conn}) })
}

const roleCols = `r.id, r.tenant_id, r.public_id, r.code, r.name, r.description, r.is_system,
	r.is_locked, r.customized, r.max_discount_percent, r.max_refund_amount,
	r.max_cash_out_amount, r.created_at, r.updated_at,
	(SELECT COUNT(*) FROM users u WHERE u.role_id = r.id) AS user_count`

// List returns a tenant's roles, system roles first.
func (r *Repo) List(ctx context.Context, tenantID uint64) ([]Role, error) {
	var out []Role
	err := r.q.SelectContext(ctx, &out,
		`SELECT `+roleCols+` FROM roles r WHERE r.tenant_id = ?
		 ORDER BY r.is_system DESC, r.is_locked DESC, r.name`, tenantID)
	return out, err
}

// ByPublicID loads one role.
func (r *Repo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Role, error) {
	var out Role
	err := r.q.GetContext(ctx, &out,
		`SELECT `+roleCols+` FROM roles r WHERE r.tenant_id = ? AND r.public_id = ?`, tenantID, publicID)
	return out, err
}

// ByID loads one role.
func (r *Repo) ByID(ctx context.Context, tenantID, id uint64) (Role, error) {
	var out Role
	err := r.q.GetContext(ctx, &out,
		`SELECT `+roleCols+` FROM roles r WHERE r.tenant_id = ? AND r.id = ?`, tenantID, id)
	return out, err
}

// ByCode loads one role by code.
func (r *Repo) ByCode(ctx context.Context, tenantID uint64, code string) (Role, error) {
	var out Role
	err := r.q.GetContext(ctx, &out,
		`SELECT `+roleCols+` FROM roles r WHERE r.tenant_id = ? AND r.code = ?`, tenantID, code)
	return out, err
}

// Insert adds a role (ignored when the code already exists) and reports
// whether a row was created.
func (r *Repo) Insert(ctx context.Context, role *Role) (bool, error) {
	res, err := r.q.ExecContext(ctx,
		`INSERT INTO roles (tenant_id, public_id, code, name, description, is_system, is_locked,
		   customized, max_discount_percent, max_refund_amount, max_cash_out_amount)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id = id`,
		role.TenantID, role.PublicID, role.Code, role.Name, role.Description, role.IsSystem,
		role.IsLocked, role.Customized, role.MaxDiscountPercent, role.MaxRefundAmount, role.MaxCashOutAmount)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	fresh, err := r.ByCode(ctx, role.TenantID, role.Code)
	if err != nil {
		return false, err
	}
	*role = fresh
	return true, nil
}

// Update saves name, description, limits and the customized flag.
func (r *Repo) Update(ctx context.Context, role Role) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE roles SET name = ?, description = ?, customized = ?, max_discount_percent = ?,
		   max_refund_amount = ?, max_cash_out_amount = ?
		 WHERE tenant_id = ? AND id = ?`,
		role.Name, role.Description, role.Customized, role.MaxDiscountPercent,
		role.MaxRefundAmount, role.MaxCashOutAmount, role.TenantID, role.ID)
	return err
}

// Delete removes a role (its grants cascade).
func (r *Repo) Delete(ctx context.Context, tenantID, id uint64) error {
	_, err := r.q.ExecContext(ctx, `DELETE FROM roles WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

// Permissions returns the codes granted to a role.
func (r *Repo) Permissions(ctx context.Context, roleID uint64) ([]string, error) {
	var out []string
	err := r.q.SelectContext(ctx, &out,
		`SELECT permission FROM role_permissions WHERE role_id = ? ORDER BY permission`, roleID)
	return out, err
}

// SetPermissions replaces a role's grants. Call inside InTx.
func (r *Repo) SetPermissions(ctx context.Context, roleID uint64, codes []string) error {
	if _, err := r.q.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, c := range codes {
		if _, err := r.q.ExecContext(ctx,
			`INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)`, roleID, c); err != nil {
			return err
		}
	}
	return nil
}

// templateRole is the subset the boot-time template sync needs.
type templateRole struct {
	ID         uint64 `db:"id"`
	TenantID   uint64 `db:"tenant_id"`
	Code       string `db:"code"`
	Customized bool   `db:"customized"`
	IsLocked   bool   `db:"is_locked"`
}

// TemplateRoles returns system roles that still follow their template,
// optionally for one tenant (0 = all tenants).
func (r *Repo) TemplateRoles(ctx context.Context, tenantID uint64) ([]templateRole, error) {
	var out []templateRole
	q := `SELECT id, tenant_id, code, customized, is_locked FROM roles
	      WHERE is_system = 1 AND customized = 0 AND is_locked = 0`
	args := []any{}
	if tenantID != 0 {
		q += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	err := r.q.SelectContext(ctx, &out, q, args...)
	return out, err
}

// CountActiveWithRole counts active users holding a role (last-owner guard).
func (r *Repo) CountActiveWithRole(ctx context.Context, tenantID, roleID uint64) (int, error) {
	var n int
	err := r.q.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND role_id = ? AND status = 'active'`, tenantID, roleID)
	return n, err
}
