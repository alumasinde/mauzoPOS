-- +goose Up
-- Subscription plans offered to tenants at onboarding. modules is the allow-list
-- of module codes the plan unlocks; limits are NULL = unlimited.
CREATE TABLE plans (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id        CHAR(26)        NOT NULL,
    name             VARCHAR(80)     NOT NULL,
    description      VARCHAR(255)    NULL,
    price_kes        DECIMAL(15,2)   NOT NULL DEFAULT 0,
    billing_interval VARCHAR(10)     NOT NULL DEFAULT 'monthly', -- monthly | yearly
    modules          JSON            NOT NULL,                   -- ["pos","sales",...]
    max_branches     INT UNSIGNED    NULL,
    max_users        INT UNSIGNED    NULL,
    is_active        TINYINT(1)      NOT NULL DEFAULT 1,
    sort             INT             NOT NULL DEFAULT 0,
    created_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_plans_public_id (public_id),
    UNIQUE KEY uq_plans_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- One row per plan assignment; the newest non-cancelled row is current.
CREATE TABLE tenant_subscriptions (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id     BIGINT UNSIGNED NOT NULL,
    plan_id       BIGINT UNSIGNED NOT NULL,
    status        VARCHAR(15)     NOT NULL DEFAULT 'trialing', -- trialing | active | past_due | cancelled
    starts_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    trial_ends_at DATETIME(3)     NULL,
    ends_at       DATETIME(3)     NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    CONSTRAINT fk_subscriptions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_subscriptions_plan FOREIGN KEY (plan_id) REFERENCES plans (id),
    KEY idx_subscriptions_tenant (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE tenant_subscriptions;
DROP TABLE plans;
