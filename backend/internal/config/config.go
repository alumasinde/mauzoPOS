// Package config loads typed application configuration from environment variables.
// A .env file is read only when APP_ENV=development (or unset), so containers rely
// purely on the real environment.
package config

import (
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/joho/godotenv"

	"mauzo/internal/core/security"
)

// Config is the root application configuration loaded from the environment.
type Config struct {
	Env            string
	AppName        string
	PublicBaseURL  string
	PlatformAppURL string
	WebAppURL      string
	HTTP           HTTPConfig
	DB             DBConfig
	Log            LogConfig
	Auth           AuthConfig
	Mail           MailConfig
	Onboarding     OnboardingConfig
	// UploadDir is the local root for stored uploads (product images).
	UploadDir string
	// MpesaCredKey encrypts stored Daraja credentials (AES-256-GCM). Required
	// in production; development falls back to a derived key.
	MpesaCredKey string
}

// OnboardingConfig holds new-tenant onboarding settings.
type OnboardingConfig struct {
	TrialDays int
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Addr            string
	AllowedOrigins  []string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	RateLimitRPS    float64
	RateLimitBurst  int
}

// DBConfig holds MySQL connection and pool settings.
type DBConfig struct {
	Host            string
	Port            int
	Name            string
	TestName        string
	User            string
	Password        string
	Params          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string
}

// AuthConfig holds session and password-hashing settings.
type AuthConfig struct {
	SessionTTL        time.Duration
	Argon2MemoryKB    uint32
	Argon2Iterations  uint32
	Argon2Parallelism uint8
	LoginMaxAttempts  int
	LoginLockout      time.Duration
}

// MailConfig holds SMTP settings for transactional email (password recovery).
// An empty Host falls back to a log-only mailer in development.
type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// Argon2Params returns the configured argon2id work factors as security params.
func (a AuthConfig) Argon2Params() security.Argon2Params {
	p := security.DefaultArgon2Params
	p.MemoryKB = a.Argon2MemoryKB
	p.Iterations = a.Argon2Iterations
	p.Parallelism = a.Argon2Parallelism
	return p
}

// DSN builds a go-sql-driver/mysql DSN for the given database name.
func (d DBConfig) DSN(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		d.User, url.QueryEscape(d.Password), d.Host, d.Port, dbName, d.Params)
}

// IsDevelopment reports whether the app runs in the development environment.
func (c Config) IsDevelopment() bool { return c.Env == "development" }

// Load reads configuration. envFiles are optional .env paths (defaults to ".env").
func Load(envFiles ...string) (Config, error) {
	if env := getStr("APP_ENV", "development"); env == "development" {
		if len(envFiles) == 0 {
			envFiles = []string{".env"}
		}
		_ = godotenv.Load(envFiles...) // missing .env is fine
	}

	e := &envReader{}
	cfg := Config{
		Env:            getStr("APP_ENV", "development"),
		AppName:        getStr("APP_NAME", "MauzoPOS"),
		PublicBaseURL:  getStr("PUBLIC_BASE_URL", "http://localhost:8080"),
		PlatformAppURL: getStr("PLATFORM_APP_URL", "http://localhost:5174"),
		WebAppURL:      getStr("WEB_APP_URL", "http://localhost:5173"),
		HTTP: HTTPConfig{
			Addr:            getStr("HTTP_ADDR", ":8080"),
			AllowedOrigins:  getList("CORS_ALLOWED_ORIGINS"),
			ReadTimeout:     e.duration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    e.duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: e.duration("HTTP_SHUTDOWN_TIMEOUT", 20*time.Second),
			RateLimitRPS:    e.float("RATE_LIMIT_RPS", 20),
			RateLimitBurst:  e.int("RATE_LIMIT_BURST", 40),
		},
		DB: DBConfig{
			Host:            getStr("DB_HOST", "127.0.0.1"),
			Port:            e.int("DB_PORT", 3306),
			Name:            getStr("DB_NAME", "mauzo"),
			TestName:        getStr("TEST_DB_NAME", "mauzo_test"),
			User:            getStr("DB_USER", "mauzo"),
			Password:        getStr("DB_PASSWORD", ""),
			Params:          getStr("DB_PARAMS", "parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci"),
			MaxOpenConns:    e.int("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    e.int("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: e.duration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Log: LogConfig{Level: getStr("LOG_LEVEL", "info")},
		Auth: AuthConfig{
			SessionTTL:        e.duration("SESSION_TOKEN_TTL", 12*time.Hour),
			Argon2MemoryKB:    uint32(clampInt(e.int("ARGON2_MEMORY_KB", 65536), 0, math.MaxInt32)), //nolint:gosec // clamped to uint32 range
			Argon2Iterations:  uint32(clampInt(e.int("ARGON2_ITERATIONS", 3), 0, math.MaxInt32)),    //nolint:gosec // clamped to uint32 range
			Argon2Parallelism: uint8(clampInt(e.int("ARGON2_PARALLELISM", 2), 0, 255)),              //nolint:gosec // clamped to uint8 range
			LoginMaxAttempts:  e.int("LOGIN_MAX_ATTEMPTS", 5),
			LoginLockout:      e.duration("LOGIN_LOCKOUT", 15*time.Minute),
		},
		Mail: MailConfig{
			Host:     getStr("SMTP_HOST", ""),
			Port:     e.int("SMTP_PORT", 587),
			User:     getStr("SMTP_USER", ""),
			Password: getStr("SMTP_PASSWORD", ""),
			From:     getStr("MAIL_FROM", "no-reply@localhost"),
		},
		Onboarding: OnboardingConfig{
			TrialDays: e.int("ONBOARDING_TRIAL_DAYS", 14),
		},
		UploadDir:    getStr("UPLOAD_DIR", "uploads"),
		MpesaCredKey: getStr("MPESA_CRED_KEY", getStr("ENCRYPTION_KEY", "")),
	}
	if e.err != nil {
		return Config{}, e.err
	}
	return cfg, nil
}
