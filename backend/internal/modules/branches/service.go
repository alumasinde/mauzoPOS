package branches

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Sentinel errors.
var (
	ErrBranchNotFound = httpx.NewError(404, "branch_not_found", "The branch was not found.")
	ErrCodeTaken      = httpx.NewError(409, "branch_code_taken", "A branch with that code already exists.")
)

// Service owns branch business rules.
type Service struct {
	conn *sqlx.DB
	repo *Repo
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo) *Service {
	return &Service{conn: conn, repo: repo}
}

// CreateRequest is POST /t/branches and the onboarding branch step.
type CreateRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	IsMain   bool   `json:"is_main"`
}

// Response is the API view of a branch.
type Response struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Phone    string `json:"phone,omitempty"`
	Location string `json:"location,omitempty"`
	IsMain   bool   `json:"is_main"`
	Status   string `json:"status"`
}

// List returns the tenant's branches.
func (s *Service) List(ctx context.Context, tenantID uint64) ([]Branch, error) {
	return s.repo.List(ctx, tenantID)
}

// Count returns the tenant's branch count.
func (s *Service) Count(ctx context.Context, tenantID uint64) (int, error) {
	return s.repo.Count(ctx, tenantID)
}

// Create adds a branch; the first branch is forced main.
func (s *Service) Create(ctx context.Context, tenantID uint64, req CreateRequest) (Branch, error) {
	b := Branch{
		TenantID: tenantID, PublicID: security.NewPublicID(),
		Code: strings.ToUpper(strings.TrimSpace(req.Code)),
		Name: strings.TrimSpace(req.Name), Status: StatusActive,
		Phone: nilIfEmpty(req.Phone), Location: nilIfEmpty(req.Location),
	}
	if b.Code == "" || b.Name == "" {
		return Branch{}, httpx.ErrValidation.WithDetails(
			map[string]any{"code": "required", "name": "required"})
	}
	n, err := s.repo.Count(ctx, tenantID)
	if err != nil {
		return Branch{}, err
	}
	b.IsMain = n == 0 || req.IsMain
	if err := s.repo.Create(ctx, &b); err != nil {
		if strings.Contains(err.Error(), "uq_branches_tenant_code") {
			return Branch{}, ErrCodeTaken
		}
		return Branch{}, err
	}
	return b, nil
}

// UpdateRequest is PATCH /t/branches/{id}.
type UpdateRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	IsMain   *bool  `json:"is_main"`
	Status   string `json:"status"` // "" keeps current
}

// Update edits a branch. Code stays unique per tenant; setting is_main clears
// it on the others. The main branch cannot be closed.
func (s *Service) Update(ctx context.Context, tenantID uint64, publicID string,
	req UpdateRequest) (Branch, error) {
	b, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return Branch{}, ErrBranchNotFound
	}
	if c := strings.ToUpper(strings.TrimSpace(req.Code)); c != "" && c != b.Code {
		b.Code = c
	}
	if n := strings.TrimSpace(req.Name); n != "" {
		b.Name = n
	}
	b.Phone = nilIfEmpty(req.Phone)
	b.Location = nilIfEmpty(req.Location)
	if req.Status != "" {
		if req.Status != StatusActive && req.Status != StatusClosed {
			return Branch{}, httpx.ErrValidation.WithDetails(
				map[string]any{"status": "active | closed"})
		}
		b.Status = req.Status
	}
	if req.IsMain != nil {
		b.IsMain = *req.IsMain
	}
	if b.Status == StatusClosed && b.IsMain {
		return Branch{}, httpx.NewError(422, "main_closed",
			"The main branch cannot be closed — make another branch main first.")
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if err := s.repo.Update(ctx, tx, b); err != nil {
			if strings.Contains(err.Error(), "uq_branches_tenant_code") {
				return ErrCodeTaken
			}
			return err
		}
		if b.IsMain {
			return s.repo.ClearMain(ctx, tx, tenantID, b.ID)
		}
		return nil
	})
	if err != nil {
		return Branch{}, err
	}
	return s.repo.ByPublicID(ctx, tenantID, publicID)
}

// ToResponse maps a Branch to its API view.
func ToResponse(b Branch) Response {
	r := Response{ID: b.PublicID, Code: b.Code, Name: b.Name, IsMain: b.IsMain, Status: b.Status}
	if b.Phone != nil {
		r.Phone = *b.Phone
	}
	if b.Location != nil {
		r.Location = *b.Location
	}
	return r
}

func nilIfEmpty(s string) *string {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return &v
}

// BranchByPublicID resolves a public id to an internal id within the tenant —
// implements terminals.BranchResolver.
func (s *Service) BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	b, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return 0, err
	}
	return b.ID, nil
}
