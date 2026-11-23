# Permission Catalogue

Format: `module.resource.action`. Each module declares its own permissions in
`backend/internal/modules/<module>/permissions.go`. At boot they register into
`core/authz`, and `migrate seed` syncs them to the DB. Code that uses an unregistered
permission string makes the boot fail.

**S** = *sensitive*. It always writes an audit entry. At the POS, a cashier without the
permission can perform it only after a supervisor enters a PIN.

Standard actions: `view`, `create`, `update`, `delete` (soft-delete or deactivate only),
`export`, `approve`.

**How enforcement works (Phase 3):**
- **Declaration:** each permission is declared with `authz.Define(Permission{Code, Module,
  Description, Sensitive})`.
- **Module gating:** a tenant permission takes effect only while its module is enabled for the
  tenant. `Can()` answers `module_disabled` (403) when the module is off, and `forbidden` (403) when
  the role lacks the permission.
- **Roles:** roles are stored per tenant (`roles`, `role_permissions`). The owner role is locked and
  holds everything. Other built-in roles follow their template until edited.
- **No escalation:** you can only grant, assign or manage what you hold yourself. This covers roles,
  employees and branch access.
- **Status:** ✅ marks permissions registered in code today. The others are reserved here and get
  registered by the phase that builds their module.

See [ADR 0001](adr/0001-module-registry-and-rbac.md) for the design.

---

## 1. Platform context (`platform.*`) — platform administrators only

| Permission | Description |
|---|---|
| `platform.dashboard.view` | Platform overview & analytics |
| `platform.tenant.view` / `.create` / `.update` | Tenant management |
| `platform.tenant.suspend` **S** | Suspend / reactivate a tenant |
| `platform.tenant.impersonate` **S** | Support login into a tenant (time-boxed, audited) |
| `platform.businesstype.view` / `.manage` | Business types & profile templates |
| `platform.module.view` / `.manage` | Module registry & feature flags |
| `platform.subscription.view` / `.manage` | Plans, subscriptions, billing |
| `platform.terminal.view` / `.manage` | Terminal licensing & activation |
| `platform.user.view` / `.manage` | Platform users & roles |
| `platform.config.manage` **S** | Global configuration |
| `platform.integration.manage` **S** | Payment / eTIMS integration config |
| `platform.appversion.manage` | Desktop app versions & release channels |
| `platform.monitoring.view` | System monitoring, sync health |
| `platform.audit.view` | Platform audit logs |

Platform roles: **Super Admin** (all), **Support** (view + impersonate), **Billing** (subscriptions), **Viewer** (view only).

---

## 2. Tenant context

| Module | Permission | S | Description |
|---|---|---|---|
| dashboard | `dashboard.view` | | Business dashboard |
| settings | `settings.business.view` / `.update` | | Business settings |
| settings | `settings.module.manage` ✅ | S | Enable/disable and configure modules (within plan) |
| users | `users.employee.view` / `.create` / `.update` ✅ | | Employees, role and branch access |
| users | `users.employee.deactivate` ✅ | S | Disable / re-enable employees |
| users | `users.employee.reset_password` ✅ | S | Set an employee's password (signs them out) |
| users | `users.employee.reset_pin` ✅ | S | Set an employee's POS PIN |
| roles | `roles.role.view` ✅ / `.manage` ✅ | S (manage) | Roles & permission assignment |
| branches | `branches.branch.view` / `.manage` | | Branches & locations |
| terminals | `terminals.terminal.view` / `.manage` | | POS terminals |
| terminals | `terminals.terminal.activate` | S | Issue an activation code |
| catalog | `catalog.product.view` / `.create` / `.update` / `.delete` | | Products |
| catalog | `catalog.product.import` / `.export` | | Bulk import/export |
| catalog | `catalog.category.manage` | | Categories, brands, UoM, attributes |
| pricing | `pricing.price.view` | | View prices & cost |
| pricing | `pricing.price.update` | S | Change selling prices / price lists |
| pricing | `pricing.cost.view` | | See purchase cost & margins |
| promotions | `promotions.promotion.manage` | | Promotions |
| inventory | `inventory.stock.view` | | Stock balances & movements |
| inventory | `inventory.adjustment.create` | S | Stock adjustments |
| inventory | `inventory.adjustment.approve` | S | Approve adjustments |
| inventory | `inventory.transfer.create` / `.receive` | | Branch transfers |
| inventory | `inventory.stocktake.manage` | | Stocktaking |
| inventory | `inventory.wastage.create` | S | Record wastage |
| purchasing | `purchasing.supplier.manage` | | Suppliers |
| purchasing | `purchasing.order.create` / `.approve` | | Purchase orders |
| purchasing | `purchasing.grn.create` | | Receive goods |
| sales | `sales.sale.view` | | Sales & transactions (own branch) |
| sales | `sales.sale.view_all_branches` | | Cross-branch sales |
| sales | `sales.sale.void` | S | Void a completed sale |
| sales | `sales.refund.create` | S | Returns & refunds |
| customers | `customers.customer.manage` | | Customers |
| payments | `payments.payment.view` | | Payment records |
| mpesa | `mpesa.transaction.view` | | Incoming M-Pesa feed |
| mpesa | `mpesa.transaction.allocate` | | Allocate to a sale (cashier level) |
| mpesa | `mpesa.transaction.reallocate` | S | Fix a wrong allocation |
| mpesa | `mpesa.config.manage` | S | Daraja configuration |
| tax | `tax.category.view` / `.manage` | S | Tax categories & rates |
| etims | `etims.invoice.view` / `.retry` | | Fiscal invoices |
| etims | `etims.config.manage` | S | eTIMS device / config |
| receipts | `receipts.template.manage` | | Receipt designer |
| shifts | `shifts.shift.view` | | Shifts |
| shifts | `shifts.shift.close_others` | S | Force-close another cashier's shift |
| shifts | `shifts.variance.approve` | S | Approve cash variance |
| reports | `reports.sales.view` / `reports.inventory.view` / `reports.tax.view` / `reports.financial.view` | | Reports |
| reports | `reports.any.export` | | Export reports |
| audit | `audit.log.view` | | Tenant audit logs |
| sync | `sync.status.view` / `sync.conflict.resolve` | | Offline sync monitoring |

