// Package receipts owns receipt layouts and issued receipts: the template
// library and designer API, per-branch assignment, the tenant logo, the
// immutable snapshot of every receipt handed out (original + audited
// reprints) and the "kit" a POS terminal needs to print offline. Layout and
// printer encoding live in pkg/receiptfmt, shared with the desktop app.
package receipts

import "mauzo/internal/core/authz"

// Receipt permissions (canonical set from docs/permissions.md). Printing and
// viewing a sale's receipt are governed by the sales permissions; reprints by
// pos.receipt.reprint (sensitive, audited).
const (
	PermTemplateManage = "receipts.template.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermTemplateManage, Module: "receipts",
			Description: "Design receipt templates, the logo and assign layouts to branches"},
	)
}
