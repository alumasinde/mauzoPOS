# Mauzo POS

Multi-tenant, configurable POS & business-management SaaS for the Kenyan market
(supermarkets, minimarts, butcheries, groceries, liquor stores, restaurants/cafés
and other retail).

One codebase, three applications, one backend:

| App | Path | Users | Tech |
|---|---|---|---|
| **Platform Admin** | `frontend/apps/platform` | SaaS operators | Svelte 5 + TS (SvelteKit, static SPA) |
| **Business Back-office** | `frontend/apps/web` | Tenant owners, managers, accountants | Svelte 5 + TS (SvelteKit, static SPA) |
| **POS Terminal (.exe)** | `frontend/apps/pos` + `backend/desktop` | Cashiers, supervisors | Svelte 5 + TS UI inside a Wails (Go) shell, local SQLite |
| **API + workers** | `backend/` | All of the above | Go, MySQL 8 / MariaDB, REST/JSON, SSE/WebSocket |

Progress is tracked in [progress.md](progress.md). Permissions are listed in
[docs/permissions.md](docs/permissions.md).

---

## 1. Guiding principles

1. **Configuration over conditionals.** Business types, modules, taxes, receipts, menus
   and categories come from data (the Business Profile Engine + Module Registry). Never
   write `if businessType == "butchery"` in feature code. Ask the registry instead:
   `modules.Enabled(ctx, "weighing_scales")`.
2. **Business logic lives in Go.** Svelte renders data and collects input. Money, tax,
   pricing, permissions and stock rules run in Go. The POS reuses the **same Go packages**
   (`backend/pkg/...`) offline, so the logic is never rewritten in TypeScript.
3. **No god files.** Split each entity into small files. Soft limit is about 300 lines per file.
   If a file grows past that, split it by entity or concern.
4. **The backend checks everything.** Tenant scope, branch scope, module enablement and
   permissions are checked in middleware or services. The UI hides things only for usability.
5. **Financial records are append-only.** Completed sales, payments, stock movements and
   fiscal invoices are never edited or deleted in place. Changes go through reversals,
   credit notes and adjustments, and each one leaves an audit trail.
6. **Idempotency everywhere money moves.** Sale completion, payments, M-Pesa callbacks and
   sync uploads all carry idempotency keys and unique constraints.
7. **Docker-ready, not Docker-dependent.** All config comes from environment variables,
   apps are stateless, logs go to stdout, and migrations run as a separate command. During
   development we run against a local MariaDB/MySQL 8.

---

## 2. Repository layout

