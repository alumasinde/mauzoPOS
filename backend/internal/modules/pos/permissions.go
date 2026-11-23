// Package pos will hold the terminal-side API (Phase 6). Its permissions are
// defined now so role templates and the role editor are complete; the API
// re-checks them when POS endpoints land and when offline sales sync.
package pos

import "mauzo/internal/core/authz"

// POS permissions (see docs/permissions.md).
const (
	PermAccess              = "pos.access"
	PermSaleCreate          = "pos.sale.create"
	PermSaleHold            = "pos.sale.hold"
	PermSaleRecall          = "pos.sale.recall"
	PermSaleCancel          = "pos.sale.cancel"
	PermLineRemove          = "pos.line.remove"
	PermPriceOverride       = "pos.price.override"
	PermDiscountApply       = "pos.discount.apply"
	PermDiscountOverride    = "pos.discount.override"
	PermDrawerOpen          = "pos.drawer.open"
	PermCashInOut           = "pos.cash.in_out"
	PermShiftOpen           = "pos.shift.open"
	PermShiftClose          = "pos.shift.close"
	PermReceiptReprint      = "pos.receipt.reprint"
	PermReportX             = "pos.report.x"
	PermReportZ             = "pos.report.z"
	PermSupervisorAuthorize = "pos.supervisor.authorize"
)

func init() {
	p := func(code, desc string, sensitive bool) authz.Permission {
		return authz.Permission{Code: code, Description: desc, Sensitive: sensitive}
	}
	authz.Define(
		p(PermAccess, "Sign in to a POS terminal", false),
		p(PermSaleCreate, "Ring up sales", false),
		p(PermSaleHold, "Hold sales", false),
		p(PermSaleRecall, "Recall held sales", false),
		p(PermSaleCancel, "Cancel an unpaid sale", false),
		p(PermLineRemove, "Remove a scanned line", true),
		p(PermPriceOverride, "Override a line price", true),
		p(PermDiscountApply, "Apply discounts within the role limit", false),
		p(PermDiscountOverride, "Apply discounts above the role limit", true),
		p(PermDrawerOpen, "Open the cash drawer without a sale", true),
		p(PermCashInOut, "Record cash in / cash out", true),
		p(PermShiftOpen, "Open own shift", false),
		p(PermShiftClose, "Close own shift", false),
		p(PermReceiptReprint, "Reprint a receipt", true),
		p(PermReportX, "Print X report", false),
		p(PermReportZ, "Print Z report", true),
		p(PermSupervisorAuthorize, "Authorise sensitive actions with a supervisor PIN", false),
	)
}
