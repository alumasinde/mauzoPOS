package terminals

import (
	"context"
	"strings"
	"time"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// BranchResolver validates a branch belongs to the tenant — implemented by
// the branches module.
type BranchResolver interface {
	BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

// Sentinel errors.
var (
	ErrInvalidBranch = httpx.NewError(422, "invalid_branch", "That branch does not exist.")
)

// Service owns terminal business rules.
type Service struct {
	repo     *Repo
	branches BranchResolver
}

// NewService builds a Service.
func NewService(repo *Repo, br BranchResolver) *Service {
	return &Service{repo: repo, branches: br}
}

// CreateRequest is POST /t/terminals and the onboarding terminal step.
type CreateRequest struct {
	BranchID string `json:"branch_id"`
	Name     string `json:"name"`
}

// Response is the API view of a terminal.
type Response struct {
	ID          string     `json:"id"`
	BranchID    uint64     `json:"branch_id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	DeviceName  *string    `json:"device_name"`
	ActivatedAt *time.Time `json:"activated_at"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
}

// List returns the tenant's terminals.
func (s *Service) List(ctx context.Context, tenantID uint64) ([]Terminal, error) {
	return s.repo.List(ctx, tenantID)
}

// Count returns the tenant's terminal count.
func (s *Service) Count(ctx context.Context, tenantID uint64) (int, error) {
	return s.repo.Count(ctx, tenantID)
}

// ErrBranchForbidden rejects terminals at branches the user cannot access.
var ErrBranchForbidden = httpx.NewError(403, "branch_forbidden", "You do not have access to that branch.")

// Create registers a pending terminal at a tenant-owned branch.
func (s *Service) Create(ctx context.Context, tenantID uint64, req CreateRequest) (Terminal, error) {
	return s.CreateAt(ctx, tenantID, req, nil)
}

// CreateAt is Create with a branch-access check (allow == nil skips it).
func (s *Service) CreateAt(ctx context.Context, tenantID uint64, req CreateRequest, allow func(branchID uint64) bool) (Terminal, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Terminal{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	branchID, err := s.branches.BranchByPublicID(ctx, tenantID, req.BranchID)
	if err != nil {
		return Terminal{}, ErrInvalidBranch
	}
	if allow != nil && !allow(branchID) {
		return Terminal{}, ErrBranchForbidden
	}
	t := Terminal{
		TenantID: tenantID, BranchID: branchID,
		PublicID: security.NewPublicID(), Name: name, Status: StatusPending,
	}
	if err := s.repo.Create(ctx, &t); err != nil {
		return Terminal{}, err
	}
	return t, nil
}

// IssueCode mints a one-time activation code (returned once, stored hashed).
func (s *Service) IssueCode(ctx context.Context, tenantID uint64, publicID string) (string, error) {
	t, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return "", ErrInvalidBranch
	}
	code, err := security.NewToken()
	if err != nil {
		return "", err
	}
	code = code[:12] // short enough to read off a screen
	if err := s.repo.SetActivationCode(ctx, t.ID, ptr(security.HashToken(code))); err != nil {
		return "", err
	}
	return code, nil
}

func ptr[T any](v T) *T { return &v }

// ToResponse maps a Terminal to its API view.
func ToResponse(t Terminal) Response {
	return Response{ID: t.PublicID, BranchID: t.BranchID, Name: t.Name, Status: t.Status,
		DeviceName: t.DeviceName, ActivatedAt: t.ActivatedAt, LastSeenAt: t.LastSeenAt}
}
