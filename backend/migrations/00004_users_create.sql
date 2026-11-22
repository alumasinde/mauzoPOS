-- +goose Up
-- Tenant employees. Back-office sign-in uses email/username + password; POS
-- sign-in will use pin_hash (Phase 6). role is an interim label until the
-- roles & permission tables land in Phase 3.
CREATE TABLE users (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id      BIGINT UNSIGNED NOT NULL,
    public_id      CHAR(26)        NOT NULL,
    name           VARCHAR(120)    NOT NULL,
    email          VARCHAR(190)    NULL,
    username       VARCHAR(60)     NULL,
    password_hash  VARCHAR(255)    NULL,   -- NULL for PIN-only cashiers
    pin_hash       VARCHAR(255)    NULL,
    role           VARCHAR(30)     NOT NULL DEFAULT 'cashier', -- owner | admin | manager | supervisor | cashier (interim)
    status         VARCHAR(20)     NOT NULL DEFAULT 'active',  -- active | disabled
    totp_secret    VARCHAR(64)     NULL,
    totp_enabled   TINYINT(1)      NOT NULL DEFAULT 0,
    recovery_codes JSON            NULL,
    last_login_at  DATETIME(3)     NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_public_id (public_id),
    UNIQUE KEY uq_users_tenant_email (tenant_id, email),
    UNIQUE KEY uq_users_tenant_username (tenant_id, username),
    KEY idx_users_tenant_status (tenant_id, status),
    CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE users;
