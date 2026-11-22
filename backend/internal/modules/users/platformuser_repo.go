package users

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// PlatformUserRepo is the platform_users data store.
type PlatformUserRepo struct {
	db *sqlx.DB
}

// NewPlatformUserRepo builds a PlatformUserRepo.
func NewPlatformUserRepo(conn *sqlx.DB) *PlatformUserRepo {
	return &PlatformUserRepo{db: conn}
}

const platformUserCols = `id, public_id, name, email, password_hash, role, status,
	totp_secret, totp_enabled, recovery_codes, last_login_at, created_at, updated_at`

// Create inserts a platform user.
func (r *PlatformUserRepo) Create(ctx context.Context, u *PlatformUser) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO platform_users (public_id, name, email, password_hash, role, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.PublicID, u.Name, u.Email, u.PasswordHash, u.Role, u.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = db.U64(id)
	fresh, err := r.ByID(ctx, u.ID)
	if err != nil {
		return err
	}
	*u = fresh
	return nil
}

// ByID loads a platform user by internal id.
func (r *PlatformUserRepo) ByID(ctx context.Context, id uint64) (PlatformUser, error) {
	var u PlatformUser
	err := r.db.GetContext(ctx, &u,
		`SELECT `+platformUserCols+` FROM platform_users WHERE id = ?`, id)
	return u, err
}

// ByPublicID loads a platform user by API id.
func (r *PlatformUserRepo) ByPublicID(ctx context.Context, publicID string) (PlatformUser, error) {
	var u PlatformUser
	err := r.db.GetContext(ctx, &u,
		`SELECT `+platformUserCols+` FROM platform_users WHERE public_id = ?`, publicID)
	return u, err
}

// ByEmail loads a platform user by sign-in email.
func (r *PlatformUserRepo) ByEmail(ctx context.Context, email string) (PlatformUser, error) {
	var u PlatformUser
	err := r.db.GetContext(ctx, &u,
		`SELECT `+platformUserCols+` FROM platform_users WHERE email = ?`, email)
	return u, err
}

// EmailExists reports whether the email is used by another platform user.
func (r *PlatformUserRepo) EmailExists(ctx context.Context, email string, excludeID uint64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM platform_users WHERE email = ? AND id <> ?`, email, excludeID)
	return n > 0, err
}

// List returns a page of platform users plus the total count.
func (r *PlatformUserRepo) List(ctx context.Context, p httpx.Page) ([]PlatformUser, int, error) {
	where := ""
	var args []any
	if p.Query != "" {
		where = ` WHERE (name LIKE ? OR email LIKE ?)`
		like := "%" + p.Query + "%"
		args = append(args, like, like)
	}
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM platform_users`+where, args...); err != nil {
		return nil, 0, err
	}
	var out []PlatformUser
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+platformUserCols+` FROM platform_users`+where+
			fmt.Sprintf(` ORDER BY %s LIMIT ? OFFSET ?`, userSort(p.Sort)),
		append(args, p.Limit(), p.Offset())...)
	return out, total, err
}

func userSort(sort string) string {
	switch sort {
	case "-created_at":
		return "created_at DESC"
	case "email":
		return "email ASC"
	default:
		return "name ASC"
	}
}

// Update applies field changes and returns the fresh row.
func (r *PlatformUserRepo) Update(ctx context.Context, u PlatformUser) (PlatformUser, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET name = ?, email = ?, role = ? WHERE id = ?`,
		u.Name, u.Email, u.Role, u.ID)
	if err != nil {
		return PlatformUser{}, err
	}
	return r.ByID(ctx, u.ID)
}

// SetStatus updates status and returns the fresh row.
func (r *PlatformUserRepo) SetStatus(ctx context.Context, id uint64, status string) (PlatformUser, error) {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET status = ? WHERE id = ?`, status, id); err != nil {
		return PlatformUser{}, err
	}
	return r.ByID(ctx, id)
}

// UpdatePassword stores a new argon2 hash.
func (r *PlatformUserRepo) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

// SetTOTP stores or clears the TOTP secret and recovery-code hashes.
func (r *PlatformUserRepo) SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET totp_secret = ?, totp_enabled = ?, recovery_codes = ? WHERE id = ?`,
		secret, enabled, codes, id)
	return err
}

// UpdateRecoveryCodes replaces the unused recovery-code hashes after one was spent.
func (r *PlatformUserRepo) UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET recovery_codes = ? WHERE id = ?`, codes, id)
	return err
}

// CountActiveSuperAdmins returns the number of enabled super admins.
func (r *PlatformUserRepo) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM platform_users WHERE role = 'super_admin' AND status = 'active'`)
	return n, err
}

// TouchLastLogin stamps the last successful sign-in.
func (r *PlatformUserRepo) TouchLastLogin(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_users SET last_login_at = ? WHERE id = ?`, time.Now(), id)
	return err
}
