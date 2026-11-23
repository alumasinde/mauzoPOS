// Package seeds holds idempotent reference-data seeders (business types,
// modules, permissions, tax defaults, role templates). Every seeder must be
// safe to run repeatedly (use INSERT ... ON DUPLICATE KEY UPDATE).
package seeds

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

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

// Ordered lets a seeder run before (negative) or after (positive) the
// default position 0. Ties keep registration order.
type Ordered interface{ Order() int }

func orderOf(s Seeder) int {
	if o, ok := s.(Ordered); ok {
		return o.Order()
	}
	return 0
}

// Run executes all registered seeders in order.
func Run(ctx context.Context, db *sqlx.DB, log *slog.Logger) error {
	if len(registry) == 0 {
		log.Info("seed: no seeders registered yet")
		return nil
	}
	ordered := slices.Clone(registry)
	slices.SortStableFunc(ordered, func(a, b Seeder) int { return orderOf(a) - orderOf(b) })
	for _, s := range ordered {
		log.Info("seed: running", "seeder", s.Name())
		if err := s.Seed(ctx, db); err != nil {
			return fmt.Errorf("seed %s: %w", s.Name(), err)
		}
	}
	return nil
}
