package users

import (
	"context"
	"time"
)

// EmployeeRequest is the body of POST /t/employees and PATCH /t/employees/{id}.
type EmployeeRequest struct {
	Name        *string   `json:"name"`
	Email       *string   `json:"email"`
	Username    *string   `json:"username"`
	Password    *string   `json:"password"`
	PIN         *string   `json:"pin"`
	RoleID      *string   `json:"role_id"`
	AllBranches *bool     `json:"all_branches"`
	Branches    *[]string `json:"branch_ids"`
}

func (r EmployeeRequest) input() EmployeeInput {
	return EmployeeInput(r)
}

// SecretRequest sets a password or PIN.
type SecretRequest struct {
	Value string `json:"value"`
}

// Ref is a compact id+name pair.
type Ref struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

// EmployeeResponse is the API view of an employee (never secrets).
type EmployeeResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       *string    `json:"email"`
	Username    *string    `json:"username"`
	Role        Ref        `json:"role"`
	AllBranches bool       `json:"all_branches"`
	Branches    []Ref      `json:"branches"`
	Status      string     `json:"status"`
	HasPassword bool       `json:"has_password"`
	HasPIN      bool       `json:"has_pin"`
	MFAEnabled  bool       `json:"mfa_enabled"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// branchIndex maps branch ids to refs for responses.
func (s *UserService) branchIndex(ctx context.Context, tenantID uint64) (map[uint64]Ref, error) {
	list, err := s.deps.Branches.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]Ref, len(list))
	for _, b := range list {
		out[b.ID] = Ref{ID: b.PublicID, Name: b.Name, Code: b.Code}
	}
	return out, nil
}

func employeeResponse(u User, branches map[uint64]Ref) EmployeeResponse {
	refs := []Ref{}
	for _, id := range u.BranchIDs {
		if ref, ok := branches[id]; ok {
			refs = append(refs, ref)
		}
	}
	return EmployeeResponse{
		ID: u.PublicID, Name: u.Name, Email: u.Email, Username: u.Username,
		Role:        Ref{ID: u.RolePublicID, Name: u.RoleName, Code: u.Role},
		AllBranches: u.AllBranches, Branches: refs, Status: u.Status,
		HasPassword: u.PasswordHash != nil, HasPIN: u.PINHash != nil, MFAEnabled: u.TOTPEnabled,
		LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt,
	}
}
