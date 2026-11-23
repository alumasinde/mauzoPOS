package users

import (
	"context"
	"slices"
	"strings"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

// EmployeeInput is a create or partial update. Nil pointers are unchanged on
// update; on create Name, RoleID and one of Email/Username are required.
type EmployeeInput struct {
	Name        *string
	Email       *string
	Username    *string
	Password    *string
	PIN         *string
	RoleID      *string   // role public id
	AllBranches *bool     // all branches vs the Branches list
	Branches    *[]string // branch public ids
}

// ListEmployees returns a page of employees with their branch ids.
func (s *UserService) ListEmployees(ctx context.Context, tenantID uint64, f ListFilter, p httpx.Page) ([]User, int, error) {
	rows, total, err := s.repo.List(ctx, tenantID, f, p)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, len(rows))
	for i, u := range rows {
		ids[i] = u.ID
	}
	br, err := s.repo.BranchIDsForUsers(ctx, tenantID, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].BranchIDs = br[rows[i].ID]
	}
	return rows, total, nil
}

// GetEmployee loads one employee with branch ids.
func (s *UserService) GetEmployee(ctx context.Context, tenantID uint64, publicID string) (User, error) {
	u, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	u.BranchIDs, err = s.repo.UserBranchIDs(ctx, tenantID, u.ID)
	return u, err
}

