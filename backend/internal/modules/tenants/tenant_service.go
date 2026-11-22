package tenants

import (
	"context"
	"regexp"
	"strings"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,59}$`)

// SessionRevoker drops every live session of a tenant. Implemented by
// modules/auth so suspension signs users out immediately.
type SessionRevoker interface {
	RevokeAllForTenant(ctx context.Context, tenantID uint64) error
}

// Service owns tenant business rules.
type Service struct {
	repo    *Repo
	revoker SessionRevoker
}

// NewService builds a Service.
func NewService(repo *Repo, revoker SessionRevoker) *Service {
	return &Service{repo: repo, revoker: revoker}
}

// Create validates and inserts a tenant.
func (s *Service) Create(ctx context.Context, req CreateTenantRequest) (Tenant, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Tenant{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(slug) {
		return Tenant{}, httpx.ErrValidation.WithDetails(map[string]any{
			"slug": "lowercase letters, digits and hyphens, 2–60 chars",
		})
	}
	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		return Tenant{}, err
	}
	if exists {
		return Tenant{}, ErrSlugTaken
	}
	t := Tenant{
		PublicID: security.NewPublicID(),
		Name:     name,
		Slug:     slug,
		Status:   StatusActive,
	}
	if v := strings.TrimSpace(req.Email); v != "" {
		t.Email = &v
	}
	if v := strings.TrimSpace(req.Phone); v != "" {
		t.Phone = &v
	}
	if err := s.repo.Create(ctx, &t); err != nil {
		return Tenant{}, err
	}
	return t, nil
}

// Get loads a tenant by public id.
func (s *Service) Get(ctx context.Context, publicID string) (Tenant, error) {
	t, err := s.repo.ByPublicID(ctx, publicID)
	if db.IsNotFound(err) {
		return Tenant{}, ErrTenantNotFound
	}
	return t, err
}

// List returns tenants and total for pagination.
func (s *Service) List(ctx context.Context, p httpx.Page) ([]Tenant, int, error) {
	return s.repo.List(ctx, p)
}

// Update applies the patch and returns the fresh tenant.
func (s *Service) Update(ctx context.Context, publicID string, req UpdateTenantRequest) (Tenant, error) {
	t, err := s.Get(ctx, publicID)
	if err != nil {
		return Tenant{}, err
	}
	if req.Name != nil {
		if name := strings.TrimSpace(*req.Name); name != "" {
			t.Name = name
		}
	}
	if req.Email != nil {
		v := strings.TrimSpace(*req.Email)
		t.Email = nil
		if v != "" {
			t.Email = &v
		}
	}
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		t.Phone = nil
		if v != "" {
			t.Phone = &v
		}
	}
	return s.repo.Update(ctx, t)
}

// Suspend blocks all access for the tenant and revokes its sessions.
func (s *Service) Suspend(ctx context.Context, publicID, reason string) (Tenant, error) {
	t, err := s.Get(ctx, publicID)
	if err != nil {
		return Tenant{}, err
	}
	if t.Status == StatusSuspended {
		return Tenant{}, ErrAlreadySuspended
	}
	reason = strings.TrimSpace(reason)
	var r *string
	if reason != "" {
		r = &reason
	}
	t, err = s.repo.SetStatus(ctx, t.ID, StatusSuspended, r)
	if err != nil {
		return Tenant{}, err
	}
	if err := s.revoker.RevokeAllForTenant(ctx, t.ID); err != nil {
		return Tenant{}, err
	}
	return t, nil
}

// Reactivate lifts a suspension.
func (s *Service) Reactivate(ctx context.Context, publicID string) (Tenant, error) {
	t, err := s.Get(ctx, publicID)
	if err != nil {
		return Tenant{}, err
	}
	if t.Status != StatusSuspended {
		return Tenant{}, ErrNotSuspended
	}
	return s.repo.SetStatus(ctx, t.ID, StatusActive, nil)
}
