package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/tenancy"
	"mauzo/internal/testutil"
)

func countAudit(t *testing.T, conn *sqlx.DB, action string) int {
	t.Helper()
	var n int
	if err := conn.Get(&n, "SELECT COUNT(*) FROM audit_logs WHERE action = ?", action); err != nil {
		t.Fatal(err)
	}
	return n
}

func insertAudit(tx *sqlx.Tx, action string) error {
	_, err := tx.Exec(`INSERT INTO audit_logs (actor_type, action) VALUES ('system', ?)`, action)
	return err
}

func TestWithTxCommitsAndRollsBack(t *testing.T) {
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "audit_logs")
	ctx := context.Background()

	if err := db.WithTx(ctx, conn, func(tx *sqlx.Tx) error { return insertAudit(tx, "test.commit") }); err != nil {
		t.Fatal(err)
	}
	if countAudit(t, conn, "test.commit") != 1 {
		t.Fatal("commit did not persist")
	}

	boom := errors.New("boom")
	err := db.WithTx(ctx, conn, func(tx *sqlx.Tx) error {
		if err := insertAudit(tx, "test.rollback"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if countAudit(t, conn, "test.rollback") != 0 {
		t.Fatal("rollback did not discard the insert")
	}
}

func TestWithTxRollsBackOnPanic(t *testing.T) {
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "audit_logs")

	func() {
		defer func() { _ = recover() }()
		_ = db.WithTx(context.Background(), conn, func(tx *sqlx.Tx) error {
			_ = insertAudit(tx, "test.panic")
			panic("boom")
		})
	}()
	if countAudit(t, conn, "test.panic") != 0 {
		t.Fatal("panic did not roll back")
	}
}

func TestForTenantRequiresTenantContext(t *testing.T) {
	conn := testutil.DB(t)
	if _, err := db.ForTenant(context.Background(), conn); !errors.Is(err, db.ErrNoTenant) {
		t.Fatalf("expected ErrNoTenant, got %v", err)
	}
	tq, err := db.ForTenant(tenancy.WithTenant(context.Background(), 42), conn)
	if err != nil || tq.ID() != 42 {
		t.Fatalf("ForTenant: %v id=%d", err, tq.ID())
	}
}
