package roles

import (
	"context"
	"time"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/cache"
	"mauzo/internal/core/identity"
)

// ModuleSource returns a tenant's effective modules (modules/registry).
type ModuleSource interface {
	EffectiveModules(ctx context.Context, tenantID uint64) ([]string, error)
}

// BranchSource returns the branch ids an employee may access (modules/users).
type BranchSource interface {
	UserBranchIDs(ctx context.Context, tenantID, userID uint64) ([]uint64, error)
}

// accessTTL bounds how long another API replica may serve stale grants.
const accessTTL = 30 * time.Second

// AccessService resolves a tenant user's identity.Access once per request,
// caching each ingredient with explicit invalidation on writes.
type AccessService struct {
	repo     *Repo
	modules  ModuleSource
	branches BranchSource
	perms    *cache.TTL[uint64, authz.Set] // by role id
	userBr   *cache.TTL[uint64, []uint64]  // by user id
}

// NewAccessService builds an AccessService.
func NewAccessService(repo *Repo, mods ModuleSource, br BranchSource) *AccessService {
	return &AccessService{
		repo: repo, modules: mods, branches: br,
		perms:  cache.New[uint64, authz.Set](accessTTL, 50_000),
		userBr: cache.New[uint64, []uint64](accessTTL, 200_000),
	}
}

// LoadAccess resolves permissions (role), enabled modules (tenant) and
// branch access (user). Implements auth.AccessLoader.
func (a *AccessService) LoadAccess(ctx context.Context, tenantID, userID, roleID uint64, allBranches bool) (identity.Access, error) {
	perms, err := a.perms.GetOrLoad(roleID, func() (authz.Set, error) {
		role, err := a.repo.ByID(ctx, tenantID, roleID)
		if err != nil {
			return authz.Set{}, err
		}
		if role.IsLocked {
			return authz.AllOf(), nil
		}
		codes, err := a.repo.Permissions(ctx, roleID)
		if err != nil {
			return authz.Set{}, err
		}
		return authz.NewSet(codes...), nil
	})
	if err != nil {
		return identity.Access{}, err
	}
	mods, err := a.modules.EffectiveModules(ctx, tenantID)
	if err != nil {
		return identity.Access{}, err
	}
	acc := identity.Access{Perms: perms, Modules: make(map[string]bool, len(mods)), AllBranches: allBranches || perms.IsAll()}
	for _, m := range mods {
		acc.Modules[m] = true
	}
	if !acc.AllBranches {
		acc.Branches, err = a.userBr.GetOrLoad(userID, func() ([]uint64, error) {
			return a.branches.UserBranchIDs(ctx, tenantID, userID)
		})
		if err != nil {
			return identity.Access{}, err
		}
	}
	return acc, nil
}

// InvalidateRole drops a role's cached permissions.
func (a *AccessService) InvalidateRole(roleID uint64) { a.perms.Delete(roleID) }

// InvalidateUser drops a user's cached branch access.
func (a *AccessService) InvalidateUser(userID uint64) { a.userBr.Delete(userID) }
