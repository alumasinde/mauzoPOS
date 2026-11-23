package branches

import "mauzo/internal/core/authz"

// Branch permissions.
const (
	PermBranchView   = "branches.branch.view"
	PermBranchManage = "branches.branch.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermBranchView, Module: "branches", Description: "View branches"},
		authz.Permission{Code: PermBranchManage, Module: "branches", Description: "Create and edit branches"},
	)
}
