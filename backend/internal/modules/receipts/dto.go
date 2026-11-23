package receipts

import (
	"encoding/json"
	"time"

	"mauzo/pkg/receiptfmt"
)

// TemplateResponse is a layout in the library.
type TemplateResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	PaperWidth int             `json:"paper_width"`
	Version    int             `json:"version"`
	BasedOn    string          `json:"based_on,omitempty"`
	Body       json.RawMessage `json:"body"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func toTemplate(t Template) TemplateResponse {
	return TemplateResponse{ID: t.PublicID, Name: t.Name, PaperWidth: t.PaperWidth, Version: t.Version,
		BasedOn: t.BasedOn, Body: json.RawMessage(t.Body), UpdatedAt: t.UpdatedAt}
}

// VersionResponse is one saved version.
type VersionResponse struct {
	Version    int             `json:"version"`
	PaperWidth int             `json:"paper_width"`
	Author     string          `json:"author,omitempty"`
	Body       json.RawMessage `json:"body"`
	CreatedAt  time.Time       `json:"created_at"`
}

// AssignmentResponse is one branch → layout row. BranchID "" = all branches.
type AssignmentResponse struct {
	BranchID     string `json:"branch_id"`
	BranchName   string `json:"branch_name"`
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
}

// AssignRequest sets or clears (template_id "") a branch's layout.
type AssignRequest struct {
	BranchID   string `json:"branch_id"`
	TemplateID string `json:"template_id"`
}

// BuiltinResponse is a shipped starting layout.
type BuiltinResponse struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Body        json.RawMessage `json:"body"`
}

// NamedDoc documents a variable or condition.
type NamedDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MetaResponse feeds the designer.
type MetaResponse struct {
	Builtins      []BuiltinResponse `json:"builtins"`
	Variables     []NamedDoc        `json:"variables"`
	Conditions    []NamedDoc        `json:"conditions"`
	SchemaVersion int               `json:"schema_version"`
	HasLogo       bool              `json:"has_logo"`
}

// PreviewResponse is a rendered layout.
type PreviewResponse struct {
	Output  receiptfmt.Output `json:"output"`
	HasLogo bool              `json:"has_logo"`
}

// IssuedResponse describes one issued copy.
type IssuedResponse struct {
	ID              string    `json:"id"`
	CopyNo          int       `json:"copy_no"`
	Channel         string    `json:"channel"`
	TemplateName    string    `json:"template_name,omitempty"`
	TemplateVersion *int      `json:"template_version,omitempty"`
	BuiltinKey      string    `json:"builtin_key,omitempty"`
	IssuedBy        string    `json:"issued_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func toIssued(i Issued) IssuedResponse {
	return IssuedResponse{ID: i.PublicID, CopyNo: i.CopyNo, Channel: i.Channel, TemplateName: i.TemplateName,
		TemplateVersion: i.TemplateVersion, BuiltinKey: i.BuiltinKey, IssuedBy: i.IssuedByName, CreatedAt: i.CreatedAt}
}

// ReceiptResponse is a printable receipt with its history.
type ReceiptResponse struct {
	Output  receiptfmt.Output `json:"output"`
	HasLogo bool              `json:"has_logo"`
	Issued  IssuedResponse    `json:"issued"`
	Copies  []IssuedResponse  `json:"copies,omitempty"`
}
