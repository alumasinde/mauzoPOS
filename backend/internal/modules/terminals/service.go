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
	ErrNotFound      = httpx.NewError(404, "terminal_not_found", "Terminal not found.")
	ErrIsDisabled    = httpx.NewError(409, "terminal_disabled",
		"This terminal is disabled. Enable it before issuing a code.")
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

// Response is the API view of a terminal. `online` derives from last_seen_at
// (the device pings /pos/heartbeat on an interval).
type Response struct {
	ID          string     `json:"id"`
	BranchID    uint64     `json:"branch_id"`
	Branch      string     `json:"branch"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	DeviceName  *string    `json:"device_name"`
	ActivatedAt *time.Time `json:"activated_at"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	Online      bool       `json:"online"`
	// CodeExpiresAt is set while an activation code is waiting (not shown
	// again — only that one exists and when it lapses).
	CodeExpiresAt *time.Time `json:"code_expires_at,omitempty"`
}

// IssuedCode is a freshly minted activation code — shown once.
type IssuedCode struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	// Repair is true when the terminal already has a device: using the code
	// moves the terminal to the new device and signs the old one out.
	Repair bool `json:"repair"`
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

// IssueCode mints a one-time activation code (returned once, stored hashed,
// valid for ActivationTTL). Issuing again replaces the previous code. For a
// terminal that already has a device it is a re-pair code.
func (s *Service) IssueCode(ctx context.Context, tenantID uint64, publicID string, allow func(uint64) bool) (IssuedCode, error) {
	t, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil || (allow != nil && !allow(t.BranchID)) {
		return IssuedCode{}, ErrNotFound
	}
	if t.Status == StatusDisabled {
		return IssuedCode{}, ErrIsDisabled
	}
	code, err := NewActivationCode()
	if err != nil {
		return IssuedCode{}, err
	}
	expires := time.Now().Add(ActivationTTL)
	if err := s.repo.SetActivationCode(ctx, t.ID, ptr(security.HashToken(NormalizeCode(code))), &expires); err != nil {
		return IssuedCode{}, err
	}
	return IssuedCode{Code: code, ExpiresAt: expires, Repair: t.ActivatedAt != nil}, nil
}

// UpdateRequest renames and/or disables / enables a terminal.
type UpdateRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"` // disabled | enabled
}

// Update applies an UpdateRequest and returns the fresh row.
func (s *Service) Update(ctx context.Context, tenantID uint64, publicID string, req UpdateRequest,
	allow func(uint64) bool) (Terminal, error) {
	t, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil || (allow != nil && !allow(t.BranchID)) {
		return Terminal{}, ErrNotFound
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 80 {
			return Terminal{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "1–80 characters"})
		}
		if err := s.repo.Rename(ctx, t.ID, name); err != nil {
			return Terminal{}, err
		}
	}
	if req.Status != nil {
		switch *req.Status {
		case "disabled":
			err = s.repo.Disable(ctx, t.ID)
		case "enabled":
			err = s.repo.Enable(ctx, t.ID)
		default:
			return Terminal{}, httpx.ErrValidation.WithDetails(map[string]any{"status": "disabled or enabled"})
		}
		if err != nil {
			return Terminal{}, err
		}
	}
	return s.repo.ByPublicID(ctx, tenantID, publicID)
}

func ptr[T any](v T) *T { return &v }

// onlineWindow is how fresh last_seen must be to count as online — the
// desktop heartbeats every 45s, so a device quiet for 3 minutes is offline.
const onlineWindow = 3 * time.Minute

// ToResponse maps a Terminal to its API view.
func ToResponse(t Terminal) Response {
	return Response{ID: t.PublicID, BranchID: t.BranchID, Branch: t.BranchName,
		Name: t.Name, Status: t.Status, DeviceName: t.DeviceName,
		ActivatedAt: t.ActivatedAt, LastSeenAt: t.LastSeenAt,
		Online:        t.Status == StatusActive && t.LastSeenAt != nil && time.Since(*t.LastSeenAt) < onlineWindow,
		CodeExpiresAt: liveExpiry(t.CodeExpiresAt)}
}

// liveExpiry hides a code that has already lapsed.
func liveExpiry(t *time.Time) *time.Time {
	if t == nil || time.Now().After(*t) {
		return nil
	}
	return t
}
