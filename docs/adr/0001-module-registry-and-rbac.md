# ADR 0001 — Module registry, roles & permissions

- **Status:** accepted (Phase 3)
- **Date:** 2026-11-23

## Context

Tenants of every business type share one codebase. Features must be switched on per tenant (by
profile, plan, the tenant itself or platform support) and every action must be authorised by the
backend, per tenant, per role and per branch. Before Phase 3 a tenant user's role was a text label
checked by `RequireUserRole`, and the module catalogue was a flat list.

## Decision

### Module registry

- **Definitions live in code** (`core/modules`): code, group, `Core` flag, `Depends`, `ConfigSchema`,
  `Nav` entries and `RecommendedFor` business types. The graph is validated at boot (unknown
  dependencies or cycles panic).
- **The `modules` table** holds only the platform-wide on/off switch. It is synced at boot, so
  operators can withdraw a module without a deploy.
- **`tenant_modules`** holds the tenant's switch, `source` (`tenant` or `platform` override) and JSON
  config. Only `modules/registry` writes it.
- **Effective modules** are computed, never stored:
  > core ∪ { enabled ∧ platform-active ∧ (platform override ∨ no plan ∨ in plan) }, then repeatedly
  > drop any module whose dependency is off.

  A plan downgrade therefore hides modules without deleting their configuration.
- **Dependencies:**
  - Enabling a module also enables its dependency closure.
  - Disabling one that others depend on fails with `module_has_dependents`, unless `cascade` is set.
  - Plans always store their dependency closure.
- **`RecommendedFor`** is a UI hint only. Any tenant may enable any module, as the spec requires.

### Permissions

- **Declaration:** each module declares `authz.Permission{Code, Module, Description, Sensitive}` in its
  `permissions.go`. Platform scope is the `platform.` prefix.
- **Database mirror:** the registry is mirrored into the `permissions` table at boot (and by
  `migrate seed`), so `role_permissions` has referential integrity. Codes removed from the code delete
  their grants by cascade.
- **Completeness guard:** a binary with a partial registry would delete real grants. Syncing is
  therefore refused unless `modules/registerall` (which links every module and calls
  `authz.MarkComplete`) is imported.
- **Module gating:** a tenant permission only takes effect while its `Module` is enabled. `Can(perm)`
  returns `module_disabled` (403) distinctly from `forbidden`.

### Roles

- **Storage:** `roles` and `role_permissions` per tenant. `users.role_id` uses a composite FK
  `(tenant_id, role_id)`, so a user can never hold another tenant's role.
- **Templates:** seven system roles come from `authz.RoleTemplates`. Grants are glob patterns expanded
  against the registry.
- **Template updates:**
  - A system role with `customized = 0` is re-expanded at every boot, so permissions added in later
    phases reach existing tenants automatically.
  - Editing a system role's permissions or limits sets `customized = 1`, after which the tenant's
    choice wins.
  - "Reset" returns the role to its template.
- **Owner:** the role is locked, implicitly holds every tenant permission, and cannot be edited or
  deleted. The last active owner cannot be disabled or demoted.
- **No escalation:** you may only grant permissions you hold. This covers creating or editing a role,
  assigning a role, granting branch access, and managing a user whose role outranks yours.

### Branch scope

- **Access rule:** `users.all_branches`, otherwise `user_branches` (composite FKs on both sides).
  Owners always have every branch.
- **Header:** `X-Branch-Id` (a public ID) is validated by the `BranchScope` middleware and stored in
  the context. Lists of branch-bound records are filtered by `Actor.CanAccessBranch`.

### Per-request access

- **Resolution:** `auth.Authenticate` resolves `identity.Access` (permissions, modules, branches) once
  per request through `roles.AccessService`.
- **Caching:** each piece uses a 30 s in-process TTL cache with explicit invalidation on writes (role
  edit, module toggle, plan change, branch change).
- **Accepted trade-off:** on a multi-replica deployment, other replicas converge within 30 s. Changes
  that must be immediate (disable user, new role or password) also revoke the user's sessions. Move
  invalidation to Redis pub/sub when running several replicas.

### Navigation

- `GET /t/navigation` returns the `Nav` entries of enabled modules that the user may see.
- The web app renders only entries whose path is a page it ships, so a module can never link to a
  screen that does not exist yet.

## Consequences

- New features declare permissions with a `Module`, add `Nav` entries when their screens land, and
  never check business types.
- New modules must be added to `core/modules/catalogue.go` and `modules/registerall`.
- POS endpoints (Phase 6) must re-check permissions on sync, because offline terminals cache them.