```
mauzoPOS/
├── README.md
├── progress.md                  # ✅ tick-list of phases / modules / features
├── docs/
│   ├── permissions.md           # permission catalogue + default role matrix
│   ├── adr/                     # architecture decision records (NNNN-title.md)
│   ├── api/                     # OpenAPI specs per module
│   └── reference/               # UI + receipt reference images
├── backend/                     # ALL Go code (one go.mod)
│   ├── cmd/
│   │   ├── api/main.go          # tiny: load config → app.New() → serve
│   │   ├── worker/main.go       # background jobs (eTIMS retries, M-Pesa, reports)
│   │   └── migrate/main.go      # migrations up/down/status + seed
│   ├── internal/
│   │   ├── app/                 # bootstrap & dependency wiring (container.go, server.go)
│   │   ├── config/              # env parsing → typed Config
│   │   ├── routes/app.go        # MASTER router: mounts every module's routes
│   │   ├── worker/              # background job runner
│   │   ├── testutil/            # integration-test DB helper (skips if DB unavailable)
│   │   ├── core/                # shared infrastructure (no business rules)
│   │   │   ├── db/              # pool, tx helper, tenant-scoped querier, migrations runner
│   │   │   ├── logger/          # slog setup
│   │   │   ├── httpx/           # JSON responses, error envelope, pagination
│   │   │   ├── middleware/      # auth, tenant, branch, permission, module, ratelimit, requestid
│   │   │   ├── authz/           # permission registry + checker
│   │   │   ├── tenancy/         # tenant context, resolver
│   │   │   ├── idempotency/     # idempotency-key store
│   │   │   ├── audit/           # audit log writer
│   │   │   ├── events/          # in-process event bus + outbox
│   │   │   ├── security/        # argon2id, tokens, MFA (TOTP)
│   │   │   └── validate/        # request validation
│   │   └── modules/             # ONE folder per business module (see §3)
│   │       ├── tenants/
│   │       ├── businesstypes/
│   │       ├── registry/        # module enablement, plan gating, navigation
│   │       ├── auth/
│   │       ├── users/
│   │       ├── roles/
│   │       ├── branches/
│   │       ├── terminals/
│   │       ├── catalog/         # products, categories, brands, barcodes, uom, attributes
│   │       ├── pricing/
│   │       ├── inventory/
│   │       ├── purchasing/
│   │       ├── sales/
│   │       ├── payments/
│   │       ├── mpesa/
│   │       ├── tax/
│   │       ├── etims/
│   │       ├── receipts/
│   │       ├── shifts/
│   │       ├── customers/
│   │       ├── promotions/
│   │       ├── reports/
│   │       ├── sync/
│   │       └── ...
│   ├── pkg/                     # PURE, dependency-free domain libs shared by API + desktop
│   │   ├── money/               # decimal money, rounding rules
│   │   ├── taxcalc/             # inclusive/exclusive VAT, breakdowns
│   │   ├── pricecalc/           # price lists, bulk pricing, promotions evaluation
│   │   ├── barcode/             # EAN/UPC validation, weighted-barcode parsing
│   │   └── receiptfmt/          # template JSON → render tree (ESC/POS, HTML, PDF)
│   ├── desktop/                 # Wails host for the POS .exe
│   │   ├── main.go
│   │   ├── wails.json           # frontend dir → ../../frontend/apps/pos
│   │   ├── bindings/            # Go methods exposed to Svelte (cart, scan, print...)
│   │   ├── localdb/             # SQLite schema + repos (modernc.org/sqlite, no CGO)
│   │   ├── syncagent/           # outbox uploader + catalog downloader
│   │   ├── printer/             # ESC/POS driver (USB / network / Windows spooler)
│   │   └── scale/               # weighing-scale serial integration
│   ├── migrations/              # SQL migrations (goose), named NNNNN_module_change.sql
│   ├── seeds/                   # business types, modules, permissions, tax defaults
│   ├── .env.example
│   └── go.mod
├── frontend/                    # ALL Svelte code (pnpm workspace)
│   ├── package.json
│   ├── pnpm-workspace.yaml
│   ├── packages/
│   │   ├── ui/                  # design tokens + reusable Svelte 5 components
│   │   ├── api-client/          # typed fetch client, error handling, auth refresh
│   │   ├── types/               # shared TS types (generated from OpenAPI where possible)
│   │   └── utils/               # formatting (KSh, dates), keyboard helpers
│   └── apps/
│       ├── platform/            # Platform Admin dashboard
│       ├── web/                 # Tenant back-office
│       └── pos/                 # POS UI (built into the Wails .exe)
└── deploy/                      # (later) Dockerfiles, compose, nginx — see §9
```

---

## 3. Backend module convention

Every module in `backend/internal/modules/<name>/` has the same shape. Files are split
**per entity**. Example for `catalog`:

```
catalog/
├── module.go              # Module descriptor: ID, deps, permissions, Register(container)
├── routes.go              # func Routes(r chi.Router, h *Handlers) — ONLY route wiring
├── permissions.go         # const PermProductView = "catalog.product.view" ... + Register()
├── errors.go              # module-specific sentinel errors
├── product_model.go       # Product struct + domain methods
├── product_dto.go         # request/response shapes + validation tags
├── product_repo.go        # SQL for products (always tenant-scoped)
├── product_service.go     # business rules, transactions, events
├── product_handler.go     # HTTP: decode → service → encode
├── product_service_test.go
├── category_model.go
├── category_repo.go
├── category_service.go
├── category_handler.go
├── barcode_model.go
├── barcode_repo.go
└── ...
```

Rules:

- `routes.go` only wires routes and attaches the permission/module guards. It holds no logic.
- Handlers stay thin. They decode, call one service method, and encode the result.
- Services own the transactions (`db.WithTx(ctx, func(tx) error {...})`), audit calls and events.
- Repositories take a tenant-scoped querier. **The `tenant_id` comes from the authenticated
  context and never from the request body.**
- Modules talk to each other through **service interfaces** that the consumer defines,
  never through each other's repos or tables.
- `main.go` stays under ~30 lines. Wiring lives in `internal/app`. Route mounting lives in
  `internal/routes/app.go`, which looks like this:

```go
// internal/routes/app.go
func New(c *app.Container) http.Handler {
    r := chi.NewRouter()            // + global middleware
    system.Routes(r, c.System)      // /healthz, /readyz
    r.Route("/api/v1", func(r chi.Router) {
        r.Mount("/platform", platformRouter(c)) // platform-admin auth context
        r.Mount("/t", tenantRouter(c))          // tenant employee auth context
        r.Mount("/pos", posRouter(c))           // terminal + cashier auth context
        r.Mount("/hooks", webhookRouter(c))     // M-Pesa / eTIMS callbacks (signed, no session)
    })
    return r
}
```

Each `xxxRouter` calls `catalog.Routes(r, c.Catalog)`, `sales.Routes(r, c.Sales)` and so on.

### Route guard example

