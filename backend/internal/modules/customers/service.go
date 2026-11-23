package customers

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Sentinel errors.
var (
	ErrNotFound = httpx.ErrNotFound
)

// Service owns customer business rules.
type Service struct{ repo *Repo }

// NewService builds a Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// Request is POST/PATCH /t/customers input.
type Request struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	Email       string  `json:"email"`
	KRAPin      string  `json:"kra_pin"`
	CreditLimit *string `json:"credit_limit"`
	Notes       *string `json:"notes"`
}

// List returns customers for the tenant.
func (s *Service) List(ctx context.Context, tenantID uint64, search string, activeOnly bool) ([]Customer, error) {
	return s.repo.List(ctx, tenantID, search, activeOnly)
}

// Get loads one customer.
func (s *Service) Get(ctx context.Context, tenantID uint64, publicID string) (Customer, error) {
	c, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return Customer{}, ErrNotFound
	}
	return c, nil
}

// Create registers a customer.
func (s *Service) Create(ctx context.Context, tenantID uint64, req Request) (Customer, error) {
	c := Customer{
		TenantID: tenantID, PublicID: security.NewPublicID(), Status: StatusActive,
		Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		Phone: strings.TrimSpace(req.Phone), Email: strings.TrimSpace(req.Email),
		KRAPin: strings.TrimSpace(req.KRAPin), Notes: req.Notes,
	}
	if c.Name == "" {
		return Customer{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	if req.CreditLimit != nil && *req.CreditLimit != "" {
		d, err := decimal.NewFromString(*req.CreditLimit)
		if err != nil || d.LessThan(decimal.Zero) {
			return Customer{}, httpx.ErrValidation.WithDetails(map[string]any{"credit_limit": "invalid"})
		}
		c.CreditLimit = &d
	}
	if err := s.repo.Create(ctx, &c); err != nil {
		return Customer{}, err
	}
	return c, nil
}

// Update edits a customer; archived customers keep their history.
func (s *Service) Update(ctx context.Context, tenantID uint64, publicID string, req Request, archive *bool) (Customer, error) {
	c, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return Customer{}, ErrNotFound
	}
	if v := strings.TrimSpace(req.Name); v != "" {
		c.Name = v
	}
	if req.Code != "" {
		c.Code = strings.TrimSpace(req.Code)
	}
	c.Phone = strings.TrimSpace(req.Phone)
	c.Email = strings.TrimSpace(req.Email)
	c.KRAPin = strings.TrimSpace(req.KRAPin)
	if req.Notes != nil {
		c.Notes = req.Notes
	}
	if req.CreditLimit != nil {
		if *req.CreditLimit == "" {
			c.CreditLimit = nil
		} else {
			d, err := decimal.NewFromString(*req.CreditLimit)
			if err != nil || d.LessThan(decimal.Zero) {
				return Customer{}, httpx.ErrValidation.WithDetails(map[string]any{"credit_limit": "invalid"})
			}
			c.CreditLimit = &d
		}
	}
	if archive != nil {
		if *archive {
			c.Status = StatusArchived
		} else {
			c.Status = StatusActive
		}
	}
	if err := s.repo.Update(ctx, &c); err != nil {
		return Customer{}, err
	}
	return c, nil
}

// OutstandingCredit returns the customer's confirmed credit-sale balance.
func (s *Service) OutstandingCredit(ctx context.Context, tenantID, customerID uint64) (decimal.Decimal, error) {
	return s.repo.OutstandingCredit(ctx, tenantID, customerID)
}
