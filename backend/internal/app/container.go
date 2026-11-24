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
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/customers"
	"mauzo/internal/modules/inventory"
	"mauzo/internal/modules/mpesa"
	"mauzo/internal/modules/onboarding"
	"mauzo/internal/modules/pos"
	"mauzo/internal/modules/pricing"
	"mauzo/internal/modules/receipts"
	_ "mauzo/internal/modules/registerall" // complete permission registry
	"mauzo/internal/modules/registry"
	"mauzo/internal/modules/roles"
	"mauzo/internal/modules/sales"
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

	// Services shared by routes (middleware) and boot tasks.
	AuthSvc     *auth.Service
	RoleSvc     *roles.Service
	RegistrySvc *registry.Service
	BranchSvc   *branches.Service

	// Module handlers (one field per module).
	System        *system.Handler
	Auth          *auth.Handler
	Tenants       *tenants.Handler
	PlatformUsers *users.PlatformUserHandler
	Employees     *users.EmployeeHandler
	AuditLogs     *auditmod.Handler
	BusinessTypes *businesstypes.Handler
	Plans         *subscriptions.Handler
	Settings      *settings.Handler
	Branches      *branches.Handler
	Terminals     *terminals.Handler
	Catalog       *catalog.Handler
	CatalogTax    *catalog.TaxonomyHandler
	Inventory     *inventory.Handler
	InventoryDocs *inventory.DocsHandler
	Pricing       *pricing.Handler
	Onboarding    *onboarding.Handler
	Roles         *roles.Handler
	Modules       *registry.Handler
	POS           *pos.Handler
	POSSvc        *pos.Service
	Customers     *customers.Handler
	Sales         *sales.Handler
	Receipts      *receipts.Handler
	MPesa         *mpesa.Handler
}

// NewContainer builds the logger, database pool and module services.
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

	// Plans gate modules; module enablement and roles feed per-request access.
	planSvc := subscriptions.NewService(subscriptions.NewRepo(conn), cfg.Onboarding.TrialDays)
	registrySvc := registry.NewService(registry.NewRepo(conn), planSvc)
	planSvc.OnPlanChange(registrySvc.Invalidate)
	roleRepo := roles.NewRepo(conn)
	accessSvc := roles.NewAccessService(roleRepo, registrySvc, userRepo)
	roleSvc := roles.NewService(roleRepo, accessSvc)

	authSvc := authStack(cfg, conn, tenantRepo, platformUserRepo, userRepo, mailer, auditW, accessSvc)

	tenantSvc, settingsStore, btSvc, branchSvc, terminalSvc := tenantTooling(conn, tenantRepo, authSvc, roleSvc, registrySvc)

	catalogSvc, taxSvc, err := catalogStack(conn, cfg) // taxonomy, products, local-disk images
	if err != nil {
		return nil, err
	}
	pricingSvc, promoSvc, invSvc := stockPricingStack(conn, catalogSvc, taxSvc, branchSvc)
	customerSvc := customers.NewService(customers.NewRepo(conn))
	salesSvc := sales.NewService(conn, sales.NewRepo(conn), catalogSvc, pricingSvc, promoSvc, invSvc, customerSvc)
	posSvc, mpesaSvc, receiptSvc, err := posPaymentsStack(conn, cfg, auditW, salesSvc)
	if err != nil {
		return nil, err
	}
	userSvc := users.NewUserService(userRepo, params,
		users.Deps{Roles: roleSvc, Branches: branchSvc, Sessions: authSvc, Seats: planSvc, Access: accessSvc})

	return &Container{
		Config:      cfg,
		Log:         log,
		DB:          conn,
		RateLimiter: middleware.NewRateLimiter(cfg.HTTP.RateLimitRPS, cfg.HTTP.RateLimitBurst),
		Audit:       auditW,
		AuthSvc:     authSvc,
		RoleSvc:     roleSvc,
		RegistrySvc: registrySvc,
		BranchSvc:   branchSvc,
		System:      system.NewHandler(Version, map[string]system.Pinger{"database": conn}),
		Auth:        auth.NewHandler(authSvc, auditW),
		Tenants:     tenants.NewHandler(tenantSvc, auditW),
		PlatformUsers: users.NewPlatformUserHandler(
			users.NewPlatformUserService(platformUserRepo, params), auditW),
		Employees:     users.NewEmployeeHandler(userSvc, auditW),
		AuditLogs:     auditmod.NewHandler(auditmod.NewRepo(conn)),
		BusinessTypes: businesstypes.NewHandler(btSvc),
		Plans:         subscriptions.NewHandler(planSvc),
		Settings:      settings.NewHandler(settingsStore),
		Branches:      branches.NewHandler(branchSvc),
		Terminals:     terminals.NewHandler(terminalSvc),
		Catalog:       catalog.NewHandler(catalogSvc, taxSvc),
		CatalogTax:    catalog.NewTaxonomyHandler(taxSvc),
		Inventory:     inventory.NewHandler(invSvc),
		InventoryDocs: inventory.NewDocsHandler(invSvc),
		Pricing:       pricing.NewHandler(pricingSvc, promoSvc, branchSvc),
		Onboarding: onboarding.NewHandler(onboarding.NewService(onboarding.Deps{
			Tenants: tenantSvc, Types: btSvc, Profile: btSvc, Plans: planSvc, Subs: planSvc,
			Mods: registrySvc, Branch: branchSvc, Admins: userSvc, Prefs: settingsStore, Term: terminalSvc,
		})),
		Roles:     roles.NewHandler(roleSvc, auditW),
		Modules:   registry.NewHandler(registrySvc, tenantSvc, auditW),
		POS:       pos.NewHandler(posSvc, registrySvc),
		POSSvc:    posSvc,
		Customers: customers.NewHandler(customerSvc),
		Sales:     sales.NewHandler(salesSvc, branchSvc),
		Receipts:  receipts.NewHandler(receiptSvc, salesSvc),
		MPesa:     mpesa.NewHandler(mpesaSvc),
	}, nil
}

