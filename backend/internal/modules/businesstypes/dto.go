package businesstypes

import "time"

// ProfileRequest is POST/PATCH /platform/business-profiles.
type ProfileRequest struct {
	BusinessType string            `json:"business_type"` // create only
	Name         string            `json:"name"`
	IsDefault    bool              `json:"is_default"`
	Definition   ProfileDefinition `json:"definition"`
}

// TypeResponse is the API view of a business type.
type TypeResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ProfileResponse is the API view of a template.
type ProfileResponse struct {
	ID           string            `json:"id"`
	BusinessType string            `json:"business_type"`
	Name         string            `json:"name"`
	IsDefault    bool              `json:"is_default"`
	Definition   ProfileDefinition `json:"definition"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func typeResponse(t BusinessType) TypeResponse {
	return TypeResponse{Code: t.Code, Name: t.Name}
}

func profileResponse(p Profile) ProfileResponse {
	return ProfileResponse{
		ID: p.PublicID, BusinessType: p.TypeCode, Name: p.Name,
		IsDefault: p.IsDefault, Definition: p.Definition, UpdatedAt: p.UpdatedAt,
	}
}
