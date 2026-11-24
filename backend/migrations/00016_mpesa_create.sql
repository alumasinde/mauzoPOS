-- +goose Up
-- Phase 9: M-Pesa (Daraja) — per-tenant credentials, the incoming payment
-- feed, callback audit and reversal tracking.

-- Daraja app credentials, one row per tenant. Secrets are AES-256-GCM blobs —
-- never plaintext, never returned by the API (masked views only). The
-- callback_token makes webhook URLs unguessable: Daraja callbacks carry no
-- signature, so the path secret is the verification boundary alongside the
-- optional source-IP allowlist at the reverse proxy.
CREATE TABLE mpesa_credentials (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id             BIGINT UNSIGNED NOT NULL,
    env                   VARCHAR(12)     NOT NULL DEFAULT 'sandbox', -- sandbox | production
    short_code            VARCHAR(20)     NOT NULL DEFAULT '',
    short_code_kind       VARCHAR(12)     NOT NULL DEFAULT 'till',    -- till (Buy Goods) | paybill
    consumer_key_enc      BLOB            NULL,
    consumer_secret_enc   BLOB            NULL,
    passkey_enc           BLOB            NULL,                       -- STK push password key
    initiator_name        VARCHAR(64)     NOT NULL DEFAULT '',        -- reversals/queries operator
    security_credential_enc BLOB          NULL,                       -- encrypted initiator credential
    callback_token        VARCHAR(64)     NOT NULL,
    c2b_registered_at     DATETIME(3)     NULL,                       -- last successful registurl
    stk_pushes_today      INT UNSIGNED    NOT NULL DEFAULT 0,         -- rate guard, reset by date check
    stk_pushes_on         DATE            NULL,
    created_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_mpesa_cred_tenant (tenant_id),
    CONSTRAINT fk_mpesa_cred_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- The incoming M-Pesa feed. One row per Daraja transaction: STK pushes start
-- as 'initiated' rows keyed by checkout_request_id; C2B confirmations land
-- keyed by TransID. (tenant_id, trans_id) is unique so a re-delivered
-- callback can never double-record money.
CREATE TABLE mpesa_transactions (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id           BIGINT UNSIGNED NOT NULL,
    public_id           VARCHAR(30)     NOT NULL,
    branch_id           BIGINT UNSIGNED NULL,           -- till that initiated it (STK) or NULL
    kind                VARCHAR(12)     NOT NULL,        -- stk | c2b | reversal
    status              VARCHAR(16)     NOT NULL DEFAULT 'initiated',
                        -- initiated | pending | success | failed | cancelled | timeout | reversal_pending | reversed
    trans_id            VARCHAR(40)     NULL, -- MpesaReceiptNumber / TransID, NULL until confirmed
    checkout_request_id VARCHAR(80)     NULL,
    merchant_request_id VARCHAR(80)     NULL,
    phone               VARCHAR(24)     NOT NULL DEFAULT '',
    payer_name          VARCHAR(160)    NOT NULL DEFAULT '',
    amount              DECIMAL(15,4)   NOT NULL,
    account_ref         VARCHAR(64)     NOT NULL DEFAULT '', -- BillRefNumber / AccountReference
    short_code          VARCHAR(20)     NOT NULL DEFAULT '',
    result_code         INT             NULL,
    result_desc         VARCHAR(255)    NOT NULL DEFAULT '',
    allocated_sale_id   BIGINT UNSIGNED NULL,
    allocated_by        BIGINT UNSIGNED NULL,
    allocated_at        DATETIME(3)     NULL,
    trans_time          DATETIME(3)     NULL,             -- payment time reported by M-Pesa
    initiated_by        BIGINT UNSIGNED NULL,             -- cashier who fired the STK push
    last_queried_at     DATETIME(3)     NULL,             -- STK query fallback pacing
    raw                 JSON            NULL,             -- last callback payload
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_mpesa_txn_public (tenant_id, public_id),
    UNIQUE KEY uq_mpesa_txn_trans (tenant_id, trans_id),
    UNIQUE KEY uq_mpesa_txn_checkout (tenant_id, checkout_request_id),
    KEY ix_mpesa_txn_feed (tenant_id, status, allocated_sale_id, created_at),
    KEY ix_mpesa_txn_phone (tenant_id, phone),
    CONSTRAINT fk_mpesa_txn_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Audit log of every inbound Daraja payload, including ones that fail to
-- match a transaction (wrong tenant token never reaches this table — those
-- 404 before decode). Store-before-process: if handling breaks we still have
-- exactly what Safaricom sent.
CREATE TABLE mpesa_events (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id  BIGINT UNSIGNED NOT NULL,
    txn_id     BIGINT UNSIGNED NULL,
    kind       VARCHAR(24)     NOT NULL, -- stk_callback | c2b_confirmation | c2b_validation | reversal_result | allocation | reversal_request | status_query | error
    ref        VARCHAR(80)     NOT NULL DEFAULT '', -- trans_id / checkout id / conversation id
    detail     VARCHAR(255)    NOT NULL DEFAULT '', -- human-readable summary for the feed view
    payload    JSON            NULL,                -- raw body
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_mpesa_events_txn (tenant_id, txn_id),
    KEY ix_mpesa_events_kind (tenant_id, kind, created_at),
    CONSTRAINT fk_mpesa_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Merchant-initiated reversals. The result arrives on a separate ResultURL;
-- status moves pending -> reversed | failed.
CREATE TABLE mpesa_reversals (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id             BIGINT UNSIGNED NOT NULL,
    txn_id                BIGINT UNSIGNED NOT NULL,
    originator_conversation_id VARCHAR(80) NOT NULL DEFAULT '',
    conversation_id       VARCHAR(80)     NOT NULL DEFAULT '',
    amount                DECIMAL(15,4)   NOT NULL,
    reason                VARCHAR(255)    NOT NULL DEFAULT '',
    status                VARCHAR(16)     NOT NULL DEFAULT 'pending', -- pending | reversed | failed
    result_code           INT             NULL,
    result_desc           VARCHAR(255)    NOT NULL DEFAULT '',
    requested_by          BIGINT UNSIGNED NULL,
    created_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_mpesa_rev_txn (txn_id),             -- one reversal per transaction
    KEY ix_mpesa_rev_conv (originator_conversation_id),
    CONSTRAINT fk_mpesa_rev_txn FOREIGN KEY (txn_id) REFERENCES mpesa_transactions (id)
);

-- +goose Down
DROP TABLE mpesa_reversals;
DROP TABLE mpesa_events;
DROP TABLE mpesa_transactions;
DROP TABLE mpesa_credentials;
