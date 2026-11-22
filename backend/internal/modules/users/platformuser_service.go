package users

import (
	"context"
	"slices"
	"strings"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

const minPasswordLen = 8

// PlatformUserService owns platform operator business rules.
type PlatformUserService struct {
	repo   *PlatformUserRepo
	params security.Argon2Params
}

// NewPlatformUserService builds a PlatformUserService.
func NewPlatformUserService(repo *PlatformUserRepo, params security.Argon2Params) *PlatformUserService {
	return &PlatformUserService{repo: repo, params: params}
}

// Create validates and inserts a platform user.
func (s *PlatformUserService) Create(ctx context.Context, req CreatePlatformUserRequest) (PlatformUser, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if name == "" || email == "" {
		return PlatformUser{}, httpx.ErrValidation.WithDetails(map[string]any{"name/email": "required"})
	}
	if !slices.Contains(authz.PlatformRoles, req.Role) {
		return PlatformUser{}, ErrInvalidRole
	}
	if len(req.Password) < minPasswordLen {
		return PlatformUser{}, ErrWeakPassword
	}
	if exists, err := s.repo.EmailExists(ctx, email, 0); err != nil {
		return PlatformUser{}, err
	} else if exists {
		return PlatformUser{}, ErrEmailTaken
	}
	hash, err := s.params.HashPassword(req.Password)
	if err != nil {
		return PlatformUser{}, err
	}
	u := PlatformUser{
		PublicID:     security.NewPublicID(),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         req.Role,
		Status:       StatusActive,
	}
	if err := s.repo.Create(ctx, &u); err != nil {
		return PlatformUser{}, err
	}
	return u, nil
}

// Get loads a platform user by public id.
func (s *PlatformUserService) Get(ctx context.Context, publicID string) (PlatformUser, error) {
	u, err := s.repo.ByPublicID(ctx, publicID)
	if db.IsNotFound(err) {
		return PlatformUser{}, ErrUserNotFound
	}
	return u, err
}

// List returns platform users and total for pagination.
func (s *PlatformUserService) List(ctx context.Context, p httpx.Page) ([]PlatformUser, int, error) {
	return s.repo.List(ctx, p)
}

// Update applies the patch, protecting the last active super admin.
func (s *PlatformUserService) Update(ctx context.Context, publicID string, req UpdatePlatformUserRequest) (PlatformUser, error) {
	u, err := s.Get(ctx, publicID)
	if err != nil {
		return PlatformUser{}, err
	}
	if req.Name != nil {
		if name := strings.TrimSpace(*req.Name); name != "" {
			u.Name = name
		}
	}
	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return PlatformUser{}, httpx.ErrValidation.WithDetails(map[string]any{"email": "required"})
		}
		if exists, err := s.repo.EmailExists(ctx, email, u.ID); err != nil {
			return PlatformUser{}, err
		} else if exists {
			return PlatformUser{}, ErrEmailTaken
		}
		u.Email = email
	}
	if req.Role != nil {
		if !slices.Contains(authz.PlatformRoles, *req.Role) {
			return PlatformUser{}, ErrInvalidRole
		}
		if err := s.guardLastSuperAdmin(ctx, u, *req.Role, u.Status); err != nil {
			return PlatformUser{}, err
		}
		u.Role = *req.Role
	}
	return s.repo.Update(ctx, u)
}

// SetStatus disables or re-enables a platform user.
func (s *PlatformUserService) SetStatus(ctx context.Context, publicID, status string, selfID uint64) (PlatformUser, error) {
	u, err := s.Get(ctx, publicID)
	if err != nil {
		return PlatformUser{}, err
	}
	switch status {
	case StatusDisabled:
		if u.Status == StatusDisabled {
			return PlatformUser{}, ErrAlreadyDisabled
		}
		if u.ID == selfID {
			return PlatformUser{}, httpx.ErrValidation.WithDetails(map[string]any{"status": "cannot disable yourself"})
		}
		if err := s.guardLastSuperAdmin(ctx, u, u.Role, status); err != nil {
			return PlatformUser{}, err
		}
	case StatusActive:
		if u.Status == StatusActive {
			return PlatformUser{}, ErrAlreadyEnabled
		}
	default:
		return PlatformUser{}, httpx.ErrValidation.WithDetails(map[string]any{"status": "active|disabled"})
	}
	return s.repo.SetStatus(ctx, u.ID, status)
}

// SetPassword replaces the password (admin reset).
func (s *PlatformUserService) SetPassword(ctx context.Context, publicID, password string) (PlatformUser, error) {
	if len(password) < minPasswordLen {
		return PlatformUser{}, ErrWeakPassword
	}
	u, err := s.Get(ctx, publicID)
	if err != nil {
		return PlatformUser{}, err
	}
	hash, err := s.params.HashPassword(password)
	if err != nil {
		return PlatformUser{}, err
	}
	if err := s.repo.UpdatePassword(ctx, u.ID, hash); err != nil {
		return PlatformUser{}, err
	}
	return u, nil
}

func (s *PlatformUserService) guardLastSuperAdmin(ctx context.Context, u PlatformUser, newRole, newStatus string) error {
	demoted := u.Role == authz.RoleSuperAdmin && newRole != authz.RoleSuperAdmin
	disabled := newStatus == StatusDisabled
	if !demoted && !disabled {
		return nil
	}
	n, err := s.repo.CountActiveSuperAdmins(ctx)
	if err != nil {
		return err
	}
	if u.Role == authz.RoleSuperAdmin && u.Status == StatusActive && n <= 1 {
		return ErrLastSuperAdmin
	}
	return nil
}
