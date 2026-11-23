package receipts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"mauzo/internal/core/db"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/sales"
	"mauzo/pkg/receiptfmt"
)

// IssueInput is who/where a receipt is issued.
type IssueInput struct {
	Channel    string
	TerminalID *uint64
	UserID     *uint64
}

// Receipt is an issued receipt ready to print or display.
type Receipt struct {
	Issued Issued
	Output receiptfmt.Output
	// Logo is the raster for logo lines (nil when none).
	Logo *receiptfmt.Raster
}

// Issue returns the sale's original receipt, creating its snapshot on first
// call. The snapshot is rendered once with the layout in force at that
// moment and never re-rendered, so later template edits cannot change a
// receipt already handed to a customer.
func (s *Service) Issue(ctx context.Context, sd sales.SaleDetail, in IssueInput) (Receipt, error) {
	tenantID := sd.Sale.TenantID
	if orig, err := s.repo.Original(ctx, tenantID, sd.Sale.ID); err == nil {
		return s.receipt(ctx, orig, "")
	} else if !errors.Is(err, ErrNotIssued) {
		return Receipt{}, err
	}
	set, err := s.settings(ctx, tenantID)
	if err != nil {
		return Receipt{}, err
	}
	res, err := s.resolve(ctx, tenantID, sd.Sale.BranchID, set)
	if err != nil {
		return Receipt{}, err
	}
	doc, logo, err := s.baseDoc(ctx, tenantID, sd.Sale.BranchID, set)
	if err != nil {
		return Receipt{}, err
	}
	if err := s.names(ctx, &sd); err != nil {
		return Receipt{}, err
	}
	fillSale(&doc, sd)
	out := receiptfmt.Render(res.tpl, doc)
	content, err := json.Marshal(out)
	if err != nil {
		return Receipt{}, err
	}
	iss := Issued{
		TenantID: tenantID, PublicID: security.NewPublicID(), SaleID: sd.Sale.ID, BranchID: sd.Sale.BranchID,
		TemplateID: res.templateID, TemplateVersion: res.version, BuiltinKey: res.builtinKey,
		PaperWidth: res.tpl.Paper, Content: content, Channel: in.Channel, TerminalID: in.TerminalID,
		IssuedBy: in.UserID,
	}
	if err := s.repo.InsertIssued(ctx, &iss); err != nil {
		if db.IsDuplicate(err) { // a concurrent call issued it first
			orig, oerr := s.repo.Original(ctx, tenantID, sd.Sale.ID)
			if oerr != nil {
				return Receipt{}, oerr
			}
			return s.receipt(ctx, orig, "")
		}
		return Receipt{}, err
	}
	return Receipt{Issued: iss, Output: out, Logo: logo}, nil
}

// names loads cashier/terminal/customer names when the caller's sale lacks
// them (the detail returned by sales.Complete is not joined).
func (s *Service) names(ctx context.Context, sd *sales.SaleDetail) error {
	if sd.Sale.CashierName != "" || sd.Sale.ID == 0 {
		return nil
	}
	c, t, cu, err := s.repo.SaleNames(ctx, sd.Sale.TenantID, sd.Sale.ID)
	if err != nil {
		return err
	}
	sd.Sale.CashierName, sd.Sale.TerminalName, sd.Sale.CustomerName = c, t, cu
	return nil
}

// ReprintInput identifies the person reprinting. The caller has already
// checked pos.receipt.reprint (or consumed a supervisor grant).
type ReprintInput struct {
	Channel      string
	TerminalID   *uint64
	UserID       *uint64
	SupervisorID *uint64
	ActorType    string // user | terminal
}

