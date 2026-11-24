// Package testutil provides helpers for integration tests that need a real
// MySQL/MariaDB database (TEST_DB_NAME, default "mauzo_test").
//
// If the test database is unreachable the test is skipped, unless
// REQUIRE_TEST_DB=1 (set in CI), in which case it fails.
package testutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/config"
	"mauzo/internal/core/db"
)

var (
	once    sync.Once
	shared  *sqlx.DB
	lock    *sql.Conn //nolint:unused // held so the MySQL named lock lives until process exit
	initErr error
	// guardErr is a fatal misconfiguration (tests pointed at a non-test DB).
	guardErr error
)

// checkTestDBName refuses to run destructive integration tests against the
// development database: tests DELETE whole tables via Truncate.
func checkTestDBName(devName, testName string) error {
	if testName == "" || strings.EqualFold(testName, devName) {
		return fmt.Errorf("testutil: TEST_DB_NAME (%q) must be set and differ from DB_NAME (%q)", testName, devName)
	}
	if !strings.Contains(strings.ToLower(testName), "test") {
		return fmt.Errorf("testutil: TEST_DB_NAME %q must contain \"test\" — refusing to truncate tables in it", testName)
	}
	return nil
}

// suiteLock serializes whole-table truncation across test packages: `go test`
// runs package binaries in parallel and they share one test database, so a
// MySQL named lock (held on a dedicated conn for the life of the binary) makes
// package suites run one at a time.
const suiteLock = "mauzo_test_suite"

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
		if guardErr = checkTestDBName(cfg.DB.Name, cfg.DB.TestName); guardErr != nil {
			return
		}
		ctx := context.Background()
		shared, initErr = db.Open(ctx, cfg.DB, cfg.DB.TestName)
		if initErr != nil {
			return
		}
		if initErr = acquireSuiteLock(ctx, shared); initErr != nil {
			return
		}
		initErr = db.MigrateUp(ctx, shared.DB)
	})
	if guardErr != nil {
		t.Fatal(guardErr) // misconfiguration: never skip silently
	}
	if initErr != nil {
		if os.Getenv("REQUIRE_TEST_DB") == "1" {
			t.Fatalf("test database unavailable: %v", initErr)
		}
		t.Skipf("skipping: test database unavailable: %v", initErr)
	}
	return shared
}

// acquireSuiteLock takes the named MySQL lock on a dedicated connection that is
// never returned to the pool — it releases automatically when the test binary
// exits. Waits up to 5 minutes for other package binaries to finish.
func acquireSuiteLock(ctx context.Context, conn *sqlx.DB) error {
	c, err := conn.Conn(ctx)
	if err != nil {
		return err
	}
	var got int
	if err := c.QueryRowContext(ctx, "SELECT GET_LOCK(?, 300)", suiteLock).Scan(&got); err != nil {
		_ = c.Close()
		return err
	}
	if got != 1 {
		_ = c.Close()
		return errors.New("GET_LOCK timed out waiting for another test package")
	}
	lock = c
	return nil
}

// EnsurePermissions inserts rows into the permissions table so tests can grant
// them in role_permissions without depending on which package last synced the
// registry. Safe to call repeatedly (INSERT IGNORE).
func EnsurePermissions(t testing.TB, conn *sqlx.DB, codes ...string) {
	t.Helper()
	for _, code := range codes {
		if _, err := conn.ExecContext(context.Background(),
			`INSERT IGNORE INTO permissions (code, scope) VALUES (?, 'tenant')`, code); err != nil {
			t.Fatalf("ensure permission %s: %v", code, err)
		}
	}
}

// Truncate empties the given tables after the test.
func Truncate(t testing.TB, conn *sqlx.DB, tables ...string) {
	t.Helper()
	t.Cleanup(func() {
		for _, tbl := range tables {
			if _, err := conn.ExecContext(context.Background(), "DELETE FROM "+tbl); err != nil {
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