```go
// modules/catalog/routes.go
func Routes(r chi.Router, h *Handlers) {
    r.Route("/products", func(r chi.Router) {
        r.Use(mw.RequireModule("catalog"))
        r.With(mw.Can(PermProductView)).Get("/", h.Product.List)
        r.With(mw.Can(PermProductCreate)).Post("/", h.Product.Create)
        r.With(mw.Can(PermProductUpdate)).Put("/{id}", h.Product.Update)
    })
}
```

---

## 4. Modules, permissions & roles

See [ADR 0001](docs/adr/0001-module-registry-and-rbac.md) for the full design.

- **Modules** are defined in `core/modules/catalogue.go`. Each has a code, dependencies, a config
  schema, sidebar `Nav` entries, and the business types it is recommended for. Feature code asks
  whether a module is enabled (`mw.RequireModule`, `actor.HasModule`) and **never checks business
  types**.
- **Effective modules** for a tenant are: core modules, plus the tenant's enabled modules, limited to
  what its plan allows (unless the platform overrode it) and what is switched on platform-wide.
  Enabling a module pulls in its dependencies. Disabling one that others need requires `cascade`.
- **Permissions** use the format `module.resource.action`. Each module declares its own in
  `permissions.go` with `authz.Define(...)`, including the module that gates it and whether it is
  `Sensitive`.
  - **Boot sync:** the API mirrors the registry into the `permissions` table at boot (`migrate seed`
    does too). This is refused unless `modules/registerall` is linked. **Add every new module there.**
  - **Unknown strings** passed to `mw.Can(...)` panic at boot.
- **Roles** are rows per tenant. Seven built-in roles come from `authz.RoleTemplates` and follow their
  template until a tenant edits them, so new permissions in later releases reach them automatically.
  - **Owner** is locked and holds everything. The last active owner is protected.
  - **No escalation:** you can only grant, assign or manage what you hold yourself.
- **Branch access:** each employee has all branches or a list. Send `X-Branch-Id` to scope a request,
  and filter branch-bound lists with `actor.CanAccessBranch`.
- **Sensitive** permissions always write an audit entry and, at the POS, need a supervisor PIN when the
  cashier lacks them.
- **Check order:** authenticated → tenant active → account active → **module enabled** →
  **permission** → **branch access**. All checks run in the backend; `/me` returns the effective
  permissions and modules only so the UI can hide controls.

### Adding a feature module (checklist)

1. Add the module to `core/modules/catalogue.go` (deps, config schema, `Nav` only once its page exists).
2. Create `internal/modules/<name>/permissions.go` with `authz.Define(...)` and the right `Module`.
3. Import the package in `internal/modules/registerall`.
4. Guard routes with `mw.Can(Perm...)`, which also enforces the module.
5. Add the page to the web app's `ROUTES` list in `+layout.svelte`.

The full catalogue is in [docs/permissions.md](docs/permissions.md).

---

## 5. Multi-tenancy & data rules

- Single MySQL schema with a `tenant_id` column on every tenant-owned table. Every unique
  index and lookup index starts with `tenant_id`.
- The repository layer only accepts a `TenantQuerier` that injects `tenant_id`, which keeps
  the code ready for dedicated databases per enterprise tenant later.
- IDs: `BIGINT UNSIGNED` auto-increment primary keys internally, plus a `public_id`
  (ULID, `CHAR(26)`) exposed in APIs. Records created offline (sales, payments, shifts)
  carry a client-generated `client_uuid` with a unique constraint, which makes sync idempotent.
- Money is `DECIMAL(15,2)`, quantity/weight is `DECIMAL(15,3)`, and rates are `DECIMAL(9,4)`.
  Go uses `shopspring/decimal` through `pkg/money`. **Never use float64 for money.**
- JSON columns are only for real configuration (module config, receipt templates, custom
  attributes). They never replace relational tables.
- **MariaDB / MySQL 8 compatibility:** stay inside the shared SQL subset. Use `JSON` (in
  MariaDB it is a LONGTEXT alias with a `JSON_VALID` check), plain CTEs and window functions,
  and no functional or multi-valued JSON indexes. CI runs against MySQL 8.

---

## 6. Tech choices (backend)

| Concern | Choice |
|---|---|
| Router | `go-chi/chi/v5` |
| DB access | `database/sql` + `jmoiron/sqlx`, hand-written SQL in `*_repo.go` |
| Driver | `go-sql-driver/mysql` (`parseTime=true`) |
| Migrations | `pressly/goose/v3` (SQL files, embedded) |
| Validation | `go-playground/validator/v10` |
| Decimal | `shopspring/decimal` |
| Hashing | `golang.org/x/crypto/argon2` (argon2id for passwords **and** PINs) |
| Logging | `log/slog` JSON to stdout |
| Sessions | Opaque tokens (hashed) in DB, with Redis cache optional later |
| Realtime | SSE for the M-Pesa feed, sync status and notifications |
| Desktop | Wails v2, `modernc.org/sqlite` (pure Go, no CGO) |
| Tests | `testing` + `testify`, integration tests against a real MySQL/MariaDB test DB |

