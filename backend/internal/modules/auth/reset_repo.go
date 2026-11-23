package auth

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// PasswordReset is a one-time reset token (SHA-256 hash stored only).
type PasswordReset struct {
	ID        uint64     `db:"id"`
	ActorType string     `db:"actor_type"`
	ActorID   uint64     `db:"actor_id"`
	TenantID  *uint64    `db:"tenant_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// ResetRepo is the auth_password_resets data store.
type ResetRepo struct {
	db *sqlx.DB
}

// NewResetRepo builds a ResetRepo.
func NewResetRepo(conn *sqlx.DB) *ResetRepo {
	return &ResetRepo{db: conn}
}

// Create stores a hashed reset token.
func (r *ResetRepo) Create(ctx context.Context, pr *PasswordReset) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO auth_password_resets (actor_type, actor_id, tenant_id, token_hash, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		pr.ActorType, pr.ActorID, pr.TenantID, pr.TokenHash, pr.ExpiresAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	pr.ID = db.U64(id)
	return nil
}

// ByTokenHash loads a reset record.
func (r *ResetRepo) ByTokenHash(ctx context.Context, hash string) (PasswordReset, error) {
	var pr PasswordReset
	err := r.db.GetContext(ctx, &pr,
		`SELECT id, actor_type, actor_id, tenant_id, token_hash, expires_at, used_at, created_at
		 FROM auth_password_resets WHERE token_hash = ?`, hash)
	return pr, err
}

// Consume atomically marks an unused, unexpired reset token as used. It
// reports false when another request already consumed it (or it expired), so
// each token sets a password at most once.
func (r *ResetRepo) Consume(ctx context.Context, id uint64, now time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE auth_password_resets SET used_at = ?
		 WHERE id = ? AND used_at IS NULL AND expires_at > ?`, now, id, now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
