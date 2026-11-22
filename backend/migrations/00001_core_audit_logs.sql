-- +goose Up
-- Append-only audit trail shared by platform and tenant contexts.
-- tenant_id is NULL for platform-level events.
CREATE TABLE audit_logs (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id     BIGINT UNSIGNED NULL,
    branch_id     BIGINT UNSIGNED NULL,
    actor_type    VARCHAR(20)     NOT NULL,           -- platform_user | user | cashier | terminal | system
    actor_id      BIGINT UNSIGNED NULL,
    supervisor_id BIGINT UNSIGNED NULL,               -- set when a supervisor PIN authorised the action
    action        VARCHAR(100)    NOT NULL,           -- permission-style key, e.g. sales.refund.create
    entity_type   VARCHAR(60)     NULL,
    entity_id     VARCHAR(40)     NULL,
    data          JSON            NULL,               -- before/after or context
    ip_address    VARCHAR(45)     NULL,
    user_agent    VARCHAR(255)    NULL,
    request_id    VARCHAR(64)     NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_audit_tenant_created (tenant_id, created_at),
    KEY idx_audit_tenant_entity (tenant_id, entity_type, entity_id),
    KEY idx_audit_tenant_action (tenant_id, action, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE audit_logs;
