package subscriptions

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/modules"
	"mauzo/internal/core/security"
)

// Sentinel errors.
var (
	ErrPlanNotFound   = httpx.NewError(404, "plan_not_found", "The plan was not found.")
	ErrPlanInactive   = httpx.NewError(422, "plan_inactive", "That plan is no longer available.")
	ErrNoSubscription = httpx.NewError(404, "no_subscription", "The tenant has no subscription.")
)

// Service owns plan and subscription business rules.
type Service struct {
	repo     *Repo
	trialD   time.Duration
	onChange []func(tenantID uint64)
}

// NewService builds a Service. trialDays is applied when a plan is assigned.
func NewService(repo *Repo, trialDays int) *Service {
	return &Service{repo: repo, trialD: time.Duration(trialDays) * 24 * time.Hour}
}

// List returns plans for the platform catalogue.
func (s *Service) List(ctx context.Context) ([]Plan, error) {
	rows, err := s.repo.ListPlans(ctx, false)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].parseModules()
	}
	return rows, nil
}

// ListActive returns purchasable plans (tenant onboarding choice list).
func (s *Service) ListActive(ctx context.Context) ([]Plan, error) {
	rows, err := s.repo.ListPlans(ctx, true)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].parseModules()
	}
	return rows, nil
}

// Create validates and inserts a plan.
func (s *Service) Create(ctx context.Context, req PlanRequest) (Plan, error) {
	p := Plan{PublicID: security.NewPublicID()}
	if err := applyRequest(&p, req); err != nil {
		return Plan{}, err
	}
	if err := s.repo.CreatePlan(ctx, &p); err != nil {
		return Plan{}, err
	}
	p.parseModules()
	return p, nil
}

// Update patches a plan.
func (s *Service) Update(ctx context.Context, publicID string, req PlanRequest) (Plan, error) {
	p, err := s.repo.PlanByPublicID(ctx, publicID)
	if err != nil {
		return Plan{}, ErrPlanNotFound
	}
	if err := applyRequest(&p, req); err != nil {
		return Plan{}, err
	}
	if err := s.repo.UpdatePlan(ctx, &p); err != nil {
		return Plan{}, err
	}
	p.parseModules()
	return p, nil
}

// Assign subscribes a tenant to a plan: the open subscription is cancelled and
// a trialing one starts. Idempotent — re-assigning the same plan is a no-op.
func (s *Service) Assign(ctx context.Context, tenantID uint64, planPublicID string) (Subscription, error) {
	p, err := s.repo.PlanByPublicID(ctx, planPublicID)
	if err != nil {
		return Subscription{}, ErrPlanNotFound
	}
	if !p.IsActive {
		return Subscription{}, ErrPlanInactive
	}
	cur, err := s.repo.Current(ctx, tenantID)
	if err == nil && cur.PlanID == p.ID {
		return cur, nil
	}
	if err := s.repo.CancelOpen(ctx, tenantID); err != nil {
		return Subscription{}, err
	}
	sub := Subscription{TenantID: tenantID, PlanID: p.ID, Status: SubStatusTrialing}
	if s.trialD > 0 {
		t := time.Now().Add(s.trialD)
		sub.TrialEndsAt = &t
	}
	if err := s.repo.InsertSubscription(ctx, &sub); err != nil {
		return Subscription{}, err
	}
	s.planChanged(tenantID)
	sub.PlanPublicID = p.PublicID
	sub.PlanName = p.Name
	return sub, nil
}

// Current returns the tenant's open subscription.
func (s *Service) Current(ctx context.Context, tenantID uint64) (Subscription, error) {
	sub, err := s.repo.Current(ctx, tenantID)
	if err != nil {
		return Subscription{}, ErrNoSubscription
	}
	return sub, nil
}

func applyRequest(p *Plan, req PlanRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	// A plan always includes the dependencies of the modules it sells.
	closure, err := modules.Closure(req.Modules)
	if err != nil {
		return httpx.ErrValidation.WithDetails(map[string]any{"modules": err.Error()})
	}
	req.Modules = closure
	switch req.BillingInterval {
	case IntervalMonthly, IntervalYearly:
	default:
		return httpx.ErrValidation.WithDetails(map[string]any{"billing_interval": "monthly|yearly"})
	}
	price, err := decimal.NewFromString(req.PriceKES)
	if err != nil || price.IsNegative() {
		return httpx.ErrValidation.WithDetails(map[string]any{"price_kes": "invalid amount"})
	}
	b, err := json.Marshal(req.Modules)
	if err != nil {
		return err
	}
	p.Name, p.Description = name, nilIfEmpty(strings.TrimSpace(req.Description))
	p.PriceKES, p.BillingInterval = price, req.BillingInterval
	p.ModulesJSON = b
	p.MaxBranches, p.MaxUsers = req.MaxBranches, req.MaxUsers
	p.IsActive, p.Sort = req.IsActive, req.Sort
	return nil
}

func (p *Plan) parseModules() {
	_ = json.Unmarshal(p.ModulesJSON, &p.Modules)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
