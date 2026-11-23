package roles

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Role errors.
var (
	ErrRoleNotFound     = httpx.NewError(http.StatusNotFound, "role_not_found", "The role was not found.")
	ErrRoleLocked       = httpx.NewError(http.StatusConflict, "role_locked", "The owner role cannot be changed.")
	ErrSystemRole       = httpx.NewError(http.StatusConflict, "role_system", "Built-in roles cannot be deleted.")
	ErrRoleInUse        = httpx.NewError(http.StatusConflict, "role_in_use", "Reassign this role's employees before deleting it.")
	ErrRoleNameTaken    = httpx.NewError(http.StatusConflict, "role_name_taken", "A role with that name already exists.")
	ErrNotTemplated     = httpx.NewError(http.StatusConflict, "role_no_template", "Only built-in roles can be reset to their template.")
	ErrUnknownPerm      = httpx.NewError(http.StatusUnprocessableEntity, "permission_unknown", "Unknown permission.")
	ErrGrantEscalation  = httpx.NewError(http.StatusForbidden, "permission_escalation", "You can only grant permissions you hold yourself.")
	ErrInvalidRoleLimit = httpx.NewError(http.StatusUnprocessableEntity, "role_limit_invalid", "Limits must be zero or positive; discount at most 100%.")
)
