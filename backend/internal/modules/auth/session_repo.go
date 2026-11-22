package auth

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// SessionRepo is the auth_sessions data store.
type SessionRepo struct {
	db *sqlx.DB
}

// NewSessionRepo builds a SessionRepo.
func NewSessionRepo(conn *sqlx.DB) *SessionRepo {
	return &SessionRepo{db: conn}
}

const sessionCols = `id, public_id, actor_type, actor_id, tenant_id, token_hash,
	mfa_verified, expires_at, revoked_at, ip_address, user_agent, last_used_at, created_at`

// Create inserts a session.
func (r *SessionRepo) Create(ctx context.Context, s *Session) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO auth_sessions
		 (public_id, actor_type, actor_id, tenant_id, token_hash, mfa_verified,
		  expires_at, ip_address, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.PublicID, s.ActorType, s.ActorID, s.TenantID, s.TokenHash, s.MFAVerified,
		s.ExpiresAt, s.IPAddress, s.UserAgent)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = db.U64(id)
	return nil
}

// ByTokenHash loads a session by the stored token hash.
func (r *SessionRepo) ByTokenHash(ctx context.Context, hash string) (Session, error) {
	var s Session
	err := r.db.GetContext(ctx, &s,
		`SELECT `+sessionCols+` FROM auth_sessions WHERE token_hash = ?`, hash)
	return s, err
}

// MarkMFAVerified flips a pending session to fully verified and extends its life.
func (r *SessionRepo) MarkMFAVerified(ctx context.Context, id uint64, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_sessions SET mfa_verified = 1, expires_at = ? WHERE id = ?`, expiresAt, id)
	return err
}

// Touch stamps last_used_at (throttled by the caller).
func (r *SessionRepo) Touch(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_sessions SET last_used_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// Revoke ends a session.
func (r *SessionRepo) Revoke(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = NOW(3) WHERE id = ? AND revoked_at IS NULL`, id)
	return err
}

// RevokeAllForActor ends every session of one account (optionally keeping one).
func (r *SessionRepo) RevokeAllForActor(ctx context.Context, actorType string, actorID uint64, keepID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = NOW(3)
		 WHERE actor_type = ? AND actor_id = ? AND revoked_at IS NULL AND id <> ?`,
		actorType, actorID, keepID)
	return err
}

// RevokeAllForTenant ends every live user session of a tenant (suspension).
func (r *SessionRepo) RevokeAllForTenant(ctx context.Context, tenantID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = NOW(3)
		 WHERE tenant_id = ? AND revoked_at IS NULL`, tenantID)
	return err
}

// ListForActor returns recent sessions of one account, newest first.
func (r *SessionRepo) ListForActor(ctx context.Context, actorType string, actorID uint64) ([]Session, error) {
	var out []Session
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+sessionCols+` FROM auth_sessions
		 WHERE actor_type = ? AND actor_id = ?
		 ORDER BY created_at DESC LIMIT 50`, actorType, actorID)
	return out, err
}

// ByPublicID loads a session by API id.
func (r *SessionRepo) ByPublicID(ctx context.Context, publicID string) (Session, error) {
	var s Session
	err := r.db.GetContext(ctx, &s,
		`SELECT `+sessionCols+` FROM auth_sessions WHERE public_id = ?`, publicID)
	return s, err
}

// DeleteExpired hard-deletes sessions past expiry (housekeeping job).
func (r *SessionRepo) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM auth_sessions WHERE expires_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
