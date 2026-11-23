package subscriptions

import (
	"context"

	"mauzo/internal/core/db"
)

// CurrentPlan returns the tenant's current plan; ok is false when the tenant
// has no open subscription.
func (s *Service) CurrentPlan(ctx context.Context, tenantID uint64) (Plan, bool, error) {
	p, err := s.repo.CurrentPlan(ctx, tenantID)
	if db.IsNotFound(err) {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, err
	}
	p.parseModules()
	return p, true, nil
}

// PlanModules implements registry.PlanSource: the module allow-list of the
// tenant's current plan (hasPlan=false when unsubscribed — no gating).
func (s *Service) PlanModules(ctx context.Context, tenantID uint64) ([]string, bool, error) {
	p, ok, err := s.CurrentPlan(ctx, tenantID)
	if err != nil || !ok {
		return nil, ok, err
	}
	return p.Modules, true, nil
}

// MaxUsers implements the employee seat limit (nil = unlimited or no plan).
func (s *Service) MaxUsers(ctx context.Context, tenantID uint64) (*int, error) {
	p, ok, err := s.CurrentPlan(ctx, tenantID)
	if err != nil || !ok {
		return nil, err
	}
	return p.MaxUsers, nil
}

// OnPlanChange registers a callback fired after a tenant's plan changes (the
// module registry uses it to drop cached enablement).
func (s *Service) OnPlanChange(fn func(tenantID uint64)) { s.onChange = append(s.onChange, fn) }

func (s *Service) planChanged(tenantID uint64) {
	for _, fn := range s.onChange {
		fn(tenantID)
	}
}
