package tenants

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/httpx"
	"mauzo/internal/testutil"
)

type fakeRevoker struct{ revoked []uint64 }

func (f *fakeRevoker) RevokeAllForTenant(_ context.Context, tenantID uint64) error {
	f.revoked = append(f.revoked, tenantID)
	return nil
}

func newSvc(t *testing.T) (*Service, *fakeRevoker, *sqlx.DB) {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "audit_logs", "auth_sessions", "user_branches", "users", "roles", "tenants")
	rev := &fakeRevoker{}
	return NewService(NewRepo(conn), rev), rev, conn
}

func TestTenantLifecycle(t *testing.T) {
	svc, rev, _ := newSvc(t)
	ctx := context.Background()

	tn, err := svc.Create(ctx, CreateTenantRequest{
		Name: "Acme Mart", Slug: "Acme-Mart", Email: "hi@acme.local", Phone: "0700000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tn.PublicID == "" || tn.Slug != "acme-mart" || tn.Status != StatusActive {
		t.Fatalf("bad tenant %+v", tn)
	}
	if tn.CreatedAt.IsZero() {
		t.Fatal("created_at not hydrated")
	}

	got, err := svc.Get(ctx, tn.PublicID)
	if err != nil || got.ID != tn.ID {
		t.Fatalf("get: %v %+v", err, got)
	}

	rows, total, err := svc.List(ctx, httpx.Page{Page: 1, PerPage: 10, Query: "acme"})
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("list: %v total=%d rows=%d", err, total, len(rows))
	}

	name, email := "Acme Supermart", "new@acme.local"
	upd, err := svc.Update(ctx, tn.PublicID, UpdateTenantRequest{Name: &name, Email: &email})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != name || upd.Email == nil || *upd.Email != email {
		t.Fatalf("update: %+v", upd)
	}

	sus, err := svc.Suspend(ctx, tn.PublicID, "non-payment")
	if err != nil {
		t.Fatal(err)
	}
	if sus.Status != StatusSuspended {
		t.Fatalf("suspend: %+v", sus)
	}
	if len(rev.revoked) != 1 || rev.revoked[0] != tn.ID {
		t.Fatalf("sessions not revoked: %v", rev.revoked)
	}
	if _, err := svc.Suspend(ctx, tn.PublicID, ""); !errors.Is(err, ErrAlreadySuspended) {
		t.Fatalf("expected ErrAlreadySuspended, got %v", err)
	}

	act, err := svc.Reactivate(ctx, tn.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if act.Status != StatusActive {
		t.Fatalf("reactivate: %+v", act)
	}
	if _, err := svc.Reactivate(ctx, tn.PublicID); !errors.Is(err, ErrNotSuspended) {
		t.Fatalf("expected ErrNotSuspended, got %v", err)
	}
}

func TestTenantValidation(t *testing.T) {
	svc, _, _ := newSvc(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateTenantRequest{Name: "", Slug: "ok-slug"}); err == nil {
		t.Fatal("expected name validation error")
	}
	if _, err := svc.Create(ctx, CreateTenantRequest{Name: "X", Slug: "bad slug!"}); err == nil {
		t.Fatal("expected slug validation error")
	}
	if _, err := svc.Create(ctx, CreateTenantRequest{Name: "One", Slug: "dup"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateTenantRequest{Name: "Two", Slug: "DUP"}); !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}
	if _, err := svc.Get(ctx, "01ZZZZZZZZZZZZZZZZZZZZZZZZ"); !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("expected ErrTenantNotFound, got %v", err)
	}
}
