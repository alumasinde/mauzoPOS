-- +goose Up
-- Business types drive the Business Profile Engine: each type has one or more
-- profile templates that seed a tenant's modules, categories, settings, receipt
-- and POS layout at onboarding.
CREATE TABLE business_types (
    id    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code  VARCHAR(30)     NOT NULL,  -- supermarket | minimart | butchery | grocery | liquor | restaurant | custom
    name  VARCHAR(80)     NOT NULL,
    sort  INT             NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uq_business_types_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE business_profiles (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id        CHAR(26)        NOT NULL,
    business_type_id BIGINT UNSIGNED NOT NULL,
    name             VARCHAR(120)    NOT NULL,
    is_default       TINYINT(1)      NOT NULL DEFAULT 0,
    -- {modules: [], categories: [], settings: {}, receipt_template, pos_layout: {}}
    definition       JSON            NOT NULL,
    created_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_business_profiles_public_id (public_id),
    UNIQUE KEY uq_business_profiles_type_name (business_type_id, name),
    CONSTRAINT fk_business_profiles_type FOREIGN KEY (business_type_id) REFERENCES business_types (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE business_profiles;
DROP TABLE business_types;
