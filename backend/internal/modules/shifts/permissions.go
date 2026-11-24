package shifts

import "mauzo/internal/core/authz"

// Tenant-scope permission codes (canonical set from docs/permissions.md).
const (
	PermShiftView       = "shifts.shift.view"
	PermShiftCloseOther = "shifts.shift.close_others"
	PermVarianceApprove = "shifts.variance.approve"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermShiftView, Module: "shifts", Description: "View cashier shifts"},
		authz.Permission{Code: PermShiftCloseOther, Module: "shifts", Description: "Force-close another cashier's shift", Sensitive: true},
		authz.Permission{Code: PermVarianceApprove, Module: "shifts", Description: "Approve or flag a cash variance", Sensitive: true},
	)
}
