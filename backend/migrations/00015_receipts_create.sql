-- +goose Up
-- Phase 8: receipt templates, per-branch assignment, logo, issued receipts.

-- A tenant's receipt layouts. `body` is a receiptfmt template (versioned
-- JSON). Every save bumps `version` and appends to receipt_template_versions,
-- so an issued receipt can always name the exact layout it was printed with.
CREATE TABLE receipt_templates (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    public_id   VARCHAR(30)     NOT NULL,
    name        VARCHAR(120)    NOT NULL,
    paper_width SMALLINT        NOT NULL DEFAULT 80, -- 58 | 80
    body        JSON            NOT NULL,
    version     INT UNSIGNED    NOT NULL DEFAULT 1,
    based_on    VARCHAR(30)     NOT NULL DEFAULT '', -- builtin key it was copied from
    archived_at DATETIME(3)     NULL,
    created_by  BIGINT UNSIGNED NULL,
    updated_by  BIGINT UNSIGNED NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_rtpl_public (tenant_id, public_id),
    KEY ix_rtpl_tenant (tenant_id, archived_at),
    CONSTRAINT fk_rtpl_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Immutable history of every saved layout.
CREATE TABLE receipt_template_versions (
    template_id BIGINT UNSIGNED NOT NULL,
    version     INT UNSIGNED    NOT NULL,
    tenant_id   BIGINT UNSIGNED NOT NULL,
    paper_width SMALLINT        NOT NULL,
    body        JSON            NOT NULL,
    created_by  BIGINT UNSIGNED NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (template_id, version),
    CONSTRAINT fk_rtplv_template FOREIGN KEY (template_id) REFERENCES receipt_templates (id) ON DELETE CASCADE,
    CONSTRAINT fk_rtplv_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id)
);

-- Which layout a branch prints. branch_id 0 = every branch without its own
-- row. No row at all = the builtin chosen by the business profile.
CREATE TABLE receipt_assignments (
    tenant_id   BIGINT UNSIGNED NOT NULL,
    branch_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    template_id BIGINT UNSIGNED NOT NULL,
    updated_by  BIGINT UNSIGNED NULL,
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (tenant_id, branch_id),
    KEY ix_rassign_template (template_id),
    CONSTRAINT fk_rassign_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_rassign_template FOREIGN KEY (template_id) REFERENCES receipt_templates (id)
);

-- The tenant's receipt logo, stored already dithered to 1 bit (receiptfmt
-- Raster) so every till prints it identically without image libraries.
CREATE TABLE receipt_logos (
    tenant_id  BIGINT UNSIGNED NOT NULL,
    width      SMALLINT        NOT NULL,
    height     SMALLINT        NOT NULL,
    bits       MEDIUMBLOB      NOT NULL,
    updated_by BIGINT UNSIGNED NULL,
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (tenant_id),
    CONSTRAINT fk_rlogo_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Every receipt handed out: the original (copy_no 0) is the rendered snapshot
-- taken when the sale was issued; reprints (copy_no 1, 2, …) re-print that
-- snapshot with a REPRINT banner and are always audited. The layout that
-- produced the content is recorded by template + version (NULL template =
-- builtin `builtin_key`).
CREATE TABLE issued_receipts (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tenant_id        BIGINT UNSIGNED NOT NULL,
    public_id        VARCHAR(30)     NOT NULL,
    sale_id          BIGINT UNSIGNED NOT NULL,
    branch_id        BIGINT UNSIGNED NOT NULL,
    copy_no          INT UNSIGNED    NOT NULL DEFAULT 0,
    template_id      BIGINT UNSIGNED NULL,
    template_version INT UNSIGNED    NULL,
    builtin_key      VARCHAR(30)     NOT NULL DEFAULT '',
    paper_width      SMALLINT        NOT NULL,
    content          JSON            NOT NULL, -- receiptfmt.Output
    channel          VARCHAR(12)     NOT NULL, -- pos | backoffice
    terminal_id      BIGINT UNSIGNED NULL,
    issued_by        BIGINT UNSIGNED NULL,     -- users.id
    supervisor_id    BIGINT UNSIGNED NULL,     -- authorised a reprint
    created_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_issued_public (tenant_id, public_id),
    UNIQUE KEY uq_issued_copy (tenant_id, sale_id, copy_no),
    KEY ix_issued_template (template_id, template_version),
    CONSTRAINT fk_issued_tenant   FOREIGN KEY (tenant_id)   REFERENCES tenants (id),
    CONSTRAINT fk_issued_sale     FOREIGN KEY (sale_id)     REFERENCES sales (id),
    CONSTRAINT fk_issued_template FOREIGN KEY (template_id) REFERENCES receipt_templates (id)
);

-- +goose Down
DROP TABLE issued_receipts;
DROP TABLE receipt_logos;
DROP TABLE receipt_assignments;
DROP TABLE receipt_template_versions;
DROP TABLE receipt_templates;
