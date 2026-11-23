package catalog

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// TaxonomyService owns categories, brands and units business rules.
type TaxonomyService struct {
	repo *TaxonomyRepo
}

// NewTaxonomyService builds a TaxonomyService.
func NewTaxonomyService(repo *TaxonomyRepo) *TaxonomyService {
	return &TaxonomyService{repo: repo}
}

// CategoryByPublic resolves a public id to the internal id (for filters).
func (s *TaxonomyService) CategoryByPublic(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	c, err := s.repo.CategoryByPublicID(ctx, tenantID, publicID)
	return c.ID, err
}

// BrandByPublic resolves a public id to the internal id (for filters).
func (s *TaxonomyService) BrandByPublic(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	b, err := s.repo.BrandByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return b.ID, nil
}

// CategoryPublicID resolves an internal category id to its public id.
func (s *TaxonomyService) CategoryPublicID(ctx context.Context, tenantID, id uint64) (string, error) {
	c, err := s.repo.CategoryByID(ctx, tenantID, id)
	return c.PublicID, err
}

// UnitCode resolves an internal unit id to its code.
func (s *TaxonomyService) UnitCode(ctx context.Context, tenantID, id uint64) (string, error) {
	u, err := s.repo.UnitByID(ctx, tenantID, id)
	return u.Code, err
}

// ListCategories returns the tenant's categories (flat list; the UI builds the tree).
func (s *TaxonomyService) ListCategories(ctx context.Context, tenantID uint64) ([]Category, error) {
	return s.repo.ListCategories(ctx, tenantID)
}

