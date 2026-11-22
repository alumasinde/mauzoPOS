package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

// MFASetup generates a pending TOTP secret for the actor's account. The secret
// is stored unverified until MFAEnable confirms a code.
func (s *Service) MFASetup(ctx context.Context, actor identity.Actor) (secret, otpauthURL string, err error) {
	acct, err := s.loadActorAccount(ctx, actor)
	if err != nil {
		return "", "", err
	}
	if acct.TOTPEnabled {
		return "", "", ErrMFAAlreadyEnabled
	}
	secret, url, err := security.GenerateTOTPSecret(s.appName, acct.Identity)
	if err != nil {
		return "", "", err
	}
	if err := s.setTOTP(ctx, actor, acct, &secret, false, nil); err != nil {
		return "", "", err
	}
	return secret, url, nil
}

// MFAEnable validates a code against the pending secret, enables TOTP and
// returns the one-time recovery codes (shown once).
func (s *Service) MFAEnable(ctx context.Context, actor identity.Actor, code string) ([]string, error) {
	acct, err := s.loadActorAccount(ctx, actor)
	if err != nil {
		return nil, err
	}
	if acct.TOTPEnabled {
		return nil, ErrMFAAlreadyEnabled
	}
	if acct.TOTPSecret == nil {
		return nil, ErrMFANotPending
	}
	if !security.ValidateTOTP(*acct.TOTPSecret, code) {
		return nil, ErrMFACodeInvalid
	}
	codes, err := security.NewRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = security.HashToken(c)
	}
	b, err := json.Marshal(hashes)
	if err != nil {
		return nil, err
	}
	if err := s.setTOTP(ctx, actor, acct, acct.TOTPSecret, true, b); err != nil {
		return nil, err
	}
	return codes, nil
}

// MFADisable clears TOTP after confirming the account password.
func (s *Service) MFADisable(ctx context.Context, actor identity.Actor, password string) error {
	acct, err := s.loadActorAccount(ctx, actor)
	if err != nil {
		return err
	}
	if !acct.TOTPEnabled {
		return ErrMFANotEnabled
	}
	if acct.PasswordHash == nil || security.VerifyPassword(password, *acct.PasswordHash) != nil {
		return httpx.NewError(http.StatusUnprocessableEntity, "password_incorrect", "The password is incorrect.")
	}
	return s.setTOTP(ctx, actor, acct, nil, false, nil)
}

func (s *Service) loadActorAccount(ctx context.Context, actor identity.Actor) (Account, error) {
	if actor.Type == identity.ActorUser {
		return s.tenant.ByID(ctx, actor.TenantID, actor.ID)
	}
	return s.platform.ByID(ctx, actor.ID)
}

func (s *Service) setTOTP(ctx context.Context, actor identity.Actor, acct Account, secret *string, enabled bool, codes []byte) error {
	if actor.Type == identity.ActorUser {
		return s.tenant.SetTOTP(ctx, acct.ID, secret, enabled, codes)
	}
	return s.platform.SetTOTP(ctx, acct.ID, secret, enabled, codes)
}
