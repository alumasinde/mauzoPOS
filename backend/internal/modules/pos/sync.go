package pos

import (
	"context"
	"time"
)

// SyncBundle is the full working set a terminal downloads: catalog, pricing,
// promotions, cashiers and tenant settings. The terminal stores it in local
// SQLite so checkout keeps working offline.
type SyncBundle struct {
	ServerTime   time.Time         `json:"server_time"`
	Cursor       time.Time         `json:"cursor"` // pass back as ?since= next sync
	Categories   []SyncCategory    `json:"categories"`
	Units        []SyncUnit        `json:"units"`
	Products     []SyncProduct     `json:"products"`
	Variants     []SyncVariant     `json:"variants"`
	Barcodes     []SyncBarcode     `json:"barcodes"`
	PriceLists   []SyncPriceList   `json:"price_lists"`
	PriceItems   []SyncPriceItem   `json:"price_items"`
	Promotions   []SyncPromotion   `json:"promotions"`
	PromoTargets []SyncPromoTarget `json:"promotion_targets"`
	Cashiers     []Cashier         `json:"cashiers"`
	Customers    []SyncCustomer    `json:"customers"`
	Settings     []SyncSetting     `json:"settings"`
	Modules      []string          `json:"modules"`
}

// Cashier is a sign-in candidate with its permissions (role-derived).
type Cashier struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	RoleName    string   `json:"role_name"`
	PINHash     *string  `json:"pin_hash"`
	Permissions []string `json:"permissions"`
	MaxDiscount *string  `json:"max_discount_percent"`
}

// ModuleLister resolves effective module codes for the tenant.
type ModuleLister interface {
	EnabledModules(ctx context.Context, tenantID uint64) ([]string, error)
}

// Sync assembles the bundle. since filters products incrementally; reference
// data (categories, units, cashiers, pricing) is small and always full.
func (s *Service) Sync(ctx context.Context, d Device, since time.Time, modules ModuleLister) (SyncBundle, error) {
	now := time.Now()
	b := SyncBundle{ServerTime: now, Cursor: now}
	var err error
	if b.Categories, err = s.repo.Categories(ctx, d.TenantID); err != nil {
		return b, err
	}
	if b.Units, err = s.repo.Units(ctx, d.TenantID); err != nil {
		return b, err
	}
	if b.Products, err = s.repo.Products(ctx, d.TenantID, since); err != nil {
		return b, err
	}
	if b.Variants, err = s.repo.Variants(ctx, d.TenantID); err != nil {
		return b, err
	}
	if b.Barcodes, err = s.repo.Barcodes(ctx, d.TenantID); err != nil {
		return b, err
	}
	if b.PriceLists, err = s.repo.PriceLists(ctx, d.TenantID, d.BranchID, now); err != nil {
		return b, err
	}
	if b.PriceItems, err = s.repo.PriceItems(ctx, d.TenantID, d.BranchID, now); err != nil {
		return b, err
	}
	if b.Promotions, err = s.repo.Promotions(ctx, d.TenantID, now); err != nil {
		return b, err
	}
	if b.PromoTargets, err = s.repo.PromoTargets(ctx, d.TenantID, now); err != nil {
		return b, err
	}
	if b.Customers, err = s.repo.Customers(ctx, d.TenantID); err != nil {
		return b, err
	}
	cashiers, err := s.repo.Cashiers(ctx, d.TenantID)
	if err != nil {
		return b, err
	}
	grants, err := s.repo.CashierBranches(ctx, d.TenantID)
	if err != nil {
		return b, err
	}
	perms, err := s.repo.CashierPermissions(ctx, d.TenantID)
	if err != nil {
		return b, err
	}
	allowed := map[uint64]bool{}
	for _, g := range grants {
		if g.BranchID == d.BranchID {
			allowed[g.UserID] = true
		}
	}
	for _, c := range cashiers {
		if !c.AllBranches && !allowed[c.ID] {
			continue // cashier has no access to this branch
		}
		b.Cashiers = append(b.Cashiers, Cashier{
			ID: c.PublicID, Name: c.Name, RoleName: c.RoleName,
			PINHash: c.PINHash, Permissions: perms[c.ID], MaxDiscount: c.MaxDiscount,
		})
	}
	if b.Settings, err = s.repo.TenantSettings(ctx, d.TenantID); err != nil {
		return b, err
	}
	b.Modules, err = modules.EnabledModules(ctx, d.TenantID)
	return b, err
}
