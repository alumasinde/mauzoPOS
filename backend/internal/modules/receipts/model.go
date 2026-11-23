package receipts

import (
	"time"
)

// Template is a saved receipt layout (receipt_templates).
type Template struct {
	ID         uint64     `db:"id"`
	TenantID   uint64     `db:"tenant_id"`
	PublicID   string     `db:"public_id"`
	Name       string     `db:"name"`
	PaperWidth int        `db:"paper_width"`
	Body       []byte     `db:"body"`
	Version    int        `db:"version"`
	BasedOn    string     `db:"based_on"`
	ArchivedAt *time.Time `db:"archived_at"`
	CreatedBy  *uint64    `db:"created_by"`
	UpdatedBy  *uint64    `db:"updated_by"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

// Version is one immutable saved layout.
type Version struct {
	TemplateID uint64    `db:"template_id"`
	Version    int       `db:"version"`
	PaperWidth int       `db:"paper_width"`
	Body       []byte    `db:"body"`
	CreatedBy  *uint64   `db:"created_by"`
	AuthorName string    `db:"author_name"`
	CreatedAt  time.Time `db:"created_at"`
}

// Assignment maps a branch (0 = all branches) to a template.
type Assignment struct {
	BranchID         uint64 `db:"branch_id"`
	BranchPublicID   string `db:"branch_public_id"`
	BranchName       string `db:"branch_name"`
	TemplateID       uint64 `db:"template_id"`
	TemplatePublicID string `db:"template_public_id"`
	TemplateName     string `db:"template_name"`
}

// Logo is the stored 1-bit logo.
type Logo struct {
	Width     int       `db:"width"`
	Height    int       `db:"height"`
	Bits      []byte    `db:"bits"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Issued is one receipt handed out (issued_receipts).
type Issued struct {
	ID              uint64    `db:"id"`
	TenantID        uint64    `db:"tenant_id"`
	PublicID        string    `db:"public_id"`
	SaleID          uint64    `db:"sale_id"`
	BranchID        uint64    `db:"branch_id"`
	CopyNo          int       `db:"copy_no"`
	TemplateID      *uint64   `db:"template_id"`
	TemplateVersion *int      `db:"template_version"`
	BuiltinKey      string    `db:"builtin_key"`
	PaperWidth      int       `db:"paper_width"`
	Content         []byte    `db:"content"`
	Channel         string    `db:"channel"`
	TerminalID      *uint64   `db:"terminal_id"`
	IssuedBy        *uint64   `db:"issued_by"`
	SupervisorID    *uint64   `db:"supervisor_id"`
	CreatedAt       time.Time `db:"created_at"`

	// Joined.
	TemplateName string `db:"template_name"`
	IssuedByName string `db:"issued_by_name"`
}

// Channels a receipt is issued through.
const (
	ChannelPOS        = "pos"
	ChannelBackoffice = "backoffice"
)

// Header is the tenant/branch context a receipt needs besides the sale.
type Header struct {
	BusinessName  string `db:"business_name"`
	Address       string `db:"address"`
	Phone         string `db:"phone"`
	Email         string `db:"email"`
	KRAPin        string `db:"kra_pin"`
	VATRegistered bool   `db:"vat_registered"`
	BranchName    string `db:"branch_name"`
	BranchCode    string `db:"branch_code"`
	BranchPhone   string `db:"branch_phone"`
	Location      string `db:"location"`
}
