package auth

import (
	"context"

	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/users"
)

// Adapters map the users/tenants repos onto the account interfaces the service
// consumes.

// PlatformAccountStore adapts users.PlatformUserRepo.
type PlatformAccountStore struct {
	repo *users.PlatformUserRepo
}

// NewPlatformAccountStore builds a PlatformAccountStore.
func NewPlatformAccountStore(repo *users.PlatformUserRepo) *PlatformAccountStore {
	return &PlatformAccountStore{repo: repo}
}

func platformAccount(u users.PlatformUser) Account {
	return Account{
		ID:            u.ID,
		PublicID:      u.PublicID,
		Name:          u.Name,
		Identity:      u.Email,
		Email:         u.Email,
		PasswordHash:  &u.PasswordHash,
		Role:          u.Role,
		Status:        u.Status,
		TOTPSecret:    u.TOTPSecret,
		TOTPEnabled:   u.TOTPEnabled,
		RecoveryCodes: u.RecoveryCodes,
	}
}

// ByEmail implements platformAccountStore.
func (s *PlatformAccountStore) ByEmail(ctx context.Context, email string) (Account, error) {
	u, err := s.repo.ByEmail(ctx, email)
	return platformAccount(u), err
}

// ByID implements platformAccountStore.
func (s *PlatformAccountStore) ByID(ctx context.Context, id uint64) (Account, error) {
	u, err := s.repo.ByID(ctx, id)
	return platformAccount(u), err
}

// UpdatePassword implements platformAccountStore.
func (s *PlatformAccountStore) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	return s.repo.UpdatePassword(ctx, id, hash)
}

// SetTOTP implements platformAccountStore.
func (s *PlatformAccountStore) SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error {
	return s.repo.SetTOTP(ctx, id, secret, enabled, codes)
}

// UpdateRecoveryCodes implements platformAccountStore.
func (s *PlatformAccountStore) UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error {
	return s.repo.UpdateRecoveryCodes(ctx, id, codes)
}

// TouchLastLogin implements platformAccountStore.
func (s *PlatformAccountStore) TouchLastLogin(ctx context.Context, id uint64) error {
	return s.repo.TouchLastLogin(ctx, id)
}

// TenantAccountStore adapts users.UserRepo.
type TenantAccountStore struct {
	repo *users.UserRepo
}

// NewTenantAccountStore builds a TenantAccountStore.
func NewTenantAccountStore(repo *users.UserRepo) *TenantAccountStore {
	return &TenantAccountStore{repo: repo}
}

func tenantAccount(u users.User) Account {
	var ident, email string
	if u.Email != nil {
		ident, email = *u.Email, *u.Email
	} else if u.Username != nil {
		ident = *u.Username
	}
	return Account{
		ID:            u.ID,
		TenantID:      u.TenantID,
		PublicID:      u.PublicID,
		Name:          u.Name,
		Identity:      ident,
		Email:         email,
		PasswordHash:  u.PasswordHash,
		RoleID:        u.RoleID,
		AllBranches:   u.AllBranches,
		Role:          u.Role,
		RoleName:      u.RoleName,
		Status:        u.Status,
		TOTPSecret:    u.TOTPSecret,
		TOTPEnabled:   u.TOTPEnabled,
		RecoveryCodes: u.RecoveryCodes,
	}
}

// ByIdentity implements tenantAccountStore.
func (s *TenantAccountStore) ByIdentity(ctx context.Context, tenantID uint64, identityName string) (Account, error) {
	u, err := s.repo.ByIdentity(ctx, tenantID, identityName)
	return tenantAccount(u), err
}

// ByID implements tenantAccountStore.
func (s *TenantAccountStore) ByID(ctx context.Context, tenantID, id uint64) (Account, error) {
	u, err := s.repo.ByID(ctx, tenantID, id)
	return tenantAccount(u), err
}

// UpdatePassword implements tenantAccountStore.
func (s *TenantAccountStore) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	return s.repo.UpdatePassword(ctx, id, hash)
}

// SetTOTP implements tenantAccountStore.
func (s *TenantAccountStore) SetTOTP(ctx context.Context, id uint64, secret *string, enabled bool, codes []byte) error {
	return s.repo.SetTOTP(ctx, id, secret, enabled, codes)
}

// UpdateRecoveryCodes implements tenantAccountStore.
func (s *TenantAccountStore) UpdateRecoveryCodes(ctx context.Context, id uint64, codes []byte) error {
	return s.repo.UpdateRecoveryCodes(ctx, id, codes)
}

// TouchLastLogin implements tenantAccountStore.
func (s *TenantAccountStore) TouchLastLogin(ctx context.Context, id uint64) error {
	return s.repo.TouchLastLogin(ctx, id)
}

// TenantStore adapts tenants.Repo.
type TenantStore struct {
	repo *tenants.Repo
}

// NewTenantStore builds a TenantStore.
func NewTenantStore(repo *tenants.Repo) *TenantStore {
	return &TenantStore{repo: repo}
}

func tenantInfo(t tenants.Tenant) TenantInfo {
	return TenantInfo{ID: t.ID, PublicID: t.PublicID, Name: t.Name, Slug: t.Slug, Status: t.Status}
}

// BySlug implements tenantStore.
func (s *TenantStore) BySlug(ctx context.Context, slug string) (TenantInfo, error) {
	t, err := s.repo.BySlug(ctx, slug)
	return tenantInfo(t), err
}

// ByID implements tenantStore.
func (s *TenantStore) ByID(ctx context.Context, id uint64) (TenantInfo, error) {
	t, err := s.repo.ByID(ctx, id)
	return tenantInfo(t), err
}
