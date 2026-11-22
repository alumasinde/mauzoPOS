package auth

import (
	"context"
	"net/http"
	"time"

	"mauzo/internal/config"
	"mauzo/internal/core/audit"
	"mauzo/internal/core/mail"
	"mauzo/internal/core/security"
)

// Account is the credential view shared by platform_users and users.
type Account struct {
	ID            uint64
	TenantID      uint64
	PublicID      string
	Name          string
	Identity      string // email for platform users, email/username for tenant users
	PasswordHash  *string
	Role          string
	Status        string
	TOTPSecret    *string
	TOTPEnabled   bool
	RecoveryCodes []byte // JSON array of sha256 hex digests
}

// TenantInfo is the sign-in context for a tenant account.
type TenantInfo struct {
	ID       uint64
	PublicID string
	Name     string
	Slug     string
	Status   string
}

// platformAccountStore is the users module's platform user persistence, as
// consumed by auth.
type platformAccountStore interface {
	ByEmail(ctx context.Context, email string) (Account, error)
	ByID(ctx context.Context, id uint64) (Account, error)
	UpdatePassword(ctx context.Context, id uint64, hash string) error
	SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error
	UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error
	TouchLastLogin(ctx context.Context, id uint64) error
}

// tenantAccountStore is the users module's tenant user persistence.
type tenantAccountStore interface {
	ByIdentity(ctx context.Context, tenantID uint64, identity string) (Account, error)
	ByID(ctx context.Context, tenantID, id uint64) (Account, error)
	UpdatePassword(ctx context.Context, id uint64, hash string) error
	SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error
	UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error
	TouchLastLogin(ctx context.Context, id uint64) error
}

// tenantStore resolves tenants for sign-in and session checks.
type tenantStore interface {
	BySlug(ctx context.Context, slug string) (TenantInfo, error)
	ByID(ctx context.Context, id uint64) (TenantInfo, error)
}

const mfaTicketTTL = 5 * time.Minute

// RequestMeta carries client metadata for history/audit rows.
type RequestMeta struct {
	IP        string
	UserAgent string
}

// LoginResult is the outcome of a successful credential check.
type LoginResult struct {
	MFARequired bool
	Ticket      string // present only when MFARequired
	Token       string // full session token
	Account     Account
	Tenant      *TenantInfo
	SessionID   uint64
}

// Service owns sign-in, sessions, MFA and password recovery.
type Service struct {
	sessions    *SessionRepo
	history     *HistoryRepo
	resets      *ResetRepo
	platform    platformAccountStore
	tenant      tenantAccountStore
	tenants     tenantStore
	mailer      mail.Mailer
	audit       *audit.Writer
	cfg         config.AuthConfig
	params      security.Argon2Params
	appName     string
	platformURL string
	webURL      string
	limiter     *lockout
}

// ServiceDeps bundles the stores and helpers a Service needs.
type ServiceDeps struct {
	Sessions *SessionRepo
	History  *HistoryRepo
	Resets   *ResetRepo
	Platform platformAccountStore
	Tenant   tenantAccountStore
	Tenants  tenantStore
	Mailer   mail.Mailer
	Audit    *audit.Writer
}

// NewService builds a Service.
func NewService(cfg config.Config, deps ServiceDeps) *Service {
	return &Service{
		sessions:    deps.Sessions,
		history:     deps.History,
		resets:      deps.Resets,
		platform:    deps.Platform,
		tenant:      deps.Tenant,
		tenants:     deps.Tenants,
		mailer:      deps.Mailer,
		audit:       deps.Audit,
		cfg:         cfg.Auth,
		params:      cfg.Auth.Argon2Params(),
		appName:     cfg.AppName,
		platformURL: cfg.PlatformAppURL,
		webURL:      cfg.WebAppURL,
		limiter:     newLockout(cfg.Auth.LoginMaxAttempts, cfg.Auth.LoginLockout),
	}
}

func metaFromRequest(r *http.Request) RequestMeta {
	return RequestMeta{IP: r.RemoteAddr, UserAgent: r.UserAgent()}
}
