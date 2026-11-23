package users

import (
	"context"
	"strings"

	"mauzo/internal/core/security"
)

// UserService owns tenant-employee write rules. Seeds use it; the CRUD
// endpoints arrive in Phase 3 with roles & permissions.
type UserService struct {
	repo   *UserRepo
	params security.Argon2Params
}

// NewUserService builds a UserService.
func NewUserService(repo *UserRepo, params security.Argon2Params) *UserService {
	return &UserService{repo: repo, params: params}
}

// CreateTenantUser inserts an employee with a hashed password (may be nil for
// PIN-only accounts later).
func (s *UserService) CreateTenantUser(ctx context.Context, tenantID uint64,
	name, email, username, password, role string) (User, error) {

	u := User{
		TenantID: tenantID,
		PublicID: security.NewPublicID(),
		Name:     strings.TrimSpace(name),
		Role:     role,
		Status:   StatusActive,
	}
	if v := strings.ToLower(strings.TrimSpace(email)); v != "" {
		if !strings.Contains(v, "@") {
			return User{}, ErrInvalidEmail
		}
		u.Email = &v
	}
	if v := strings.TrimSpace(username); v != "" {
		// Sign-in matches "email = ? OR username = ?", so a username must never
		// look like an email or it could shadow another user's email.
		if strings.Contains(v, "@") {
			return User{}, ErrInvalidUsername
		}
		u.Username = &v
	}
	if password != "" {
		hash, err := s.params.HashPassword(password)
		if err != nil {
			return User{}, err
		}
		u.PasswordHash = &hash
	}
	if u.Role == "" {
		u.Role = UserRoleCashier
	}
	if err := s.repo.Create(ctx, &u); err != nil {
		return User{}, err
	}
	return u, nil
}
