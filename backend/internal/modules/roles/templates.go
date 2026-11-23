package roles

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/security"
)

func nullDec(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

// templatePermissions expands a template's grants against the registry.
// Locked roles hold everything implicitly and store no rows.
func templatePermissions(t authz.RoleTemplate) []string {
	if t.Locked {
		return nil
	}
	return authz.Expand(authz.ScopeTenant, t.Grants)
}

// ProvisionTenant implements tenants.Provisioner: creates any missing system
// role for the tenant and brings template-following roles up to date.
func (s *Service) ProvisionTenant(ctx context.Context, tenantID uint64) error {
	for _, t := range authz.RoleTemplates {
		desc := t.Description
		role := Role{
			TenantID: tenantID, PublicID: security.NewPublicID(), Code: t.Code, Name: t.Name,
			Description: &desc, IsSystem: true, IsLocked: t.Locked,
			MaxDiscountPercent: nullDec(t.MaxDiscountPercent),
			MaxRefundAmount:    nullDec(t.MaxRefundAmount),
			MaxCashOutAmount:   nullDec(t.MaxCashOutAmount),
		}
		if _, err := s.repo.Insert(ctx, &role); err != nil {
			return err
		}
	}
	return s.syncTemplates(ctx, tenantID)
}

// SyncTemplates re-applies templates to every system role that the tenant has
// not customised, so permissions added by later releases reach them. Writes
// only when the grant set differs.
func (s *Service) SyncTemplates(ctx context.Context) error { return s.syncTemplates(ctx, 0) }

func (s *Service) syncTemplates(ctx context.Context, tenantID uint64) error {
	rows, err := s.repo.templateRoles(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		t, ok := authz.Template(row.Code)
		if !ok {
			continue
		}
		want := templatePermissions(t)
		have, err := s.repo.Permissions(ctx, row.ID)
		if err != nil {
			return err
		}
		slices.Sort(have) // DB collation order may differ from byte order
		if slices.Equal(want, have) {
			continue
		}
		err = s.repo.InTx(ctx, func(r *Repo) error { return r.SetPermissions(ctx, row.ID, want) })
		if err != nil {
			return err
		}
		s.access.InvalidateRole(row.ID)
	}
	return nil
}
