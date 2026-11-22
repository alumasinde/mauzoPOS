// Package config loads typed application configuration from environment variables.
// A .env file is read only when APP_ENV=development (or unset), so containers rely
// purely on the real environment.
package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/joho/godotenv"
)

// Config is the root application configuration loaded from the environment.
type Config struct {
	Env     string
	AppName string
	HTTP    HTTPConfig
	DB      DBConfig
	Log     LogConfig
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
		Env:     getStr("APP_ENV", "development"),
		AppName: getStr("APP_NAME", "MauzoPOS"),
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
	}
	if e.err != nil {
		return Config{}, e.err
	}
	return cfg, nil
}
