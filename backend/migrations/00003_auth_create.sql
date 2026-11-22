-- +goose Up
-- Unified auth tables for both login contexts. actor_type discriminates:
-- 'platform_user' (SaaS operators) vs 'user' (tenant employees). tenant_id is
-- denormalised onto rows for tenant actors so "revoke all of a tenant's
-- sessions" and suspension enforcement stay cheap.

CREATE TABLE platform_users (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id      CHAR(26)        NOT NULL,
    name           VARCHAR(120)    NOT NULL,
    email          VARCHAR(190)    NOT NULL,
    password_hash  VARCHAR(255)    NOT NULL,
    role           VARCHAR(30)     NOT NULL,                    -- super_admin | support | billing | viewer
    status         VARCHAR(20)     NOT NULL DEFAULT 'active',   -- active | disabled
    totp_secret    VARCHAR(64)     NULL,
    totp_enabled   TINYINT(1)      NOT NULL DEFAULT 0,
    recovery_codes JSON            NULL,                        -- sha256 hashes of unused codes
    last_login_at  DATETIME(3)     NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_platform_users_public_id (public_id),
    UNIQUE KEY uq_platform_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Opaque session tokens; only the SHA-256 of the client token is stored.
-- mfa_verified=0 marks a short-lived pre-MFA ticket session.
CREATE TABLE auth_sessions (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id    CHAR(26)        NOT NULL,
    actor_type   VARCHAR(20)     NOT NULL,                      -- platform_user | user
    actor_id     BIGINT UNSIGNED NOT NULL,
    tenant_id    BIGINT UNSIGNED NULL,
    token_hash   CHAR(64)        NOT NULL,
    mfa_verified TINYINT(1)      NOT NULL DEFAULT 0,
    expires_at   DATETIME(3)     NOT NULL,
    revoked_at   DATETIME(3)     NULL,
    ip_address   VARCHAR(45)     NULL,
    user_agent   VARCHAR(255)    NULL,
    last_used_at DATETIME(3)     NULL,
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_auth_sessions_public_id (public_id),
    UNIQUE KEY uq_auth_sessions_token (token_hash),
    KEY idx_auth_sessions_actor (actor_type, actor_id),
    KEY idx_auth_sessions_tenant (tenant_id),
    KEY idx_auth_sessions_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE auth_login_history (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_type     VARCHAR(20)     NOT NULL,
    actor_id       BIGINT UNSIGNED NULL,                        -- NULL when identity unknown
    tenant_id      BIGINT UNSIGNED NULL,
    identity       VARCHAR(190)    NOT NULL,                    -- email or username attempted
    success        TINYINT(1)      NOT NULL,
    failure_reason VARCHAR(40)     NULL,                        -- bad_password | locked | mfa_failed | disabled | suspended | unknown_identity
    ip_address     VARCHAR(45)     NULL,
    user_agent     VARCHAR(255)    NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_history_actor (actor_type, actor_id, created_at),
    KEY idx_history_identity (actor_type, identity, created_at),
    KEY idx_history_tenant (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE auth_password_resets (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_type VARCHAR(20)     NOT NULL,
    actor_id   BIGINT UNSIGNED NOT NULL,
    tenant_id  BIGINT UNSIGNED NULL,
    token_hash CHAR(64)        NOT NULL,
    expires_at DATETIME(3)     NOT NULL,
    used_at    DATETIME(3)     NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_resets_token (token_hash),
    KEY idx_resets_actor (actor_type, actor_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE auth_password_resets;
DROP TABLE auth_login_history;
DROP TABLE auth_sessions;
DROP TABLE platform_users;
