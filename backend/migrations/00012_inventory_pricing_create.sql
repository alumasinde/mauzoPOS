-- +goose Up
-- Phase 5: inventory (locations, ledger, balances, documents) & pricing.

CREATE TABLE stock_locations (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id   BIGINT UNSIGNED NOT NULL,
  branch_id   BIGINT UNSIGNED NOT NULL,
  public_id   VARCHAR(30)     NOT NULL,
  name        VARCHAR(120)    NOT NULL,
  kind        VARCHAR(30)     NOT NULL DEFAULT 'storage', -- floor | storage | shelf | fridge | other
  is_default  TINYINT(1)      NOT NULL DEFAULT 0,
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_stock_locations_pid (tenant_id, public_id),
  UNIQUE KEY uq_stock_locations_name (tenant_id, branch_id, name),
  KEY ix_stock_locations_branch (tenant_id, branch_id),
  CONSTRAINT fk_loc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_loc_branch FOREIGN KEY (branch_id) REFERENCES branches(id)
);

CREATE TABLE product_batches (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id    BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,        -- 0 = no variant
  batch_no     VARCHAR(60)      NOT NULL,
  expiry_date  DATE             NULL,
  received_at  DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  created_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_batches_no (tenant_id, product_id, variant_id, batch_no),
  KEY ix_batches_expiry (tenant_id, expiry_date),
  CONSTRAINT fk_batch_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_batch_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- stock_movements is an append-only ledger: the only way stock ever changes.
-- qty is a signed delta in the product's base unit.
CREATE TABLE stock_movements (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id    BIGINT UNSIGNED NOT NULL,
  branch_id    BIGINT UNSIGNED NOT NULL,
  location_id  BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  batch_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  qty          DECIMAL(16,4)    NOT NULL,                 -- signed delta
  unit_cost    DECIMAL(16,4)    NULL,                     -- cost basis for inbound moves
  kind         VARCHAR(20)      NOT NULL,                 -- receive|sale|adjust|transfer_out|transfer_in|count|waste|return
  ref_type     VARCHAR(30)      NULL,                     -- adjustment|transfer|count|sale|wastage|receive
  ref_public_id VARCHAR(30)     NULL,
  reason       VARCHAR(40)      NULL,
  note         VARCHAR(255)     NULL,
  actor_id     BIGINT UNSIGNED  NULL,
  created_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY ix_mov_product (tenant_id, product_id, created_at),
  KEY ix_mov_branch  (tenant_id, branch_id, kind, created_at),
  KEY ix_mov_ref     (tenant_id, ref_type, ref_public_id),
  CONSTRAINT fk_mov_tenant   FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_mov_branch   FOREIGN KEY (branch_id) REFERENCES branches(id),
  CONSTRAINT fk_mov_location FOREIGN KEY (location_id) REFERENCES stock_locations(id),
  CONSTRAINT fk_mov_product  FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Derived balances; only ever written inside the same transaction as a
-- stock_movements insert. avg_cost is the weighted-average cost for the row.
CREATE TABLE stock_balances (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id    BIGINT UNSIGNED NOT NULL,
  branch_id    BIGINT UNSIGNED NOT NULL,
  location_id  BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  batch_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  qty          DECIMAL(16,4)    NOT NULL DEFAULT 0,
  avg_cost     DECIMAL(16,4)    NOT NULL DEFAULT 0,
  updated_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_balances_row (tenant_id, location_id, product_id, variant_id, batch_id),
  KEY ix_balances_product (tenant_id, product_id),
  CONSTRAINT fk_bal_tenant   FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_bal_location FOREIGN KEY (location_id) REFERENCES stock_locations(id),
  CONSTRAINT fk_bal_product  FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE stock_adjustments (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id    BIGINT UNSIGNED NOT NULL,
  branch_id    BIGINT UNSIGNED NOT NULL,
  public_id    VARCHAR(30)      NOT NULL,
  status       VARCHAR(15)      NOT NULL DEFAULT 'pending', -- pending|approved|posted|rejected
  reason       VARCHAR(40)      NOT NULL,
  note         VARCHAR(255)     NULL,
  created_by   BIGINT UNSIGNED  NOT NULL,
  approved_by  BIGINT UNSIGNED  NULL,
  posted_at    DATETIME(3)      NULL,
  created_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_adjustments_pid (tenant_id, public_id),
  CONSTRAINT fk_adj_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_adj_branch FOREIGN KEY (branch_id) REFERENCES branches(id)
);

CREATE TABLE stock_adjustment_items (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  adjustment_id BIGINT UNSIGNED NOT NULL,
  location_id   BIGINT UNSIGNED NOT NULL,
  product_id    BIGINT UNSIGNED NOT NULL,
  variant_id    BIGINT UNSIGNED NOT NULL DEFAULT 0,
  batch_id      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  qty           DECIMAL(16,4)   NOT NULL,                 -- signed delta
  CONSTRAINT fk_adjitem_adj FOREIGN KEY (adjustment_id) REFERENCES stock_adjustments(id) ON DELETE CASCADE
);

CREATE TABLE stock_transfers (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id     BIGINT UNSIGNED NOT NULL,
  public_id     VARCHAR(30)     NOT NULL,
  from_branch_id BIGINT UNSIGNED NOT NULL,
  to_branch_id  BIGINT UNSIGNED NOT NULL,
  status        VARCHAR(15)     NOT NULL DEFAULT 'draft',  -- draft|sent|received|cancelled
  note          VARCHAR(255)    NULL,
  created_by    BIGINT UNSIGNED NOT NULL,
  sent_at       DATETIME(3)     NULL,
  received_at   DATETIME(3)     NULL,
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_transfers_pid (tenant_id, public_id),
  CONSTRAINT fk_tr_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_tr_from   FOREIGN KEY (from_branch_id) REFERENCES branches(id),
  CONSTRAINT fk_tr_to     FOREIGN KEY (to_branch_id) REFERENCES branches(id)
);

CREATE TABLE stock_transfer_items (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  public_id    VARCHAR(30)     NOT NULL,
  transfer_id  BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  qty_sent     DECIMAL(16,4)   NOT NULL,
  qty_received DECIMAL(16,4)   NULL,
  CONSTRAINT fk_tritem_tr FOREIGN KEY (transfer_id) REFERENCES stock_transfers(id) ON DELETE CASCADE
);

CREATE TABLE stock_counts (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id   BIGINT UNSIGNED NOT NULL,
  branch_id   BIGINT UNSIGNED NOT NULL,
  public_id   VARCHAR(30)     NOT NULL,
  status      VARCHAR(15)     NOT NULL DEFAULT 'open',     -- open|posted|cancelled
  location_id BIGINT UNSIGNED NULL,                        -- NULL = whole branch
  note        VARCHAR(255)    NULL,
  created_by  BIGINT UNSIGNED NOT NULL,
  posted_at   DATETIME(3)     NULL,
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_counts_pid (tenant_id, public_id),
  CONSTRAINT fk_cnt_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_cnt_branch FOREIGN KEY (branch_id) REFERENCES branches(id)
);

CREATE TABLE stock_count_items (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  public_id    VARCHAR(30)     NOT NULL,
  count_id     BIGINT UNSIGNED NOT NULL,
  location_id  BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  batch_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  expected_qty DECIMAL(16,4)   NOT NULL,
  counted_qty  DECIMAL(16,4)   NULL,                       -- NULL = not yet counted
  CONSTRAINT fk_cntitem_cnt FOREIGN KEY (count_id) REFERENCES stock_counts(id) ON DELETE CASCADE
);

CREATE TABLE reorder_levels (
  tenant_id   BIGINT UNSIGNED NOT NULL,
  branch_id   BIGINT UNSIGNED NOT NULL,
  product_id  BIGINT UNSIGNED NOT NULL,
  variant_id  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  min_qty     DECIMAL(16,4)   NOT NULL DEFAULT 0,
  target_qty  DECIMAL(16,4)   NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, branch_id, product_id, variant_id),
  CONSTRAINT fk_rl_tenant  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_rl_product FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE price_lists (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id      BIGINT UNSIGNED NOT NULL,
  public_id      VARCHAR(30)     NOT NULL,
  name           VARCHAR(120)    NOT NULL,
  branch_id      BIGINT UNSIGNED NULL,                     -- NULL = all branches
  customer_group VARCHAR(40)     NULL,                     -- NULL = everyone
  priority       INT             NOT NULL DEFAULT 0,       -- higher wins
  is_active      TINYINT(1)      NOT NULL DEFAULT 1,
  valid_from     DATETIME(3)     NULL,
  valid_to       DATETIME(3)     NULL,
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_price_lists_pid (tenant_id, public_id),
  UNIQUE KEY uq_price_lists_name (tenant_id, name),
  CONSTRAINT fk_pl_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_pl_branch FOREIGN KEY (branch_id) REFERENCES branches(id)
);

-- Tiered pricing: several rows per product with increasing min_qty; the
-- largest min_qty <= sold qty wins.
CREATE TABLE price_list_items (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  list_id      BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  variant_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  min_qty      DECIMAL(16,4)   NOT NULL DEFAULT 1,
  price        DECIMAL(16,4)   NOT NULL,
  UNIQUE KEY uq_plitems_row (list_id, product_id, variant_id, min_qty),
  CONSTRAINT fk_pli_list    FOREIGN KEY (list_id) REFERENCES price_lists(id) ON DELETE CASCADE,
  CONSTRAINT fk_pli_product FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE promotions (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id   BIGINT UNSIGNED NOT NULL,
  public_id   VARCHAR(30)     NOT NULL,
  name        VARCHAR(120)    NOT NULL,
  kind        VARCHAR(15)     NOT NULL,                    -- percent|amount|bxgy
  config      JSON           NOT NULL,                     -- {"percent":10} | {"amount":50} | {"buy":2,"get":1,"get_percent":100}
  starts_at   DATETIME(3)     NULL,
  ends_at     DATETIME(3)     NULL,
  days_mask   TINYINT UNSIGNED NULL,                       -- bit 0=Mon … bit 6=Sun; NULL = every day
  priority    INT            NOT NULL DEFAULT 0,
  is_active   TINYINT(1)     NOT NULL DEFAULT 1,
  created_at  DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_promotions_pid (tenant_id, public_id),
  CONSTRAINT fk_promo_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- A target row with all NULLs means "the whole catalog".
CREATE TABLE promotion_targets (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  promotion_id  BIGINT UNSIGNED NOT NULL,
  product_id    BIGINT UNSIGNED NULL,
  category_id   BIGINT UNSIGNED NULL,
  CONSTRAINT fk_pt_promo    FOREIGN KEY (promotion_id) REFERENCES promotions(id) ON DELETE CASCADE,
  CONSTRAINT fk_pt_product  FOREIGN KEY (product_id) REFERENCES products(id),
  CONSTRAINT fk_pt_category FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE price_history (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tenant_id   BIGINT UNSIGNED NOT NULL,
  product_id  BIGINT UNSIGNED NOT NULL,
  variant_id  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  source      VARCHAR(20)     NOT NULL,                    -- base | price_list
  ref_public_id VARCHAR(30)   NULL,                        -- price list public id when source=price_list
  old_price   DECIMAL(16,4)  NULL,
  new_price   DECIMAL(16,4)  NOT NULL,
  actor_id    BIGINT UNSIGNED NULL,
  created_at  DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY ix_ph_product (tenant_id, product_id, created_at),
  CONSTRAINT fk_ph_tenant  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_ph_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- +goose Down
DROP TABLE IF EXISTS price_history;
DROP TABLE IF EXISTS promotion_targets;
DROP TABLE IF EXISTS promotions;
DROP TABLE IF EXISTS price_list_items;
DROP TABLE IF EXISTS price_lists;
DROP TABLE IF EXISTS reorder_levels;
DROP TABLE IF EXISTS stock_count_items;
DROP TABLE IF EXISTS stock_counts;
DROP TABLE IF EXISTS stock_transfer_items;
DROP TABLE IF EXISTS stock_transfers;
DROP TABLE IF EXISTS stock_adjustment_items;
DROP TABLE IF EXISTS stock_adjustments;
DROP TABLE IF EXISTS stock_balances;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS product_batches;
DROP TABLE IF EXISTS stock_locations;
