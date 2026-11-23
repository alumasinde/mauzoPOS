package users

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// UserRepo is the tenant users data store. Every tenant-facing method is
// tenant-scoped; the id-only writers are used by auth for the signed-in
// account itself.
type UserRepo struct {
	db   db.Querier
	conn *sqlx.DB
}

// NewUserRepo builds a UserRepo.
func NewUserRepo(conn *sqlx.DB) *UserRepo {
	return &UserRepo{db: conn, conn: conn}
}

// InTx runs fn with a repo bound to one transaction.
func (r *UserRepo) InTx(ctx context.Context, fn func(txr *UserRepo) error) error {
	return db.WithTx(ctx, r.conn, func(tx *sqlx.Tx) error { return fn(&UserRepo{db: tx, conn: r.conn}) })
}

const userCols = `u.id, u.tenant_id, u.public_id, u.name, u.email, u.username, u.password_hash,
	u.pin_hash, u.role_id, r.code AS role, r.name AS role_name, r.public_id AS role_public_id,
	u.all_branches, u.status, u.totp_secret, u.totp_enabled, u.recovery_codes,
	u.last_login_at, u.created_at, u.updated_at`

const userFrom = ` FROM users u JOIN roles r ON r.id = u.role_id `

// Create inserts a tenant user.
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (tenant_id, public_id, name, email, username, password_hash, pin_hash,
		   role_id, all_branches, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.TenantID, u.PublicID, u.Name, u.Email, u.Username, u.PasswordHash, u.PINHash,
		u.RoleID, u.AllBranches, u.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = db.U64(id)
	fresh, err := r.ByID(ctx, u.TenantID, u.ID)
	if err != nil {
		return err
	}
	*u = fresh
	return nil
}

// ByID loads a user within its tenant.
func (r *UserRepo) ByID(ctx context.Context, tenantID, id uint64) (User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT `+userCols+userFrom+`WHERE u.tenant_id = ? AND u.id = ?`, tenantID, id)
	return u, err
}

// ByPublicID loads a user by API id within its tenant.
func (r *UserRepo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT `+userCols+userFrom+`WHERE u.tenant_id = ? AND u.public_id = ?`, tenantID, publicID)
	return u, err
}

// ByIdentity loads a user by email or username within a tenant (for sign-in).
func (r *UserRepo) ByIdentity(ctx context.Context, tenantID uint64, identity string) (User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT `+userCols+userFrom+`WHERE u.tenant_id = ? AND (u.email = ? OR u.username = ?)`,
		tenantID, identity, identity)
	return u, err
}

// ListFilter narrows List.
type ListFilter struct {
	Query  string
	Status string
	RoleID uint64
}

// List returns a page of employees and the total.
func (r *UserRepo) List(ctx context.Context, tenantID uint64, f ListFilter, p httpx.Page) ([]User, int, error) {
	where := ` WHERE u.tenant_id = ?`
	args := []any{tenantID}
	if f.Query != "" {
		where += ` AND (u.name LIKE ? OR u.email LIKE ? OR u.username LIKE ?)`
		like := "%" + f.Query + "%"
		args = append(args, like, like, like)
	}
	if f.Status != "" {
		where += ` AND u.status = ?`
		args = append(args, f.Status)
	}
	if f.RoleID != 0 {
		where += ` AND u.role_id = ?`
		args = append(args, f.RoleID)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*)`+userFrom+where, args...); err != nil {
		return nil, 0, err
	}
	var out []User
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+userCols+userFrom+where+` ORDER BY u.status, u.name LIMIT ? OFFSET ?`,
		append(args, p.Limit(), p.Offset())...)
	return out, total, err
}

// Update saves the editable profile fields.
func (r *UserRepo) Update(ctx context.Context, u User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET name = ?, email = ?, username = ?, role_id = ?, all_branches = ?
		 WHERE tenant_id = ? AND id = ?`,
		u.Name, u.Email, u.Username, u.RoleID, u.AllBranches, u.TenantID, u.ID)
	return err
}

// SetStatus activates or disables a user.
func (r *UserRepo) SetStatus(ctx context.Context, tenantID, id uint64, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = ? WHERE tenant_id = ? AND id = ?`, status, tenantID, id)
	return err
}

// SetPINHash stores a new argon2 PIN hash.
func (r *UserRepo) SetPINHash(ctx context.Context, tenantID, id uint64, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET pin_hash = ? WHERE tenant_id = ? AND id = ?`, hash, tenantID, id)
	return err
}

// IdentityTaken reports whether email or username is used by another user of
// the tenant (in either column, since sign-in matches both).
func (r *UserRepo) IdentityTaken(ctx context.Context, tenantID uint64, value string, excludeID uint64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND id <> ? AND (email = ? OR username = ?)`,
		tenantID, excludeID, value, value)
	return n > 0, err
}

// CountActive counts active employees (plan seat limit).
func (r *UserRepo) CountActive(ctx context.Context, tenantID uint64) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND status = 'active'`, tenantID)
	return n, err
}

// UpdatePassword stores a new argon2 hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

// SetTOTP stores or clears the TOTP secret and recovery-code hashes.
func (r *UserRepo) SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_secret = ?, totp_enabled = ?, recovery_codes = ? WHERE id = ?`,
		secret, enabled, codes, id)
	return err
}

// UpdateRecoveryCodes replaces the unused recovery-code hashes after one was spent.
func (r *UserRepo) UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET recovery_codes = ? WHERE id = ?`, codes, id)
	return err
}

// TouchLastLogin stamps the last successful sign-in.
func (r *UserRepo) TouchLastLogin(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), id)
	return err
}