Frontend: pnpm workspace, Vite 8, Svelte 5 (runes), TypeScript 6 strict, SvelteKit 3 with
`adapter-static` for platform/web, plain Vite + Svelte for POS, shared `@mauzo/ui` design tokens.

SvelteKit 3 notes:
- There is no `svelte.config.js`. Kit options are passed to `sveltekit({...})` in `vite.config.ts`.
- Use the package `imports` alias `#lib/*` instead of `$lib`, e.g. `import { api } from '#lib/api.ts'`.
- Shared packages are consumed as TypeScript/Svelte source (no build step).

---

## 7. Getting started (local development)

### Prerequisites

- Go ≥ 1.24 (installed: 1.26)
- Node ≥ 20 (installed: 26) and **pnpm** (`npm i -g pnpm`)
- MariaDB ≥ 10.11 or MySQL 8 (installed: MariaDB 12.3)
- Wails CLI for the POS app: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 runtime (ships with Windows 10/11)

### Database

```sql
CREATE DATABASE mauzo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE mauzo_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'mauzo'@'localhost' IDENTIFIED BY 'change-me';
GRANT ALL ON mauzo.* TO 'mauzo'@'localhost';
GRANT ALL ON mauzo_test.* TO 'mauzo'@'localhost';
```

### Backend

```powershell
cd backend
copy .env.example .env      # then edit DB credentials
go run ./cmd/migrate up
go run ./cmd/migrate seed
go run ./cmd/api            # http://localhost:8080
go run ./cmd/worker         # separate terminal
```

### Frontend

```powershell
cd frontend
pnpm install
pnpm --filter platform dev  # http://localhost:5174
pnpm --filter web dev       # http://localhost:5173
```

### POS desktop

```powershell
cd backend/desktop
wails dev                   # hot reload, uses frontend/apps/pos (port 5175)
wails build                 # → build/bin/MauzoPOS.exe
```

The POS `vite build` writes straight into `backend/desktop/frontend/dist`, which Wails embeds.
Generated JS bindings land in `frontend/apps/pos/src/wailsjs` (git-ignored). Go methods are
exposed as `window.go.bindings.App.*`. Set `MAUZO_API_URL` to point the terminal at a
different API (default `http://localhost:8080`).

### Tests

```powershell
cd backend; go test ./...
cd frontend; pnpm -r test
```

---

## 8. API conventions

- Base path `/api/v1`, JSON in and out, `snake_case` fields.
- Error envelope: `{"error": {"code": "sale_not_open", "message": "...", "details": {...}}}`.
- List endpoints accept `?page=&per_page=&sort=&q=` and return `{data, meta:{page, per_page, total}}`.
- Mutations that move money or stock require an `Idempotency-Key` header.
- Webhooks (`/api/v1/hooks/...`) use source verification (IP allow-list + secret path token)
  and are deduplicated by provider transaction ID.

---

## 9. Docker / production readiness (deferred)

Docker is **not** used during development, but the code is built to drop into it:

- All config comes from env vars (`internal/config`), with `.env` loaded only when `APP_ENV=development`.
- The API and worker are stateless. Files go to an object-storage interface (local disk in dev, S3-compatible in prod).
- `GET /healthz` (liveness) and `GET /readyz` (DB/Redis reachable) exist from Phase 0.
- Migrations run as a separate command (`migrate up`) and never on API boot.
- Graceful shutdown on SIGTERM.
- Later, `deploy/` will hold `Dockerfile.api`, `Dockerfile.worker`, `docker-compose.yml`
  (mysql, redis, api, worker, nginx) and an nginx config serving `web` and `platform` static builds.

---

## 10. Reference material

- `docs/reference/pos-checkout-reference.webp` is the target POS checkout layout: cart on the
  left, category tiles and product grid on the right, function keys (Esc/F5–F10) along the
  bottom, payment method buttons, and the big **Pay** button.
- `docs/reference/receipts/` is for the three receipt photographs (supermarket, minimart with
  M-Pesa, restaurant fiscal/eTIMS). **They still need to be added.**

## 11. Contributing workflow

1. Pick the next unchecked item in [progress.md](progress.md) and mark it 🟡.
2. Build it end to end: migration → model/repo → service (+ tests) → handler/routes →
   permissions → UI → docs.
3. Mark it ✅ only when every column is done. **A screen without a working backend is not done.**
4. Record significant design decisions as an ADR in `docs/adr/`.
