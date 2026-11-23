-- +goose Up
-- Onboarding + configuration fields on tenants.
ALTER TABLE tenants
    ADD COLUMN business_type   VARCHAR(30)     NULL,
    ADD COLUMN kra_pin         VARCHAR(20)     NULL,
    ADD COLUMN vat_registered  TINYINT(1)      NOT NULL DEFAULT 0,
    ADD COLUMN address         VARCHAR(255)    NULL,
    ADD COLUMN onboarding_step INT             NOT NULL DEFAULT 0,
    ADD COLUMN onboarded_at    DATETIME(3)     NULL;

-- Arbitrary key/value configuration, validated against registered JSON schemas.
CREATE TABLE tenant_settings (
    tenant_id  BIGINT UNSIGNED NOT NULL,
    `key`      VARCHAR(120)    NOT NULL,
    `value`    JSON            NOT NULL,
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (tenant_id, `key`),
    CONSTRAINT fk_tenant_settings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Enabled modules per tenant. Module codes come from profile templates, plans
-- and (Phase 3) the module registry.
CREATE TABLE tenant_modules (
    tenant_id BIGINT UNSIGNED NOT NULL,
    module    VARCHAR(60)     NOT NULL,
    enabled   TINYINT(1)      NOT NULL DEFAULT 1,
    config    JSON            NULL,
    PRIMARY KEY (tenant_id, module),
    CONSTRAINT fk_tenant_modules_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Minimal branches table (full CRUD + shifts land in Phase 12). The first
-- branch is flagged main and is required before a terminal can be registered.
CREATE TABLE branches (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    public_id  CHAR(26)        NOT NULL,
    code       VARCHAR(20)     NOT NULL,
    name       VARCHAR(120)    NOT NULL,
    phone      VARCHAR(40)     NULL,
    location   VARCHAR(190)    NULL,
    is_main    TINYINT(1)      NOT NULL DEFAULT 0,
    status     VARCHAR(20)     NOT NULL DEFAULT 'active', -- active | closed
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_branches_public_id (public_id),
    UNIQUE KEY uq_branches_tenant_code (tenant_id, code),
    CONSTRAINT fk_branches_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- POS terminals. Activation (device binding + codes) lands in Phase 6; a
-- terminal created at onboarding starts as pending.
CREATE TABLE pos_terminals (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    branch_id  BIGINT UNSIGNED NOT NULL,
    public_id  CHAR(26)        NOT NULL,
    name       VARCHAR(120)    NOT NULL,
    status     VARCHAR(20)     NOT NULL DEFAULT 'pending', -- pending | active | disabled
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_pos_terminals_public_id (public_id),
    CONSTRAINT fk_pos_terminals_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_pos_terminals_branch FOREIGN KEY (branch_id) REFERENCES branches (id),
    KEY idx_pos_terminals_tenant (tenant_id, branch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE pos_terminals;
DROP TABLE branches;
DROP TABLE tenant_modules;
DROP TABLE tenant_settings;
ALTER TABLE tenants
    DROP COLUMN business_type,
    DROP COLUMN kra_pin,
    DROP COLUMN vat_registered,
    DROP COLUMN address,
    DROP COLUMN onboarding_step,
    DROP COLUMN onboarded_at;
