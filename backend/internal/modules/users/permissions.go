package users

import "mauzo/internal/core/authz"

// User permissions: platform operators and tenant employees.
const (
	PermPlatformUserView   = "platform.user.view"
	PermPlatformUserManage = "platform.user.manage"

	PermEmployeeView          = "users.employee.view"
	PermEmployeeCreate        = "users.employee.create"
	PermEmployeeUpdate        = "users.employee.update"
	PermEmployeeDeactivate    = "users.employee.deactivate"
	PermEmployeeResetPassword = "users.employee.reset_password"
	PermEmployeeResetPIN      = "users.employee.reset_pin"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermPlatformUserView, Description: "View platform operators"},
		authz.Permission{Code: PermPlatformUserManage, Description: "Manage platform operators", Sensitive: true},

		authz.Permission{Code: PermEmployeeView, Module: "users", Description: "View employees"},
		authz.Permission{Code: PermEmployeeCreate, Module: "users", Description: "Add employees"},
		authz.Permission{Code: PermEmployeeUpdate, Module: "users", Description: "Edit employees, their role and branch access"},
		authz.Permission{Code: PermEmployeeDeactivate, Module: "users", Description: "Disable and re-enable employees", Sensitive: true},
		authz.Permission{Code: PermEmployeeResetPassword, Module: "users", Description: "Set an employee's password", Sensitive: true},
		authz.Permission{Code: PermEmployeeResetPIN, Module: "users", Description: "Set an employee's POS PIN", Sensitive: true},
	)
}
