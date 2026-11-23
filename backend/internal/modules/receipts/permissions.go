// Package receipts will own receipt templates, the designer and printing
// (Phase 8). Its permissions are defined now so role templates, the role
// editor and the sidebar are complete before it ships.
package receipts

import "mauzo/internal/core/authz"

// Receipt permissions (canonical set from docs/permissions.md).
const (
	PermTemplateManage = "receipts.template.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermTemplateManage, Module: "receipts", Description: "Design receipt templates"},
	)
}
