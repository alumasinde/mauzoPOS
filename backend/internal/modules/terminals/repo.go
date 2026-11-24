package terminals

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the pos_terminals store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const cols = `id, tenant_id, branch_id, public_id, name, status, device_name, activated_at, last_seen_at,
	activation_code_expires_at, created_at, updated_at`

const colsJoin = `t.id, t.tenant_id, t.branch_id, t.public_id, t.name, t.status, t.device_name, t.activated_at,
	t.last_seen_at, t.activation_code_expires_at, t.created_at, t.updated_at`

// SetActivationCode stores the hashed one-time code and its expiry; a nil
// hash clears any waiting code.
func (r *Repo) SetActivationCode(ctx context.Context, terminalID uint64, hash *string, expires *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pos_terminals SET activation_code_hash = ?, activation_code_expires_at = ?,
		   activation_code_issued_at = IF(? IS NULL, NULL, NOW(3))
		 WHERE id = ?`, hash, expires, hash, terminalID)
	return err
}

// Rename changes a terminal's display name.
func (r *Repo) Rename(ctx context.Context, terminalID uint64, name string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE pos_terminals SET name = ? WHERE id = ?`, name, terminalID)
	return err
}

// Disable blocks a terminal: its device token and any waiting code stop
// working at once (the till is signed out on its next request).
func (r *Repo) Disable(ctx context.Context, terminalID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pos_terminals SET status = 'disabled', device_token_hash = NULL,
		   activation_code_hash = NULL, activation_code_expires_at = NULL
		 WHERE id = ?`, terminalID)
	return err
}

// Enable lifts a disable. The device must be paired again with a new code
// (its token was cleared), so the terminal returns to pending.
func (r *Repo) Enable(ctx context.Context, terminalID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pos_terminals SET status = 'pending' WHERE id = ? AND status = 'disabled'`, terminalID)
	return err
}

// List returns a tenant's terminals with branch names.
func (r *Repo) List(ctx context.Context, tenantID uint64) ([]Terminal, error) {
	var out []Terminal
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+colsJoin+`, b.name AS branch_name
		 FROM pos_terminals t JOIN branches b ON b.id = t.branch_id
		 WHERE t.tenant_id = ? ORDER BY t.name`, tenantID)
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
