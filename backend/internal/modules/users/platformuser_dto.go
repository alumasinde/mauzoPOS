package users

import "time"

// CreatePlatformUserRequest is the POST /platform/users body.
type CreatePlatformUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdatePlatformUserRequest is the PATCH /platform/users/{id} body.
type UpdatePlatformUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Role  *string `json:"role"`
}

// SetPasswordRequest is the POST /platform/users/{id}/password body.
type SetPasswordRequest struct {
	Password string `json:"password"`
}

// PlatformUserResponse is the API representation (never includes hashes/secrets).
type PlatformUserResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	MFAEnabled  bool       `json:"mfa_enabled"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toPlatformUserResponse(u PlatformUser) PlatformUserResponse {
	return PlatformUserResponse{
		ID:          u.PublicID,
		Name:        u.Name,
		Email:       u.Email,
		Role:        u.Role,
		Status:      u.Status,
		MFAEnabled:  u.TOTPEnabled,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}
