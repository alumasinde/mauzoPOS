package registry

import (
	"context"
	"encoding/json"
	"slices"
	"sort"

	"mauzo/internal/core/identity"
	"mauzo/internal/core/modules"
)

// ModuleState is the tenant-facing view of one module.
type ModuleState struct {
	Module         modules.Module
	Enabled        bool   // stored switch
	Effective      bool   // actually on for the tenant
	Source         string // tenant | platform ("" when never toggled)
	Reason         string // why an enabled module is not effective
	InPlan         bool
	PlatformActive bool
	Config         json.RawMessage
}

// TenantView lists every module with its state for a tenant.
func (s *Service) TenantView(ctx context.Context, tenantID uint64) ([]ModuleState, error) {
	f, err := s.loadFacts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	on, reasons := f.effective()
	out := make([]ModuleState, 0, len(modules.Codes()))
	for _, m := range modules.All() {
		row := f.rows[m.Code]
		out = append(out, ModuleState{
			Module:         m,
			Enabled:        m.Core || row.Enabled,
			Effective:      on[m.Code],
			Source:         row.Source,
			Reason:         reasons[m.Code],
			InPlan:         f.inPlan(m.Code),
			PlatformActive: f.isActive(m.Code),
			Config:         json.RawMessage(row.Config),
		})
	}
	return out, nil
}

// PlatformState is one row of the platform module screen.
type PlatformState struct {
	Module   modules.Module
	IsActive bool
}

// PlatformView lists every module with its platform switch.
func (s *Service) PlatformView(ctx context.Context) ([]PlatformState, error) {
	active, err := s.activeMap(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PlatformState, 0, len(modules.Codes()))
	for _, m := range modules.All() {
		on, ok := active[m.Code]
		out = append(out, PlatformState{Module: m, IsActive: m.Core || !ok || on})
	}
	return out, nil
}

// SetPlatformActive flips a module platform-wide. Core modules cannot be
// switched off. Every tenant's cache is dropped so the change is immediate on
// this replica.
func (s *Service) SetPlatformActive(ctx context.Context, code string, active bool) error {
	m, err := lookup(code)
	if err != nil {
		return err
	}
	if m.Core {
		return ErrCoreModule
	}
	if err := s.repo.SetActive(ctx, code, active); err != nil {
		return err
	}
	s.active.Clear()
	s.eff.Clear()
	return nil
}

// Navigation returns the sidebar entries the actor may see: items of enabled
// modules whose permission the actor holds, ordered by section then position.
func Navigation(a identity.Actor) []modules.NavItem {
	var out []modules.NavItem
	for _, m := range modules.All() {
		if !a.HasModule(m.Code) {
			continue
		}
		for _, n := range m.Nav {
			if n.Permission == "" || a.Can(n.Permission) {
				out = append(out, n)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, _ := modules.SectionByKey(out[i].Section)
		sj, _ := modules.SectionByKey(out[j].Section)
		if si != sj {
			return si < sj
		}
		return out[i].Order < out[j].Order
	})
	return slices.Clip(out)
}