// Reprint issues copy N of a sale's receipt: the original snapshot with a
// REPRINT banner, recorded in issued_receipts and the audit log in one
// transaction.
func (s *Service) Reprint(ctx context.Context, sd sales.SaleDetail, in ReprintInput) (Receipt, error) {
	orig, err := s.Issue(ctx, sd, IssueInput{Channel: in.Channel, TerminalID: in.TerminalID, UserID: in.UserID})
	if err != nil {
		return Receipt{}, err
	}
	var copyRow Issued
	err = s.inTx(ctx, func(r *Repo) error {
		n, err := r.NextCopyNo(ctx, sd.Sale.TenantID, sd.Sale.ID)
		if err != nil {
			return err
		}
		copyRow = orig.Issued
		copyRow.ID = 0
		copyRow.PublicID = security.NewPublicID()
		copyRow.CopyNo = n
		copyRow.Channel, copyRow.TerminalID = in.Channel, in.TerminalID
		copyRow.IssuedBy, copyRow.SupervisorID = in.UserID, in.SupervisorID
		if err := r.InsertIssued(ctx, &copyRow); err != nil {
			return err
		}
		actorType := in.ActorType
		if actorType == "" {
			actorType = "user"
		}
		return r.InsertAudit(ctx, sd.Sale.TenantID, sd.Sale.BranchID, actorType, in.UserID, in.SupervisorID,
			"receipt.reprinted", sd.Sale.PublicID, map[string]any{
				"receipt_no": sd.Sale.ReceiptNo, "copy_no": n, "channel": in.Channel,
				"total": sd.Sale.Total.String(),
			})
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.receipt(ctx, copyRow, sd.Sale.Status)
}

// ForSale returns the original receipt of a sale (issuing it lazily for sales
// completed before receipts existed or uploaded from an offline till).
func (s *Service) ForSale(ctx context.Context, sd sales.SaleDetail, userID uint64) (Receipt, error) {
	r, err := s.Issue(ctx, sd, IssueInput{Channel: ChannelBackoffice, UserID: &userID})
	if err != nil {
		return r, err
	}
	if sd.Sale.Status == sales.StatusVoided && !hasBanner(r.Output, "VOIDED") {
		r.Output = receiptfmt.Mark(r.Output, "*** VOIDED ***")
	}
	return r, nil
}

// Copies lists every issued copy of a sale (original + reprints).
func (s *Service) Copies(ctx context.Context, tenantID, saleID uint64) ([]Issued, error) {
	return s.repo.Copies(ctx, tenantID, saleID)
}

// receipt decodes a stored snapshot; reprints get their banner, sales voided
// after issue get a VOIDED banner.
func (s *Service) receipt(ctx context.Context, iss Issued, saleStatus string) (Receipt, error) {
	var out receiptfmt.Output
	if err := json.Unmarshal(iss.Content, &out); err != nil {
		return Receipt{}, err
	}
	if iss.CopyNo > 0 {
		out = receiptfmt.Mark(out, receiptfmt.ReprintBanner(iss.CopyNo))
	}
	if saleStatus == sales.StatusVoided && !hasBanner(out, "VOIDED") {
		out = receiptfmt.Mark(out, "*** VOIDED ***")
	}
	logo, err := s.Logo(ctx, iss.TenantID)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{Issued: iss, Output: out, Logo: logo}, nil
}

func hasBanner(o receiptfmt.Output, word string) bool {
	for _, l := range o.Lines {
		if l.Kind == receiptfmt.LineText && strings.Contains(l.Text, "*** "+word) {
			return true
		}
	}
	return false
}

// PreviewInput renders an unsaved layout.
type PreviewInput struct {
	Body json.RawMessage `json:"body"`
	// BranchID (public) supplies the header; empty = first branch / sample.
	BranchID string `json:"branch_id,omitempty"`
	// Sale (public id) previews a real sale; empty uses sample data.
	Sale string `json:"sale_id,omitempty"`
}

// Preview renders body against sample data or a real sale.
func (s *Service) Preview(ctx context.Context, tenantID uint64, body json.RawMessage, branchID uint64,
	sd *sales.SaleDetail) (receiptfmt.Output, *receiptfmt.Raster, error) {
	tpl, _, err := parseBody(body)
	if err != nil {
		return receiptfmt.Output{}, nil, err
	}
	set, err := s.settings(ctx, tenantID)
	if err != nil {
		return receiptfmt.Output{}, nil, err
	}
	if sd != nil {
		branchID = sd.Sale.BranchID
	}
	doc, logo, err := s.baseDoc(ctx, tenantID, branchID, set)
	if err != nil {
		return receiptfmt.Output{}, nil, err
	}
	if sd != nil {
		fillSale(&doc, *sd)
	} else {
		sample := receiptfmt.SampleDoc()
		doc.Sale, doc.Items, doc.Payments = sample.Sale, sample.Items, sample.Payments
		if doc.Branch.Name == "" {
			doc.Branch = sample.Branch
		}
		if len(doc.Extra) == 0 {
			doc.Extra = sample.Extra
		}
		doc.Flags.Sample = true
	}
	return receiptfmt.Render(tpl, doc), logo, nil
}

// Kit is what a terminal downloads to print (also offline).
func (s *Service) Kit(ctx context.Context, tenantID, branchID uint64) (receiptfmt.Kit, error) {
	set, err := s.settings(ctx, tenantID)
	if err != nil {
		return receiptfmt.Kit{}, err
	}
	res, err := s.resolve(ctx, tenantID, branchID, set)
	if err != nil {
		return receiptfmt.Kit{}, err
	}
	doc, logo, err := s.baseDoc(ctx, tenantID, branchID, set)
	if err != nil {
		return receiptfmt.Kit{}, err
	}
	k := receiptfmt.Kit{
		TemplateID: res.publicID, BuiltinKey: res.builtinKey, Template: res.raw,
		Business: doc.Business, Branch: doc.Branch, Footer: doc.Footer, Extra: doc.Extra,
		PricesIncludeVAT: true, Logo: logo, Policy: set.policy,
	}
	if res.version != nil {
		k.TemplateVersion = *res.version
	}
	return k, nil
}

// BranchID resolves a branch public id ("" → 0).
func (s *Service) BranchID(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	if publicID == "" {
		return 0, nil
	}
	return s.repo.BranchIDByPublicID(ctx, tenantID, publicID)
}

// InvoiceDoc is the full receipt data of a sale for the A4 invoice view.
func (s *Service) InvoiceDoc(ctx context.Context, sd sales.SaleDetail) (receiptfmt.Doc, error) {
	set, err := s.settings(ctx, sd.Sale.TenantID)
	if err != nil {
		return receiptfmt.Doc{}, err
	}
	doc, _, err := s.baseDoc(ctx, sd.Sale.TenantID, sd.Sale.BranchID, set)
	if err != nil {
		return receiptfmt.Doc{}, err
	}
	fillSale(&doc, sd)
	doc.Taxes = receiptfmt.TaxBreakdown(doc.Items, doc.Sale.PricesIncludeVAT)
	return doc, nil
}
