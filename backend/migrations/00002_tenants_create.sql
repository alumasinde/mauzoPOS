-- +goose Up
-- Tenants are the businesses that use the POS. Everything tenant-owned hangs
-- off tenants.id. Business profile and subscription fields arrive in Phase 2.
CREATE TABLE tenants (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id      CHAR(26)        NOT NULL,
    name           VARCHAR(160)    NOT NULL,
    slug           VARCHAR(60)     NOT NULL,                  -- sign-in company code
    email          VARCHAR(190)    NULL,
    phone          VARCHAR(40)     NULL,
    status         VARCHAR(20)     NOT NULL DEFAULT 'active', -- active | suspended
    suspended_at   DATETIME(3)     NULL,
    suspend_reason VARCHAR(255)    NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_tenants_public_id (public_id),
    UNIQUE KEY uq_tenants_slug (slug),
    KEY idx_tenants_status (status),
    KEY idx_tenants_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE tenants;
