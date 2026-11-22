// Package audit writes the append-only audit trail (audit_logs, migration
// 00001). Audit failures are logged but never fail the request.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/identity"
)

// Entry is one audit record. Nil pointers become SQL NULLs.
type Entry struct {
	TenantID     *uint64
	BranchID     *uint64
	ActorType    string
	ActorID      *uint64
	SupervisorID *uint64
	Action       string
	EntityType   string
	EntityID     string
	Data         map[string]any
	IPAddress    string
	UserAgent    string
	RequestID    string
}

// Writer inserts audit rows.
type Writer struct {
	db  *sqlx.DB
	log *slog.Logger
}

// NewWriter builds a Writer.
func NewWriter(db *sqlx.DB, log *slog.Logger) *Writer {
	return &Writer{db: db, log: log}
}

// Log writes an audit entry, reporting errors to the logger instead of the caller.
func (w *Writer) Log(ctx context.Context, e Entry) {
	var data any
	if e.Data != nil {
		b, err := json.Marshal(e.Data)
		if err != nil {
			w.log.Warn("audit: marshal data", "err", err)
		} else {
			data = string(b)
		}
	}
	_, err := w.db.ExecContext(ctx,
		`INSERT INTO audit_logs
		 (tenant_id, branch_id, actor_type, actor_id, supervisor_id, action,
		  entity_type, entity_id, data, ip_address, user_agent, request_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.TenantID, e.BranchID, e.ActorType, e.ActorID, e.SupervisorID, e.Action,
		nilIfEmpty(e.EntityType), nilIfEmpty(e.EntityID), data,
		nilIfEmpty(e.IPAddress), nilIfEmpty(e.UserAgent), nilIfEmpty(e.RequestID))
	if err != nil {
		w.log.Error("audit: insert failed", "action", e.Action, "err", err)
	}
}

// FromRequest fills actor, IP, user-agent and request ID fields from the request
// context. Services call it then set action/entity/data.
func FromRequest(r *http.Request, e Entry) Entry {
	if a, ok := identity.ActorFrom(r.Context()); ok && !a.IsZero() {
		e.ActorType = string(a.Type)
		e.ActorID = &a.ID
		e.TenantID = nilIfZero(a.TenantID)
	}
	e.IPAddress = r.RemoteAddr
	e.UserAgent = r.UserAgent()
	e.RequestID = chimw.GetReqID(r.Context())
	return e
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nilIfZero(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}
