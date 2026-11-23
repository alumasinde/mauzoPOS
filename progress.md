# Mauzo POS — Progress Tracker

**Legend:** ⬜ Not started · 🟡 In progress · ✅ Done · ⛔ Blocked · ➖ Not applicable

**Columns:** **DB** = schema + migration + seed · **API** = repo, service, handler, routes,
permissions, validation · **UI** = working Svelte screen(s) wired to the API · **Tests** =
automated unit/integration tests · **Status** = overall (✅ only when every applicable
column is ✅ and docs are updated).

> Rule: a feature that only has a visual interface is **not done**. Money, payment,
> inventory, tax and authorization features must work end to end before you tick them.

---

## Summary

| # | Phase | Status |
|---|---|---|
| 0 | Project foundation & tooling | 🟡 (CI not yet run — no remote) |
| 1 | Platform foundation, tenants & authentication | ✅ |
| 2 | Business profiles & onboarding | ✅ |
| 3 | Module registry & permissions | ⬜ |
| 4 | Products, categories & barcodes | ⬜ |
| 5 | Inventory & pricing | ⬜ |
| 6 | POS desktop & checkout | ⬜ |
| 7 | Sales & payments | ⬜ |
| 8 | Receipt printing & designer | ⬜ |
| 9 | M-Pesa integrations | ⬜ |
| 10 | Tax engine & eTIMS | ⬜ |
| 11 | Purchasing & suppliers | ⬜ |
| 12 | Branches & cashier shifts | ⬜ |
| 13 | Reports & analytics | ⬜ |
| 14 | Offline synchronization | ⬜ |
| 15 | Production deployment & monitoring (Docker) | ⬜ |
| 16 | Industry modules (restaurant, butchery, loyalty…) | ⬜ |

---

## Phase 0 — Project foundation & tooling

| Feature | Status |
|---|---|
| Repo structure (`backend/`, `frontend/`, `docs/`) | ✅ |
| README.md, progress.md, docs/permissions.md | ✅ |
| `backend/go.mod`, small `cmd/api/main.go`, `cmd/worker`, `cmd/migrate` | ✅ |
| `internal/config` (env vars + `.env` in dev) | ✅ |
| `internal/app` container wiring + graceful shutdown | ✅ |
| `internal/routes/app.go` master router (`/platform`, `/t`, `/pos`, `/hooks` contexts) | ✅ |
| `core/db` pool, `WithTx`, tenant-scoped querier (+ integration tests) | ✅ |
| `core/httpx` JSON responses, error envelope, strict decode, pagination (+ tests) | ✅ |
| Middleware: request ID, slog logging, recover, CORS, rate limit (+ tests) | ✅ |
| `/healthz` and `/readyz` (+ tests, verified live against MariaDB) | ✅ |
| goose migrations (embedded) + `migrate up/down/status/seed/create` | ✅ |
| `core` audit_logs table (migration 00001) | ✅ |
| Background worker runner (`internal/worker`, panic-safe, + tests) | ✅ |
| `pkg/money` decimal helpers, rounding modes, allocation, cash increments (+ tests) | ✅ |
| Integration-test harness against the `mauzo_test` DB (`internal/testutil`) | ✅ |
| Frontend pnpm workspace (`packages/ui`, `api-client`, `types`, `utils`) | ✅ |
| Design tokens + base components (Button, TextField, DataTable, Modal, Toaster, Spinner, StatusDot) | ✅ |
| Scaffold `apps/platform`, `apps/web`, `apps/pos` (Svelte 5 + TS strict, SvelteKit 3) | ✅ |
| Wails shell `backend/desktop` building `MauzoPOS.exe` (Go↔UI binding, API check) | ✅ |
| Lint/format: `gofmt`, ESLint, Prettier, `svelte-check` passing | ✅ |
| `golangci-lint` config (`backend/.golangci.yml`) — run locally, **0 issues** | ✅ |
| `.gitignore`, `.gitattributes` | ✅ |
| CI pipeline `.github/workflows/ci.yml` — written, not yet run (no remote) | 🟡 |

