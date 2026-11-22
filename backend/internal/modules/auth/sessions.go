package auth

import (
	"context"
	"time"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/tenants"
)

// ErrSessionExpired rejects expired tokens with a distinct code so clients can
// send the user back to sign-in rather than a generic 401.
var ErrSessionExpired = httpx.NewError(401, "session_expired", "Your session expired. Sign in again.")

// Authenticate resolves a bearer token to an actor. It implements the
// middleware.SessionAuthenticator interface.
func (s *Service) Authenticate(ctx context.Context, token string) (identity.Actor, error) {
	sess, err := s.sessions.ByTokenHash(ctx, security.HashToken(token))
	if db.IsNotFound(err) {
		return identity.Actor{}, httpx.ErrUnauthorized
	}
	if err != nil {
		return identity.Actor{}, err
	}
	if sess.RevokedAt != nil {
		return identity.Actor{}, httpx.ErrUnauthorized
	}
	if time.Now().After(sess.ExpiresAt) {
		return identity.Actor{}, ErrSessionExpired
	}
	if !sess.MFAVerified {
		return identity.Actor{}, httpx.ErrUnauthorized
	}
	acct, t, err := s.loadAccount(ctx, sess)
	if err != nil {
		return identity.Actor{}, err
	}
	if acct.Status != "active" {
		return identity.Actor{}, ErrAccountDisabled
	}
	if t != nil && t.Status == tenants.StatusSuspended {
		return identity.Actor{}, tenants.ErrTenantSuspended
	}
	if sess.LastUsedAt == nil || time.Since(*sess.LastUsedAt) > time.Minute {
		_ = s.sessions.Touch(ctx, sess.ID)
	}
	return identity.Actor{
		Type:      identity.ActorType(sess.ActorType),
		ID:        acct.ID,
		SessionID: sess.ID,
		TenantID:  acct.TenantID,
		Role:      acct.Role,
	}, nil
}

// Logout revokes the actor's current session.
func (s *Service) Logout(ctx context.Context, actor identity.Actor) error {
	return s.sessions.Revoke(ctx, actor.SessionID)
}

// Sessions lists the actor's sessions, marking the current one.
func (s *Service) Sessions(ctx context.Context, actor identity.Actor) ([]Session, error) {
	out, err := s.sessions.ListForActor(ctx, string(actor.Type), actor.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	live := out[:0]
	for _, ss := range out {
		if ss.RevokedAt == nil && now.Before(ss.ExpiresAt) {
			live = append(live, ss)
		}
	}
	return live, nil
}

// RevokeSession ends one of the actor's own sessions by public id.
func (s *Service) RevokeSession(ctx context.Context, actor identity.Actor, publicID string) error {
	sess, err := s.sessions.ByPublicID(ctx, publicID)
	if db.IsNotFound(err) {
		return ErrSessionNotFound
	}
	if err != nil {
		return err
	}
	if sess.ActorType != string(actor.Type) || sess.ActorID != actor.ID {
		return ErrSessionNotFound // no cross-account visibility
	}
	return s.sessions.Revoke(ctx, sess.ID)
}

// RevokeAllForTenant implements tenants.SessionRevoker.
func (s *Service) RevokeAllForTenant(ctx context.Context, tenantID uint64) error {
	return s.sessions.RevokeAllForTenant(ctx, tenantID)
}

// RevokeAllForActor signs an account out everywhere (used after password change).
func (s *Service) RevokeAllForActor(ctx context.Context, actor identity.Actor, keepSession bool) error {
	var keep uint64
	if keepSession {
		keep = actor.SessionID
	}
	return s.sessions.RevokeAllForActor(ctx, string(actor.Type), actor.ID, keep)
}

// LoginHistory returns the actor's recent sign-in attempts.
func (s *Service) LoginHistory(ctx context.Context, actor identity.Actor) ([]LoginAttempt, error) {
	return s.history.ListForActor(ctx, string(actor.Type), actor.ID, 50)
}

// Profile resolves the actor's account for the /me endpoint.
func (s *Service) Profile(ctx context.Context, actor identity.Actor) (Account, *TenantInfo, error) {
	var acct Account
	var err error
	if actor.Type == identity.ActorUser {
		acct, err = s.tenant.ByID(ctx, actor.TenantID, actor.ID)
	} else {
		acct, err = s.platform.ByID(ctx, actor.ID)
	}
	if err != nil {
		return Account{}, nil, err
	}
	if actor.Type == identity.ActorUser {
		t, err := s.tenants.ByID(ctx, actor.TenantID)
		if err != nil {
			return Account{}, nil, err
		}
		return acct, &t, nil
	}
	return acct, nil, nil
}
