package subscriptions

import "time"

// PlanRequest is POST/PATCH /platform/plans.
type PlanRequest struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PriceKES        string   `json:"price_kes"` // decimal string
	BillingInterval string   `json:"billing_interval"`
	Modules         []string `json:"modules"`
	MaxBranches     *int     `json:"max_branches"`
	MaxUsers        *int     `json:"max_users"`
	IsActive        bool     `json:"is_active"`
	Sort            int      `json:"sort"`
}

// PlanResponse is the API view of a plan.
type PlanResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	PriceKES        string   `json:"price_kes"`
	BillingInterval string   `json:"billing_interval"`
	Modules         []string `json:"modules"`
	MaxBranches     *int     `json:"max_branches"`
	MaxUsers        *int     `json:"max_users"`
	IsActive        bool     `json:"is_active"`
	Sort            int      `json:"sort"`
}

// AssignRequest is POST /platform/tenants/{id}/subscription and onboarding.
type AssignRequest struct {
	PlanID string `json:"plan_id"`
}

// SubscriptionResponse is the API view of a tenant subscription.
type SubscriptionResponse struct {
	PlanID      string     `json:"plan_id"`
	PlanName    string     `json:"plan_name"`
	Status      string     `json:"status"`
	StartsAt    time.Time  `json:"starts_at"`
	TrialEndsAt *time.Time `json:"trial_ends_at"`
}

func planResponse(p Plan) PlanResponse {
	out := PlanResponse{
		ID: p.PublicID, Name: p.Name, PriceKES: p.PriceKES.StringFixed(2),
		BillingInterval: p.BillingInterval, Modules: p.Modules,
		MaxBranches: p.MaxBranches, MaxUsers: p.MaxUsers,
		IsActive: p.IsActive, Sort: p.Sort,
	}
	if p.Description != nil {
		out.Description = *p.Description
	}
	return out
}