## Phase 1 — Platform foundation, tenants & authentication

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| `core/security`: argon2id hashing, token generation | ➖ | ✅ | ➖ | ✅ | ✅ |
| Platform users (CRUD, roles) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Platform login / logout / sessions | ✅ | ✅ | ✅ | ✅ | ✅ |
| Platform MFA (TOTP) for privileged users | ✅ | ✅ | ✅ | ✅ | ✅ |
| Password recovery (email token) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Login history & session management (revoke) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tenants: create / activate / suspend / view | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tenant context middleware (tenant from token, never from the request) | ➖ | ✅ | ➖ | ✅ | ✅ |
| Tenant employee login (email/username + password) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tenant optional MFA | ✅ | ✅ | ✅ | ✅ | ✅ |
| Audit log writer + platform audit log viewer | ✅ | ✅ | ✅ | ✅ | ✅ |
| Login rate limiting / lockout | ✅ | ✅ | ➖ | ✅ | ✅ |
| Platform dashboard shell (layout, nav, auth guard) | ➖ | ➖ | ✅ | ➖ | ✅ |
| Back-office shell (layout, auth guard) | ➖ | ➖ | ✅ | ➖ | ✅ |

## Phase 2 — Business profiles & onboarding

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Business types (supermarket, minimart, butchery, grocery, liquor, restaurant, custom) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Business profile templates (default modules, categories, settings, receipt, POS layout) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Profile Engine: apply profile to tenant (idempotent) | ✅ | ✅ | ➖ | ✅ | ✅ |
| Subscription plans & tenant subscriptions | ✅ | ✅ | ✅ | ✅ | ✅ |
| Onboarding wizard step 1–3: business info, type, profile | ✅ | ✅ | ✅ | ✅ | ✅ |
| Wizard step 4–5: plan, enabled modules | ✅ | ✅ | ✅ | ✅ | ✅ |
| Wizard step 6–7: branches, initial administrator | ✅ | ✅ | ✅ | ✅ | ✅ |
| Wizard step 8–10: tax, payment methods, receipt config | ✅ | ✅ | ✅ | ✅ | ✅ |
| Wizard step 11–12: terminal config, activation | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tenant settings (key/value + JSON-schema validation) | ✅ | ✅ | ✅ | ✅ | ✅ |

