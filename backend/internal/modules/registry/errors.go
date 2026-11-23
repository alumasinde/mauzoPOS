package registry

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Module errors.
var (
	ErrUnknownModule     = httpx.NewError(http.StatusNotFound, "module_not_found", "The module was not found.")
	ErrCoreModule        = httpx.NewError(http.StatusConflict, "module_core", "Core modules are always enabled.")
	ErrModuleUnavailable = httpx.NewError(http.StatusConflict, "module_unavailable", "This module is currently unavailable on the platform.")
	ErrNotInPlan         = httpx.NewError(http.StatusPaymentRequired, "module_not_in_plan", "Your subscription plan does not include this module.")
	ErrHasDependents     = httpx.NewError(http.StatusConflict, "module_has_dependents", "Other enabled modules depend on this one.")
	ErrNoConfig          = httpx.NewError(http.StatusUnprocessableEntity, "module_no_config", "This module has no settings.")
	ErrModuleNotEnabled  = httpx.NewError(http.StatusConflict, "module_not_enabled", "Enable the module before configuring it.")
)
