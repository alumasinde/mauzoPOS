package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	_ "mauzo/internal/modules/registerall" // full permission registry
	"mauzo/internal/modules/users"
	"mauzo/internal/testutil"
	"mauzo/internal/testutil/rbactest"
)

var fastParams = security.Argon2Params{MemoryKB: 8192, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

type revoked []uint64

func (r *revoked) RevokeUserSessions(_ context.Context, id uint64) error {
	*r = append(*r, id)
	return nil
}

type seats struct{ max *int }

func (s seats) MaxUsers(context.Context, uint64) (*int, error) { return s.max, nil }

type env struct {
	svc     *users.UserService
	rbac    rbactest.Services
	conn    *sqlx.DB
	tid     uint64
	owner   identity.Actor
	revoked *revoked
	seats   *seats
	b1, b2  string // branch public ids
	b1ID    uint64
}

func ptr[T any](v T) *T { return &v }

func errCode(err error) string {
	var e *httpx.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func setup(t *testing.T) *env {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn, rbactest.Tables...)
	repo := users.NewUserRepo(conn)
	rbac := rbactest.Setup(t, conn, nil, repo)
	e := &env{rbac: rbac, conn: conn, revoked: &revoked{}, seats: &seats{}}
	e.tid = rbac.Tenant(t, conn, "emp")
	e.b1ID, e.b1 = rbactest.Branch(t, conn, e.tid, "B1")
	_, e.b2 = rbactest.Branch(t, conn, e.tid, "B2")
	e.svc = users.NewUserService(repo, fastParams, users.Deps{
		Roles: rbac.Roles, Branches: branches.NewService(branches.NewRepo(conn)),
		Sessions: e.revoked, Seats: e.seats, Access: rbac.Access,
	})
	ownerUser, err := e.svc.CreateTenantUser(context.Background(), e.tid, "Owner", "owner@emp.test", "", "password123", authz.RoleOwner)
	if err != nil {
		t.Fatal(err)
	}
	e.owner = rbac.Actor(t, e.tid, ownerUser.ID, authz.RoleOwner, true)
	return e
}

func (e *env) rolePID(t *testing.T, code string) string {
	t.Helper()
	r, _, err := e.rbac.Roles.ByCodeForAssign(context.Background(), e.tid, code)
	if err != nil {
		t.Fatal(err)
	}
	return r.PublicID
}

func TestCreateEmployeeWithBranches(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	u, err := e.svc.CreateEmployee(ctx, e.owner, users.EmployeeInput{
		Name: ptr("Jane"), Username: ptr("jane"), PIN: ptr("4821"),
		RoleID: ptr(e.rolePID(t, authz.RoleCashier)), Branches: &[]string{e.b1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.AllBranches || len(u.BranchIDs) != 1 || u.BranchIDs[0] != e.b1ID || u.PINHash == nil || u.Role != authz.RoleCashier {
		t.Fatalf("employee: %+v", u)
	}
	if err := security.VerifyPassword("4821", *u.PINHash); err != nil {
		t.Fatal("PIN must be argon2id-hashed")
	}
	if _, err := e.svc.CreateEmployee(ctx, e.owner, users.EmployeeInput{
		Name: ptr("Dup"), Username: ptr("jane"), RoleID: ptr(e.rolePID(t, authz.RoleCashier)),
	}); errCode(err) != "username_taken" {
		t.Fatalf("duplicate username: %v", err)
	}
	if _, err := e.svc.CreateEmployee(ctx, e.owner, users.EmployeeInput{
		Name: ptr("Weak"), Username: ptr("weak"), PIN: ptr("1234"), RoleID: ptr(e.rolePID(t, authz.RoleCashier)),
	}); errCode(err) != "weak_pin" {
		t.Fatalf("sequential PIN: %v", err)
	}
}

func TestNoEscalation(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	mgrUser, err := e.svc.CreateEmployee(ctx, e.owner, users.EmployeeInput{
		Name: ptr("Mgr"), Username: ptr("mgr"), RoleID: ptr(e.rolePID(t, authz.RoleManager)), AllBranches: ptr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	mgr := e.rbac.Actor(t, e.tid, mgrUser.ID, authz.RoleManager, true)

	// A manager cannot create an admin or an owner…
	for _, code := range []string{authz.RoleAdmin, authz.RoleOwner} {
		if _, err := e.svc.CreateEmployee(ctx, mgr, users.EmployeeInput{
			Name: ptr("X"), Username: ptr("x-" + code), RoleID: ptr(e.rolePID(t, code)),
		}); errCode(err) != "role_escalation" {
			t.Fatalf("manager creating %s: %v", code, err)
		}
	}
	// …nor manage the owner.
	ownerUser, _, _ := e.svc.ListEmployees(ctx, e.tid, users.ListFilter{Query: "Owner"}, httpx.Page{Page: 1, PerPage: 5})
	if _, err := e.svc.SetEmployeeStatus(ctx, mgr, ownerUser[0].PublicID, users.StatusDisabled); errCode(err) != "user_outranks_you" {
		t.Fatalf("manager disabling owner: %v", err)
	}
	// A branch-restricted manager cannot grant all-branch access.
	restricted := e.rbac.Actor(t, e.tid, mgrUser.ID, authz.RoleManager, false, e.b1ID)
	if _, err := e.svc.CreateEmployee(ctx, restricted, users.EmployeeInput{
		Name: ptr("Y"), Username: ptr("y"), RoleID: ptr(e.rolePID(t, authz.RoleCashier)), AllBranches: ptr(true),
	}); errCode(err) != "branch_forbidden" {
		t.Fatalf("restricted manager granting all branches: %v", err)
	}
}

func TestOwnerAndSelfGuards(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	rows, _, _ := e.svc.ListEmployees(ctx, e.tid, users.ListFilter{}, httpx.Page{Page: 1, PerPage: 5})
	ownerPID := rows[0].PublicID

	if _, err := e.svc.SetEmployeeStatus(ctx, e.owner, ownerPID, users.StatusDisabled); errCode(err) != "self_change" {
		t.Fatalf("self disable: %v", err)
	}
	if _, err := e.svc.UpdateEmployee(ctx, e.owner, ownerPID, users.EmployeeInput{
		RoleID: ptr(e.rolePID(t, authz.RoleCashier)),
	}); errCode(err) != "self_change" {
		t.Fatalf("self demote: %v", err)
	}

	second, err := e.svc.CreateEmployee(ctx, e.owner, users.EmployeeInput{
		Name: ptr("Co-owner"), Email: ptr("co@emp.test"), RoleID: ptr(e.rolePID(t, authz.RoleOwner)),
	})
	if err != nil {
		t.Fatal(err)
	}
	coOwner := e.rbac.Actor(t, e.tid, second.ID, authz.RoleOwner, true)
	// With two owners, one may demote the other…
	if _, err := e.svc.UpdateEmployee(ctx, coOwner, ownerPID, users.EmployeeInput{
		RoleID: ptr(e.rolePID(t, authz.RoleAdmin)),
	}); err != nil {
		t.Fatal(err)
	}
	// …but not the last one.
	if _, err := e.svc.SetEmployeeStatus(ctx, e.owner, second.PublicID, users.StatusDisabled); errCode(err) != "last_owner" {
		t.Fatalf("disabling last owner: %v", err)
	}
	if len(*e.revoked) == 0 {
		t.Fatal("a role change must sign the user out")
	}
}

func TestSeatLimit(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.seats.max = ptr(2)
	in := func(name string) users.EmployeeInput {
		return users.EmployeeInput{Name: ptr(name), Username: ptr(name), RoleID: ptr(e.rolePID(t, authz.RoleCashier))}
	}
	if _, err := e.svc.CreateEmployee(ctx, e.owner, in("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateEmployee(ctx, e.owner, in("two")); errCode(err) != "seat_limit" {
		t.Fatalf("expected seat_limit, got %v", err)
	}
}
