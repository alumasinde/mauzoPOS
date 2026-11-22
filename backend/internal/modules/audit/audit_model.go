// Package audit exposes the append-only audit trail to platform admins and
// (interim, until Phase 3 roles) tenant owners/admins.
package audit

import (
	"encoding/json"
	"time"
)

// Log is one audit_logs row.
type Log struct {
	ID           uint64           `db:"id"`
	TenantID     *uint64          `db:"tenant_id"`
	BranchID     *uint64          `db:"branch_id"`
	ActorType    string           `db:"actor_type"`
	ActorID      *uint64          `db:"actor_id"`
	SupervisorID *uint64          `db:"supervisor_id"`
	Action       string           `db:"action"`
	EntityType   *string          `db:"entity_type"`
	EntityID     *string          `db:"entity_id"`
	Data         *json.RawMessage `db:"data"`
	IPAddress    *string          `db:"ip_address"`
	UserAgent    *string          `db:"user_agent"`
	RequestID    *string          `db:"request_id"`
	CreatedAt    time.Time        `db:"created_at"`
}

// Response is the API representation of a Log row.
type Response struct {
	ID         uint64          `json:"id"`
	ActorType  string          `json:"actor_type"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type,omitempty"`
	EntityID   string          `json:"entity_id,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	IP         string          `json:"ip_address,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

func toResponse(l Log) Response {
	r := Response{ID: l.ID, ActorType: l.ActorType, Action: l.Action, CreatedAt: l.CreatedAt}
	if l.Data != nil {
		r.Data = *l.Data
	}
	if l.EntityType != nil {
		r.EntityType = *l.EntityType
	}
	if l.EntityID != nil {
		r.EntityID = *l.EntityID
	}
	if l.IPAddress != nil {
		r.IP = *l.IPAddress
	}
	if l.UserAgent != nil {
		r.UserAgent = *l.UserAgent
	}
	return r
}
