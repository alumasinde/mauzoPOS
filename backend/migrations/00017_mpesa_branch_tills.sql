-- +goose Up
-- Which branch an M-Pesa till / store number belongs to. A C2B confirmation
-- carries the business short code it was paid to; mapping it to a branch
-- lets only that branch's tills see the payment. Unmapped codes (one till
-- number for the whole business) stay visible to every branch.
CREATE TABLE mpesa_branch_tills (
    tenant_id  BIGINT UNSIGNED NOT NULL,
    short_code VARCHAR(20)     NOT NULL,
    branch_id  BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (tenant_id, short_code),
    KEY ix_mbt_branch (tenant_id, branch_id),
    CONSTRAINT fk_mbt_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_mbt_branch FOREIGN KEY (tenant_id, branch_id) REFERENCES branches (tenant_id, id)
);

-- The till feed query uses ix_mpesa_txn_feed from 00016.

-- +goose Down
DROP TABLE mpesa_branch_tills;
