package terminals

import "mauzo/internal/core/authz"

// Terminal permissions (interim: RequireUserRole enforces them). Platform
// variants are registered for Phase 6 device administration.
const (
	PermTerminalView       = "terminals.terminal.view"
	PermTerminalManage     = "terminals.terminal.manage"
	PermTerminalActivate   = "terminals.terminal.activate"
	PermPlatformTermView   = "platform.terminal.view"
	PermPlatformTermManage = "platform.terminal.manage"
)

func init() {
	authz.Register(PermTerminalView, PermTerminalManage, PermTerminalActivate,
		PermPlatformTermView, PermPlatformTermManage)
}