// CreateCategory adds a category, optionally under a parent.
func (s *TaxonomyService) CreateCategory(ctx context.Context, tenantID uint64, req CategoryRequest) (Category, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Category{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	var parentID *uint64
	if req.ParentID != nil && *req.ParentID != "" {
		parent, err := s.repo.CategoryByPublicID(ctx, tenantID, *req.ParentID)
		if db.IsNotFound(err) {
			return Category{}, ErrNotFound
		}
		if err != nil {
			return Category{}, err
		}
		parentID = &parent.ID
	}
	c := Category{TenantID: tenantID, PublicID: security.NewPublicID(),
		ParentID: parentID, Name: name, Sort: req.Sort}
	if err := s.repo.CreateCategory(ctx, &c); err != nil {
		if db.IsDuplicate(err) {
			return Category{}, ErrNameTaken
		}
		return Category{}, err
	}
	return c, nil
}

// UpdateCategory renames, re-sorts or reparents a category.
func (s *TaxonomyService) UpdateCategory(ctx context.Context, tenantID uint64, publicID string, req CategoryRequest) (Category, error) {
	c, err := s.repo.CategoryByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name != "" {
		c.Name = name
	}
	c.Sort = req.Sort
	if req.ParentID != nil {
		if *req.ParentID == "" {
			c.ParentID = nil // promote to root
		} else {
			parent, err := s.repo.CategoryByPublicID(ctx, tenantID, *req.ParentID)
			if db.IsNotFound(err) {
				return Category{}, ErrNotFound
			}
			if err != nil {
				return Category{}, err
			}
			if parent.ID == c.ID {
				return Category{}, httpx.ErrValidation.WithDetails(map[string]any{"parent_id": "cannot be its own parent"})
			}
			c.ParentID = &parent.ID
		}
	}
	if err := s.repo.UpdateCategory(ctx, &c); err != nil {
		if db.IsDuplicate(err) {
			return Category{}, ErrNameTaken
		}
		return Category{}, err
	}
	return c, nil
}

// DeleteCategory removes an empty, unused category.
func (s *TaxonomyService) DeleteCategory(ctx context.Context, tenantID uint64, publicID string) error {
	c, err := s.repo.CategoryByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	kids, err := s.repo.CategoryHasChildren(ctx, c.ID)
	if err != nil {
		return err
	}
	if kids {
		return ErrCategoryInUse
	}
	used, err := s.repo.CategoryInUse(ctx, c.ID)
	if err != nil {
		return err
	}
	if used {
		return ErrCategoryInUse
	}
	return s.repo.DeleteCategory(ctx, c.ID)
}

// ListBrands returns the tenant's brands.
func (s *TaxonomyService) ListBrands(ctx context.Context, tenantID uint64) ([]Brand, error) {
	return s.repo.ListBrands(ctx, tenantID)
}

// CreateBrand adds a brand.
func (s *TaxonomyService) CreateBrand(ctx context.Context, tenantID uint64, req BrandRequest) (Brand, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Brand{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	b := Brand{TenantID: tenantID, PublicID: security.NewPublicID(), Name: name}
	if err := s.repo.CreateBrand(ctx, &b); err != nil {
		if db.IsDuplicate(err) {
			return Brand{}, ErrNameTaken
		}
		return Brand{}, err
	}
	return b, nil
}

// UpdateBrand renames a brand.
func (s *TaxonomyService) UpdateBrand(ctx context.Context, tenantID uint64, publicID string, req BrandRequest) (Brand, error) {
	b, err := s.repo.BrandByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return Brand{}, ErrNotFound
	}
	if err != nil {
		return Brand{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name != "" {
		b.Name = name
	}
	if err := s.repo.UpdateBrand(ctx, &b); err != nil {
		if db.IsDuplicate(err) {
			return Brand{}, ErrNameTaken
		}
		return Brand{}, err
	}
	return b, nil
}

// DeleteBrand removes an unused brand.
func (s *TaxonomyService) DeleteBrand(ctx context.Context, tenantID uint64, publicID string) error {
	b, err := s.repo.BrandByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	used, err := s.repo.BrandInUse(ctx, b.ID)
	if err != nil {
		return err
	}
	if used {
		return ErrCategoryInUse
	}
	return s.repo.DeleteBrand(ctx, b.ID)
}

// ListUnits returns built-in plus tenant units.
func (s *TaxonomyService) ListUnits(ctx context.Context, tenantID uint64) ([]Unit, error) {
	return s.repo.ListUnits(ctx, tenantID)
}

// CreateUnit adds a tenant unit. An empty BaseCode makes the unit its kind's
// base; otherwise Factor is the multiplier to the named base unit.
func (s *TaxonomyService) CreateUnit(ctx context.Context, tenantID uint64, req UnitRequest) (Unit, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return Unit{}, httpx.ErrValidation.WithDetails(map[string]any{"code": "required", "name": "required"})
	}
	switch req.Kind {
	case UnitKindCount, UnitKindMass, UnitKindVolume, UnitKindLength, UnitKindTime:
	default:
		return Unit{}, httpx.ErrValidation.WithDetails(map[string]any{"kind": "must be count, mass, volume, length or time"})
	}
	u := Unit{TenantID: &tenantID, Code: code, Name: name, Kind: req.Kind}
	if req.BaseCode == "" {
		one := decimal.NewFromInt(1)
		u.Factor = &one
	} else {
		base, err := s.repo.UnitByCode(ctx, tenantID, req.BaseCode)
		if db.IsNotFound(err) {
			return Unit{}, httpx.ErrValidation.WithDetails(map[string]any{"base_code": "unknown unit"})
		}
		if err != nil {
			return Unit{}, err
		}
		if base.Kind != u.Kind {
			return Unit{}, httpx.ErrValidation.WithDetails(map[string]any{"base_code": "base unit has a different kind"})
		}
		f, err := decimal.NewFromString(req.Factor)
		if err != nil || !f.IsPositive() {
			return Unit{}, httpx.ErrValidation.WithDetails(map[string]any{"factor": "must be a positive number"})
		}
		u.BaseUnitID = &base.ID
		u.Factor = &f
	}
	if err := s.repo.CreateUnit(ctx, &u); err != nil {
		if db.IsDuplicate(err) {
			return Unit{}, ErrNameTaken
		}
		return Unit{}, err
	}
	return u, nil
}

// Convert returns qty expressed in to unit, both within the same kind.
func (s *TaxonomyService) Convert(ctx context.Context, tenantID uint64, qty decimal.Decimal, fromCode, toCode string) (decimal.Decimal, error) {
	from, err := s.repo.UnitByCode(ctx, tenantID, fromCode)
	if err != nil {
		return decimal.Zero, err
	}
	to, err := s.repo.UnitByCode(ctx, tenantID, toCode)
	if err != nil {
		return decimal.Zero, err
	}
	if from.Kind != to.Kind {
		return decimal.Zero, ErrBadUnit
	}
	if from.Factor == nil || to.Factor == nil {
		return decimal.Zero, ErrBadUnit
	}
	base := qty.Mul(*from.Factor)
	return base.Div(*to.Factor), nil
}
