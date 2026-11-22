package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"mauzo/migrations"
)

func init() {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		panic(err)
	}
	goose.SetTableName("schema_migrations")
}

// MigrateUp applies all pending migrations.
func MigrateUp(ctx context.Context, conn *sql.DB) error {
	if err := goose.UpContext(ctx, conn, "."); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the latest migration.
func MigrateDown(ctx context.Context, conn *sql.DB) error {
	return goose.DownContext(ctx, conn, ".")
}

// MigrateReset rolls back all migrations (used by tests only).
func MigrateReset(ctx context.Context, conn *sql.DB) error {
	return goose.ResetContext(ctx, conn, ".")
}

// MigrateStatus prints the status of every migration.
func MigrateStatus(ctx context.Context, conn *sql.DB) error {
	return goose.StatusContext(ctx, conn, ".")
}
