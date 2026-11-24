-- +goose Up
-- Phase 12: cashier shifts — float, cash movements, blind-count close and
-- variance. A shift belongs to one terminal's drawer; one shift per terminal
-- may be open at a time (enforced by uq_shift_open on the open marker).

CREATE TABLE cash_shifts (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT UNSIGNED NOT NULL,
    branch_id       BIGINT UNSIGNED NOT NULL,
    terminal_id     BIGINT UNSIGNED NOT NULL,
    public_id       VARCHAR(30)     NOT NULL,
    opened_by       BIGINT UNSIGNED NOT NULL, -- users.id (cashier)
    closed_by       BIGINT UNSIGNED NULL,     -- NULL when force-closed by back-office? no — reviewer id lives on variance fields
    status          VARCHAR(12)     NOT NULL DEFAULT 'open', -- open | closed
    float_amount    DECIMAL(15,4)   NOT NULL DEFAULT 0,
    opened_at       DATETIME(3)     NOT NULL,
    closed_at       DATETIME(3)     NULL,
    -- Z-report snapshot at close. expected/declared/variance stay NULL on a
    -- force-close from the back office (no blind count was taken).
    expected_amount DECIMAL(15,4)   NULL,
    declared_amount DECIMAL(15,4)   NULL,
    variance        DECIMAL(15,4)   NULL,
    close_note      VARCHAR(255)    NOT NULL DEFAULT '',
    -- Variance review (shifts.variance.approve): pending | approved | flagged.
    variance_status VARCHAR(16)     NOT NULL DEFAULT 'pending',
    variance_note   VARCHAR(255)    NOT NULL DEFAULT '',
    reviewed_by     BIGINT UNSIGNED NULL,
    reviewed_at     DATETIME(3)     NULL,
    open_marker     TINYINT         NULL DEFAULT 1, -- 1 while open, NULL when closed
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_shift_public (tenant_id, public_id),
    UNIQUE KEY uq_shift_open (tenant_id, terminal_id, open_marker), -- one open shift per till
    KEY ix_shifts_branch (tenant_id, branch_id, opened_at),
    KEY ix_shifts_cashier (tenant_id, opened_by),
    KEY ix_shifts_status (tenant_id, status),
    CONSTRAINT fk_shifts_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_shifts_branch   FOREIGN KEY (branch_id)   REFERENCES branches (id),
    CONSTRAINT fk_shifts_terminal FOREIGN KEY (terminal_id) REFERENCES pos_terminals (id),
    CONSTRAINT fk_shifts_opener   FOREIGN KEY (opened_by)   REFERENCES users (id),
    CONSTRAINT fk_shifts_closer   FOREIGN KEY (closed_by)   REFERENCES users (id)
);

-- Manual drawer movements inside a shift: cash added (drop, change fund),
-- cash removed (paid-out, bank drop) and audited no-sale drawer opens.
-- shift_id is nullable because a drawer can be kicked with no shift open.
CREATE TABLE cash_movements (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    shift_id    BIGINT UNSIGNED NULL,
    branch_id   BIGINT UNSIGNED NOT NULL,
    terminal_id BIGINT UNSIGNED NOT NULL,
    public_id   VARCHAR(30)     NOT NULL,
    kind        VARCHAR(12)     NOT NULL, -- cash_in | cash_out | drawer_open
    amount      DECIMAL(15,4)   NOT NULL DEFAULT 0,
    reason      VARCHAR(160)    NOT NULL DEFAULT '',
    created_by  BIGINT UNSIGNED NOT NULL, -- cashier users.id
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_move_public (tenant_id, public_id),
    KEY ix_move_shift (shift_id, kind),
    KEY ix_move_terminal (tenant_id, terminal_id, created_at),
    CONSTRAINT fk_move_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_move_shift    FOREIGN KEY (shift_id)    REFERENCES cash_shifts (id),
    CONSTRAINT fk_move_terminal FOREIGN KEY (terminal_id) REFERENCES pos_terminals (id),
    CONSTRAINT fk_move_cashier  FOREIGN KEY (created_by)  REFERENCES users (id)
);

-- Sales are stamped with the till's open shift at completion; a delayed
-- offline upload falls into the shift window that covered its sold_at.
ALTER TABLE sales
    ADD COLUMN shift_id BIGINT UNSIGNED NULL AFTER cashier_id,
    ADD KEY ix_sales_shift (tenant_id, shift_id),
    ADD CONSTRAINT fk_sales_shift FOREIGN KEY (shift_id) REFERENCES cash_shifts (id);

-- +goose Down
ALTER TABLE sales DROP FOREIGN KEY fk_sales_shift;
ALTER TABLE sales DROP KEY ix_sales_shift;
ALTER TABLE sales DROP COLUMN shift_id;
DROP TABLE cash_movements;
DROP TABLE cash_shifts;
