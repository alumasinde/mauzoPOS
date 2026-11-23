package users

import (
	"context"
	"strings"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/roles"
)

// RoleDirectory resolves tenant roles (modules/roles).
type RoleDirectory interface {
	RoleIDByCode(ctx context.Context, tenantID uint64, code string) (uint64, error)
	ByPublicIDForAssign(ctx context.Context, tenantID uint64, publicID string) (roles.Role, authz.Set, error)
	ByCodeForAssign(ctx context.Context, tenantID uint64, code string) (roles.Role, authz.Set, error)
	CountActiveWithRole(ctx context.Context, tenantID, roleID uint64) (int, error)
	List(ctx context.Context, tenantID uint64) ([]roles.Role, error)
}

// BranchDirectory lists a tenant's branches (modules/branches).
type BranchDirectory interface {
	List(ctx context.Context, tenantID uint64) ([]branches.Branch, error)
}

// SessionRevoker signs a user out everywhere (modules/auth).
type SessionRevoker interface {
	RevokeUserSessions(ctx context.Context, userID uint64) error
}

// SeatLimiter returns the plan's employee limit (modules/subscriptions).
type SeatLimiter interface {
	MaxUsers(ctx context.Context, tenantID uint64) (*int, error)
}

// AccessInvalidator drops cached branch access (modules/roles).
type AccessInvalidator interface {
	InvalidateUser(userID uint64)
}

// Deps are the collaborators of UserService. Seats, Sessions and Access may
// be nil in seeds.
type Deps struct {
	Roles    RoleDirectory
	Branches BranchDirectory
	Sessions SessionRevoker
	Seats    SeatLimiter
	Access   AccessInvalidator
}

// UserService owns tenant-employee rules.
type UserService struct {
	repo   *UserRepo
	params security.Argon2Params
	deps   Deps
}

// NewUserService builds a UserService.
func NewUserService(repo *UserRepo, params security.Argon2Params, deps Deps) *UserService {
	return &UserService{repo: repo, params: params, deps: deps}
}

// CreateTenantUser inserts an employee in the role with template code
// roleCode (onboarding and seeds). Branch access follows the template.
func (s *UserService) CreateTenantUser(ctx context.Context, tenantID uint64,
	name, email, username, password, roleCode string) (User, error) {

	if roleCode == "" {
		roleCode = authz.RoleCashier
	}
	roleID, err := s.deps.Roles.RoleIDByCode(ctx, tenantID, roleCode)
	if err != nil {
		return User{}, err
	}
	tpl, _ := authz.Template(roleCode)
	u := User{
		TenantID: tenantID, PublicID: security.NewPublicID(), Name: strings.TrimSpace(name),
		RoleID: roleID, AllBranches: tpl.AllBranches || tpl.Code == "", Status: StatusActive,
	}
	if err := s.applyIdentity(ctx, &u, email, username); err != nil {
		return User{}, err
	}
	if password != "" {
		if len(password) < 8 {
			return User{}, ErrWeakPassword
		}
		hash, err := s.params.HashPassword(password)
		if err != nil {
			return User{}, err
		}
		u.PasswordHash = &hash
	}
	if err := s.repo.Create(ctx, &u); err != nil {
		return User{}, mapDup(err)
	}
	return u, nil
}

// CreateTenantUserAs is CreateTenantUser on behalf of a signed-in actor: the
// actor must hold every permission of the role (onboarding's admin step).
func (s *UserService) CreateTenantUserAs(ctx context.Context, actor identity.Actor,
	name, email, username, password, roleCode string) (User, error) {

	if roleCode == "" {
		roleCode = authz.RoleAdmin
	}
	_, set, err := s.deps.Roles.ByCodeForAssign(ctx, actor.TenantID, roleCode)
	if err != nil {
		return User{}, ErrInvalidRole
	}
	if !canManage(actor, set) {
		return User{}, ErrRoleEscalation
	}
	if err := s.checkSeat(ctx, actor.TenantID); err != nil {
		return User{}, err
	}
	return s.CreateTenantUser(ctx, actor.TenantID, name, email, username, password, roleCode)
}

// applyIdentity validates and sets email/username, checking uniqueness across
// both columns (sign-in matches either).
func (s *UserService) applyIdentity(ctx context.Context, u *User, email, username string) error {
	u.Email, u.Username = nil, nil
	if v := strings.ToLower(strings.TrimSpace(email)); v != "" {
		if !strings.Contains(v, "@") || len(v) > 190 {
			return ErrInvalidEmail
		}
		if taken, err := s.repo.IdentityTaken(ctx, u.TenantID, v, u.ID); err != nil {
			return err
		} else if taken {
			return ErrEmailTaken
		}
		u.Email = &v
	}
	if v := strings.TrimSpace(username); v != "" {
		// A username must never look like an email or it could shadow
		// another user's email at sign-in.
		if strings.Contains(v, "@") || len(v) > 60 {
			return ErrInvalidUsername
		}
		if taken, err := s.repo.IdentityTaken(ctx, u.TenantID, v, u.ID); err != nil {
			return err
		} else if taken {
			return ErrUsernameTaken
		}
		u.Username = &v
	}
	if u.Email == nil && u.Username == nil {
		return ErrIdentityRequired
	}
	if u.Name == "" || len(u.Name) > 120 {
		return ErrNameRequired
	}
	return nil
}

// canManage reports whether actor may manage target: the target's role must
// not grant anything the actor lacks, and only owners manage owners.
func canManage(actor identity.Actor, target authz.Set) bool {
	if target.IsAll() {
		return actor.Access.Perms.IsAll()
	}
	return actor.Access.Perms.Contains(target.Codes(authz.ScopeTenant))
}
