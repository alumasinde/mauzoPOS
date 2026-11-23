package registry

import (
	"encoding/json"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/modules"
)

// ModuleResponse describes a module for both the tenant and platform screens.
type ModuleResponse struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Group          string          `json:"group"`
	Core           bool            `json:"core"`
	Depends        []string        `json:"depends"`
	RecommendedFor []string        `json:"recommended_for"`
	Permissions    []string        `json:"permissions"`
	HasConfig      bool            `json:"has_config"`
	ConfigSchema   map[string]any  `json:"config_schema,omitempty"`
	Enabled        *bool           `json:"enabled,omitempty"`
	Effective      *bool           `json:"effective,omitempty"`
	Source         string          `json:"source,omitempty"`
	Reason         string          `json:"reason,omitempty"`
	InPlan         *bool           `json:"in_plan,omitempty"`
	PlatformActive *bool           `json:"platform_active,omitempty"`
	Config         json.RawMessage `json:"config,omitempty"`
}

func describe(m modules.Module) ModuleResponse {
	var perms []string
	for _, p := range authz.All() {
		if p.Module == m.Code {
			perms = append(perms, p.Code)
		}
	}
	return ModuleResponse{
		Code: m.Code, Name: m.Name, Description: m.Description, Group: string(m.Group),
		Core: m.Core, Depends: nonNil(m.Depends), RecommendedFor: nonNil(m.RecommendedFor),
		Permissions: nonNil(perms), HasConfig: m.ConfigSchema != nil, ConfigSchema: m.ConfigSchema,
	}
}

func stateResponse(st ModuleState) ModuleResponse {
	r := describe(st.Module)
	r.Enabled, r.Effective, r.InPlan, r.PlatformActive = &st.Enabled, &st.Effective, &st.InPlan, &st.PlatformActive
	r.Source, r.Reason = st.Source, st.Reason
	if len(st.Config) > 0 && string(st.Config) != "null" {
		r.Config = st.Config
	}
	return r
}

func platformResponse(st PlatformState) ModuleResponse {
	r := describe(st.Module)
	r.PlatformActive = &st.IsActive
	return r
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ToggleRequest enables or disables a module.
type ToggleRequest struct {
	Enabled bool `json:"enabled"`
	// Cascade also disables dependent modules.
	Cascade bool `json:"cascade"`
}

// ToggleResponse reports what changed.
type ToggleResponse struct {
	Changed []string         `json:"changed"`
	Modules []ModuleResponse `json:"modules"`
}

// ActiveRequest flips the platform switch.
type ActiveRequest struct {
	Active bool `json:"active"`
}

// NavResponse is the dynamic sidebar.
type NavResponse struct {
	Items []modules.NavItem `json:"items"`
}
