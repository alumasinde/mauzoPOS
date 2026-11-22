// Package seeds holds idempotent reference-data seeders (business types,
// modules, permissions, tax defaults, role templates). Every seeder must be
// safe to run repeatedly (use INSERT ... ON DUPLICATE KEY UPDATE).
package seeds

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// Seeder inserts reference data idempotently.
type Seeder interface {
	Name() string
	Seed(ctx context.Context, db *sqlx.DB) error
}

// registry lists seeders in dependency order. Modules append theirs here.
var registry []Seeder

// Register adds a seeder. Call from an init() in the seeder's file.
func Register(s Seeder) { registry = append(registry, s) }

// Run executes all registered seeders in order.
func Run(ctx context.Context, db *sqlx.DB, log *slog.Logger) error {
	if len(registry) == 0 {
		log.Info("seed: no seeders registered yet")
		return nil
	}
	for _, s := range registry {
		log.Info("seed: running", "seeder", s.Name())
		if err := s.Seed(ctx, db); err != nil {
			return fmt.Errorf("seed %s: %w", s.Name(), err)
		}
	}
	return nil
}
