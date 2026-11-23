package catalog

import (
	"context"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// TaxonomyRepo is the categories/brands/units store.
type TaxonomyRepo struct {
	db *sqlx.DB
}

// NewTaxonomyRepo builds a TaxonomyRepo.
func NewTaxonomyRepo(conn *sqlx.DB) *TaxonomyRepo { return &TaxonomyRepo{db: conn} }

const catCols = `id, tenant_id, public_id, parent_id, name, sort, created_at`

// ListCategories returns a tenant's categories ordered for display.
func (r *TaxonomyRepo) ListCategories(ctx context.Context, tenantID uint64) ([]Category, error) {
	var out []Category
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+catCols+`,
			(SELECT COUNT(*) FROM products p WHERE p.category_id = categories.id) AS product_count
		 FROM categories WHERE tenant_id = ? ORDER BY parent_id IS NOT NULL, parent_id, sort, name`,
		tenantID)
	return out, err
}

// CategoryByPublicID loads one category scoped to a tenant.
func (r *TaxonomyRepo) CategoryByPublicID(ctx context.Context, tenantID uint64, publicID string) (Category, error) {
	var c Category
	err := r.db.GetContext(ctx, &c,
		`SELECT `+catCols+` FROM categories WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return c, err
}

// CategoryHasChildren reports whether any category hangs below id.
func (r *TaxonomyRepo) CategoryHasChildren(ctx context.Context, id uint64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM categories WHERE parent_id = ?`, id)
	return n > 0, err
}

// CategoryInUse reports whether products reference the category.
func (r *TaxonomyRepo) CategoryInUse(ctx context.Context, id uint64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM products WHERE category_id = ?`, id)
	return n > 0, err
}

// CategoryByID loads one category by internal id, scoped to a tenant.
func (r *TaxonomyRepo) CategoryByID(ctx context.Context, tenantID, id uint64) (Category, error) {
	var c Category
	err := r.db.GetContext(ctx, &c,
		`SELECT `+catCols+` FROM categories WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return c, err
}

// CreateCategory inserts a category.
func (r *TaxonomyRepo) CreateCategory(ctx context.Context, c *Category) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (tenant_id, public_id, parent_id, name, sort) VALUES (?, ?, ?, ?, ?)`,
		c.TenantID, c.PublicID, c.ParentID, c.Name, c.Sort)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = db.U64(id)
	fresh, err := r.CategoryByID(ctx, c.TenantID, c.ID)
	if err != nil {
		return err
	}
	*c = fresh
	return nil
}

// UpdateCategory renames/reparents/sorts a category.
func (r *TaxonomyRepo) UpdateCategory(ctx context.Context, c *Category) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories SET parent_id = ?, name = ?, sort = ? WHERE id = ?`,
		c.ParentID, c.Name, c.Sort, c.ID)
	return err
}

// DeleteCategory removes a category row.
func (r *TaxonomyRepo) DeleteCategory(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	return err
}

const brandCols = `id, tenant_id, public_id, name, created_at`

// ListBrands returns a tenant's brands alphabetically.
func (r *TaxonomyRepo) ListBrands(ctx context.Context, tenantID uint64) ([]Brand, error) {
	var out []Brand
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+brandCols+`,
			(SELECT COUNT(*) FROM products p WHERE p.brand_id = brands.id) AS product_count
		 FROM brands WHERE tenant_id = ? ORDER BY name`, tenantID)
	return out, err
}

// BrandByID loads one brand by internal id, scoped to a tenant.
func (r *TaxonomyRepo) BrandByID(ctx context.Context, tenantID, id uint64) (Brand, error) {
	var b Brand
	err := r.db.GetContext(ctx, &b,
		`SELECT `+brandCols+` FROM brands WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return b, err
}

// BrandByName finds a brand by exact name (used by CSV import).
func (r *TaxonomyRepo) BrandByName(ctx context.Context, tenantID uint64, name string) (Brand, error) {
	var b Brand
	err := r.db.GetContext(ctx, &b,
		`SELECT `+brandCols+` FROM brands WHERE tenant_id = ? AND name = ?`, tenantID, name)
	return b, err
}

// BrandByPublicID loads one brand scoped to a tenant.
func (r *TaxonomyRepo) BrandByPublicID(ctx context.Context, tenantID uint64, publicID string) (Brand, error) {
	var b Brand
	err := r.db.GetContext(ctx, &b,
		`SELECT `+brandCols+` FROM brands WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return b, err
}

// CreateBrand inserts a brand.
func (r *TaxonomyRepo) CreateBrand(ctx context.Context, b *Brand) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO brands (tenant_id, public_id, name) VALUES (?, ?, ?)`,
		b.TenantID, b.PublicID, b.Name)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = db.U64(id)
	return nil
}

// UpdateBrand renames a brand.
func (r *TaxonomyRepo) UpdateBrand(ctx context.Context, b *Brand) error {
	_, err := r.db.ExecContext(ctx, `UPDATE brands SET name = ? WHERE id = ?`, b.Name, b.ID)
	return err
}

// BrandInUse reports whether products reference the brand.
func (r *TaxonomyRepo) BrandInUse(ctx context.Context, id uint64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM products WHERE brand_id = ?`, id)
	return n > 0, err
}

// DeleteBrand removes a brand row.
func (r *TaxonomyRepo) DeleteBrand(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM brands WHERE id = ?`, id)
	return err
}

const unitCols = `id, tenant_id, code, name, kind, base_unit_id, factor`

// ListUnits returns built-ins plus the tenant's own units.
func (r *TaxonomyRepo) ListUnits(ctx context.Context, tenantID uint64) ([]Unit, error) {
	var out []Unit
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+unitCols+` FROM units WHERE tenant_id IS NULL OR tenant_id = ? ORDER BY kind, code`,
		tenantID)
	return out, err
}

// UnitByID loads a unit visible to the tenant (own or built-in).
func (r *TaxonomyRepo) UnitByID(ctx context.Context, tenantID, id uint64) (Unit, error) {
	var u Unit
	err := r.db.GetContext(ctx, &u,
		`SELECT `+unitCols+` FROM units WHERE id = ? AND (tenant_id IS NULL OR tenant_id = ?)`,
		id, tenantID)
	return u, err
}

// UnitByCode finds a unit by code for the tenant (own units shadow built-ins).
func (r *TaxonomyRepo) UnitByCode(ctx context.Context, tenantID uint64, code string) (Unit, error) {
	var u Unit
	err := r.db.GetContext(ctx, &u,
		`SELECT `+unitCols+` FROM units WHERE code = ? AND (tenant_id IS NULL OR tenant_id = ?)
		 ORDER BY tenant_id IS NULL LIMIT 1`,
		code, tenantID)
	return u, err
}

// CreateUnit inserts a tenant unit.
func (r *TaxonomyRepo) CreateUnit(ctx context.Context, u *Unit) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO units (tenant_id, code, name, kind, base_unit_id, factor) VALUES (?, ?, ?, ?, ?, ?)`,
		u.TenantID, u.Code, u.Name, u.Kind, u.BaseUnitID, u.Factor)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = db.U64(id)
	return nil
}
