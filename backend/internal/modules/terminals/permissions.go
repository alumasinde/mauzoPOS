package terminals

import "mauzo/internal/core/authz"

// Terminal permissions.
const (
	PermTerminalView       = "terminals.terminal.view"
	PermTerminalManage     = "terminals.terminal.manage"
	PermTerminalActivate   = "terminals.terminal.activate"
	PermPlatformTermView   = "platform.terminal.view"
	PermPlatformTermManage = "platform.terminal.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermTerminalView, Module: "terminals", Description: "View POS terminals"},
		authz.Permission{Code: PermTerminalManage, Module: "terminals", Description: "Create and edit POS terminals"},
		authz.Permission{Code: PermTerminalActivate, Module: "terminals", Description: "Issue terminal activation codes", Sensitive: true},
		authz.Permission{Code: PermPlatformTermView, Description: "View terminal licences"},
		authz.Permission{Code: PermPlatformTermManage, Description: "Manage terminal licences"},
	)
}
