package audit

import (
	"context"
	"log/slog"
	"testing"

	coraudit "mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/testutil"
)

func TestWriteAndList(t *testing.T) {
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, "audit_logs", "roles", "tenants")
	ctx := context.Background()

	tn := tenants.Tenant{PublicID: security.NewPublicID(), Name: "Auditee", Slug: "aud", Status: tenants.StatusActive}
	if err := tenants.NewRepo(conn).Create(ctx, &tn); err != nil {
		t.Fatal(err)
	}

	w := coraudit.NewWriter(conn, slog.Default())
	w.Log(ctx, coraudit.Entry{
		ActorType: "platform_user", Action: "tenant.suspend",
		EntityType: "tenant", EntityID: "pub-1", IPAddress: "127.0.0.1",
	})
	w.Log(ctx, coraudit.Entry{
		TenantID: &tn.ID, ActorType: "user", Action: "sale.complete",
		EntityType: "sale", EntityID: "pub-2",
	})

	repo := NewRepo(conn)
	page := httpx.Page{Page: 1, PerPage: 10}

	// Platform view: everything.
	rows, total, err := repo.List(ctx, nil, "", page)
	if err != nil || total != 2 || len(rows) != 2 {
		t.Fatalf("platform list: %v total=%d", err, total)
	}

	// Tenant view: only that tenant's rows.
	rows, total, err = repo.List(ctx, &tn.ID, "", page)
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("tenant list: %v total=%d", err, total)
	}
	if rows[0].Action != "sale.complete" {
		t.Fatalf("tenant row: %+v", rows[0])
	}

	// Filters: actor_type and free-text search.
	if _, total, err := repo.List(ctx, nil, "user", page); err != nil || total != 1 {
		t.Fatalf("actor_type filter: %v total=%d", err, total)
	}
	if _, total, err := repo.List(ctx, nil, "", httpx.Page{Page: 1, PerPage: 10, Query: "suspend"}); err != nil || total != 1 {
		t.Fatalf("query filter: %v total=%d", err, total)
	}
}
