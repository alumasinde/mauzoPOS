// Package auth implements sign-in for both contexts: platform operators and
// tenant employees. It owns sessions, login history, password resets, MFA
// (TOTP + recovery codes) and login lockout.
package auth

import "time"

// Session is an opaque-token auth session (unified for both actor types).
type Session struct {
	ID          uint64     `db:"id"`
	PublicID    string     `db:"public_id"`
	ActorType   string     `db:"actor_type"`
	ActorID     uint64     `db:"actor_id"`
	TenantID    *uint64    `db:"tenant_id"`
	TokenHash   string     `db:"token_hash"`
	MFAVerified bool       `db:"mfa_verified"`
	ExpiresAt   time.Time  `db:"expires_at"`
	RevokedAt   *time.Time `db:"revoked_at"`
	IPAddress   string     `db:"ip_address"`
	UserAgent   string     `db:"user_agent"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	CreatedAt   time.Time  `db:"created_at"`
}
