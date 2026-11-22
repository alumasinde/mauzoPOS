package auth

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mauzo/internal/core/db"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/tenants"
)

// LoginPlatform verifies a platform operator's credentials and opens a session.
// When the account has TOTP enabled the result carries a short-lived ticket
// instead of a full session token.
func (s *Service) LoginPlatform(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	key := lockoutKey(identity.ActorPlatformUser, "", email)
	if until := s.limiter.lockedUntil(key); time.Now().Before(until) {
		s.record(ctx, identity.ActorPlatformUser, nil, nil, email, "locked", meta)
		return nil, ErrTooManyAttempts
	}
	acct, err := s.platform.ByEmail(ctx, email)
	if db.IsNotFound(err) {
		s.limiter.fail(key)
		s.record(ctx, identity.ActorPlatformUser, nil, nil, email, "unknown_identity", meta)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return s.finishLogin(ctx, key, acct, nil, email, password, meta)
}

// LoginTenant verifies a tenant employee's credentials. The company slug picks
// the tenant; suspended tenants cannot sign in.
func (s *Service) LoginTenant(ctx context.Context, slug, identityName, password string, meta RequestMeta) (*LoginResult, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	identityName = strings.TrimSpace(identityName)
	key := lockoutKey(identity.ActorUser, slug, identityName)
	if until := s.limiter.lockedUntil(key); time.Now().Before(until) {
		s.record(ctx, identity.ActorUser, nil, nil, identityName, "locked", meta)
		return nil, ErrTooManyAttempts
	}
	t, err := s.tenants.BySlug(ctx, slug)
	if db.IsNotFound(err) {
		s.limiter.fail(key)
		s.record(ctx, identity.ActorUser, nil, nil, identityName, "unknown_identity", meta)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	acct, err := s.tenant.ByIdentity(ctx, t.ID, identityName)
	if db.IsNotFound(err) {
		s.limiter.fail(key)
		s.record(ctx, identity.ActorUser, nil, &t.ID, identityName, "unknown_identity", meta)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return s.finishLogin(ctx, key, acct, &t, identityName, password, meta)
}

func (s *Service) finishLogin(ctx context.Context, key string, acct Account, t *TenantInfo,
	identityName, password string, meta RequestMeta) (*LoginResult, error) {

	actorType := identity.ActorPlatformUser
	if acct.TenantID != 0 {
		actorType = identity.ActorUser
	}
	if acct.Status != "active" {
		s.limiter.fail(key)
		s.record(ctx, actorType, &acct.ID, nilIfZero(acct.TenantID), identityName, "disabled", meta)
		return nil, ErrAccountDisabled
	}
	if t != nil && t.Status == tenants.StatusSuspended {
		s.limiter.fail(key)
		s.record(ctx, actorType, &acct.ID, &t.ID, identityName, "suspended", meta)
		return nil, tenants.ErrTenantSuspended
	}
	if acct.PasswordHash == nil || security.VerifyPassword(password, *acct.PasswordHash) != nil {
		s.limiter.fail(key)
		s.record(ctx, actorType, &acct.ID, nilIfZero(acct.TenantID), identityName, "bad_password", meta)
		return nil, ErrInvalidCredentials
	}
	s.limiter.succeed(key)

	token, err := security.NewToken()
	if err != nil {
		return nil, err
	}
	ttl := s.cfg.SessionTTL
	if acct.TOTPEnabled {
		ttl = mfaTicketTTL
	}
	sess := Session{
		PublicID:    security.NewPublicID(),
		ActorType:   string(actorType),
		ActorID:     acct.ID,
		TenantID:    nilIfZero(acct.TenantID),
		TokenHash:   security.HashToken(token),
		MFAVerified: !acct.TOTPEnabled,
		ExpiresAt:   time.Now().Add(ttl),
		IPAddress:   meta.IP,
		UserAgent:   meta.UserAgent,
	}
	if err := s.sessions.Create(ctx, &sess); err != nil {
		return nil, err
	}
	s.recordSuccess(ctx, actorType, acct, meta)
	s.touchLastLogin(ctx, actorType, acct)

	res := &LoginResult{Account: acct, Tenant: t, SessionID: sess.ID}
	if acct.TOTPEnabled {
		res.MFARequired = true
		res.Ticket = token
	} else {
		res.Token = token
	}
	return res, nil
}

// VerifyMFA completes a sign-in that returned an MFA ticket. The ticket is the
// pending session's token; a valid TOTP or unused recovery code upgrades it to
// a full session.
func (s *Service) VerifyMFA(ctx context.Context, ticket, code string, meta RequestMeta) (*LoginResult, error) {
	sess, err := s.sessions.ByTokenHash(ctx, security.HashToken(ticket))
	if db.IsNotFound(err) || sess.RevokedAt != nil || sess.MFAVerified || time.Now().After(sess.ExpiresAt) {
		return nil, ErrMFATicketInvalid
	}
	if err != nil {
		return nil, err
	}
	acct, t, err := s.loadAccount(ctx, sess)
	if err != nil {
		return nil, err
	}
	if !s.mfaCodeValid(ctx, acct, code) {
		var tid *uint64
		if t != nil {
			tid = &t.ID
		}
		s.record(ctx, identity.ActorType(sess.ActorType), &acct.ID, tid, acct.Identity, "mfa_failed", meta)
		return nil, ErrMFACodeInvalid
	}
	if err := s.sessions.MarkMFAVerified(ctx, sess.ID, time.Now().Add(s.cfg.SessionTTL)); err != nil {
		return nil, err
	}
	actorType := identity.ActorType(sess.ActorType)
	s.recordSuccess(ctx, actorType, acct, meta)
	s.touchLastLogin(ctx, actorType, acct)
	return &LoginResult{Token: ticket, Account: acct, Tenant: t, SessionID: sess.ID}, nil
}

// mfaCodeValid accepts a TOTP code or consumes a recovery code.
func (s *Service) mfaCodeValid(ctx context.Context, acct Account, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" || acct.TOTPSecret == nil {
		return false
	}
	if security.ValidateTOTP(*acct.TOTPSecret, code) {
		return true
	}
	return s.consumeRecoveryCode(ctx, acct, code)
}

// consumeRecoveryCode removes a spent code's hash and persists the rest.
func (s *Service) consumeRecoveryCode(ctx context.Context, acct Account, code string) bool {
	var hashes []string
	if len(acct.RecoveryCodes) == 0 || json.Unmarshal(acct.RecoveryCodes, &hashes) != nil {
		return false
	}
	norm := security.HashToken(strings.ToLower(code))
	keep := hashes[:0]
	matched := false
	for _, h := range hashes {
		if h == norm && !matched {
			matched = true
			continue
		}
		keep = append(keep, h)
	}
	if !matched {
		return false
	}
	b, err := json.Marshal(keep)
	if err != nil {
		return false
	}
	if acct.TenantID != 0 {
		return s.tenant.UpdateRecoveryCodes(ctx, acct.ID, b) == nil
	}
	return s.platform.UpdateRecoveryCodes(ctx, acct.ID, b) == nil
}

// loadAccount resolves the account (and tenant) behind a session.
func (s *Service) loadAccount(ctx context.Context, sess Session) (Account, *TenantInfo, error) {
	var acct Account
	var err error
	switch identity.ActorType(sess.ActorType) {
	case identity.ActorUser:
		var tenantID uint64
		if sess.TenantID != nil {
			tenantID = *sess.TenantID
		}
		acct, err = s.tenant.ByID(ctx, tenantID, sess.ActorID)
		if err != nil {
			return Account{}, nil, err
		}
		t, err := s.tenants.ByID(ctx, tenantID)
		if err != nil {
			return Account{}, nil, err
		}
		return acct, &t, nil
	default:
		acct, err = s.platform.ByID(ctx, sess.ActorID)
		return acct, nil, err
	}
}

func (s *Service) record(ctx context.Context, actorType identity.ActorType, actorID, tenantID *uint64,
	identityName, reason string, meta RequestMeta) {
	s.history.Record(ctx, LoginAttempt{
		ActorType:     string(actorType),
		ActorID:       actorID,
		TenantID:      tenantID,
		Identity:      identityName,
		Success:       false,
		FailureReason: &reason,
		IPAddress:     meta.IP,
		UserAgent:     meta.UserAgent,
	})
}

func (s *Service) recordSuccess(ctx context.Context, actorType identity.ActorType, acct Account, meta RequestMeta) {
	var tid *uint64
	if acct.TenantID != 0 {
		tid = &acct.TenantID
	}
	s.history.Record(ctx, LoginAttempt{
		ActorType: string(actorType), ActorID: &acct.ID, TenantID: tid,
		Identity: acct.Identity, Success: true, IPAddress: meta.IP, UserAgent: meta.UserAgent,
	})
}

func (s *Service) touchLastLogin(ctx context.Context, actorType identity.ActorType, acct Account) {
	if actorType == identity.ActorUser {
		_ = s.tenant.TouchLastLogin(ctx, acct.ID)
		return
	}
	_ = s.platform.TouchLastLogin(ctx, acct.ID)
}

func lockoutKey(t identity.ActorType, scope, identityName string) string {
	return string(t) + "|" + scope + "|" + strings.ToLower(identityName)
}

func nilIfZero(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}
