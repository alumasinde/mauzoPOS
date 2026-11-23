package inventory

import (
	"context"
	"slices"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
)

// ---- adjustments ---------------------------------------------------------------

// AdjustmentRequest is the create payload.
type AdjustmentRequest struct {
	BranchID string                  `json:"branch_id"` // public id
	Reason   string                  `json:"reason"`
	Note     string                  `json:"note"`
	Items    []AdjustmentItemRequest `json:"items"`
}

// AdjustmentItemRequest is one adjustment line (qty is signed).
type AdjustmentItemRequest struct {
	ProductID  string  `json:"product_id"`
	VariantID  string  `json:"variant_id,omitempty"`
	LocationID string  `json:"location_id,omitempty"`
	BatchNo    string  `json:"batch_no,omitempty"`
	Expiry     *string `json:"expiry,omitempty"`
	Qty        string  `json:"qty"` // signed delta
}

// CreateAdjustment drafts a pending adjustment.
func (s *Service) CreateAdjustment(ctx context.Context, actor identity.Actor, req AdjustmentRequest) (Adjustment, error) {
	if !slices.Contains(AdjustReasons, req.Reason) {
		return Adjustment{}, ErrBadReason
	}
	branchID, err := s.branch(ctx, actor, req.BranchID)
	if err != nil {
		return Adjustment{}, err
	}
	a := Adjustment{
		TenantID: actor.TenantID, BranchID: branchID, PublicID: security.NewPublicID(),
		Status: StatusPending, Reason: req.Reason, CreatedBy: actor.ID,
	}
	if req.Note != "" {
		a.Note = &req.Note
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		for _, in := range req.Items {
			line, err := s.resolveDocItem(ctx, tx, actor.TenantID, branchID, in.ProductID, in.VariantID, in.LocationID, in.BatchNo, in.Expiry)
			if err != nil {
				return err
			}
			qty, err := decimal.NewFromString(in.Qty)
			if err != nil || qty.IsZero() {
				return ErrBadQty
			}
			a.Items = append(a.Items, AdjustmentItem{
				LocationID: line.location.ID, ProductID: line.product.ID,
				VariantID: line.variantID(), BatchID: line.batchID, Qty: qty,
			})
		}
		if len(a.Items) == 0 {
			return ErrNoItems
		}
		return s.docs.CreateAdjustment(ctx, tx, &a)
	})
	return a, err
}

// ApproveAdjustment marks a pending adjustment approved (post-ready). The
// creator cannot approve their own document.
func (s *Service) ApproveAdjustment(ctx context.Context, actor identity.Actor, publicID string, approve bool) (Adjustment, error) {
	a, err := s.docs.Adjustment(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return a, err
	}
	if a.Status != StatusPending {
		return a, ErrBadState
	}
	if a.CreatedBy == actor.ID {
		return a, ErrSelfApprove
	}
	next := StatusApproved
	if !approve {
		next = StatusRejected
	}
	if err := s.docs.SetAdjustmentStatus(ctx, s.conn, a.ID, next, &actor.ID); err != nil {
		return a, err
	}
	return s.docs.Adjustment(ctx, s.conn, actor.TenantID, publicID)
}

// PostAdjustment writes an approved adjustment to the ledger.
func (s *Service) PostAdjustment(ctx context.Context, actor identity.Actor, publicID string) (Adjustment, error) {
	a, err := s.docs.Adjustment(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return a, err
	}
	if a.Status != StatusApproved {
		return a, ErrBadState
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		for _, it := range a.Items {
			m := Movement{
				TenantID: actor.TenantID, BranchID: a.BranchID, LocationID: it.LocationID,
				ProductID: it.ProductID, VariantID: it.VariantID, BatchID: it.BatchID,
				Qty: it.Qty, Kind: MoveAdjust, RefType: ptr("adjustment"), RefPublicID: &a.PublicID,
				Reason: &a.Reason, ActorID: &actor.ID,
			}
			// No cost basis → avg_cost is left unchanged.
			if err := s.postOne(ctx, tx, m); err != nil {
				return err
			}
		}
		return s.docs.SetAdjustmentStatus(ctx, tx, a.ID, StatusPosted, nil)
	})
	if err != nil {
		return a, err
	}
	return s.docs.Adjustment(ctx, s.conn, actor.TenantID, publicID)
}

// Adjustments lists adjustments.
func (s *Service) Adjustments(ctx context.Context, actor identity.Actor, branchPub, status string, pg httpx.Page) ([]Adjustment, int, error) {
	var branchID uint64
	if branchPub != "" {
		var err error
		if branchID, err = s.branch(ctx, actor, branchPub); err != nil {
			return nil, 0, err
		}
	}
	return s.docs.Adjustments(ctx, actor.TenantID, branchID, status, pg)
}

