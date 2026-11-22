package auth

import "time"

// PlatformLoginRequest is POST /platform/auth/login.
type PlatformLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TenantLoginRequest is POST /t/auth/login.
type TenantLoginRequest struct {
	Tenant   string `json:"tenant"` // company slug
	Identity string `json:"identity"`
	Password string `json:"password"`
}

// MFAVerifyRequest is POST */auth/mfa/verify.
type MFAVerifyRequest struct {
	Ticket string `json:"ticket"`
	Code   string `json:"code"`
}

// ForgotPasswordRequest is POST */auth/password/forgot.
type ForgotPasswordRequest struct {
	Tenant   string `json:"tenant"` // tenant context only
	Identity string `json:"identity"`
}

// ResetPasswordRequest is POST */auth/password/reset.
type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// MFACodeRequest is POST */auth/mfa/enable.
type MFACodeRequest struct {
	Code string `json:"code"`
}

// MFADisableRequest is POST */auth/mfa/disable.
type MFADisableRequest struct {
	Password string `json:"password"`
}

// TenantRef is embedded in tenant-context responses.
type TenantRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ProfileResponse is the authenticated account view.
type ProfileResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Identity   string     `json:"identity"`
	Role       string     `json:"role"`
	MFAEnabled bool       `json:"mfa_enabled"`
	Tenant     *TenantRef `json:"tenant,omitempty"`
}

// LoginResponse is the result of */auth/login or */auth/mfa/verify.
type LoginResponse struct {
	MFARequired bool             `json:"mfa_required,omitempty"`
	Ticket      string           `json:"ticket,omitempty"`
	Token       string           `json:"token,omitempty"`
	User        *ProfileResponse `json:"user,omitempty"`
}

// SessionResponse is one live session.
type SessionResponse struct {
	ID         string     `json:"id"`
	IP         string     `json:"ip_address,omitempty"`
	UserAgent  string     `json:"user_agent,omitempty"`
	Current    bool       `json:"current"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// HistoryResponse is one login-history entry.
type HistoryResponse struct {
	Identity      string    `json:"identity"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failure_reason,omitempty"`
	IP            string    `json:"ip_address,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// MFASetupResponse carries the new secret + QR provisioning URL.
type MFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

// MFAEnableResponse returns the one-time recovery codes.
type MFAEnableResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

func loginResponse(res *LoginResult) LoginResponse {
	out := LoginResponse{MFARequired: res.MFARequired}
	if res.MFARequired {
		out.Ticket = res.Ticket
		return out
	}
	out.Token = res.Token
	out.User = profileResponse(res.Account, res.Tenant)
	return out
}

func profileResponse(acct Account, t *TenantInfo) *ProfileResponse {
	p := &ProfileResponse{
		ID:         acct.PublicID,
		Name:       acct.Name,
		Identity:   acct.Identity,
		Role:       acct.Role,
		MFAEnabled: acct.TOTPEnabled,
	}
	if t != nil {
		p.Tenant = &TenantRef{ID: t.PublicID, Name: t.Name, Slug: t.Slug}
	}
	return p
}
