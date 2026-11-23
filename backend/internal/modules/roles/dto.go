package roles

import (
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/modules"
)

// RoleRequest is the body of POST/PATCH /t/roles. Limits are decimal
// strings; null/empty means unlimited.
type RoleRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Permissions        []string `json:"permissions"`
	MaxDiscountPercent *string  `json:"max_discount_percent"`
	MaxRefundAmount    *string  `json:"max_refund_amount"`
	MaxCashOutAmount   *string  `json:"max_cash_out_amount"`
}

func (r RoleRequest) input() (Input, error) {
	parse := func(field string, s *string) (decimal.NullDecimal, error) {
		if s == nil || strings.TrimSpace(*s) == "" {
			return decimal.NullDecimal{}, nil
		}
		d, err := decimal.NewFromString(strings.TrimSpace(*s))
		if err != nil {
			return decimal.NullDecimal{}, httpx.ErrValidation.WithDetails(map[string]any{field: "must be a number"})
		}
		return decimal.NullDecimal{Decimal: d, Valid: true}, nil
	}
	var in Input
	var err error
	in.Name, in.Description, in.Permissions = r.Name, r.Description, r.Permissions
	if in.Limits.MaxDiscountPercent, err = parse("max_discount_percent", r.MaxDiscountPercent); err != nil {
		return in, err
	}
	if in.Limits.MaxRefundAmount, err = parse("max_refund_amount", r.MaxRefundAmount); err != nil {
		return in, err
	}
	in.Limits.MaxCashOutAmount, err = parse("max_cash_out_amount", r.MaxCashOutAmount)
	return in, err
}

// RoleResponse is the API view of a role.
type RoleResponse struct {
	ID                 string   `json:"id"`
	Code               string   `json:"code"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	IsSystem           bool     `json:"is_system"`
	IsLocked           bool     `json:"is_locked"`
	Customized         bool     `json:"customized"`
	UserCount          int      `json:"user_count"`
	Permissions        []string `json:"permissions"`
	MaxDiscountPercent *string  `json:"max_discount_percent"`
	MaxRefundAmount    *string  `json:"max_refund_amount"`
	MaxCashOutAmount   *string  `json:"max_cash_out_amount"`
}

func decOut(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	s := d.Decimal.String()
	return &s
}

func toResponse(r Role) RoleResponse {
	desc := ""
	if r.Description != nil {
		desc = *r.Description
	}
	perms := r.Permissions
	if perms == nil {
		perms = []string{}
	}
	return RoleResponse{
		ID: r.PublicID, Code: r.Code, Name: r.Name, Description: desc,
		IsSystem: r.IsSystem, IsLocked: r.IsLocked, Customized: r.Customized, UserCount: r.UserCount,
		Permissions:        perms,
		MaxDiscountPercent: decOut(r.MaxDiscountPercent), MaxRefundAmount: decOut(r.MaxRefundAmount),
		MaxCashOutAmount: decOut(r.MaxCashOutAmount),
	}
}

// PermissionItem is one entry of the permission catalogue.
type PermissionItem struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Sensitive   bool   `json:"sensitive"`
}

// PermissionGroup groups permissions by module for the role editor.
type PermissionGroup struct {
	Key         string           `json:"key"`
	Name        string           `json:"name"`
	Module      string           `json:"module,omitempty"`
	Enabled     bool             `json:"enabled"`
	Permissions []PermissionItem `json:"permissions"`
}

// catalogue builds the tenant permission catalogue. Groups follow module
// order; core permissions without a module are grouped by code prefix.
func catalogue(enabled map[string]bool) []PermissionGroup {
	groups := map[string]*PermissionGroup{}
	var keys []string
	for _, p := range authz.All() {
		if p.Scope() != authz.ScopeTenant {
			continue
		}
		key, name, enabledG := p.Module, "", true
		if key != "" {
			m, _ := modules.Get(key)
			name, enabledG = m.Name, enabled[key]
		} else {
			key, _, _ = strings.Cut(p.Code, ".")
			name = map[string]string{"pos": "Point of sale", "audit": "Audit"}[key]
			if name == "" {
				name = strings.ToUpper(key[:1]) + key[1:]
			}
		}
		g, ok := groups[key]
		if !ok {
			g = &PermissionGroup{Key: key, Name: name, Module: p.Module, Enabled: enabledG}
			groups[key] = g
			keys = append(keys, key)
		}
		g.Permissions = append(g.Permissions, PermissionItem{Code: p.Code, Description: p.Description, Sensitive: p.Sensitive})
	}
	order := map[string]int{}
	for i, c := range modules.Codes() {
		order[c] = i
	}
	rank := func(k string) int {
		if i, ok := order[k]; ok {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(keys, func(a, b string) int { return rank(a) - rank(b) })
	out := make([]PermissionGroup, 0, len(keys))
	for _, k := range keys {
		out = append(out, *groups[k])
	}
	return out
}
