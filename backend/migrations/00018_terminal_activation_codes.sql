-- +goose Up
-- Activation codes expire (a code read off a screen must not stay valid
-- forever) and record when they were issued, so the back office can show
-- "code waiting — expires 14:05".
ALTER TABLE pos_terminals
  ADD COLUMN activation_code_expires_at DATETIME(3) NULL AFTER activation_code_hash,
  ADD COLUMN activation_code_issued_at  DATETIME(3) NULL AFTER activation_code_expires_at;

-- +goose Down
ALTER TABLE pos_terminals
  DROP COLUMN activation_code_issued_at,
  DROP COLUMN activation_code_expires_at;
