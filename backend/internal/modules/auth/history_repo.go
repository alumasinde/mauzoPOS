package auth

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// LoginAttempt is one sign-in try, success or failure.
type LoginAttempt struct {
	ID            uint64    `db:"id"`
	ActorType     string    `db:"actor_type"`
	ActorID       *uint64   `db:"actor_id"`
	TenantID      *uint64   `db:"tenant_id"`
	Identity      string    `db:"identity"`
	Success       bool      `db:"success"`
	FailureReason *string   `db:"failure_reason"`
	IPAddress     string    `db:"ip_address"`
	UserAgent     string    `db:"user_agent"`
	CreatedAt     time.Time `db:"created_at"`
}

// HistoryRepo is the auth_login_history data store.
type HistoryRepo struct {
	db *sqlx.DB
}

// NewHistoryRepo builds a HistoryRepo.
func NewHistoryRepo(conn *sqlx.DB) *HistoryRepo {
	return &HistoryRepo{db: conn}
}

// Record writes one attempt.
func (r *HistoryRepo) Record(ctx context.Context, a LoginAttempt) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO auth_login_history
		 (actor_type, actor_id, tenant_id, identity, success, failure_reason, ip_address, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ActorType, a.ActorID, a.TenantID, a.Identity, a.Success, a.FailureReason, a.IPAddress, a.UserAgent)
	if err != nil {
		// Login history must never break sign-in; log at service level if needed.
		_ = err
	}
}

// ListForActor returns an account's recent attempts, newest first.
func (r *HistoryRepo) ListForActor(ctx context.Context, actorType string, actorID uint64, limit int) ([]LoginAttempt, error) {
	var out []LoginAttempt
	err := r.db.SelectContext(ctx, &out,
		`SELECT id, actor_type, actor_id, tenant_id, identity, success, failure_reason,
		        ip_address, user_agent, created_at
		 FROM auth_login_history
		 WHERE actor_type = ? AND actor_id = ?
		 ORDER BY created_at DESC LIMIT ?`, actorType, actorID, limit)
	return out, err
}
