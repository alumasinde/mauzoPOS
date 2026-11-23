// Package onboarding drives the new-tenant wizard: a step state machine whose
// steps fan out to the tenants, businesstypes, subscriptions, branches,
// settings, users and terminals modules through narrow interfaces.
package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"mauzo/internal/core/httpx"
	coresettings "mauzo/internal/core/settings"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/businesstypes"
	"mauzo/internal/modules/subscriptions"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/terminals"
	"mauzo/internal/modules/users"
)

// Wizard step numbers persisted on tenants.onboarding_step.
const (
	StepBusiness    = 1
	StepProfile     = 2
	StepPlan        = 3
	StepModules     = 4
	StepBranch      = 5
	StepAdmin       = 6
	StepPreferences = 7
	StepTerminal    = 8
)

// LastStep is the final numbered step before completion.
const LastStep = StepTerminal

// Deps lists the cross-module capabilities the wizard needs, satisfied by
// real services in the container and stubs in tests.
type Deps struct {
	Tenants TenantStore
	Types   TypeLister
	Profile ProfileApplier
	Plans   PlanCatalog
	Subs    SubscriptionAssigner
	Mods    ModuleSetter
	Branch  BranchMaker
	Admins  AdminMaker
	Prefs   PreferenceWriter
	Term    TerminalMaker
}

// TenantStore exposes tenant onboarding state + business fields.
type TenantStore interface {
	ByID(ctx context.Context, id uint64) (tenants.Tenant, error)
	SetBusinessFields(ctx context.Context, tenantID uint64, name string, kraPin, address *string, vat bool) error
	AdvanceOnboarding(ctx context.Context, tenantID uint64, step int) error
	CompleteOnboarding(ctx context.Context, tenantID uint64) error
}

// TypeLister lists business types.
type TypeLister interface {
	ListTypes(ctx context.Context) ([]businesstypes.BusinessType, error)
}

// ProfileApplier seeds a tenant from a profile and sets modules.
type ProfileApplier interface {
	Apply(ctx context.Context, tenantID uint64, profilePublicID string) (businesstypes.Profile, error)
	ListProfiles(ctx context.Context, typeCode string) ([]businesstypes.Profile, error)
}

// ModuleSetter sets/enumerates enabled modules.
type ModuleSetter interface {
	SetModules(ctx context.Context, tenantID uint64, mods []string) error
	EnabledModules(ctx context.Context, tenantID uint64) ([]string, error)
}

// PlanCatalog lists purchasable plans.
type PlanCatalog interface {
	ListActive(ctx context.Context) ([]subscriptions.Plan, error)
}

// SubscriptionAssigner assigns/reads a tenant subscription.
type SubscriptionAssigner interface {
	Assign(ctx context.Context, tenantID uint64, planPublicID string) (subscriptions.Subscription, error)
	Current(ctx context.Context, tenantID uint64) (subscriptions.Subscription, error)
}

// BranchMaker creates/lists branches.
type BranchMaker interface {
	Create(ctx context.Context, tenantID uint64, req branches.CreateRequest) (branches.Branch, error)
	List(ctx context.Context, tenantID uint64) ([]branches.Branch, error)
}

// AdminMaker creates an extra tenant admin/employee.
type AdminMaker interface {
	CreateTenantUser(ctx context.Context, tenantID uint64, name, email, username, password, role string) (users.User, error)
}

// PreferenceWriter writes one settings key.
type PreferenceWriter interface {
	Set(ctx context.Context, tenantID uint64, key string, value json.RawMessage) error
}

// TerminalMaker creates/lists terminals.
type TerminalMaker interface {
	Create(ctx context.Context, tenantID uint64, req terminals.CreateRequest) (terminals.Terminal, error)
	List(ctx context.Context, tenantID uint64) ([]terminals.Terminal, error)
}

// Service orchestrates onboarding steps.
type Service struct {
	deps Deps
}