## Phase 3 — Module registry & permissions

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Module registry (id, deps, permissions, config schema, availability, plan) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Dependency resolution on enable/disable | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Tenant module overrides + config | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| `RequireModule` middleware (backend enforcement) | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| Permission registry (`permissions.go` per module → DB sync, boot validation) | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Roles CRUD + default role templates per tenant | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| `Can(perm)` middleware + branch-scope check | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| Employees: CRUD, roles, branch assignment | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Dynamic navigation API (modules × permissions) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Platform: module & feature management screen | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 4 — Products, categories & barcodes

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Categories / departments (tree) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Brands | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Units of measure + conversions | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Products: standard, service, packaged | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Variable-weight products (price per kg) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Product variants | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Composite / deposit products | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Multiple barcodes per product (+ pack-size barcodes) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| `pkg/barcode`: EAN/UPC validation, weighted-barcode parsing rules | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Custom attributes | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Product images (storage interface) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Import / export (CSV/XLSX) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Barcode label printing | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 5 — Inventory & pricing

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Stock locations per branch | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Stock movement ledger (only way to change stock) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Stock balances (per branch / location) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Stock adjustments (reason codes, approval) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Stock transfers between branches | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Stocktaking / cycle counts | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Batch & expiry tracking + expiry alerts | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Wastage | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Reorder levels + low-stock alerts | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Inventory valuation (weighted average cost) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Price lists (per branch / customer group) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Bulk / tiered pricing | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Promotions engine (`pkg/pricecalc`) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Price change history + audit | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 6 — POS desktop & checkout

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Terminal registration & activation (device identity, licence) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Local SQLite schema (`desktop/localdb`) | ⬜ | ➖ | ➖ | ⬜ | ⬜ |
| Initial catalog / price / tax / cashier download | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Cashier selection + PIN login (argon2id, lockout) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Auto-lock + cashier switching | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Supervisor PIN authorization | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Checkout screen layout (per reference image) | ➖ | ➖ | ⬜ | ⬜ | ⬜ |
| Persistent scanner focus + HID scanner handling | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Barcode lookup (local) + auto add / qty increment | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Unknown barcode handling | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Weighted / price-embedded barcode parsing | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Product search + category tiles | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Cart: qty, remove, line discount | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Price check (F6) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Hold sale (F5) / recall held sales | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Keyboard shortcuts (Esc, F5–F10) | ➖ | ➖ | ⬜ | ⬜ | ⬜ |
| Business-type adaptive POS layout | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Online / offline / printer status indicators | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Terminal settings screen | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 7 — Sales & payments

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Sale lifecycle (draft → active → held → pending payment → completed / cancelled / returned) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Sale completion in one DB transaction (sale + items + payments + stock + audit) | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Idempotency keys / duplicate-sale prevention | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Payment engine (method registry) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Cash payment + change | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Card payment (reference capture) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Bank transfer / customer credit | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Split payments | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Order / cart discounts (permission + supervisor) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Voids (controlled, audited) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Returns & refunds (linked to original sale) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Customers (basic accounts) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Transaction lookup (POS + back-office) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Back-office sales list & sale detail | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 8 — Receipt printing & designer

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Receipt reference photos added to `docs/reference/receipts/` | ➖ | ➖ | ➖ | ➖ | ⛔ |
| Versioned JSON receipt template schema | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| `pkg/receiptfmt` render tree (header, transaction, items, payments, tax, fiscal, footer, conditionals) | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| Default templates: supermarket, minimart (M-Pesa), restaurant (fiscal) | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Branch-specific templates | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Visual receipt designer + live preview | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Issued-receipt snapshot (content + template version) | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| ESC/POS driver 58mm / 80mm (USB, network, spooler) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Logo + QR code printing | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| Auto print on completion + reprint (audited) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| A4 invoice / PDF receipt | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Printer configuration screen | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 9 — M-Pesa integrations (Daraja)

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Per-tenant Daraja credentials (encrypted, server-only) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| C2B URL registration (Till / Paybill) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| C2B confirmation/validation callbacks (dedupe by transaction code) | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Incoming payment feed to POS (SSE) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Manual allocation to sale (all verification rules) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Matching suggestions (never amount-only proof) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| STK Push initiate | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| STK callback handling + status query fallback | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Reconciliation screen (unallocated, allocated, disputed) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Refund / reversal tracking | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 10 — Tax engine & eTIMS

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Tax categories (standard, zero-rated, exempt, non-VAT, other) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Effective-dated tax rates | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| `pkg/taxcalc`: inclusive/exclusive, discounts, rounding, breakdowns | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| Tax snapshot on each sale line | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Credit notes / reversals | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| **Verify current KRA eTIMS requirements (OSCU/VSCU)** | ➖ | ➖ | ➖ | ➖ | ⬜ |
| eTIMS integration layer (device init, item/code sync, invoice submit) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Fiscal invoice records (CU/SCU info, signature, QR from KRA only) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Submission queue, retries, status tracking, reconciliation | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Buyer PIN capture at POS | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 11 — Purchasing & suppliers

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Suppliers | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Purchase orders (draft → approved → received / partial) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Goods received notes (stock in + cost update + batches) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Supplier returns | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Supplier invoices & balances | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 12 — Branches & cashier shifts

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Branches CRUD + branch settings | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Branch-level access enforcement | ➖ | ⬜ | ➖ | ⬜ | ⬜ |
| POS terminal management (back-office) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Shift open (float) / close (blind count) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Cash in / cash out / drawer open (audited) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| X-report / Z-report | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Shift variance review | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 13 — Reports & analytics

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Business dashboard widgets (configurable) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Sales reports (by day, product, category, cashier, branch, payment) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Tax / VAT report | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Inventory reports (valuation, movement, expiry, dead stock) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| M-Pesa reconciliation report | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Consolidated multi-branch reports | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Export (CSV / XLSX / PDF) | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Platform analytics & tenant usage stats | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 14 — Offline synchronization

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Local transactional outbox (SQLite) | ⬜ | ➖ | ➖ | ⬜ | ⬜ |
| Offline cash sales with client UUIDs | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Upload endpoint: idempotent, acknowledgements | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Delta download (catalog, prices, tax, cashiers, config) by version cursor | ⬜ | ⬜ | ➖ | ⬜ | ⬜ |
| Retry with backoff + dead-letter view | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Conflict detection (stock oversell, price change) + review | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Offline M-Pesa never marked confirmed | ➖ | ⬜ | ⬜ | ⬜ | ⬜ |
| Sync status indicator (POS) + monitoring (back-office/platform) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

## Phase 15 — Production deployment & monitoring

| Feature | Status |
|---|---|
| `deploy/Dockerfile.api`, `Dockerfile.worker` (multi-stage, distroless) | ⬜ |
| `deploy/docker-compose.yml` (mysql, redis, api, worker, nginx) | ⬜ |
| nginx: TLS, static `web` + `platform`, `/api` proxy, SSE settings | ⬜ |
| Secrets management (env / vault), no secrets in images | ⬜ |
| Redis (sessions cache, rate limit, SSE fan-out) | ⬜ |
| Background job queue for workers | ⬜ |
| Structured logs + error tracking | ⬜ |
| Metrics & uptime monitoring | ⬜ |
| Automated DB backups + **tested restore** | ⬜ |
| CI/CD automated deployment | ⬜ |
| App version management + POS auto-update channel | ⬜ |
| Code signing for the POS .exe + installer | ⬜ |

