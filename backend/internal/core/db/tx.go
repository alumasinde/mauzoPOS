package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Querier is satisfied by both *sqlx.DB and *sqlx.Tx so repositories work in
// and out of transactions.
type Querier interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

var (
	_ Querier = (*sqlx.DB)(nil)
	_ Querier = (*sqlx.Tx)(nil)
)

// WithTx runs fn inside a transaction. It commits when fn returns nil and rolls
// back on error or panic (re-panicking afterwards).
func WithTx(ctx context.Context, conn *sqlx.DB, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err = fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("db: rollback: %w", rbErr))
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

// IsNotFound reports whether err is sql.ErrNoRows.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// U64 converts a non-negative LastInsertId to uint64.
func U64(id int64) uint64 {
	return uint64(max(id, 0))
}
