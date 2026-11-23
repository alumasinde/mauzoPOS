-- +goose Up
-- Phase 7: customers, sales ledger, payments and returns.

CREATE TABLE customers (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id     BIGINT UNSIGNED NOT NULL,
    public_id     VARCHAR(30)     NOT NULL,
    code          VARCHAR(40)     NOT NULL DEFAULT '',
    name          VARCHAR(160)    NOT NULL,
    phone         VARCHAR(40)     NOT NULL DEFAULT '',
    email         VARCHAR(160)    NOT NULL DEFAULT '',
    kra_pin       VARCHAR(20)     NOT NULL DEFAULT '',
    credit_limit  DECIMAL(15,4)   NULL,
    notes         TEXT            NULL,
    status        VARCHAR(20)     NOT NULL DEFAULT 'active', -- active | archived
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_customers_public_id (tenant_id, public_id),
    KEY ix_customers_name (tenant_id, name),
    KEY ix_customers_phone (tenant_id, phone),
    CONSTRAINT fk_customers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Per-tenant document sequences (sales receipts, returns, ...). Rows are
-- locked FOR UPDATE inside the completing transaction so numbers never gap
-- on rollback or collide under concurrency.
CREATE TABLE tenant_counters (
    tenant_id BIGINT UNSIGNED NOT NULL,
    name      VARCHAR(40)     NOT NULL,
    value     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, name),
    CONSTRAINT fk_counters_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

CREATE TABLE sales (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT UNSIGNED NOT NULL,
    branch_id       BIGINT UNSIGNED NOT NULL,
    terminal_id     BIGINT UNSIGNED NULL,
    cashier_id      BIGINT UNSIGNED NULL, -- users.id who rang it up
    customer_id     BIGINT UNSIGNED NULL,
    public_id       VARCHAR(30)     NOT NULL,
    receipt_no      BIGINT UNSIGNED NOT NULL,
    status          VARCHAR(30)     NOT NULL DEFAULT 'completed',
                    -- completed | voided | returned | partially_returned
    source          VARCHAR(10)     NOT NULL DEFAULT 'pos', -- pos | backoffice
    idempotency_key VARCHAR(64)     NULL,
    currency        CHAR(3)         NOT NULL DEFAULT 'KES',
    subtotal        DECIMAL(15,4)   NOT NULL DEFAULT 0, -- qty x price, before discounts
    discount        DECIMAL(15,4)   NOT NULL DEFAULT 0, -- promo + manual discounts
    tax             DECIMAL(15,4)   NOT NULL DEFAULT 0,
    total           DECIMAL(15,4)   NOT NULL DEFAULT 0,
    paid            DECIMAL(15,4)   NOT NULL DEFAULT 0, -- applied payment total
    change_due      DECIMAL(15,4)   NOT NULL DEFAULT 0,
    discount_note   VARCHAR(160)    NOT NULL DEFAULT '', -- supervisor/over-limit note
    voided_at       DATETIME(3)     NULL,
    voided_by       BIGINT UNSIGNED NULL,
    void_reason     VARCHAR(255)    NOT NULL DEFAULT '',
    sold_at         DATETIME(3)     NOT NULL, -- device clock for offline sales
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_sales_public_id (tenant_id, public_id),
    UNIQUE KEY uq_sales_receipt (tenant_id, receipt_no),
    UNIQUE KEY uq_sales_idem (tenant_id, idempotency_key),
    KEY ix_sales_branch_date (tenant_id, branch_id, sold_at),
    KEY ix_sales_cashier (tenant_id, cashier_id),
    KEY ix_sales_customer (tenant_id, customer_id),
    CONSTRAINT fk_sales_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_sales_branch   FOREIGN KEY (branch_id)   REFERENCES branches (id),
    CONSTRAINT fk_sales_terminal FOREIGN KEY (terminal_id) REFERENCES pos_terminals (id),
    CONSTRAINT fk_sales_cashier  FOREIGN KEY (cashier_id)  REFERENCES users (id),
    CONSTRAINT fk_sales_customer FOREIGN KEY (customer_id) REFERENCES customers (id)
);

CREATE TABLE sale_items (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id     BIGINT UNSIGNED NOT NULL,
    sale_id       BIGINT UNSIGNED NOT NULL,
    line_no       INT             NOT NULL,
    product_id    BIGINT UNSIGNED NOT NULL,
    variant_id    BIGINT UNSIGNED NULL,
    name          VARCHAR(200)    NOT NULL, -- snapshot incl. variant name
    sku           VARCHAR(64)     NOT NULL DEFAULT '',
    qty           DECIMAL(15,4)   NOT NULL,
    unit_price    DECIMAL(15,4)   NOT NULL, -- after list price, before discounts
    discount      DECIMAL(15,4)   NOT NULL DEFAULT 0,
    tax_rate      DECIMAL(6,4)    NULL,
    tax           DECIMAL(15,4)   NOT NULL DEFAULT 0,
    total         DECIMAL(15,4)   NOT NULL,
    returned_qty  DECIMAL(15,4)   NOT NULL DEFAULT 0,
    price_override TINYINT(1)     NOT NULL DEFAULT 0,
    promotion     VARCHAR(120)    NOT NULL DEFAULT '', -- applied promo name
    PRIMARY KEY (id),
    UNIQUE KEY uq_sale_items_line (sale_id, line_no),
    KEY ix_sale_items_product (tenant_id, product_id),
    CONSTRAINT fk_items_tenant  FOREIGN KEY (tenant_id)  REFERENCES tenants (id),
    CONSTRAINT fk_items_sale    FOREIGN KEY (sale_id)    REFERENCES sales (id) ON DELETE CASCADE,
    CONSTRAINT fk_items_product FOREIGN KEY (product_id) REFERENCES products (id)
);

CREATE TABLE sale_payments (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    sale_id     BIGINT UNSIGNED NOT NULL,
    method      VARCHAR(20)     NOT NULL, -- cash | card | bank | credit
    amount      DECIMAL(15,4)   NOT NULL, -- applied to the sale
    tendered    DECIMAL(15,4)   NULL,     -- cash handed over
    change_due  DECIMAL(15,4)   NULL,
    reference   VARCHAR(80)     NOT NULL DEFAULT '', -- card/bank ref
    status      VARCHAR(20)     NOT NULL DEFAULT 'confirmed', -- confirmed | voided | refunded
    created_by  BIGINT UNSIGNED NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_payments_sale (sale_id),
    KEY ix_payments_method (tenant_id, method, created_at),
    CONSTRAINT fk_payments_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_payments_sale   FOREIGN KEY (sale_id)   REFERENCES sales (id) ON DELETE CASCADE
);

CREATE TABLE sale_returns (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id      BIGINT UNSIGNED NOT NULL,
    sale_id        BIGINT UNSIGNED NOT NULL,
    public_id      VARCHAR(30)     NOT NULL,
    return_no      BIGINT UNSIGNED NOT NULL,
    reason         VARCHAR(255)    NOT NULL DEFAULT '',
    refund_method  VARCHAR(20)     NOT NULL DEFAULT 'cash',
    refund_amount  DECIMAL(15,4)   NOT NULL,
    status         VARCHAR(20)     NOT NULL DEFAULT 'completed', -- completed | voided
    supervisor_id  BIGINT UNSIGNED NULL,
    created_by     BIGINT UNSIGNED NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_returns_public_id (tenant_id, public_id),
    UNIQUE KEY uq_returns_no (tenant_id, return_no),
    KEY ix_returns_sale (sale_id),
    CONSTRAINT fk_returns_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT fk_returns_sale   FOREIGN KEY (sale_id)   REFERENCES sales (id)
);

CREATE TABLE sale_return_items (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id    BIGINT UNSIGNED NOT NULL,
    return_id    BIGINT UNSIGNED NOT NULL,
    sale_item_id BIGINT UNSIGNED NOT NULL,
    qty          DECIMAL(15,4)   NOT NULL,
    restock      TINYINT(1)      NOT NULL DEFAULT 1,
    amount       DECIMAL(15,4)   NOT NULL, -- refunded line value
    PRIMARY KEY (id),
    KEY ix_ritems_return (return_id),
    CONSTRAINT fk_ritems_tenant FOREIGN KEY (tenant_id)    REFERENCES tenants (id),
    CONSTRAINT fk_ritems_return FOREIGN KEY (return_id)    REFERENCES sale_returns (id) ON DELETE CASCADE,
    CONSTRAINT fk_ritems_item   FOREIGN KEY (sale_item_id) REFERENCES sale_items (id)
);

-- +goose Down
DROP TABLE sale_return_items;
DROP TABLE sale_returns;
DROP TABLE sale_payments;
DROP TABLE sale_items;
DROP TABLE sales;
DROP TABLE tenant_counters;
DROP TABLE customers;
