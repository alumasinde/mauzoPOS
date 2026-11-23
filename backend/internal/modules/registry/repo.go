package registry

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo stores modules and tenant_modules. q is the pool or, inside InTx, the
// transaction.
type Repo struct {
	q    db.Querier
	conn *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{q: conn, conn: conn} }

// WithQuerier binds the repo to q so callers can join their transaction.
func (r *Repo) WithQuerier(q db.Querier) *Repo { return &Repo{q: q, conn: r.conn} }

// InTx runs fn with a repo bound to one transaction.
func (r *Repo) InTx(ctx context.Context, fn func(txr *Repo) error) error {
	return db.WithTx(ctx, r.conn, func(tx *sqlx.Tx) error { return fn(r.WithQuerier(tx)) })
}

// ─── modules (platform switch) ──────────────────────────────────────────────

// SyncCodes inserts any registry code missing from the modules table (active).
func (r *Repo) SyncCodes(ctx context.Context, codes []string) error {
	for _, c := range codes {
		if _, err := r.q.ExecContext(ctx,
			`INSERT INTO modules (code, is_active) VALUES (?, 1) ON DUPLICATE KEY UPDATE code = code`, c); err != nil {
			return err
		}
	}
	return nil
}

// PlatformModules returns every modules row.
func (r *Repo) PlatformModules(ctx context.Context) ([]PlatformModule, error) {
	var out []PlatformModule
	err := r.q.SelectContext(ctx, &out, `SELECT code, is_active, updated_at FROM modules ORDER BY code`)
	return out, err
}

// SetActive flips the platform switch for a module.
func (r *Repo) SetActive(ctx context.Context, code string, active bool) error {
	_, err := r.q.ExecContext(ctx, `UPDATE modules SET is_active = ? WHERE code = ?`, active, code)
	return err
}

// ─── tenant_modules ─────────────────────────────────────────────────────────

const tmCols = `tenant_id, module, enabled, source, config, updated_at`

// TenantModules returns every tenant_modules row of a tenant.
func (r *Repo) TenantModules(ctx context.Context, tenantID uint64) ([]TenantModule, error) {
	var out []TenantModule
	err := r.q.SelectContext(ctx, &out,
		`SELECT `+tmCols+` FROM tenant_modules WHERE tenant_id = ? ORDER BY module`, tenantID)
	return out, err
}

// SetEnabled upserts enabled/source for one module, keeping its config.
func (r *Repo) SetEnabled(ctx context.Context, tenantID uint64, code string, enabled bool, source string) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO tenant_modules (tenant_id, module, enabled, source) VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE enabled = VALUES(enabled), source = VALUES(source)`,
		tenantID, code, enabled, source)
	return err
}

// SetConfig stores a module's validated config.
func (r *Repo) SetConfig(ctx context.Context, tenantID uint64, code string, cfg json.RawMessage) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE tenant_modules SET config = ? WHERE tenant_id = ? AND module = ?`, cfg, tenantID, code)
	return err
}

// TenantsWithModule lists tenant ids that have code enabled (cache
// invalidation after a platform switch).
func (r *Repo) TenantsWithModule(ctx context.Context, code string) ([]uint64, error) {
	var out []uint64
	err := r.q.SelectContext(ctx, &out,
		`SELECT tenant_id FROM tenant_modules WHERE module = ? AND enabled = 1`, code)
	return out, err
}