## Phase 16 — Industry & optional modules

| Feature | DB | API | UI | Tests | Status |
|---|---|---|---|---|---|
| Weighing scale integration (serial) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Butchery: cuts, yield tracking, carcass breakdown | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Restaurant: tables & floor plan | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Restaurant: menu management & modifiers | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Restaurant: kitchen orders (KOT / KDS) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Recipes (stock deduction by ingredients) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Service charge, waiters/servers | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Liquor: age-check prompt, licence hours | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Loyalty & rewards | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Customer credit accounts | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Delivery management | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

---

## Change log

| Date | Change |
|---|---|
| 2026-11-22 | Project initialised: README.md, progress.md, docs/permissions.md, `.env.example`, POS reference image |
| 2026-11-22 | Phase 0: Go backend skeleton (config, db, httpx, middleware, health, migrations, worker, `pkg/money`), pnpm workspace with 4 shared packages, web/platform/pos apps, Wails POS .exe, lint + CI config. Go tests incl. DB integration pass; 14 frontend tests pass; all apps build. |
| 2026-11-22 | `golangci-lint run` → 0 issues. Fixed 58 findings: errcheck (deferred `Close`), errorlint (`errors.Is` for `http.ErrAbortHandler`), gocritic exitAfterDefer (`cmd/*` restructured to `run() error`), revive exported doc comments. |
| 2026-11-23 | Phase 1 (mostly): `core/security` (argon2id, tokens, TOTP, ULID), `core/identity`, `core/authz`, auth middleware, `core/mail`, `core/audit`; migrations 00002–00004 (tenants+slug, auth sessions/history/resets, platform+tenant users); modules tenants/auth/users/audit with platform + tenant auth contexts; foundation seed (`admin@mauzo.local` / `ChangeMe123!`, demo tenant `demo` / `owner@demo.local`); 5 auth integration tests vs `mauzo_test`; platform app (login+MFA challenge, forgot/reset, shell, tenants, users, audit) + web app (slug login+MFA, forgot/reset, shell, audit). Verified live via Vite proxy: platform + tenant login, `/me`. All checks green: `golangci-lint` 0 issues, `go test`, `svelte-check`, eslint+prettier, `pnpm -r build`. |
| 2026-11-23 | Phase 1 completed: `MfaPanel` (shared TOTP setup/disable, recovery codes) + Account page (sessions list + revoke, sign-in history) in both apps; tenant edit modal (PATCH); integration tests for tenants/users/audit; `testutil` serialized on a MySQL named lock to stop cross-package truncation races; `audit.Log.Data` NULL-scan fix. |
| 2026-11-23 | Phase 2: migrations 00005–00007 (business_types/profiles, plans/tenant_subscriptions, tenant cols + tenant_settings + tenant_modules + branches + pos_terminals); `core/modules` catalogue, `core/settings` key/value store + JSON-schema validator; modules businesstypes (profile CRUD + idempotent apply engine), subscriptions (plan CRUD + assign/cancel + trial), settings (`/t/settings`), branches + terminals (minimal create/list), onboarding (8-step wizard state machine: business → profile → plan → modules → branch → admin → preferences → terminal → complete, guard: branch+terminal required); `ONBOARDING_TRIAL_DAYS` config; catalogue seed (7 business types + default profiles, 3 plans); platform UI (Business types, Plans); web UI (onboarding wizard `/onboarding`, `/settings`, dashboard banner). Integration test walks the whole wizard against `mauzo_test`. Verified live: full wizard run end-to-end on demo tenant. `golangci-lint` 0 issues, all tests green, frontend check/lint/build green. |
| 2026-11-23 | Review fixes. **Auth:** MFA tickets revoked after `LOGIN_MAX_ATTEMPTS` bad codes and redeemable once (conditional `MarkMFAVerified`); password checked before account/tenant status; argon2 timing equaliser for unknown/PIN-only identities; lockout map pruned and bounded; reset tokens consumed atomically before the password change; reset mail goes to `Account.Email` only (username-only/PIN-only accounts skipped). **Users:** usernames may not contain `@`, emails must. **Tests:** `testutil` refuses a `TEST_DB_NAME` equal to `DB_NAME` or without "test". **Profiles:** `Apply` and profile save run in transactions, DB errors no longer reported as 404, `SetDefaults` no longer uses `INSERT IGNORE`. **DB:** migration 00008 composite FK `pos_terminals(tenant_id, branch_id) → branches(tenant_id, id)` (validated up/down on `mauzo_test`, applied to dev). Build + vet clean; tests not run (new `lockout_test.go` included). |
