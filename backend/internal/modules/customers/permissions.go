package customers

import "mauzo/internal/core/authz"

// Customer permissions (docs/permissions.md).
const (
	PermCustomerManage = "customers.customer.manage"
	PermCustomerView   = "customers.customer.view"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermCustomerView, Module: "customers", Description: "View customers"},
		authz.Permission{Code: PermCustomerManage, Module: "customers", Description: "Create and edit customers"},
	)
}