// GetAdjustment loads one adjustment.
func (s *Service) GetAdjustment(ctx context.Context, actor identity.Actor, publicID string) (Adjustment, error) {
	a, err := s.docs.Adjustment(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return a, err
	}
	if !actor.CanAccessBranch(a.BranchID) {
		return a, ErrBranchAccess
	}
	return a, nil
}

// ---- transfers -----------------------------------------------------------------

// TransferRequest is the create payload.
type TransferRequest struct {
	FromBranchID string                `json:"from_branch_id"` // public ids
	ToBranchID   string                `json:"to_branch_id"`
	Note         string                `json:"note"`
	Items        []TransferItemRequest `json:"items"`
}

// TransferItemRequest is one transfer line.
type TransferItemRequest struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	Qty       string `json:"qty"` // positive
}

// CreateTransfer drafts an inter-branch transfer.
func (s *Service) CreateTransfer(ctx context.Context, actor identity.Actor, req TransferRequest) (Transfer, error) {
	fromID, err := s.branch(ctx, actor, req.FromBranchID)
	if err != nil {
		return Transfer{}, err
	}
	toID, err := s.branches.BranchByPublicID(ctx, actor.TenantID, req.ToBranchID)
	if err != nil {
		return Transfer{}, err
	}
	if fromID == toID {
		return Transfer{}, ErrSameBranch
	}
	t := Transfer{
		TenantID: actor.TenantID, PublicID: security.NewPublicID(),
		FromBranchID: fromID, ToBranchID: toID, Status: StatusDraft, CreatedBy: actor.ID,
	}
	if req.Note != "" {
		t.Note = &req.Note
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		for _, in := range req.Items {
			line, err := s.resolveDocItem(ctx, tx, actor.TenantID, fromID, in.ProductID, in.VariantID, "", "", nil)
			if err != nil {
				return err
			}
			qty, err := decimal.NewFromString(in.Qty)
			if err != nil || qty.LessThanOrEqual(decimal.Zero) {
				return ErrBadQty
			}
			t.Items = append(t.Items, TransferItem{
				PublicID: security.NewPublicID(), ProductID: line.product.ID,
				VariantID: line.variantID(), QtySent: qty,
			})
		}
		if len(t.Items) == 0 {
			return ErrNoItems
		}
		return s.docs.CreateTransfer(ctx, tx, &t)
	})
	return t, err
}

// SendTransfer posts transfer_out movements at the source branch's default
// location and marks the transfer in transit.
func (s *Service) SendTransfer(ctx context.Context, actor identity.Actor, publicID string) (Transfer, error) {
	t, err := s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return t, err
	}
	if !actor.CanAccessBranch(t.FromBranchID) {
		return t, ErrBranchAccess
	}
	if t.Status != StatusDraft {
		return t, ErrBadState
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		loc, err := s.repo.DefaultLocation(ctx, tx, t.TenantID, t.FromBranchID)
		if err != nil {
			return err
		}
		for _, it := range t.Items {
			m := Movement{
				TenantID: t.TenantID, BranchID: t.FromBranchID, LocationID: loc.ID,
				ProductID: it.ProductID, VariantID: it.VariantID,
				Qty: it.QtySent.Neg(), Kind: MoveTransferOut,
				RefType: ptr("transfer"), RefPublicID: &t.PublicID, ActorID: &actor.ID,
			}
			if err := s.postOne(ctx, tx, m); err != nil {
				return err
			}
		}
		return s.docs.SetTransferStatus(ctx, tx, t.ID, StatusSent)
	})
	if err != nil {
		return t, err
	}
	return s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
}

// ReceiveItemRequest overrides one line's received quantity.
type ReceiveItemRequest struct {
	ItemID string `json:"item_id"` // transfer item public id
	Qty    string `json:"qty"`
}

// ReceiveRequest carries per-item received quantities (defaults to sent).
type ReceiveRequest struct {
	Items []ReceiveItemRequest `json:"items"`
}

