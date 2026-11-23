package businesstypes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/modules"
	"mauzo/internal/core/security"
	"mauzo/internal/core/settings"
)

// TenantWriter sets the business_type column on a tenant — implemented by
// modules/tenants so this module never writes another module's table.
type TenantWriter interface {
	SetBusinessType(ctx context.Context, tenantID uint64, code string) error
}

// ModuleEnabler enables modules (with dependencies) inside a caller's
// transaction — implemented by modules/registry, which owns tenant_modules.
type ModuleEnabler interface {
	EnableAll(ctx context.Context, q db.Querier, tenantID uint64, codes []string) error
	Invalidate(tenantID uint64)
}

// Service owns business-type and profile business rules plus the profile
// engine.
type Service struct {
	repo     *Repo
	settings *settings.Store
	tenants  TenantWriter
	modules  ModuleEnabler
}

// NewService builds a Service.
func NewService(repo *Repo, st *settings.Store, tw TenantWriter, me ModuleEnabler) *Service {
	return &Service{repo: repo, settings: st, tenants: tw, modules: me}
}

// ErrProfileNotFound is returned for unknown profile public ids.
var ErrProfileNotFound = httpx.NewError(404, "profile_not_found", "The business profile was not found.")

// ListTypes returns all business types.
func (s *Service) ListTypes(ctx context.Context) ([]BusinessType, error) {
	return s.repo.ListTypes(ctx)
}

// ListProfiles returns templates, optionally for one type ("" = all).
func (s *Service) ListProfiles(ctx context.Context, typeCode string) ([]Profile, error) {
	var typeID *uint64
	if typeCode != "" {
		t, err := s.repo.TypeByCode(ctx, typeCode)
		if db.IsNotFound(err) {
			return nil, ErrProfileNotFound
		}
		if err != nil {
			return nil, err
		}
		typeID = &t.ID
	}
	return s.repo.ListProfiles(ctx, typeID)
}

// Profile fetches and unmarshals one template.
func (s *Service) Profile(ctx context.Context, publicID string) (Profile, error) {
	p, err := s.repo.ProfileByPublicID(ctx, publicID)
	if db.IsNotFound(err) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, err // real DB failures surface as 500, not 404
	}
	return p, p.ParseDefinition()
}

// CreateProfile validates and inserts a template.
func (s *Service) CreateProfile(ctx context.Context, req ProfileRequest) (Profile, error) {
	t, err := s.repo.TypeByCode(ctx, req.BusinessType)
	if db.IsNotFound(err) {
		return Profile{}, httpx.ErrValidation.WithDetails(map[string]any{"business_type": "unknown"})
	}
	if err != nil {
		return Profile{}, err
	}
	if err := req.Definition.Validate(); err != nil {
		return Profile{}, err
	}
	p := Profile{
		PublicID:       security.NewPublicID(),
		BusinessTypeID: t.ID,
		Name:           strings.TrimSpace(req.Name),
		IsDefault:      req.IsDefault,
		TypeCode:       t.Code,
	}
	if p.Name == "" {
		return Profile{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	p.DefinitionJSON, err = json.Marshal(req.Definition)
	if err != nil {
		return Profile{}, err
	}
	if err := s.saveProfile(ctx, &p, true); err != nil {
		return Profile{}, err
	}
	return p, p.ParseDefinition()
}

// UpdateProfile patches a template.
func (s *Service) UpdateProfile(ctx context.Context, publicID string, req ProfileRequest) (Profile, error) {
	p, err := s.Profile(ctx, publicID)
	if err != nil {
		return Profile{}, err
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		p.Name = name
	}
	p.IsDefault = req.IsDefault
	if err := req.Definition.Validate(); err != nil {
		return Profile{}, err
	}
	if p.DefinitionJSON, err = json.Marshal(req.Definition); err != nil {
		return Profile{}, err
	}
	if err := s.saveProfile(ctx, &p, false); err != nil {
		return Profile{}, err
	}
	return p, p.ParseDefinition()
}

// saveProfile inserts or updates p and, when it is the default, clears the
// flag on its siblings — in one transaction so a type never has two defaults.
func (s *Service) saveProfile(ctx context.Context, p *Profile, create bool) error {
	return s.repo.InTx(ctx, func(_ *sqlx.Tx, r *Repo) error {
		var err error
		if create {
			err = r.CreateProfile(ctx, p)
		} else {
			err = r.UpdateProfile(ctx, p)
		}
		if err != nil || !p.IsDefault {
			return err
		}
		return r.ClearDefault(ctx, p.BusinessTypeID, p.ID)
	})
}

// Apply seeds a tenant from a profile: enables the profile modules and writes
// settings defaults in one transaction, then records the business type. Every
// step is idempotent, so a retry after a failure completes the job.
func (s *Service) Apply(ctx context.Context, tenantID uint64, profilePublicID string) (Profile, error) {
	p, err := s.Profile(ctx, profilePublicID)
	if err != nil {
		return Profile{}, err
	}
	defaults, err := profileDefaults(p.Definition)
	if err != nil {
		return Profile{}, err
	}
	err = s.repo.InTx(ctx, func(tx *sqlx.Tx, r *Repo) error {
		if err := s.modules.EnableAll(ctx, tx, tenantID, p.Definition.Modules); err != nil {
			return err
		}
		if s.settings == nil {
			return nil
		}
		return s.settings.WithQuerier(tx).SetDefaults(ctx, tenantID, defaults)
	})
	if err != nil {
		return Profile{}, err
	}
	s.modules.Invalidate(tenantID)
	// The tenants module owns its table, so this write is outside the
	// transaction. It goes last: business_type marks the profile as applied.
	if s.tenants != nil {
		if err := s.tenants.SetBusinessType(ctx, tenantID, p.TypeCode); err != nil {
			return Profile{}, err
		}
	}
	return p, nil
}

// profileDefaults builds the settings defaults a profile writes on a tenant.
func profileDefaults(d ProfileDefinition) (map[string]json.RawMessage, error) {
	defaults := map[string]json.RawMessage{}
	for k, v := range d.Settings {
		defaults[k] = v
	}
	add := func(key string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		defaults[key] = b
		return nil
	}
	if len(d.Categories) > 0 {
		if err := add("catalog.default_categories", d.Categories); err != nil {
			return nil, err
		}
	}
	if d.ReceiptTemplate != "" {
		if err := add("profile.receipt_template", d.ReceiptTemplate); err != nil {
			return nil, err
		}
	}
	if len(d.PosLayout) > 0 {
		if err := add("profile.pos_layout", d.PosLayout); err != nil {
			return nil, err
		}
	}
	return defaults, nil
}

// ParseDefinition unmarshals the stored JSON column.
func (p *Profile) ParseDefinition() error {
	if len(p.DefinitionJSON) == 0 {
		return nil
	}
	return json.Unmarshal(p.DefinitionJSON, &p.Definition)
}

// Validate rejects unknown module codes.
func (d ProfileDefinition) Validate() error {
	for _, m := range d.Modules {
		if !modules.Known(m) {
			return httpx.ErrValidation.WithDetails(map[string]any{"modules": "unknown module " + m})
		}
	}
	for k := range d.Settings {
		if err := settings.Validate(k, d.Settings[k]); err != nil {
			if errors.Is(err, settings.ErrUnknownKey) {
				return httpx.ErrValidation.WithDetails(map[string]any{"settings": "unregistered key " + k})
			}
			return httpx.ErrValidation.WithDetails(map[string]any{"settings": k})
		}
	}
	return nil
}
