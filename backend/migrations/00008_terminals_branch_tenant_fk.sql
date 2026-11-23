-- +goose Up
-- Tenant isolation at the database level: a terminal's branch must belong to
-- the same tenant as the terminal. The composite FK (tenant_id, branch_id) ->
-- branches (tenant_id, id) makes a cross-tenant branch reference impossible,
-- even if application code passes the wrong branch id.
ALTER TABLE branches
    ADD UNIQUE KEY uq_branches_tenant_id (tenant_id, id);

ALTER TABLE pos_terminals
    DROP FOREIGN KEY fk_pos_terminals_branch;

ALTER TABLE pos_terminals
    ADD CONSTRAINT fk_pos_terminals_tenant_branch
        FOREIGN KEY (tenant_id, branch_id) REFERENCES branches (tenant_id, id);

-- +goose Down
ALTER TABLE pos_terminals
    DROP FOREIGN KEY fk_pos_terminals_tenant_branch;

ALTER TABLE pos_terminals
    ADD CONSTRAINT fk_pos_terminals_branch FOREIGN KEY (branch_id) REFERENCES branches (id);

ALTER TABLE branches
    DROP KEY uq_branches_tenant_id;
