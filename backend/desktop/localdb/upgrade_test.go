package localdb

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// A till database created by an older version (cashiers without
// max_discount, pending_sales without last_error) is upgraded in place on
// open, keeping its data.
func TestOpenUpgradesOldDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sqlx.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE cashiers (id TEXT PRIMARY KEY, name TEXT NOT NULL, role_name TEXT NOT NULL,
		 pin_hash TEXT NOT NULL, permissions TEXT NOT NULL DEFAULT '[]')`,
		`INSERT INTO cashiers (id, name, role_name, pin_hash) VALUES ('u1', 'Mary', 'Cashier', 'h')`,
		`CREATE TABLE pending_sales (id TEXT PRIMARY KEY, cart TEXT NOT NULL, totals TEXT NOT NULL,
		 payment TEXT NOT NULL DEFAULT 'cash', cashier_id TEXT NOT NULL, created_at TEXT NOT NULL,
		 uploaded INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err := old.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	_ = old.Close()

	d, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open old db: %v", err)
	}
	defer func() { _ = d.Close() }()
	if _, err := d.ExecContext(ctx, `UPDATE cashiers SET max_discount = '5' WHERE id = 'u1'`); err != nil {
		t.Fatalf("max_discount not added: %v", err)
	}
	if err := d.MarkSaleFailed(ctx, "x", "reason"); err != nil {
		t.Fatalf("last_error not added: %v", err)
	}
	var name string
	if err := d.GetContext(ctx, &name, `SELECT name FROM cashiers WHERE id = 'u1'`); err != nil || name != "Mary" {
		t.Fatalf("data lost: %q %v", name, err)
	}
	// Opening again is a no-op.
	_ = d.Close()
	if d2, err := Open(ctx, path); err != nil {
		t.Fatalf("reopen: %v", err)
	} else {
		_ = d2.Close()
	}
}

func TestAddable(t *testing.T) {
	for in, want := range map[string]string{
		"TEXT NOT NULL":              "TEXT",
		"TEXT NOT NULL DEFAULT '[]'": "TEXT NOT NULL DEFAULT '[]'",
		"INTEGER PRIMARY KEY":        "INTEGER",
		"TEXT":                       "TEXT",
	} {
		if got := addable(in); got != want {
			t.Errorf("addable(%q) = %q, want %q", in, got, want)
		}
	}
}