### POS-specific (checked on the terminal **and** re-checked by the API on sync)

| Permission | S | Description |
|---|---|---|
| `pos.access` | | Can log in to a terminal |
| `pos.sale.create` | | Ring up sales |
| `pos.sale.hold` / `pos.sale.recall` | | Hold / recall sales |
| `pos.sale.cancel` | | Cancel an open (unpaid) sale |
| `pos.line.remove` | S | Remove a scanned line |
| `pos.price.override` | S | Override a line price |
| `pos.discount.apply` | | Discount within the role's max % |
| `pos.discount.override` | S | Discount above the role's max % |
| `pos.drawer.open` | S | Open drawer with no sale |
| `pos.cash.in_out` | S | Cash in / cash out |
| `pos.shift.open` / `pos.shift.close` | | Own shift |
| `pos.receipt.reprint` | S | Reprint a receipt |
| `pos.report.x` / `pos.report.z` | S (Z) | X / Z reports |
| `pos.supervisor.authorize` | | Can authorize **S** actions with their PIN |

Role-level limits (stored on the role and not encoded in the permission): `max_discount_percent`,
`max_refund_amount`, `max_cash_out_amount`.

---

## 3. Default role templates (seeded per tenant, editable)

| Permission group | Owner | Admin | Manager | Accountant | Inventory Mgr | Supervisor | Cashier |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| Settings / modules / roles | ✔ | ✔ | | | | | |
| Employees | ✔ | ✔ | ✔ | | | | |
| Branches / terminals | ✔ | ✔ | ✔ | | | | |
| Catalog & categories | ✔ | ✔ | ✔ | | ✔ | | |
| Price update / cost view | ✔ | ✔ | ✔ | view cost | view cost | | |
| Inventory adjust / approve | ✔ | ✔ | ✔ approve | | ✔ create | | |
| Purchasing | ✔ | ✔ | ✔ | view | ✔ | | |
| Sales view / void / refund | ✔ | ✔ | ✔ | view | | ✔ | |
| M-Pesa allocate / reallocate | ✔ | ✔ | ✔ | ✔ reallocate | | ✔ | allocate |
| Tax / eTIMS config | ✔ | ✔ | | ✔ | | | |
| Shifts / variance | ✔ | ✔ | ✔ | ✔ | | ✔ | own |
| Reports | all | all | all | financial, tax, sales | inventory | sales (branch) | |
| Audit log | ✔ | ✔ | | ✔ | | | |
| POS basic (`pos.access`, sale, hold, shift) | ✔ | ✔ | ✔ | | | ✔ | ✔ |
| POS sensitive + `pos.supervisor.authorize` | ✔ | ✔ | ✔ | | | ✔ | |
