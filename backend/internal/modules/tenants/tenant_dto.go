package tenants

import "time"

// CreateTenantRequest is the POST /platform/tenants body.
type CreateTenantRequest struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// UpdateTenantRequest is the PATCH /platform/tenants/{id} body. Nil means unchanged.
type UpdateTenantRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

// SuspendRequest is the POST /platform/tenants/{id}/suspend body.
type SuspendRequest struct {
	Reason string `json:"reason"`
}

// TenantResponse is the API representation of a tenant.
type TenantResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	Email         string     `json:"email,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Status        string     `json:"status"`
	SuspendedAt   *time.Time `json:"suspended_at,omitempty"`
	SuspendReason string     `json:"suspend_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func toResponse(t Tenant) TenantResponse {
	res := TenantResponse{
		ID:     t.PublicID,
		Name:   t.Name,
		Slug:   t.Slug,
		Status: t.Status,
	}
	if t.Email != nil {
		res.Email = *t.Email
	}
	if t.Phone != nil {
		res.Phone = *t.Phone
	}
	res.SuspendedAt = t.SuspendedAt
	if t.SuspendReason != nil {
		res.SuspendReason = *t.SuspendReason
	}
	res.CreatedAt = t.CreatedAt
	return res
}
