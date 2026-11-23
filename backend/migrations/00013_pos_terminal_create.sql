-- +goose Up
-- Phase 6: terminal device identity + held sales.

ALTER TABLE pos_terminals
  ADD COLUMN activation_code_hash CHAR(64)     NULL AFTER status,
  ADD COLUMN device_name          VARCHAR(120) NULL AFTER activation_code_hash,
  ADD COLUMN device_id            VARCHAR(64)  NULL AFTER device_name,
  ADD COLUMN device_token_hash    CHAR(64)     NULL AFTER device_id,
  ADD COLUMN activated_at         DATETIME(3)  NULL AFTER device_token_hash,
  ADD COLUMN last_seen_at         DATETIME(3)  NULL AFTER activated_at,
  ADD UNIQUE KEY uq_pos_terminals_device_token (device_token_hash),
  ADD UNIQUE KEY uq_pos_terminals_activation (activation_code_hash);

CREATE TABLE held_sales (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    branch_id   BIGINT UNSIGNED NOT NULL,
    terminal_id BIGINT UNSIGNED NOT NULL,
    user_id     BIGINT UNSIGNED NOT NULL,
    public_id   CHAR(26)        NOT NULL,
    label       VARCHAR(80)     NOT NULL DEFAULT '',
    cart        JSON            NOT NULL, -- lines as held: product/variant/qty/price/discount
    status      VARCHAR(20)     NOT NULL DEFAULT 'held', -- held | recalled | cancelled
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_held_sales_public_id (public_id),
    KEY idx_held_sales_branch (tenant_id, branch_id, status),
    CONSTRAINT fk_held_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_held_branch   FOREIGN KEY (branch_id)   REFERENCES branches (id),
    CONSTRAINT fk_held_terminal FOREIGN KEY (terminal_id) REFERENCES pos_terminals (id),
    CONSTRAINT fk_held_user     FOREIGN KEY (user_id)     REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE held_sales;
ALTER TABLE pos_terminals
  DROP KEY uq_pos_terminals_device_token,
  DROP KEY uq_pos_terminals_activation,
  DROP COLUMN activation_code_hash,
  DROP COLUMN device_name,
  DROP COLUMN device_id,
  DROP COLUMN device_token_hash,
  DROP COLUMN activated_at,
  DROP COLUMN last_seen_at;
