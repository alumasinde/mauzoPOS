package branches

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the branches store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const cols = `id, tenant_id, public_id, code, name, phone, location,
	is_main, status, created_at, updated_at`

// List returns a tenant's branches, main first.
func (r *Repo) List(ctx context.Context, tenantID uint64) ([]Branch, error) {
	var out []Branch
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+cols+` FROM branches WHERE tenant_id = ? ORDER BY is_main DESC, name`, tenantID)
	return out, err
}

// Count returns how many branches a tenant has.
func (r *Repo) Count(ctx context.Context, tenantID uint64) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM branches WHERE tenant_id = ?`, tenantID)
	return n, err
}

// ByPublicID loads one branch scoped to a tenant.
func (r *Repo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Branch, error) {
	var b Branch
	err := r.db.GetContext(ctx, &b,
		`SELECT `+cols+` FROM branches WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return b, err
}

// Create inserts a branch and hydrates the row.
func (r *Repo) Create(ctx context.Context, b *Branch) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO branches (tenant_id, public_id, code, name, phone, location, is_main, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.TenantID, b.PublicID, b.Code, b.Name, b.Phone, b.Location, b.IsMain, b.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = db.U64(id)
	fresh, err := r.ByPublicID(ctx, b.TenantID, b.PublicID)
	if err != nil {
		return err
	}
	*b = fresh
	return nil
}

// Update writes an edited branch inside tx.
func (r *Repo) Update(ctx context.Context, tx *sqlx.Tx, b Branch) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE branches SET code=?, name=?, phone=?, location=?, status=?, is_main=?
		 WHERE id=? AND tenant_id=?`,
		b.Code, b.Name, b.Phone, b.Location, b.Status, b.IsMain, b.ID, b.TenantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ClearMain un-mains every branch except the one just made main.
func (r *Repo) ClearMain(ctx context.Context, tx *sqlx.Tx, tenantID, keepID uint64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE branches SET is_main=0 WHERE tenant_id=? AND id<>?`, tenantID, keepID)
	return err
}
