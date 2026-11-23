package registry

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"mauzo/internal/core/cache"
	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/modules"
	"mauzo/internal/core/settings"
)

// PlanSource reports the module allow-list of a tenant's current plan.
// Implemented by modules/subscriptions.
type PlanSource interface {
	PlanModules(ctx context.Context, tenantID uint64) (mods []string, hasPlan bool, err error)
}

// cacheTTL bounds how long another API replica may serve stale enablement.
const cacheTTL = 30 * time.Second

// Service owns module enablement rules.
type Service struct {
	repo   *Repo
	plans  PlanSource
	eff    *cache.TTL[uint64, []string]
	active *cache.TTL[string, map[string]bool]
}

// NewService builds a Service. plans may be nil (no plan gating).
func NewService(repo *Repo, plans PlanSource) *Service {
	return &Service{
		repo:   repo,
		plans:  plans,
		eff:    cache.New[uint64, []string](cacheTTL, 50_000),
		active: cache.New[string, map[string]bool](cacheTTL, 1),
	}
}

// SyncRegistry inserts missing module codes into the modules table (boot).
func (s *Service) SyncRegistry(ctx context.Context) error {
	return s.repo.SyncCodes(ctx, modules.Codes())
}

func lookup(code string) (modules.Module, error) {
	m, ok := modules.Get(code)
	if !ok {
		return modules.Module{}, ErrUnknownModule
	}
	return m, nil
}

