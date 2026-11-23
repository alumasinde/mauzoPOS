-- +goose Up
-- Phase 4: products, categories & barcodes.

-- Category tree (departments = root categories). Root rows have parent_id NULL;
-- name uniqueness among siblings is enforced in the service layer because
-- MySQL treats NULLs as distinct in unique indexes.
CREATE TABLE categories (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    public_id  CHAR(26)        NOT NULL,
    parent_id  BIGINT UNSIGNED NULL,
    name       VARCHAR(120)    NOT NULL,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_categories_pub (tenant_id, public_id),
    UNIQUE KEY uq_categories_name (tenant_id, parent_id, name),
    KEY ix_categories_parent (tenant_id, parent_id),
    CONSTRAINT fk_categories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE brands (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    public_id  CHAR(26)        NOT NULL,
    name       VARCHAR(120)    NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_brands_pub (tenant_id, public_id),
    UNIQUE KEY uq_brands_name (tenant_id, name),
    CONSTRAINT fk_brands_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Units of measure. tenant_id NULL rows are shared built-ins; tenants add their
-- own. Conversions are expressed as factor-to-base within a kind:
-- qty_in_base = qty * factor (kg base of mass, litre base of volume, piece of
-- count). base_unit_id links a unit to its kind's base.
CREATE TABLE units (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id    BIGINT UNSIGNED NULL,
    code         VARCHAR(30)     NOT NULL,
    name         VARCHAR(60)     NOT NULL,
    kind         ENUM('count','mass','volume','length','time') NOT NULL,
    base_unit_id BIGINT UNSIGNED NULL,
    factor       DECIMAL(24,8)   NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_units_code (tenant_id, code),
    CONSTRAINT fk_units_base FOREIGN KEY (base_unit_id) REFERENCES units (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Built-in units shared by every tenant (tenant_id NULL).
INSERT INTO units (tenant_id, code, name, kind, base_unit_id, factor) VALUES
    (NULL, 'pc',    'Piece',      'count',  NULL, 1),
    (NULL, 'dozen', 'Dozen',      'count',  NULL, 12),
    (NULL, 'kg',    'Kilogram',   'mass',   NULL, 1),
    (NULL, 'g',     'Gram',       'mass',   NULL, 0.001),
    (NULL, 'l',     'Litre',      'volume', NULL, 1),
    (NULL, 'ml',    'Millilitre', 'volume', NULL, 0.001),
    (NULL, 'm',     'Metre',      'length', NULL, 1),
    (NULL, 'cm',    'Centimetre', 'length', NULL, 0.01);

UPDATE units SET base_unit_id = (SELECT id FROM (SELECT id FROM units WHERE code = 'pc') t)    WHERE tenant_id IS NULL AND kind = 'count'  AND code <> 'pc';
UPDATE units SET base_unit_id = (SELECT id FROM (SELECT id FROM units WHERE code = 'kg') t)    WHERE tenant_id IS NULL AND kind = 'mass'   AND code <> 'kg';
UPDATE units SET base_unit_id = (SELECT id FROM (SELECT id FROM units WHERE code = 'l')  t)    WHERE tenant_id IS NULL AND kind = 'volume' AND code <> 'l';
UPDATE units SET base_unit_id = (SELECT id FROM (SELECT id FROM units WHERE code = 'cm') t)    WHERE tenant_id IS NULL AND code = 'cm';

CREATE TABLE products (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    public_id   CHAR(26)        NOT NULL,
    sku         VARCHAR(64)     NOT NULL,
    name        VARCHAR(180)    NOT NULL,
    description TEXT            NULL,
    kind        ENUM('standard','service','packaged','composite') NOT NULL DEFAULT 'standard',
    category_id BIGINT UNSIGNED NULL,
    brand_id    BIGINT UNSIGNED NULL,
    unit_id     BIGINT UNSIGNED NULL,
    -- 'weight' products are variable-weight: price is per unit (e.g. per kg)
    -- and the POS expects a measured quantity or a scale barcode.
    sell_by     ENUM('unit','weight') NOT NULL DEFAULT 'unit',
    price       DECIMAL(15,2)   NOT NULL DEFAULT 0,
    cost        DECIMAL(15,2)   NULL,
    vat_rate    DECIMAL(5,2)    NULL,
    deposit     DECIMAL(15,2)   NULL,
    track_stock TINYINT(1)      NOT NULL DEFAULT 1,
    attributes  JSON            NULL,
    status      ENUM('active','archived') NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_products_pub (tenant_id, public_id),
    UNIQUE KEY uq_products_sku (tenant_id, sku),
    KEY ix_products_name (tenant_id, name),
    KEY ix_products_category (tenant_id, category_id),
    CONSTRAINT fk_products_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories (id),
    CONSTRAINT fk_products_brand    FOREIGN KEY (brand_id)    REFERENCES brands (id),
    CONSTRAINT fk_products_unit     FOREIGN KEY (unit_id)     REFERENCES units (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE product_variants (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    product_id BIGINT UNSIGNED NOT NULL,
    public_id  CHAR(26)        NOT NULL,
    sku        VARCHAR(64)     NULL,
    name       VARCHAR(120)    NOT NULL,
    price      DECIMAL(15,2)   NULL,   -- NULL inherits the parent price
    attributes JSON            NULL,   -- e.g. {"size":"L","colour":"red"}
    PRIMARY KEY (id),
    UNIQUE KEY uq_variants_pub (tenant_id, public_id),
    UNIQUE KEY uq_variants_sku (tenant_id, sku),
    KEY ix_variants_product (product_id),
    CONSTRAINT fk_variants_tenant  FOREIGN KEY (tenant_id)  REFERENCES tenants (id),
    CONSTRAINT fk_variants_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Every scannable code for a product or variant. pack_qty > 1 means one scan
-- moves that many sell units (pack-size barcodes).
CREATE TABLE product_barcodes (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    product_id BIGINT UNSIGNED NOT NULL,
    variant_id BIGINT UNSIGNED NULL,
    barcode    VARCHAR(64)     NOT NULL,
    pack_qty   DECIMAL(15,3)   NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_barcodes_code (tenant_id, barcode),
    KEY ix_barcodes_product (product_id),
    CONSTRAINT fk_barcodes_tenant  FOREIGN KEY (tenant_id)  REFERENCES tenants (id),
    CONSTRAINT fk_barcodes_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE CASCADE,
    CONSTRAINT fk_barcodes_variant FOREIGN KEY (variant_id) REFERENCES product_variants (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Composite products are assembled from other products (a breakfast platter,
-- a gift hamper). Selling one deducts qty of each component in Phase 5.
CREATE TABLE product_components (
    product_id   BIGINT UNSIGNED NOT NULL,
    component_id BIGINT UNSIGNED NOT NULL,
    qty          DECIMAL(15,3)   NOT NULL,
    PRIMARY KEY (product_id, component_id),
    CONSTRAINT fk_components_product   FOREIGN KEY (product_id)   REFERENCES products (id) ON DELETE CASCADE,
    CONSTRAINT fk_components_component FOREIGN KEY (component_id) REFERENCES products (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Product images. `path` is opaque to the DB — the service's ImageStore
-- interface decides where bytes live (local disk now, object storage later).
CREATE TABLE product_images (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id BIGINT UNSIGNED NOT NULL,
    path       VARCHAR(255)    NOT NULL,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_images_product (product_id),
    CONSTRAINT fk_images_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_components;
DROP TABLE IF EXISTS product_barcodes;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS units;
DROP TABLE IF EXISTS brands;
DROP TABLE IF EXISTS categories;
