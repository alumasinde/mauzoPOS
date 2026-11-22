// Package db owns the MySQL/MariaDB connection pool and transaction helpers.
package db

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver
	"github.com/jmoiron/sqlx"

	"mauzo/internal/config"
)

// Open creates a pooled connection and verifies it with a ping.
func Open(ctx context.Context, cfg config.DBConfig, dbName string) (*sqlx.DB, error) {
	conn, err := sqlx.Open("mysql", cfg.DSN(dbName))
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: ping %s@%s:%d/%s: %w", cfg.User, cfg.Host, cfg.Port, dbName, err)
	}
	return conn, nil
}