// posPaymentsStack builds the till, M-Pesa and receipt services and binds the
// cross-module deps: sales consumes confirmed txns inside Complete, mpesa
// reads sales for allocation verification, the till initiates pushes and
// streams the feed.
func posPaymentsStack(conn *sqlx.DB, cfg config.Config, auditW *audit.Writer,
	salesSvc *sales.Service) (*pos.Service, *mpesa.Service, *receipts.Service, error) {
	posSvc := pos.NewService(conn, pos.NewRepo(conn))
	posSvc.BindSales(salesSvc)
	mpesaSvc, err := mpesa.NewService(conn, mpesa.NewRepo(conn),
		mpesaKey(cfg), cfg.PublicBaseURL, auditW)
	if err != nil {
		return nil, nil, nil, err
	}
	salesSvc.BindMpesa(mpesaSvc)
	mpesaSvc.BindSales(salesSvc)
	posSvc.BindMpesa(mpesaSvc)
	receiptSvc := receipts.NewService(conn, receipts.NewRepo(conn))
	posSvc.BindReceipts(receiptSvc)
	return posSvc, mpesaSvc, receiptSvc, nil
}

// mpesaKey resolves the credential-encryption key. Production requires the env
// var; development derives a stable local key so saved configs keep working.
func mpesaKey(cfg config.Config) string {
	if cfg.MpesaCredKey != "" {
		return cfg.MpesaCredKey
	}
	if cfg.IsDevelopment() {
		return "dev-mpesa-key:" + cfg.DB.Name
	}
	return "" // NewService errors — a production boot without a key must fail
}

// tenantTooling builds the tenant-scoped services shared by onboarding and the
// back-office modules (kept out of NewContainer for readability).
func tenantTooling(
	conn *sqlx.DB, tenantRepo *tenants.Repo, authSvc *auth.Service,
	roleSvc *roles.Service, registrySvc *registry.Service,
) (*tenants.Service, *coresettings.Store, *businesstypes.Service, *branches.Service, *terminals.Service) {
	tenantSvc := tenants.NewService(tenantRepo, authSvc)
	tenantSvc.AddProvisioner(roleSvc)
	settingsStore := coresettings.NewStore(conn)
	btSvc := businesstypes.NewService(businesstypes.NewRepo(conn), settingsStore, tenantSvc, registrySvc)
	branchSvc := branches.NewService(branches.NewRepo(conn))
	return tenantSvc, settingsStore, btSvc, branchSvc,
		terminals.NewService(terminals.NewRepo(conn), branchSvc)
}

// authStack builds the auth service with its repos and account stores (kept
// out of NewContainer for readability).
func authStack(
	cfg config.Config, conn *sqlx.DB, tenantRepo *tenants.Repo,
	platformUserRepo *users.PlatformUserRepo, userRepo *users.UserRepo,
	mailer mail.Mailer, auditW *audit.Writer, accessSvc *roles.AccessService,
) *auth.Service {
	return auth.NewService(cfg, auth.ServiceDeps{
		Sessions: auth.NewSessionRepo(conn),
		History:  auth.NewHistoryRepo(conn),
		Resets:   auth.NewResetRepo(conn),
		Platform: auth.NewPlatformAccountStore(platformUserRepo),
		Tenant:   auth.NewTenantAccountStore(userRepo),
		Tenants:  auth.NewTenantStore(tenantRepo),
		Mailer:   mailer,
		Audit:    auditW,
		Access:   accessSvc,
	})
}

// catalogStack builds the catalog module's services (kept out of NewContainer
// for readability).
func catalogStack(conn *sqlx.DB, cfg config.Config) (*catalog.Service, *catalog.TaxonomyService, error) {
	imageStore, err := catalog.NewDiskImageStore(cfg.UploadDir)
	if err != nil {
		return nil, nil, err
	}
	taxRepo := catalog.NewTaxonomyRepo(conn)
	return catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, imageStore),
		catalog.NewTaxonomyService(taxRepo), nil
}

// stockPricingStack builds the inventory + pricing services (kept out of
// NewContainer for readability).
func stockPricingStack(conn *sqlx.DB, catalogSvc *catalog.Service, taxSvc *catalog.TaxonomyService,
	branchSvc *branches.Service) (*pricing.Service, *pricing.PromoService, *inventory.Service) {
	repo := pricing.NewRepo(conn)
	pricingSvc := pricing.NewService(conn, repo, catalogSvc, branchSvc)
	promoSvc := pricing.NewPromoService(conn, repo, catalogSvc, taxSvc)
	catalogSvc.WatchPrices(pricingSvc.RecordPrice)
	invSvc := inventory.NewService(conn, inventory.NewRepo(conn),
		inventory.NewDocsRepo(conn), catalogSvc, branchSvc)
	return pricingSvc, promoSvc, invSvc
}

// Close releases container resources (the database pool).
func (c *Container) Close() error {
	return c.DB.Close()
}
