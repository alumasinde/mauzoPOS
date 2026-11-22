// Package testutil provides helpers for integration tests that need a real
// MySQL/MariaDB database (TEST_DB_NAME, default "mauzo_test").
//
// If the test database is unreachable the test is skipped, unless
// REQUIRE_TEST_DB=1 (set in CI), in which case it fails.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/config"
	"mauzo/internal/core/db"
)

var (
	once    sync.Once
	shared  *sqlx.DB
	initErr error
)

// DB returns a migrated connection to the test database, shared across the
// tests in one package. Tests must clean up rows they create (see Truncate).
func DB(t testing.TB) *sqlx.DB {
	t.Helper()
	once.Do(func() {
		cfg, err := config.Load(filepath.Join(backendRoot(), ".env"))
		if err != nil {
			initErr = err
			return
		}
		ctx := context.Background()
		shared, initErr = db.Open(ctx, cfg.DB, cfg.DB.TestName)
		if initErr != nil {
			return
		}
		initErr = db.MigrateUp(ctx, shared.DB)
	})
	if initErr != nil {
		if os.Getenv("REQUIRE_TEST_DB") == "1" {
			t.Fatalf("test database unavailable: %v", initErr)
		}
		t.Skipf("skipping: test database unavailable: %v", initErr)
	}
	return shared
}

// Truncate empties the given tables after the test.
func Truncate(t testing.TB, conn *sqlx.DB, tables ...string) {
	t.Helper()
	t.Cleanup(func() {
		for _, tbl := range tables {
			if _, err := conn.Exec("DELETE FROM " + tbl); err != nil {
				t.Errorf("truncate %s: %v", tbl, err)
			}
		}
	})
}

// backendRoot walks up from the working directory to the folder with go.mod.
func backendRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