// Enable switches code on together with any dependency that is off. Tenant
// enablement is plan-gated; platform overrides are not. It returns the codes
// that were switched on.
func (s *Service) Enable(ctx context.Context, tenantID uint64, code, source string) ([]string, error) {
	m, err := lookup(code)
	if err != nil {
		return nil, err
	}
	if m.Core {
		return nil, ErrCoreModule
	}
	f, err := s.loadFacts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	closure, err := modules.Closure([]string{code})
	if err != nil {
		return nil, err
	}
	var notInPlan, toEnable []string
	for _, c := range closure {
		if dep, _ := modules.Get(c); dep.Core {
			continue
		}
		if !f.isActive(c) {
			return nil, ErrModuleUnavailable.WithDetails(map[string]any{"module": c})
		}
		if source == SourceTenant && !f.inPlan(c) {
			notInPlan = append(notInPlan, c)
		}
		if r, ok := f.rows[c]; !ok || !r.Enabled || (source == SourcePlatform && r.Source != SourcePlatform && c == code) {
			toEnable = append(toEnable, c)
		}
	}
	if len(notInPlan) > 0 {
		return nil, ErrNotInPlan.WithDetails(map[string]any{"modules": notInPlan})
	}
	err = s.repo.InTx(ctx, func(r *Repo) error {
		for _, c := range toEnable {
			if err := r.SetEnabled(ctx, tenantID, c, true, source); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.Invalidate(tenantID)
	return toEnable, nil
}

// Disable switches code off. When other enabled modules depend on it the call
// fails with ErrHasDependents unless cascade is set, in which case those are
// switched off too. It returns the codes switched off.
func (s *Service) Disable(ctx context.Context, tenantID uint64, code string, cascade bool) ([]string, error) {
	m, err := lookup(code)
	if err != nil {
		return nil, err
	}
	if m.Core {
		return nil, ErrCoreModule
	}
	rows, err := s.repo.TenantModules(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var enabled []string
	sources := map[string]string{}
	for _, r := range rows {
		if r.Enabled {
			enabled = append(enabled, r.Module)
			sources[r.Module] = r.Source
		}
	}
	if !slices.Contains(enabled, code) {
		return nil, nil // already off
	}
	dependents := modules.Dependents(code, enabled)
	if len(dependents) > 0 && !cascade {
		return nil, ErrHasDependents.WithDetails(map[string]any{"dependents": dependents})
	}
	off := append(slices.Clone(dependents), code)
	err = s.repo.InTx(ctx, func(r *Repo) error {
		for _, c := range off {
			if err := r.SetEnabled(ctx, tenantID, c, false, sources[c]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.Invalidate(tenantID)
	return off, nil
}

// SetModules makes the tenant-enabled set exactly mods plus their
// dependencies (onboarding). Platform overrides are left untouched.
func (s *Service) SetModules(ctx context.Context, tenantID uint64, mods []string) error {
	closure, err := modules.Closure(mods)
	if err != nil {
		var unk modules.ErrUnknown
		if errors.As(err, &unk) {
			return httpx.ErrValidation.WithDetails(map[string]any{"modules": "unknown module " + unk.Code})
		}
		return err
	}
	f, err := s.loadFacts(ctx, tenantID)
	if err != nil {
		return err
	}
	var notInPlan []string
	for _, c := range closure {
		if !f.inPlan(c) {
			notInPlan = append(notInPlan, c)
		}
		if !f.isActive(c) {
			return ErrModuleUnavailable.WithDetails(map[string]any{"module": c})
		}
	}
	if len(notInPlan) > 0 {
		return ErrNotInPlan.WithDetails(map[string]any{"modules": notInPlan})
	}
	err = s.repo.InTx(ctx, func(r *Repo) error {
		for _, c := range closure {
			if dep, _ := modules.Get(c); dep.Core {
				continue
			}
			if row, ok := f.rows[c]; ok && row.Enabled {
				continue
			}
			if err := r.SetEnabled(ctx, tenantID, c, true, SourceTenant); err != nil {
				return err
			}
		}
		for c, row := range f.rows {
			if row.Enabled && row.Source != SourcePlatform && !slices.Contains(closure, c) {
				if err := r.SetEnabled(ctx, tenantID, c, false, row.Source); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.Invalidate(tenantID)
	return nil
}

// EnableAll enables codes and their dependencies on q (which may be a caller's
// transaction) without plan gating — used when a business profile is applied.
// The caller must call Invalidate after committing.
func (s *Service) EnableAll(ctx context.Context, q db.Querier, tenantID uint64, codes []string) error {
	closure, err := modules.Closure(codes)
	if err != nil {
		return err
	}
	r := s.repo.WithQuerier(q)
	for _, c := range closure {
		if m, _ := modules.Get(c); m.Core {
			continue
		}
		if _, err := r.q.ExecContext(ctx,
			`INSERT INTO tenant_modules (tenant_id, module, enabled, source) VALUES (?, ?, 1, ?)
			 ON DUPLICATE KEY UPDATE enabled = 1`, tenantID, c, SourceTenant); err != nil {
			return err
		}
	}
	return nil
}

// SetConfig validates and stores a module's tenant config.
func (s *Service) SetConfig(ctx context.Context, tenantID uint64, code string, raw json.RawMessage) error {
	m, err := lookup(code)
	if err != nil {
		return err
	}
	if m.ConfigSchema == nil {
		return ErrNoConfig
	}
	if err := settings.ValidateSchema(m.ConfigSchema, raw, code); err != nil {
		return httpx.ErrValidation.WithDetails(map[string]any{"config": err.Error()})
	}
	on, err := s.EffectiveModules(ctx, tenantID)
	if err != nil {
		return err
	}
	if !slices.Contains(on, code) {
		return ErrModuleNotEnabled
	}
	return s.repo.SetConfig(ctx, tenantID, code, raw)
}

// Config returns a module's stored config (nil when unset). Feature modules
// read their settings through this.
func (s *Service) Config(ctx context.Context, tenantID uint64, code string) (json.RawMessage, error) {
	rows, err := s.repo.TenantModules(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Module == code {
			return json.RawMessage(r.Config), nil
		}
	}
	return nil, nil
}

// EnabledModules is the onboarding view: the tenant's effective modules.
func (s *Service) EnabledModules(ctx context.Context, tenantID uint64) ([]string, error) {
	return s.EffectiveModules(ctx, tenantID)
}
