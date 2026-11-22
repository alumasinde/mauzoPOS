package users

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// UserRepo is the tenant users data store. Every method is tenant-scoped.
type UserRepo struct {
	db *sqlx.DB
}

// NewUserRepo builds a UserRepo.
func NewUserRepo(conn *sqlx.DB) *UserRepo {
	return &UserRepo{db: conn}
}

const userCols = `id, tenant_id, public_id, name, email, username, password_hash,
	pin_hash, role, status, totp_secret, totp_enabled, recovery_codes,
	last_login_at, created_at, updated_at`

// Create inserts a tenant user.
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (tenant_id, public_id, name, email, username, password_hash, role, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.TenantID, u.PublicID, u.Name, u.Email, u.Username, u.PasswordHash, u.Role, u.Status)
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
		`SELECT `+userCols+` FROM users WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return u, err
}

// ByIdentity loads a user by email or username within a tenant (for sign-in).
func (r *UserRepo) ByIdentity(ctx context.Context, tenantID uint64, identity string) (User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT `+userCols+` FROM users WHERE tenant_id = ? AND (email = ? OR username = ?)`,
		tenantID, identity, identity)
	return u, err
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
