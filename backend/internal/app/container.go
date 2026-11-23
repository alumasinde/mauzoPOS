// Package app wires configuration, infrastructure and modules together.
// Each module contributes its handlers here; routes/app.go mounts them.
package app

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/config"
	"mauzo/internal/core/audit"
	"mauzo/internal/core/db"
	"mauzo/internal/core/logger"
	"mauzo/internal/core/mail"
	"mauzo/internal/core/middleware"
	coresettings "mauzo/internal/core/settings"
	auditmod "mauzo/internal/modules/audit"
	"mauzo/internal/modules/auth"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/businesstypes"
	"mauzo/internal/modules/onboarding"
	"mauzo/internal/modules/settings"
	"mauzo/internal/modules/subscriptions"
	"mauzo/internal/modules/system"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/terminals"
	"mauzo/internal/modules/users"
)

// Version is overridden at build time: -ldflags "-X mauzo/internal/app.Version=1.2.3"
var Version = "dev"

// Container holds shared infrastructure and every module's handlers.
type Container struct {
	Config      config.Config
	Log         *slog.Logger
	DB          *sqlx.DB
	RateLimiter *middleware.RateLimiter
	Audit       *audit.Writer

	// Auth service is shared by routes (Authenticate middleware) and handlers.
	AuthSvc *auth.Service

	// Module handlers (one field per module).
	System        *system.Handler
	Auth          *auth.Handler
	Tenants       *tenants.Handler
	PlatformUsers *users.PlatformUserHandler
	AuditLogs     *auditmod.Handler
	BusinessTypes *businesstypes.Handler
	Plans         *subscriptions.Handler
	Settings      *settings.Handler
	Branches      *branches.Handler
	Terminals     *terminals.Handler
	Onboarding    *onboarding.Handler
}

// NewContainer builds the logger, database pool and module handlers from cfg.
func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	log := logger.New(cfg.Log.Level, cfg.IsDevelopment())
	slog.SetDefault(log)

	conn, err := db.Open(ctx, cfg.DB, cfg.DB.Name)
	if err != nil {
		return nil, err
	}

	auditW := audit.NewWriter(conn, log)
	mailer := mail.New(cfg.Mail, log)
	params := cfg.Auth.Argon2Params()

	tenantRepo := tenants.NewRepo(conn)
	platformUserRepo := users.NewPlatformUserRepo(conn)
	userRepo := users.NewUserRepo(conn)

	authSvc := auth.NewService(cfg, auth.ServiceDeps{
		Sessions: auth.NewSessionRepo(conn),
		History:  auth.NewHistoryRepo(conn),
		Resets:   auth.NewResetRepo(conn),
		Platform: auth.NewPlatformAccountStore(platformUserRepo),
		Tenant:   auth.NewTenantAccountStore(userRepo),
		Tenants:  auth.NewTenantStore(tenantRepo),
		Mailer:   mailer,
		Audit:    auditW,
	})

	// Phase 2 services.
	tenantSvc := tenants.NewService(tenantRepo, authSvc)
	settingsStore := coresettings.NewStore(conn)
	btSvc := businesstypes.NewService(businesstypes.NewRepo(conn), settingsStore, tenantSvc)
	planSvc := subscriptions.NewService(subscriptions.NewRepo(conn), cfg.Onboarding.TrialDays)
	branchSvc := branches.NewService(branches.NewRepo(conn))
	terminalSvc := terminals.NewService(terminals.NewRepo(conn), branchSvc)
	userSvc := users.NewUserService(userRepo, params)
	onbSvc := onboarding.NewService(onboarding.Deps{
		Tenants: tenantSvc,
		Types:   btSvc,
		Profile: btSvc,
		Plans:   planSvc,
		Subs:    planSvc,
		Mods:    btSvc,
		Branch:  branchSvc,
		Admins:  userSvc,
		Prefs:   settingsStore,
		Term:    terminalSvc,
	})

	return &Container{
		Config:      cfg,
		Log:         log,
		DB:          conn,
		RateLimiter: middleware.NewRateLimiter(cfg.HTTP.RateLimitRPS, cfg.HTTP.RateLimitBurst),
		Audit:       auditW,
		AuthSvc:     authSvc,
		System:      system.NewHandler(Version, map[string]system.Pinger{"database": conn}),
		Auth:        auth.NewHandler(authSvc, auditW),
		Tenants:     tenants.NewHandler(tenantSvc, auditW),
		PlatformUsers: users.NewPlatformUserHandler(
			users.NewPlatformUserService(platformUserRepo, params), auditW),
		AuditLogs:     auditmod.NewHandler(auditmod.NewRepo(conn)),
		BusinessTypes: businesstypes.NewHandler(btSvc),
		Plans:         subscriptions.NewHandler(planSvc),
		Settings:      settings.NewHandler(settingsStore),
		Branches:      branches.NewHandler(branchSvc),
		Terminals:     terminals.NewHandler(terminalSvc),
		Onboarding:    onboarding.NewHandler(onbSvc),
	}, nil
}

// Close releases container resources (the database pool).
func (c *Container) Close() error {
	return c.DB.Close()
}
