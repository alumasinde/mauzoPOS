package roles

import (
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

// Service owns role business rules.
type Service struct {
	repo   *Repo
	access *AccessService
}

// NewService builds a Service.
func NewService(repo *Repo, access *AccessService) *Service {
	return &Service{repo: repo, access: access}
}

// Input is the editable part of a role.
type Input struct {
	Name        string
	Description string
	Permissions []string
	Limits      Limits
}

// List returns the tenant's roles with their permissions.
func (s *Service) List(ctx context.Context, tenantID uint64) ([]Role, error) {
	rows, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].Permissions, err = s.permissionsOf(ctx, rows[i]); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// Get returns one role with its permissions.
func (s *Service) Get(ctx context.Context, tenantID uint64, publicID string) (Role, error) {
	role, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, err
	}
	role.Permissions, err = s.permissionsOf(ctx, role)
	return role, err
}

func (s *Service) permissionsOf(ctx context.Context, r Role) ([]string, error) {
	if r.IsLocked {
		return authz.InScope(authz.ScopeTenant), nil
	}
	perms, err := s.repo.Permissions(ctx, r.ID)
	slices.Sort(perms) // byte order, independent of DB collation
	return perms, err
}

// Create adds a custom role.
func (s *Service) Create(ctx context.Context, actor identity.Actor, in Input) (Role, error) {
	perms, err := s.validate(actor, in)
	if err != nil {
		return Role{}, err
	}
	existing, err := s.repo.List(ctx, actor.TenantID)
	if err != nil {
		return Role{}, err
	}
	name := strings.TrimSpace(in.Name)
	codes := make([]string, len(existing))
	for i, r := range existing {
		if strings.EqualFold(r.Name, name) {
			return Role{}, ErrRoleNameTaken
		}
		codes[i] = r.Code
	}
	role := Role{
		TenantID: actor.TenantID, PublicID: security.NewPublicID(), Code: uniqueCode(name, codes),
		Name: name, Description: nilIfEmpty(in.Description),
		MaxDiscountPercent: in.Limits.MaxDiscountPercent, MaxRefundAmount: in.Limits.MaxRefundAmount,
		MaxCashOutAmount: in.Limits.MaxCashOutAmount,
	}
	err = s.repo.InTx(ctx, func(r *Repo) error {
		if _, err := r.Insert(ctx, &role); err != nil {
			return err
		}
		return r.SetPermissions(ctx, role.ID, perms)
	})
	if err != nil {
		return Role{}, err
	}
	role.Permissions = perms
	return role, nil
}

// Update edits a role. Editing a built-in role's permissions or limits marks
// it customized, so template updates no longer overwrite it.
func (s *Service) Update(ctx context.Context, actor identity.Actor, publicID string, in Input) (Role, error) {
	role, err := s.Get(ctx, actor.TenantID, publicID)
	if err != nil {
		return Role{}, err
	}
	if role.IsLocked {
		return Role{}, ErrRoleLocked
	}
	// You may only edit roles that grant nothing beyond what you hold.
	if !actor.Access.Perms.Contains(role.Permissions) {
		return Role{}, ErrGrantEscalation
	}
	perms, err := s.validate(actor, in)
	if err != nil {
		return Role{}, err
	}
	name := strings.TrimSpace(in.Name)
	if !strings.EqualFold(name, role.Name) {
		others, err := s.repo.List(ctx, actor.TenantID)
		if err != nil {
			return Role{}, err
		}
		for _, o := range others {
			if o.ID != role.ID && strings.EqualFold(o.Name, name) {
				return Role{}, ErrRoleNameTaken
			}
		}
	}
	changed := !slices.Equal(perms, role.Permissions) || !limitsEqual(in.Limits, role.Limits())
	role.Name, role.Description = name, nilIfEmpty(in.Description)
	role.MaxDiscountPercent, role.MaxRefundAmount, role.MaxCashOutAmount =
		in.Limits.MaxDiscountPercent, in.Limits.MaxRefundAmount, in.Limits.MaxCashOutAmount
	role.Customized = role.Customized || (role.IsSystem && changed)
	err = s.repo.InTx(ctx, func(r *Repo) error {
		if err := r.Update(ctx, role); err != nil {
			return err
		}
		return r.SetPermissions(ctx, role.ID, perms)
	})
	if err != nil {
		return Role{}, err
	}
	s.access.InvalidateRole(role.ID)
	role.Permissions = perms
	return role, nil
}

// Delete removes an unused custom role.
func (s *Service) Delete(ctx context.Context, actor identity.Actor, publicID string) error {
	role, err := s.Get(ctx, actor.TenantID, publicID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrSystemRole
	}
	if role.UserCount > 0 {
		return ErrRoleInUse.WithDetails(map[string]any{"users": role.UserCount})
	}
	if !actor.Access.Perms.Contains(role.Permissions) {
		return ErrGrantEscalation
	}
	if err := s.repo.Delete(ctx, actor.TenantID, role.ID); err != nil {
		return err
	}
	s.access.InvalidateRole(role.ID)
	return nil
}