// CreateEmployee adds an employee. The actor may only assign a role and
// branches within their own reach, within the plan's seat limit.
func (s *UserService) CreateEmployee(ctx context.Context, actor identity.Actor, in EmployeeInput) (User, error) {
	if in.Name == nil || in.RoleID == nil {
		return User{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required", "role_id": "required"})
	}
	if err := s.checkSeat(ctx, actor.TenantID); err != nil {
		return User{}, err
	}
	u := User{TenantID: actor.TenantID, PublicID: security.NewPublicID(), Name: strings.TrimSpace(*in.Name), Status: StatusActive}
	if err := s.applyIdentity(ctx, &u, deref(in.Email), deref(in.Username)); err != nil {
		return User{}, err
	}
	if err := s.assignRole(ctx, actor, &u, *in.RoleID); err != nil {
		return User{}, err
	}
	branchIDs, err := s.resolveBranches(ctx, actor, &u, in)
	if err != nil {
		return User{}, err
	}
	if err := s.applySecrets(&u, in); err != nil {
		return User{}, err
	}
	err = s.repo.InTx(ctx, func(r *UserRepo) error {
		if err := r.Create(ctx, &u); err != nil {
			return mapDup(err)
		}
		return r.SetUserBranches(ctx, u.TenantID, u.ID, branchIDs)
	})
	if err != nil {
		return User{}, err
	}
	u.BranchIDs = branchIDs
	return u, nil
}

// UpdateEmployee edits profile, role and branch access. You cannot change
// your own role, branch access or credentials here (use the account page).
func (s *UserService) UpdateEmployee(ctx context.Context, actor identity.Actor, publicID string, in EmployeeInput) (User, error) {
	u, err := s.manageable(ctx, actor, publicID)
	if err != nil {
		return User{}, err
	}
	self := u.ID == actor.ID
	if self && (in.RoleID != nil || in.AllBranches != nil || in.Branches != nil || in.Password != nil || in.PIN != nil) {
		return User{}, ErrSelfChange
	}
	if in.Name != nil {
		u.Name = strings.TrimSpace(*in.Name)
	}
	email, username := derefOr(in.Email, u.Email), derefOr(in.Username, u.Username)
	if err := s.applyIdentity(ctx, &u, email, username); err != nil {
		return User{}, err
	}
	oldRole := u.RoleID
	if in.RoleID != nil {
		if err := s.assignRole(ctx, actor, &u, *in.RoleID); err != nil {
			return User{}, err
		}
		if u.RoleID != oldRole {
			if err := s.guardLastOwner(ctx, u.TenantID, oldRole, u.Status); err != nil {
				return User{}, err
			}
		}
	}
	branchIDs := u.BranchIDs
	branchesChanged := in.AllBranches != nil || in.Branches != nil
	if branchesChanged {
		if branchIDs, err = s.resolveBranches(ctx, actor, &u, in); err != nil {
			return User{}, err
		}
	}
	if err := s.applySecrets(&u, in); err != nil {
		return User{}, err
	}
	err = s.repo.InTx(ctx, func(r *UserRepo) error {
		if err := r.Update(ctx, u); err != nil {
			return mapDup(err)
		}
		if in.Password != nil {
			if err := r.UpdatePassword(ctx, u.ID, *u.PasswordHash); err != nil {
				return err
			}
		}
		if in.PIN != nil {
			if err := r.SetPINHash(ctx, u.TenantID, u.ID, *u.PINHash); err != nil {
				return err
			}
		}
		if branchesChanged {
			return r.SetUserBranches(ctx, u.TenantID, u.ID, branchIDs)
		}
		return nil
	})
	if err != nil {
		return User{}, err
	}
	s.invalidate(u.ID)
	// A new role or password takes effect everywhere: sign the user out.
	if (in.RoleID != nil && u.RoleID != oldRole) || in.Password != nil || branchesChanged {
		s.revoke(ctx, u.ID)
	}
	return s.GetEmployee(ctx, u.TenantID, u.PublicID)
}

// SetEmployeeStatus disables (signing the user out) or re-enables an employee.
func (s *UserService) SetEmployeeStatus(ctx context.Context, actor identity.Actor, publicID, status string) (User, error) {
	u, err := s.manageable(ctx, actor, publicID)
	if err != nil {
		return User{}, err
	}
	if u.ID == actor.ID {
		return User{}, ErrSelfChange
	}
	switch status {
	case u.Status:
		if status == StatusDisabled {
			return User{}, ErrAlreadyDisabled
		}
		return User{}, ErrAlreadyEnabled
	case StatusDisabled:
		if err := s.guardLastOwner(ctx, u.TenantID, u.RoleID, u.Status); err != nil {
			return User{}, err
		}
	case StatusActive:
		if err := s.checkSeat(ctx, u.TenantID); err != nil {
			return User{}, err
		}
	}
	if err := s.repo.SetStatus(ctx, u.TenantID, u.ID, status); err != nil {
		return User{}, err
	}
	if status == StatusDisabled {
		s.revoke(ctx, u.ID)
	}
	return s.GetEmployee(ctx, u.TenantID, u.PublicID)
}

// manageable loads the target and checks the actor outranks it.
func (s *UserService) manageable(ctx context.Context, actor identity.Actor, publicID string) (User, error) {
	u, err := s.GetEmployee(ctx, actor.TenantID, publicID)
	if err != nil {
		return User{}, err
	}
	if u.ID == actor.ID {
		return u, nil
	}
	_, set, err := s.deps.Roles.ByPublicIDForAssign(ctx, actor.TenantID, u.RolePublicID)
	if err != nil {
		return User{}, err
	}
	if !canManage(actor, set) {
		return User{}, ErrOutranked
	}
	return u, nil
}

// assignRole resolves the role and enforces "only grant what you hold".
func (s *UserService) assignRole(ctx context.Context, actor identity.Actor, u *User, rolePublicID string) error {
	role, set, err := s.deps.Roles.ByPublicIDForAssign(ctx, actor.TenantID, rolePublicID)
	if err != nil {
		return ErrInvalidRole
	}
	if !canManage(actor, set) {
		return ErrRoleEscalation
	}
	u.RoleID, u.Role, u.RoleName, u.RolePublicID = role.ID, role.Code, role.Name, role.PublicID
	return nil
}

// resolveBranches applies all_branches / branch list within the actor's reach.
func (s *UserService) resolveBranches(ctx context.Context, actor identity.Actor, u *User, in EmployeeInput) ([]uint64, error) {
	if in.AllBranches != nil {
		u.AllBranches = *in.AllBranches
	} else if in.Branches != nil {
		u.AllBranches = false
	}
	if u.AllBranches {
		if !actor.Access.AllBranches {
			return nil, ErrBranchNotAllowed
		}
		return []uint64{}, nil
	}
	if in.Branches == nil {
		return u.BranchIDs, nil
	}
	list, err := s.deps.Branches.List(ctx, actor.TenantID)
	if err != nil {
		return nil, err
	}
	byPublic := map[string]uint64{}
	for _, b := range list {
		byPublic[b.PublicID] = b.ID
	}
	ids := []uint64{}
	for _, pid := range *in.Branches {
		id, ok := byPublic[pid]
		if !ok {
			return nil, ErrUnknownBranch.WithDetails(map[string]any{"branch": pid})
		}
		if !actor.CanAccessBranch(id) {
			return nil, ErrBranchNotAllowed
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// applySecrets hashes a new password and/or PIN.
func (s *UserService) applySecrets(u *User, in EmployeeInput) error {
	if in.Password != nil {
		if len(*in.Password) < 8 {
			return ErrWeakPassword
		}
		h, err := s.params.HashPassword(*in.Password)
		if err != nil {
			return err
		}
		u.PasswordHash = &h
	}
	if in.PIN != nil {
		if !validPIN(*in.PIN) {
			return ErrWeakPIN
		}
		h, err := s.params.HashPassword(*in.PIN) // argon2id, same parameters as passwords
		if err != nil {
			return err
		}
		u.PINHash = &h
	}
	return nil
}

// guardLastOwner refuses to remove the last active owner (oldRole leaving).
func (s *UserService) guardLastOwner(ctx context.Context, tenantID, oldRole uint64, status string) error {
	ownerID, err := s.deps.Roles.RoleIDByCode(ctx, tenantID, authz.RoleOwner)
	if err != nil || oldRole != ownerID || status != StatusActive {
		return err
	}
	n, err := s.deps.Roles.CountActiveWithRole(ctx, tenantID, ownerID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastOwner
	}
	return nil
}

func (s *UserService) checkSeat(ctx context.Context, tenantID uint64) error {
	if s.deps.Seats == nil {
		return nil
	}
	limit, err := s.deps.Seats.MaxUsers(ctx, tenantID)
	if err != nil || limit == nil {
		return err
	}
	n, err := s.repo.CountActive(ctx, tenantID)
	if err != nil {
		return err
	}
	if n >= *limit {
		return ErrSeatLimit.WithDetails(map[string]any{"limit": *limit})
	}
	return nil
}

func (s *UserService) revoke(ctx context.Context, userID uint64) {
	if s.deps.Sessions != nil {
		_ = s.deps.Sessions.RevokeUserSessions(ctx, userID)
	}
}

func (s *UserService) invalidate(userID uint64) {
	if s.deps.Access != nil {
		s.deps.Access.InvalidateUser(userID)
	}
}

// validPIN accepts 4–6 digits that are not all the same or a straight run.
func validPIN(pin string) bool {
	if len(pin) < 4 || len(pin) > 6 {
		return false
	}
	asc, desc, same := true, true, true
	for i := range len(pin) {
		if pin[i] < '0' || pin[i] > '9' {
			return false
		}
		if i > 0 {
			d := int(pin[i]) - int(pin[i-1])
			asc, desc, same = asc && d == 1, desc && d == -1, same && d == 0
		}
	}
	return !asc && !desc && !same
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefOr(in, cur *string) string {
	if in != nil {
		return *in
	}
	return deref(cur)
}

// AssignableRoles lists the roles the actor may give an employee: those whose
// permissions the actor holds (owner only for owner).
func (s *UserService) AssignableRoles(ctx context.Context, actor identity.Actor) ([]Ref, error) {
	all, err := s.deps.Roles.List(ctx, actor.TenantID)
	if err != nil {
		return nil, err
	}
	out := []Ref{}
	for _, r := range all {
		set := authz.NewSet(r.Permissions...)
		if r.IsLocked {
			set = authz.AllOf()
		}
		if canManage(actor, set) {
			out = append(out, Ref{ID: r.PublicID, Name: r.Name, Code: r.Code})
		}
	}
	return out, nil
}
