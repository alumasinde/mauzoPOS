package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

const resetTokenTTL = 30 * time.Minute

// ForgotPassword issues a reset token and emails a link. It never reveals
// whether the identity exists — callers always get nil unless mailing fails.
func (s *Service) ForgotPassword(ctx context.Context, actorType identity.ActorType, slug, identityName string) error {
	identityName = strings.TrimSpace(identityName)
	var acct Account
	var email, appURL string

	if actorType == identity.ActorUser {
		t, err := s.tenants.BySlug(ctx, strings.ToLower(strings.TrimSpace(slug)))
		if err != nil {
			return nil // silent: no tenant enumeration
		}
		acct, err = s.tenant.ByIdentity(ctx, t.ID, identityName)
		if err != nil {
			return nil
		}
		if acct.Identity == "" {
			return nil // passwordless (PIN-only) accounts cannot self-reset
		}
		email = acct.Identity
		appURL = s.webURL
	} else {
		acct2, err := s.platform.ByEmail(ctx, strings.ToLower(identityName))
		if err != nil {
			return nil
		}
		acct = acct2
		email = identityName
		appURL = s.platformURL
	}

	token, err := security.NewToken()
	if err != nil {
		return err
	}
	pr := PasswordReset{
		ActorType: string(actorType),
		ActorID:   acct.ID,
		TenantID:  nilIfZero(acct.TenantID),
		TokenHash: security.HashToken(token),
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}
	if err := s.resets.Create(ctx, &pr); err != nil {
		return err
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", appURL, token)
	body := fmt.Sprintf("A password reset was requested for your %s account.\n\n%s\n\nThe link expires in 30 minutes. If you did not request it, ignore this message.", s.appName, link)
	if err := s.mailer.Send(ctx, email, s.appName+" password reset", body); err != nil {
		return fmt.Errorf("send reset mail: %w", err)
	}
	return nil
}

// ResetPassword consumes a token, sets the new password and signs out all
// sessions of the account.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return httpxWeakPassword()
	}
	pr, err := s.resets.ByTokenHash(ctx, security.HashToken(token))
	if db.IsNotFound(err) || pr.UsedAt != nil || time.Now().After(pr.ExpiresAt) {
		return ErrResetTokenInvalid
	}
	if err != nil {
		return err
	}
	hash, err := s.params.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if identity.ActorType(pr.ActorType) == identity.ActorUser {
		if err := s.tenant.UpdatePassword(ctx, pr.ActorID, hash); err != nil {
			return err
		}
	} else {
		if err := s.platform.UpdatePassword(ctx, pr.ActorID, hash); err != nil {
			return err
		}
	}
	if err := s.resets.MarkUsed(ctx, pr.ID); err != nil {
		return err
	}
	return s.sessions.RevokeAllForActor(ctx, pr.ActorType, pr.ActorID, 0)
}

func httpxWeakPassword() error {
	return httpx.ErrValidation.WithDetails(map[string]any{"password": "must be at least 8 characters"})
}