// ReceiveTransfer posts transfer_in movements at the destination branch and
// completes the transfer.
func (s *Service) ReceiveTransfer(ctx context.Context, actor identity.Actor, publicID string, req ReceiveRequest) (Transfer, error) {
	t, err := s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return t, err
	}
	if !actor.CanAccessBranch(t.ToBranchID) {
		return t, ErrBranchAccess
	}
	if t.Status != StatusSent {
		return t, ErrBadState
	}
	got := map[string]decimal.Decimal{}
	for _, in := range req.Items {
		q, err := decimal.NewFromString(in.Qty)
		if err != nil || q.LessThanOrEqual(decimal.Zero) {
			return t, ErrBadQty
		}
		got[in.ItemID] = q
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		loc, err := s.repo.DefaultLocation(ctx, tx, t.TenantID, t.ToBranchID)
		if err != nil {
			return err
		}
		for _, it := range t.Items {
			qty := it.QtySent
			if q, ok := got[it.PublicID]; ok {
				qty = q
			}
			cost, err := s.repo.AvgCost(ctx, tx, t.TenantID, t.FromBranchID, it.ProductID, it.VariantID)
			if err != nil {
				return err
			}
			m := Movement{
				TenantID: t.TenantID, BranchID: t.ToBranchID, LocationID: loc.ID,
				ProductID: it.ProductID, VariantID: it.VariantID,
				Qty: qty, Kind: MoveTransferIn, UnitCost: &cost,
				RefType: ptr("transfer"), RefPublicID: &t.PublicID, ActorID: &actor.ID,
			}
			if err := s.postOne(ctx, tx, m); err != nil {
				return err
			}
			if err := s.docs.MarkReceived(ctx, tx, t.ID, it.PublicID, qty.String()); err != nil {
				return err
			}
		}
		return s.docs.SetTransferStatus(ctx, tx, t.ID, StatusReceived)
	})
	if err != nil {
		return t, err
	}
	return s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
}

// CancelTransfer cancels a draft transfer (sent transfers must be received).
func (s *Service) CancelTransfer(ctx context.Context, actor identity.Actor, publicID string) (Transfer, error) {
	t, err := s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return t, err
	}
	if t.Status != StatusDraft {
		return t, ErrBadState
	}
	if err := s.docs.SetTransferStatus(ctx, s.conn, t.ID, StatusCancelled); err != nil {
		return t, err
	}
	return s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
}

// Transfers lists transfers involving the branch.
func (s *Service) Transfers(ctx context.Context, actor identity.Actor, branchPub string, pg httpx.Page) ([]Transfer, int, error) {
	var branchID uint64
	if branchPub != "" {
		var err error
		if branchID, err = s.branch(ctx, actor, branchPub); err != nil {
			return nil, 0, err
		}
	}
	return s.docs.Transfers(ctx, actor.TenantID, branchID, pg)
}

// GetTransfer loads one transfer.
func (s *Service) GetTransfer(ctx context.Context, actor identity.Actor, publicID string) (Transfer, error) {
	t, err := s.docs.Transfer(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return t, err
	}
	if !actor.CanAccessBranch(t.FromBranchID) && !actor.CanAccessBranch(t.ToBranchID) {
		return t, ErrBranchAccess
	}
	return t, nil
}

// ---- stocktakes ------------------------------------------------------------------

// CreateCount opens a stocktake and snapshots expected balances.
func (s *Service) CreateCount(ctx context.Context, actor identity.Actor, branchPub, locationPub, note string) (Count, error) {
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return Count{}, err
	}
	c := Count{
		TenantID: actor.TenantID, BranchID: branchID, PublicID: security.NewPublicID(),
		Status: StatusOpen, CreatedBy: actor.ID,
	}
	if note != "" {
		c.Note = &note
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if locationPub != "" {
			loc, err := s.repo.LocationByPublicID(ctx, actor.TenantID, branchID, locationPub)
			if err != nil {
				return err
			}
			c.LocationID = &loc.ID
		}
		rows, err := s.repo.ExpectedRows(ctx, actor.TenantID, branchID, c.LocationID)
		if err != nil {
			return err
		}
		for _, b := range rows {
			c.Items = append(c.Items, CountItem{
				PublicID: security.NewPublicID(), LocationID: b.LocationID,
				ProductID: b.ProductID, VariantID: b.VariantID,
				BatchID: b.BatchID, ExpectedQty: b.Qty,
			})
		}
		return s.docs.CreateCount(ctx, tx, &c)
	})
	return c, err
}

// SetCounted records counted quantities on an open count, keyed by item
// public id.
func (s *Service) SetCounted(ctx context.Context, actor identity.Actor, publicID string, counts map[string]string) (Count, error) {
	c, err := s.openCount(ctx, actor, publicID)
	if err != nil {
		return c, err
	}
	valid := map[string]bool{}
	for _, it := range c.Items {
		valid[it.PublicID] = true
	}
	for itemPubID, qtyStr := range counts {
		if !valid[itemPubID] {
			return c, ErrDocNotFound
		}
		if _, err := decimal.NewFromString(qtyStr); err != nil {
			return c, ErrBadQty
		}
		if err := s.docs.SetCounted(ctx, c.ID, itemPubID, qtyStr); err != nil {
			return c, err
		}
	}
	return s.docs.Count(ctx, s.conn, actor.TenantID, publicID)
}