// NewService builds a Service.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// State is GET /t/onboarding — where the wizard left off plus option lists.
type State struct {
	Step          int                                 `json:"step"`
	Done          bool                                `json:"done"`
	BusinessType  *string                             `json:"business_type"`
	BusinessTypes []businesstypes.TypeResponse        `json:"business_types"`
	Profiles      []businesstypes.ProfileResponse     `json:"profiles"`
	Plans         []subscriptions.PlanResponse        `json:"plans"`
	Modules       []string                            `json:"modules"`
	Branches      []branches.Response                 `json:"branches"`
	Terminals     []terminals.Response                `json:"terminals"`
	Subscription  *subscriptions.SubscriptionResponse `json:"subscription,omitempty"`
}

// Current builds the wizard state for the tenant.
func (s *Service) Current(ctx context.Context, tenantID uint64) (State, error) {
	t, err := s.deps.Tenants.ByID(ctx, tenantID)
	if err != nil {
		return State{}, err
	}
	st := State{Step: t.OnboardingStep, Done: t.OnboardedAt != nil, BusinessType: t.BusinessType}
	types, err := s.deps.Types.ListTypes(ctx)
	if err != nil {
		return State{}, err
	}
	for _, ty := range types {
		st.BusinessTypes = append(st.BusinessTypes, businesstypes.TypeResponse{Code: ty.Code, Name: ty.Name})
	}
	profiles, err := s.deps.Profile.ListProfiles(ctx, "")
	if err != nil {
		return State{}, err
	}
	for _, p := range profiles {
		_ = p.ParseDefinition()
		st.Profiles = append(st.Profiles, businesstypes.ProfileResponse{
			ID: p.PublicID, BusinessType: p.TypeCode, Name: p.Name,
			IsDefault: p.IsDefault, Definition: p.Definition, UpdatedAt: p.UpdatedAt,
		})
	}
	plans, err := s.deps.Plans.ListActive(ctx)
	if err != nil {
		return State{}, err
	}
	for _, p := range plans {
		pr := subscriptions.PlanResponse{
			ID: p.PublicID, Name: p.Name, PriceKES: p.PriceKES.StringFixed(2),
			BillingInterval: p.BillingInterval, Modules: p.Modules,
			MaxBranches: p.MaxBranches, MaxUsers: p.MaxUsers, IsActive: p.IsActive, Sort: p.Sort,
		}
		if p.Description != nil {
			pr.Description = *p.Description
		}
		st.Plans = append(st.Plans, pr)
	}
	if st.Modules, err = s.deps.Mods.EnabledModules(ctx, tenantID); err != nil {
		return State{}, err
	}
	brs, err := s.deps.Branch.List(ctx, tenantID)
	if err != nil {
		return State{}, err
	}
	for _, b := range brs {
		st.Branches = append(st.Branches, branches.ToResponse(b))
	}
	terms, err := s.deps.Term.List(ctx, tenantID)
	if err != nil {
		return State{}, err
	}
	for _, tm := range terms {
		st.Terminals = append(st.Terminals, terminals.ToResponse(tm))
	}
	if sub, err := s.deps.Subs.Current(ctx, tenantID); err == nil {
		st.Subscription = &subscriptions.SubscriptionResponse{
			PlanID: sub.PlanPublicID, PlanName: sub.PlanName, Status: sub.Status,
			StartsAt: sub.StartsAt, TrialEndsAt: sub.TrialEndsAt,
		}
	}
	return st, nil
}

func (s *Service) advance(ctx context.Context, tenantID uint64, step int) error {
	return s.deps.Tenants.AdvanceOnboarding(ctx, tenantID, step)
}

// BusinessRequest is step 1.
type BusinessRequest struct {
	Name          string `json:"name"`
	KRAPin        string `json:"kra_pin"`
	Address       string `json:"address"`
	VATRegistered bool   `json:"vat_registered"`
}

