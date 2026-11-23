package registry

import (
	"context"
	"slices"

	"mauzo/internal/core/modules"
)

// Why a stored-enabled module is not effective.
const (
	ReasonPlatformInactive = "platform_inactive"
	ReasonNotInPlan        = "not_in_plan"
	ReasonDependencyOff    = "dependency_off"
)

// facts is everything needed to compute a tenant's effective modules.
type facts struct {
	rows    map[string]TenantModule
	active  map[string]bool
	plan    []string
	hasPlan bool
}

func (s *Service) loadFacts(ctx context.Context, tenantID uint64) (facts, error) {
	rows, err := s.repo.TenantModules(ctx, tenantID)
	if err != nil {
		return facts{}, err
	}
	active, err := s.activeMap(ctx)
	if err != nil {
		return facts{}, err
	}
	f := facts{rows: map[string]TenantModule{}, active: active}
	for _, r := range rows {
		f.rows[r.Module] = r
	}
	if s.plans != nil {
		if f.plan, f.hasPlan, err = s.plans.PlanModules(ctx, tenantID); err != nil {
			return facts{}, err
		}
	}
	return f, nil
}

// inPlan reports whether the plan allows code (core and no-plan always do).
func (f facts) inPlan(code string) bool {
	m, _ := modules.Get(code)
	return m.Core || !f.hasPlan || slices.Contains(f.plan, code)
}

// isActive reports the platform switch (missing row = active).
func (f facts) isActive(code string) bool {
	m, _ := modules.Get(code)
	if m.Core {
		return true
	}
	on, ok := f.active[code]
	return !ok || on
}

// effective computes the enabled set and, for stored-enabled modules that are
// off, the reason. Core modules are always on; a module whose dependency is
// off is off too (fixpoint).
func (f facts) effective() (on map[string]bool, reasons map[string]string) {
	on, reasons = map[string]bool{}, map[string]string{}
	for _, c := range modules.CoreCodes() {
		on[c] = true
	}
	for code, r := range f.rows {
		if !r.Enabled || !modules.Known(code) {
			continue
		}
		switch {
		case !f.isActive(code):
			reasons[code] = ReasonPlatformInactive
		case r.Source != SourcePlatform && !f.inPlan(code):
			reasons[code] = ReasonNotInPlan
		default:
			on[code] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for code := range on {
			m, _ := modules.Get(code)
			for _, d := range m.Depends {
				if !on[d] {
					delete(on, code)
					reasons[code] = ReasonDependencyOff
					changed = true
					break
				}
			}
		}
	}
	return on, reasons
}

// EffectiveModules returns the tenant's enabled module codes (cached). This
// is what RequireModule/Can enforce.
func (s *Service) EffectiveModules(ctx context.Context, tenantID uint64) ([]string, error) {
	return s.eff.GetOrLoad(tenantID, func() ([]string, error) {
		f, err := s.loadFacts(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		on, _ := f.effective()
		out := make([]string, 0, len(on))
		for _, c := range modules.Codes() {
			if on[c] {
				out = append(out, c)
			}
		}
		return out, nil
	})
}

// activeMap returns the platform switch per module (cached).
func (s *Service) activeMap(ctx context.Context) (map[string]bool, error) {
	return s.active.GetOrLoad("", func() (map[string]bool, error) {
		rows, err := s.repo.PlatformModules(ctx)
		if err != nil {
			return nil, err
		}
		m := make(map[string]bool, len(rows))
		for _, r := range rows {
			m[r.Code] = r.IsActive
		}
		return m, nil
	})
}

// Invalidate drops a tenant's cached effective modules. Call after any write
// that can change them (module toggles, plan changes).
func (s *Service) Invalidate(tenantID uint64) { s.eff.Delete(tenantID) }