// PostCount writes 'count' movements for every variance and closes the count.
func (s *Service) PostCount(ctx context.Context, actor identity.Actor, publicID string) (Count, error) {
	c, err := s.openCount(ctx, actor, publicID)
	if err != nil {
		return c, err
	}
	for _, it := range c.Items {
		if it.CountedQty == nil {
			return c, ErrUncounted
		}
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		for _, it := range c.Items {
			delta := it.CountedQty.Sub(it.ExpectedQty)
			if delta.IsZero() {
				continue
			}
			m := Movement{
				TenantID: c.TenantID, BranchID: c.BranchID, LocationID: it.LocationID,
				ProductID: it.ProductID, VariantID: it.VariantID, BatchID: it.BatchID,
				Qty: delta, Kind: MoveCount, RefType: ptr("count"), RefPublicID: &c.PublicID,
				ActorID: &actor.ID,
			}
			if err := s.postOne(ctx, tx, m); err != nil {
				return err
			}
		}
		return s.docs.CloseCount(ctx, tx, c.ID, StatusPosted)
	})
	if err != nil {
		return c, err
	}
	return s.docs.Count(ctx, s.conn, actor.TenantID, publicID)
}

// CancelCount abandons an open count.
func (s *Service) CancelCount(ctx context.Context, actor identity.Actor, publicID string) (Count, error) {
	c, err := s.openCount(ctx, actor, publicID)
	if err != nil {
		return c, err
	}
	if err := s.docs.CloseCount(ctx, s.conn, c.ID, StatusCancelled); err != nil {
		return c, err
	}
	return s.docs.Count(ctx, s.conn, actor.TenantID, publicID)
}

// Counts lists stocktakes.
func (s *Service) Counts(ctx context.Context, actor identity.Actor, branchPub string, pg httpx.Page) ([]Count, int, error) {
	var branchID uint64
	if branchPub != "" {
		var err error
		if branchID, err = s.branch(ctx, actor, branchPub); err != nil {
			return nil, 0, err
		}
	}
	return s.docs.Counts(ctx, actor.TenantID, branchID, pg)
}

// GetCount loads one count.
func (s *Service) GetCount(ctx context.Context, actor identity.Actor, publicID string) (Count, error) {
	c, err := s.docs.Count(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return c, err
	}
	if !actor.CanAccessBranch(c.BranchID) {
		return c, ErrBranchAccess
	}
	return c, nil
}

// ---- shared doc helpers ---------------------------------------------------------

// openCount loads a count and checks it is open + branch-accessible.
func (s *Service) openCount(ctx context.Context, actor identity.Actor, publicID string) (Count, error) {
	c, err := s.docs.Count(ctx, s.conn, actor.TenantID, publicID)
	if err != nil {
		return c, err
	}
	if !actor.CanAccessBranch(c.BranchID) {
		return c, ErrBranchAccess
	}
	if c.Status != StatusOpen {
		return c, ErrBadState
	}
	return c, nil
}

// postOne appends one movement and applies its balance delta inside tx.
// Outbound rows without an explicit batch are FEFO-allocated across batches.
func (s *Service) postOne(ctx context.Context, tx *sqlx.Tx, m Movement) error {
	movs, err := s.allocateBatches(ctx, tx, m.TenantID, []Movement{m})
	if err != nil {
		return err
	}
	if err := s.repo.InsertMovements(ctx, tx, movs); err != nil {
		return err
	}
	for _, row := range movs {
		if err := s.repo.AddBalance(ctx, tx, row); err != nil {
			return err
		}
	}
	return nil
}

// resolveDocItem resolves product/variant/location for a document line.
func (s *Service) resolveDocItem(ctx context.Context, q db.Querier, tenantID, branchID uint64, productPub, variantPub, locationPub, batchNo string, expiry *string) (stockLine, error) {
	items, err := s.resolveLines(ctx, q, tenantID, branchID, []LineRequest{{
		ProductID: productPub, VariantID: variantPub, LocationID: locationPub,
		BatchNo: batchNo, Expiry: expiry, Qty: "1",
	}})
	if err != nil {
		return stockLine{}, err
	}
	return items[0], nil
}

// variantID is the resolved variant's internal id or 0.
func (l stockLine) variantID() uint64 {
	if l.hasVar {
		return l.variant.ID
	}
	return 0
}

func ptr[T any](v T) *T { return &v }
