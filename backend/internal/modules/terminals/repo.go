package terminals

import (
	"context"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the pos_terminals store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const cols = `id, tenant_id, branch_id, public_id, name, status, created_at, updated_at`

// List returns a tenant's terminals.
func (r *Repo) List(ctx context.Context, tenantID uint64) ([]Terminal, error) {
	var out []Terminal
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+cols+` FROM pos_terminals WHERE tenant_id = ? ORDER BY name`, tenantID)
	return out, err
}

// Count returns how many terminals a tenant has.
func (r *Repo) Count(ctx context.Context, tenantID uint64) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM pos_terminals WHERE tenant_id = ?`, tenantID)
	return n, err
}

// ByPublicID loads one terminal scoped to a tenant.
func (r *Repo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Terminal, error) {
	var t Terminal
	err := r.db.GetContext(ctx, &t,
		`SELECT `+cols+` FROM pos_terminals WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return t, err
}

// Create inserts a terminal and hydrates the row.
func (r *Repo) Create(ctx context.Context, t *Terminal) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pos_terminals (tenant_id, branch_id, public_id, name, status)
		 VALUES (?, ?, ?, ?, ?)`,
		t.TenantID, t.BranchID, t.PublicID, t.Name, t.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = db.U64(id)
	fresh, err := r.ByPublicID(ctx, t.TenantID, t.PublicID)
	if err != nil {
		return err
	}
	*t = fresh
	return nil
}
