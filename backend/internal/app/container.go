// Package app wires configuration, infrastructure and modules together.
// Each module contributes its handlers here; routes/app.go mounts them.
package app

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/config"
	"mauzo/internal/core/db"
	"mauzo/internal/core/logger"
	"mauzo/internal/core/middleware"
	"mauzo/internal/modules/system"
)

// Version is overridden at build time: -ldflags "-X mauzo/internal/app.Version=1.2.3"
var Version = "dev"

// Container holds shared infrastructure and every module's handlers.
type Container struct {
	Config      config.Config
	Log         *slog.Logger
	DB          *sqlx.DB
	RateLimiter *middleware.RateLimiter

	// Module handlers (one field per module).
	System *system.Handler
}

// NewContainer builds the logger, database pool and module handlers from cfg.
func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	log := logger.New(cfg.Log.Level, cfg.IsDevelopment())
	slog.SetDefault(log)

	conn, err := db.Open(ctx, cfg.DB, cfg.DB.Name)
	if err != nil {
		return nil, err
	}

	return &Container{
		Config:      cfg,
		Log:         log,
		DB:          conn,
		RateLimiter: middleware.NewRateLimiter(cfg.HTTP.RateLimitRPS, cfg.HTTP.RateLimitBurst),
		System:      system.NewHandler(Version, map[string]system.Pinger{"database": conn}),
	}, nil
}

// Close releases container resources (the database pool).
func (c *Container) Close() error {
	return c.DB.Close()
}
