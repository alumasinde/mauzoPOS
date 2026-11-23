package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/tenancy"
)

func init() {
	authz.Define(authz.Permission{Code: "guardtest.sales.view", Module: "sales"})
}

func serveAs(h http.Handler, a *identity.Actor, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	if a != nil {
		req = req.WithContext(identity.WithActor(req.Context(), *a))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func tenantUser(perms authz.Set, mods ...string) *identity.Actor {
	m := map[string]bool{}
	for _, c := range mods {
		m[c] = true
	}
	return &identity.Actor{Type: identity.ActorUser, TenantID: 1, Access: identity.Access{Perms: perms, Modules: m}}
}

func TestCanDistinguishesModuleAndPermission(t *testing.T) {
	h := Can("guardtest.sales.view")(okHandler)
	if rec := serveAs(h, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
	rec := serveAs(h, tenantUser(authz.NewSet("guardtest.sales.view")), nil)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "module_disabled") {
		t.Fatalf("module off should be module_disabled: %d %s", rec.Code, rec.Body)
	}
	rec = serveAs(h, tenantUser(authz.NewSet(), "sales"), nil)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"forbidden"`) {
		t.Fatalf("missing permission should be forbidden: %d %s", rec.Code, rec.Body)
	}
	if rec := serveAs(h, tenantUser(authz.NewSet("guardtest.sales.view"), "sales"), nil); rec.Code != http.StatusOK {
		t.Fatalf("granted: %d", rec.Code)
	}
}

func TestCanPanicsOnUnknownPermission(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	Can("nope.not.registered")
}

func TestRequireModule(t *testing.T) {
	h := RequireModule("sales")(okHandler)
	if rec := serveAs(h, tenantUser(authz.AllOf()), nil); rec.Code != http.StatusForbidden {
		t.Fatalf("disabled module: %d", rec.Code)
	}
	if rec := serveAs(h, tenantUser(authz.NewSet(), "sales"), nil); rec.Code != http.StatusOK {
		t.Fatalf("enabled module: %d", rec.Code)
	}
}

type fakeBranches map[string]uint64

func (f fakeBranches) BranchByPublicID(_ context.Context, _ uint64, pid string) (uint64, error) {
	if id, ok := f[pid]; ok {
		return id, nil
	}
	return 0, errors.New("not found")
}

func TestBranchScope(t *testing.T) {
	var got uint64
	h := BranchScope(fakeBranches{"A": 1, "B": 2})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = tenancy.BranchID(r.Context())
	}))
	a := tenantUser(authz.NewSet())
	a.Access.Branches = []uint64{1}

	if rec := serveAs(h, a, map[string]string{BranchHeader: "A"}); rec.Code != http.StatusOK || got != 1 {
		t.Fatalf("own branch: %d got=%d", rec.Code, got)
	}
	if rec := serveAs(h, a, map[string]string{BranchHeader: "B"}); rec.Code != http.StatusForbidden {
		t.Fatalf("other branch: %d", rec.Code)
	}
	if rec := serveAs(h, a, map[string]string{BranchHeader: "ZZ"}); rec.Code != http.StatusForbidden {
		t.Fatalf("unknown branch: %d", rec.Code)
	}
	got = 99
	if rec := serveAs(h, a, nil); rec.Code != http.StatusOK || got != 0 {
		t.Fatalf("no header passes through without a branch: %d got=%d", rec.Code, got)
	}
}