// Reset restores a built-in role to its template and resumes template sync.
func (s *Service) Reset(ctx context.Context, actor identity.Actor, publicID string) (Role, error) {
	role, err := s.Get(ctx, actor.TenantID, publicID)
	if err != nil {
		return Role{}, err
	}
	t, ok := authz.Template(role.Code)
	if !role.IsSystem || !ok {
		return Role{}, ErrNotTemplated
	}
	if role.IsLocked {
		return Role{}, ErrRoleLocked
	}
	perms := templatePermissions(t)
	if !actor.Access.Perms.Contains(perms) || !actor.Access.Perms.Contains(role.Permissions) {
		return Role{}, ErrGrantEscalation
	}
	role.Customized = false
	role.MaxDiscountPercent, role.MaxRefundAmount, role.MaxCashOutAmount =
		nullDec(t.MaxDiscountPercent), nullDec(t.MaxRefundAmount), nullDec(t.MaxCashOutAmount)
	err = s.repo.InTx(ctx, func(r *Repo) error {
		if err := r.Update(ctx, role); err != nil {
			return err
		}
		return r.SetPermissions(ctx, role.ID, perms)
	})
	if err != nil {
		return Role{}, err
	}
	s.access.InvalidateRole(role.ID)
	role.Permissions = perms
	return role, nil
}

// validate normalises the input and enforces registry membership, scope and
// no-escalation. It returns the sorted, de-duplicated permission list.
func (s *Service) validate(actor identity.Actor, in Input) ([]string, error) {
	if strings.TrimSpace(in.Name) == "" || len(strings.TrimSpace(in.Name)) > 80 {
		return nil, httpx.ErrValidation.WithDetails(map[string]any{"name": "required, at most 80 characters"})
	}
	if len(in.Description) > 255 {
		return nil, httpx.ErrValidation.WithDetails(map[string]any{"description": "at most 255 characters"})
	}
	if err := validateLimits(in.Limits); err != nil {
		return nil, err
	}
	perms := slices.Clone(in.Permissions)
	slices.Sort(perms)
	perms = slices.Compact(perms)
	for _, c := range perms {
		p, ok := authz.Lookup(c)
		if !ok || p.Scope() != authz.ScopeTenant {
			return nil, ErrUnknownPerm.WithDetails(map[string]any{"permission": c})
		}
	}
	if !actor.Access.Perms.Contains(perms) {
		return nil, ErrGrantEscalation
	}
	if perms == nil {
		perms = []string{}
	}
	return perms, nil
}

func validateLimits(l Limits) error {
	hundred := decimal.NewFromInt(100)
	for _, d := range []decimal.NullDecimal{l.MaxDiscountPercent, l.MaxRefundAmount, l.MaxCashOutAmount} {
		if d.Valid && d.Decimal.IsNegative() {
			return ErrInvalidRoleLimit
		}
	}
	if l.MaxDiscountPercent.Valid && l.MaxDiscountPercent.Decimal.GreaterThan(hundred) {
		return ErrInvalidRoleLimit
	}
	return nil
}

func limitsEqual(a, b Limits) bool {
	eq := func(x, y decimal.NullDecimal) bool {
		return x.Valid == y.Valid && (!x.Valid || x.Decimal.Equal(y.Decimal))
	}
	return eq(a.MaxDiscountPercent, b.MaxDiscountPercent) &&
		eq(a.MaxRefundAmount, b.MaxRefundAmount) && eq(a.MaxCashOutAmount, b.MaxCashOutAmount)
}

var nonCode = regexp.MustCompile(`[^a-z0-9]+`)

// uniqueCode derives a stable role code from its name, avoiding template
// codes and existing roles.
func uniqueCode(name string, taken []string) string {
	base := strings.Trim(nonCode.ReplaceAllString(strings.ToLower(name), "_"), "_")
	if base == "" {
		base = "role"
	}
	base = "custom_" + base
	if len(base) > 34 {
		base = base[:34]
	}
	code := base
	for i := 2; slices.Contains(taken, code); i++ {
		code = base + "_" + strconv.Itoa(i)
	}
	return code
}

func nilIfEmpty(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return &s
}

// ByPublicIDForAssign returns a role and its permission set, for the employee
// service's role assignment checks.
func (s *Service) ByPublicIDForAssign(ctx context.Context, tenantID uint64, publicID string) (Role, authz.Set, error) {
	role, err := s.Get(ctx, tenantID, publicID)
	if err != nil {
		return Role{}, authz.Set{}, err
	}
	if role.IsLocked {
		return role, authz.AllOf(), nil
	}
	return role, authz.NewSet(role.Permissions...), nil
}

// ByCodeForAssign is ByPublicIDForAssign keyed by role code.
func (s *Service) ByCodeForAssign(ctx context.Context, tenantID uint64, code string) (Role, authz.Set, error) {
	role, err := s.repo.ByCode(ctx, tenantID, code)
	if db.IsNotFound(err) {
		return Role{}, authz.Set{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, authz.Set{}, err
	}
	return s.ByPublicIDForAssign(ctx, tenantID, role.PublicID)
}

// RoleIDByCode resolves a role id (onboarding and seeds pass template codes).
func (s *Service) RoleIDByCode(ctx context.Context, tenantID uint64, code string) (uint64, error) {
	role, err := s.repo.ByCode(ctx, tenantID, code)
	if db.IsNotFound(err) {
		return 0, ErrRoleNotFound.WithDetails(map[string]any{"role": code})
	}
	return role.ID, err
}

// CountActiveWithRole counts active users in a role (last-owner guard).
func (s *Service) CountActiveWithRole(ctx context.Context, tenantID, roleID uint64) (int, error) {
	return s.repo.CountActiveWithRole(ctx, tenantID, roleID)
}
