package businesstypes

import (
	"context"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the business_types + business_profiles + tenant_modules store.
// q is the pool or, inside InTx, the transaction.
type Repo struct {
	q    db.Querier
	conn *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{q: conn, conn: conn} }

// InTx runs fn with a Repo bound to a single transaction; tx is passed so
// other stores (e.g. settings) can join the same transaction.
func (r *Repo) InTx(ctx context.Context, fn func(tx *sqlx.Tx, txr *Repo) error) error {
	return db.WithTx(ctx, r.conn, func(tx *sqlx.Tx) error {
		return fn(tx, &Repo{q: tx, conn: r.conn})
	})
}

const typeCols = `id, code, name, sort`

// ListTypes returns every business type ordered by sort.
func (r *Repo) ListTypes(ctx context.Context) ([]BusinessType, error) {
	var out []BusinessType
	err := r.q.SelectContext(ctx, &out, `SELECT `+typeCols+` FROM business_types ORDER BY sort, name`)
	return out, err
}

// TypeByID loads one business type.
func (r *Repo) TypeByID(ctx context.Context, id uint64) (BusinessType, error) {
	var t BusinessType
	err := r.q.GetContext(ctx, &t, `SELECT `+typeCols+` FROM business_types WHERE id = ?`, id)
	return t, err
}

// TypeByCode loads one business type by code.
func (r *Repo) TypeByCode(ctx context.Context, code string) (BusinessType, error) {
	var t BusinessType
	err := r.q.GetContext(ctx, &t, `SELECT `+typeCols+` FROM business_types WHERE code = ?`, code)
	return t, err
}

const profileCols = `p.id, p.public_id, p.business_type_id, p.name, p.is_default,
	p.definition, p.created_at, p.updated_at, bt.code AS type_code`

const profileJoin = ` FROM business_profiles p JOIN business_types bt ON bt.id = p.business_type_id `

// ListProfiles returns profiles, optionally restricted to one type.
func (r *Repo) ListProfiles(ctx context.Context, typeID *uint64) ([]Profile, error) {
	var out []Profile
	q := `SELECT ` + profileCols + profileJoin
	args := []any{}
	if typeID != nil {
		q += ` WHERE p.business_type_id = ?`
		args = append(args, *typeID)
	}
	q += ` ORDER BY bt.sort, p.is_default DESC, p.name`
	err := r.q.SelectContext(ctx, &out, q, args...)
	return out, err
}

// ProfileByPublicID loads one profile by public id.
func (r *Repo) ProfileByPublicID(ctx context.Context, publicID string) (Profile, error) {
	var p Profile
	err := r.q.GetContext(ctx, &p,
		`SELECT `+profileCols+profileJoin+`WHERE p.public_id = ?`, publicID)
	return p, err
}

// DefaultProfile returns the default profile for a business type, or sql.ErrNoRows.
func (r *Repo) DefaultProfile(ctx context.Context, typeID uint64) (Profile, error) {
	var p Profile
	err := r.q.GetContext(ctx, &p,
		`SELECT `+profileCols+profileJoin+`WHERE p.business_type_id = ? AND p.is_default = 1 LIMIT 1`, typeID)
	return p, err
}

// CreateProfile inserts a profile and hydrates the row.
func (r *Repo) CreateProfile(ctx context.Context, p *Profile) error {
	res, err := r.q.ExecContext(ctx,
		`INSERT INTO business_profiles (public_id, business_type_id, name, is_default, definition)
		 VALUES (?, ?, ?, ?, ?)`,
		p.PublicID, p.BusinessTypeID, p.Name, p.IsDefault, p.DefinitionJSON)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = db.U64(id)
	fresh, err := r.ProfileByPublicID(ctx, p.PublicID)
	if err != nil {
		return err
	}
	*p = fresh
	return nil
}

// UpdateProfile updates name/flag/definition and hydrates the row.
func (r *Repo) UpdateProfile(ctx context.Context, p *Profile) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE business_profiles SET name = ?, is_default = ?, definition = ? WHERE id = ?`,
		p.Name, p.IsDefault, p.DefinitionJSON, p.ID)
	if err != nil {
		return err
	}
	fresh, err := r.ProfileByPublicID(ctx, p.PublicID)
	if err != nil {
		return err
	}
	*p = fresh
	return nil
}

// ClearDefault unsets is_default on every other profile of the same type.
func (r *Repo) ClearDefault(ctx context.Context, typeID, keepID uint64) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE business_profiles SET is_default = 0 WHERE business_type_id = ? AND id <> ?`,
		typeID, keepID)
	return err
}

// ─── tenant_modules ─────────────────────────────────────────────────────────

// ListModules returns all tenant_modules rows for a tenant.
func (r *Repo) ListModules(ctx context.Context, tenantID uint64) ([]TenantModule, error) {
	var out []TenantModule
	err := r.q.SelectContext(ctx, &out,
		`SELECT tenant_id, module, enabled FROM tenant_modules WHERE tenant_id = ? ORDER BY module`, tenantID)
	return out, err
}

// EnableModules upserts enabled=1 for each code without touching others —
// idempotent profile application.
func (r *Repo) EnableModules(ctx context.Context, tenantID uint64, mods []string) error {
	for _, m := range mods {
		if _, err := r.q.ExecContext(ctx,
			`INSERT INTO tenant_modules (tenant_id, module, enabled) VALUES (?, ?, 1)
			 ON DUPLICATE KEY UPDATE enabled = 1`, tenantID, m); err != nil {
			return err
		}
	}
	return nil
}

// SetModules marks exactly the given codes enabled and everything else
// disabled (used by the wizard so unchecked modules are switched off).
func (r *Repo) SetModules(ctx context.Context, tenantID uint64, mods []string) error {
	if err := r.EnableModules(ctx, tenantID, mods); err != nil {
		return err
	}
	if len(mods) == 0 {
		_, err := r.q.ExecContext(ctx,
			`UPDATE tenant_modules SET enabled = 0 WHERE tenant_id = ?`, tenantID)
		return err
	}
	args := make([]any, 0, len(mods)+1)
	args = append(args, tenantID)
	marks := ""
	for i, m := range mods {
		if i > 0 {
			marks += ","
		}
		marks += "?"
		args = append(args, m)
	}
	_, err := r.q.ExecContext(ctx,
		`UPDATE tenant_modules SET enabled = 0 WHERE tenant_id = ? AND module NOT IN (`+marks+`)`, args...)
	return err
}
