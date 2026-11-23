-- +goose Up
-- Phase 3: module registry + roles & permissions + branch access.

-- Mirror of the code permission registry (core/authz), synced at boot. Kept
-- in the DB so role_permissions has referential integrity and reports/UI can
-- join descriptions. Removing a permission from code cascades its grants away.
CREATE TABLE permissions (
    code        VARCHAR(100)    NOT NULL,
    scope       VARCHAR(10)     NOT NULL,              -- platform | tenant
    module      VARCHAR(60)     NULL,                  -- NULL = core (always available)
    description VARCHAR(255)    NOT NULL DEFAULT '',
    is_sensitive TINYINT(1)     NOT NULL DEFAULT 0,
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (code),
    KEY idx_permissions_scope_module (scope, module)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Platform-wide module switch. Module definitions (deps, config schema, nav)
-- live in code (core/modules); this row lets operators withdraw a module from
-- every tenant without a deploy. Synced at boot (missing codes inserted).
CREATE TABLE modules (
    code       VARCHAR(60)     NOT NULL,
    is_active  TINYINT(1)      NOT NULL DEFAULT 1,
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Who enabled a tenant module: the tenant (plan-gated) or a platform override.
ALTER TABLE tenant_modules
    ADD COLUMN source     VARCHAR(10) NOT NULL DEFAULT 'tenant',
    ADD COLUMN updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3);

-- Tenant roles. System roles come from core/authz templates; customized=0
-- means the role still follows its template (re-expanded at boot).
CREATE TABLE roles (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id            BIGINT UNSIGNED NOT NULL,
    public_id            CHAR(26)        NOT NULL,
    code                 VARCHAR(40)     NOT NULL,
    name                 VARCHAR(80)     NOT NULL,
    description          VARCHAR(255)    NULL,
    is_system            TINYINT(1)      NOT NULL DEFAULT 0,
    is_locked            TINYINT(1)      NOT NULL DEFAULT 0,
    customized           TINYINT(1)      NOT NULL DEFAULT 0,
    max_discount_percent DECIMAL(5,2)    NULL,
    max_refund_amount    DECIMAL(15,2)   NULL,
    max_cash_out_amount  DECIMAL(15,2)   NULL,
    created_at           DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at           DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_roles_public_id (public_id),
    UNIQUE KEY uq_roles_tenant_code (tenant_id, code),
    UNIQUE KEY uq_roles_tenant_id (tenant_id, id),
    CONSTRAINT fk_roles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE role_permissions (
    role_id    BIGINT UNSIGNED NOT NULL,
    permission VARCHAR(100)    NOT NULL,
    PRIMARY KEY (role_id, permission),
    KEY idx_role_permissions_permission (permission),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission) REFERENCES permissions (code)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- users.role_id is backfilled (and made NOT NULL) by 00010; all_branches=1
-- keeps every existing employee's current access.
ALTER TABLE users
    ADD COLUMN role_id      BIGINT UNSIGNED NULL AFTER pin_hash,
    ADD COLUMN all_branches TINYINT(1)      NOT NULL DEFAULT 1 AFTER role_id,
    ADD UNIQUE KEY uq_users_tenant_id (tenant_id, id);

-- Branch access for employees with all_branches = 0. Composite FKs make a
-- cross-tenant user/branch pairing impossible.
CREATE TABLE user_branches (
    tenant_id BIGINT UNSIGNED NOT NULL,
    user_id   BIGINT UNSIGNED NOT NULL,
    branch_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id, branch_id),
    KEY idx_user_branches_tenant_user (tenant_id, user_id),
    KEY idx_user_branches_tenant_branch (tenant_id, branch_id),
    CONSTRAINT fk_user_branches_user FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_user_branches_branch FOREIGN KEY (tenant_id, branch_id) REFERENCES branches (tenant_id, id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE user_branches;
ALTER TABLE users
    DROP KEY uq_users_tenant_id,
    DROP COLUMN all_branches,
    DROP COLUMN role_id;
DROP TABLE role_permissions;
DROP TABLE roles;
ALTER TABLE tenant_modules
    DROP COLUMN updated_at,
    DROP COLUMN source;
DROP TABLE modules;
DROP TABLE permissions;
