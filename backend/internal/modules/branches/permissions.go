package branches

import "mauzo/internal/core/authz"

// Branch permissions (interim: RequireUserRole enforces them).
const (
	PermBranchView   = "branches.branch.view"
	PermBranchManage = "branches.branch.manage"
)

func init() {
	authz.Register(PermBranchView, PermBranchManage)
}