// SaveBusiness stores step 1 details.
func (s *Service) SaveBusiness(ctx context.Context, tenantID uint64, req BusinessRequest) error {
	var kra, addr *string
	if req.KRAPin != "" {
		kra = &req.KRAPin
	}
	if req.Address != "" {
		addr = &req.Address
	}
	if req.Name == "" {
		return httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	if err := s.deps.Tenants.SetBusinessFields(ctx, tenantID, req.Name, kra, addr, req.VATRegistered); err != nil {
		return err
	}
	return s.advance(ctx, tenantID, StepBusiness)
}

// ProfileRequest is step 2.
type ProfileRequest struct {
	ProfileID string `json:"profile_id"`
}

// SaveProfile applies the chosen profile (idempotent).
func (s *Service) SaveProfile(ctx context.Context, tenantID uint64, req ProfileRequest) error {
	if req.ProfileID == "" {
		return httpx.ErrValidation.WithDetails(map[string]any{"profile_id": "required"})
	}
	if _, err := s.deps.Profile.Apply(ctx, tenantID, req.ProfileID); err != nil {
		return err
	}
	return s.advance(ctx, tenantID, StepProfile)
}

// SavePlan assigns the chosen plan (step 3).
func (s *Service) SavePlan(ctx context.Context, tenantID uint64, planID string) error {
	if planID == "" {
		return httpx.ErrValidation.WithDetails(map[string]any{"plan_id": "required"})
	}
	if _, err := s.deps.Subs.Assign(ctx, tenantID, planID); err != nil {
		return err
	}
	return s.advance(ctx, tenantID, StepPlan)
}

// ModulesRequest is step 4.
type ModulesRequest struct {
	Modules []string `json:"modules"`
}

// SaveModules enables exactly the chosen modules (validated ⊆ plan allowed at
// a later phase — Phase 3 registry does entitlement checks).
func (s *Service) SaveModules(ctx context.Context, tenantID uint64, req ModulesRequest) error {
	if err := s.deps.Mods.SetModules(ctx, tenantID, req.Modules); err != nil {
		return err
	}
	return s.advance(ctx, tenantID, StepModules)
}

// SaveBranch adds a branch (step 5).
func (s *Service) SaveBranch(ctx context.Context, tenantID uint64, req branches.CreateRequest) (branches.Branch, error) {
	b, err := s.deps.Branch.Create(ctx, tenantID, req)
	if err != nil {
		return branches.Branch{}, err
	}
	return b, s.advance(ctx, tenantID, StepBranch)
}

// AdminRequest is step 6 — optional extra employee.
type AdminRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// SaveAdmin creates an additional employee; empty request skips the step.
func (s *Service) SaveAdmin(ctx context.Context, tenantID uint64, req AdminRequest) error {
	if req.Name != "" || req.Email != "" {
		role := req.Role
		if role == "" {
			role = users.UserRoleAdmin
		}
		if _, err := s.deps.Admins.CreateTenantUser(ctx, tenantID,
			req.Name, req.Email, req.Username, req.Password, role); err != nil {
			return err
		}
	}
	return s.advance(ctx, tenantID, StepAdmin)
}

// PreferencesRequest is step 7 — arbitrary validated settings.
type PreferencesRequest struct {
	Values map[string]json.RawMessage `json:"values"`
}

// SavePreferences writes each provided settings key.
func (s *Service) SavePreferences(ctx context.Context, tenantID uint64, req PreferencesRequest) error {
	for k, v := range req.Values {
		if err := s.deps.Prefs.Set(ctx, tenantID, k, v); err != nil {
			if errors.Is(err, coresettings.ErrUnknownKey) {
				return httpx.ErrValidation.WithDetails(map[string]any{k: "unknown key"})
			}
			return httpx.ErrValidation.WithDetails(map[string]any{k: err.Error()})
		}
	}
	return s.advance(ctx, tenantID, StepPreferences)
}

// SaveTerminal registers a POS terminal (step 8).
func (s *Service) SaveTerminal(ctx context.Context, tenantID uint64, req terminals.CreateRequest) (terminals.Terminal, error) {
	t, err := s.deps.Term.Create(ctx, tenantID, req)
	if err != nil {
		return terminals.Terminal{}, err
	}
	return t, s.advance(ctx, tenantID, StepTerminal)
}

// Complete requires a branch + terminal then stamps onboarded_at.
func (s *Service) Complete(ctx context.Context, tenantID uint64) error {
	brs, err := s.deps.Branch.List(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(brs) == 0 {
		return httpx.NewError(http.StatusUnprocessableEntity, "incomplete",
			"Add a branch before finishing setup.")
	}
	ts, err := s.deps.Term.List(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(ts) == 0 {
		return httpx.NewError(http.StatusUnprocessableEntity, "incomplete",
			"Register a POS terminal before finishing setup.")
	}
	return s.deps.Tenants.CompleteOnboarding(ctx, tenantID)
}
