package inventory

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/catalog"
)

// Products is the slice of the catalog service inventory needs: resolve a
// product (with variants) to internal ids and check it can hold stock.
type Products interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (catalog.Product, error)
}

// Branches resolves branch public ids to internal ids within a tenant.
type Branches interface {
	BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

// Service implements the stock ledger and its documents.
type Service struct {
	conn     *sqlx.DB
	repo     *Repo
	docs     *DocsRepo
	products Products
	branches Branches
}

// NewService wires the inventory service.
func NewService(conn *sqlx.DB, repo *Repo, docs *DocsRepo, products Products, branches Branches) *Service {
	return &Service{conn: conn, repo: repo, docs: docs, products: products, branches: branches}
}

// ---- locations --------------------------------------------------------------

// CreateLocation adds a stock location to a branch.
func (s *Service) CreateLocation(ctx context.Context, actor identity.Actor, branchPub, name, kind string) (Location, error) {
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return Location{}, err
	}
	if kind == "" {
		kind = "storage"
	}
	l := Location{TenantID: actor.TenantID, BranchID: branchID, PublicID: security.NewPublicID(), Name: name, Kind: kind}
	if err := s.repo.CreateLocation(ctx, s.conn, &l); err != nil {
		return Location{}, err
	}
	return l, nil
}

// Locations lists a branch's locations.
func (s *Service) Locations(ctx context.Context, actor identity.Actor, branchPub string) ([]Location, error) {
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return nil, err
	}
	return s.repo.Locations(ctx, actor.TenantID, branchID)
}

// DeleteLocation removes an unused location.
func (s *Service) DeleteLocation(ctx context.Context, actor identity.Actor, branchPub, publicID string) error {
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return err
	}
	l, err := s.repo.LocationByPublicID(ctx, actor.TenantID, branchID, publicID)
	if err != nil {
		return err
	}
	if used, err := s.repo.LocationUsed(ctx, l.ID); err != nil {
		return err
	} else if used {
		return ErrLocationInUse
	}
	return s.repo.DeleteLocation(ctx, actor.TenantID, branchID, publicID)
}

// ---- stock line resolution ----------------------------------------------------

// stockLine is one resolved item ready to post.
type stockLine struct {
	location Location
	product  catalog.Product
	variant  catalog.Variant
	hasVar   bool
	batchID  uint64
	qty      decimal.Decimal
	cost     *decimal.Decimal
}

// LineRequest is one inbound stock line (receive/wastage share the shape).
type LineRequest struct {
	ProductID  string  `json:"product_id"`            // public id
	VariantID  string  `json:"variant_id,omitempty"`  // public id, optional
	LocationID string  `json:"location_id,omitempty"` // public id, optional → default location
	BatchNo    string  `json:"batch_no,omitempty"`    // optional; creates/links a batch
	Expiry     *string `json:"expiry,omitempty"`      // YYYY-MM-DD, used with BatchNo
	Qty        string  `json:"qty"`                   // positive decimal
	UnitCost   *string `json:"unit_cost,omitempty"`   // optional cost basis
}

// resolveLines validates item rows: stocked products, same-branch locations,
// variant and batch ownership. Quantities must parse positive.
func (s *Service) resolveLines(ctx context.Context, q db.Querier, tenantID, branchID uint64, items []LineRequest) ([]stockLine, error) {
	if len(items) == 0 {
		return nil, ErrNoItems
	}
	out := make([]stockLine, 0, len(items))
	for i, in := range items {
		var line stockLine
		field := func(f string) *httpx.Error {
			return httpx.ErrValidation.WithDetails(map[string]any{
				"items." + strconv.Itoa(i) + "." + f: "required or invalid",
			})
		}
		p, err := s.products.Get(ctx, tenantID, in.ProductID)
		if err != nil {
			return nil, field("product_id")
		}
		if p.Kind == catalog.KindService || !p.TrackStock {
			return nil, ErrNotStocked
		}
		line.product = p
		if in.VariantID != "" {
			found := false
			for _, v := range p.Variants {
				if v.PublicID == in.VariantID {
					line.variant, line.hasVar, found = v, true, true
					break
				}
			}
			if !found {
				return nil, field("variant_id")
			}
		}
		qty, err := decimal.NewFromString(in.Qty)
		if err != nil || qty.LessThanOrEqual(decimal.Zero) {
			return nil, field("qty")
		}
		line.qty = qty
		if in.UnitCost != nil {
			c, err := decimal.NewFromString(*in.UnitCost)
			if err != nil || c.LessThan(decimal.Zero) {
				return nil, field("unit_cost")
			}
			line.cost = &c
		}
		line.location, err = s.locationOrDefault(ctx, q, tenantID, branchID, in.LocationID)
		if err != nil {
			return nil, err
		}
		if in.BatchNo != "" {
			b := Batch{TenantID: tenantID, ProductID: p.ID, BatchNo: in.BatchNo}
			if line.hasVar {
				b.VariantID = line.variant.ID
			}
			if in.Expiry != nil {
				t, err := time.Parse("2006-01-02", *in.Expiry)
				if err != nil {
					return nil, field("expiry")
				}
				b.ExpiryDate = &t
			}
			if err := s.repo.UpsertBatch(ctx, q, &b); err != nil {
				return nil, err
			}
			line.batchID = b.ID
		}
		out = append(out, line)
	}
	return out, nil
}

func (s *Service) locationOrDefault(ctx context.Context, q db.Querier, tenantID, branchID uint64, pub string) (Location, error) {
	if pub == "" {
		return s.repo.DefaultLocation(ctx, q, tenantID, branchID)
	}
	return s.repo.LocationByPublicID(ctx, tenantID, branchID, pub)
}

// post appends ledger rows and updates balances in one transaction. All stock
// changes — receive, wastage, documents — funnel through here.
func (s *Service) post(ctx context.Context, tenantID, actorID uint64, kind, refType, refID, reason, note string, lines []stockLine, sign int) error {
	return db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		return s.postInTx(ctx, tx, tenantID, actorID, kind, refType, refID, reason, note, lines, sign)
	})
}

// PostStockInTx resolves item rows and posts ledger movements inside a
// caller-owned transaction — the sales module decrements stock atomically
// with the sale row this way. sign < 0 decrements (sale/wastage), > 0
// increments (receive/return restock).
func (s *Service) PostStockInTx(ctx context.Context, tx *sqlx.Tx, tenantID, actorID, branchID uint64,
	kind, refType, refID, reason, note string, items []LineRequest, sign int) error {
	lines, err := s.resolveLines(ctx, tx, tenantID, branchID, items)
	if err != nil {
		return err
	}
	return s.postInTx(ctx, tx, tenantID, actorID, kind, refType, refID, reason, note, lines, sign)
}

// postInTx is the transaction body of post: movements + balances.
func (s *Service) postInTx(ctx context.Context, tx *sqlx.Tx, tenantID, actorID uint64,
	kind, refType, refID, reason, note string, lines []stockLine, sign int) error {
	{
		movs := make([]Movement, 0, len(lines))
		for _, l := range lines {
			qty := l.qty
			if sign < 0 {
				qty = qty.Neg()
			}
			m := Movement{
				TenantID: tenantID, BranchID: l.location.BranchID, LocationID: l.location.ID,
				ProductID: l.product.ID, Qty: qty, Kind: kind, ActorID: &actorID,
			}
			if l.hasVar {
				m.VariantID = l.variant.ID
			}
			m.BatchID = l.batchID
			if sign > 0 {
				m.UnitCost = l.cost // inbound moves carry cost for valuation
			}
			setOpt(&m.RefType, refType)
			setOpt(&m.RefPublicID, refID)
			setOpt(&m.Reason, reason)
			setOpt(&m.Note, note)
			movs = append(movs, m)
		}
		expanded, err := s.allocateBatches(ctx, tx, tenantID, movs)
		if err != nil {
			return err
		}
		if err := s.repo.InsertMovements(ctx, tx, expanded); err != nil {
			return err
		}
		for _, m := range expanded {
			if err := s.repo.AddBalance(ctx, tx, m); err != nil {
				return err
			}
		}
		return nil
	}
}

// allocateBatches splits outbound movements that carry no batch across the
// product's batches FEFO (earliest expiry first, NULLs last), so stock never
// goes negative on a phantom non-batch row while batches hold the units.
// Whatever batches can't cover stays on batch 0 (unbatched stock).
func (s *Service) allocateBatches(ctx context.Context, q db.Querier, tenantID uint64, movs []Movement) ([]Movement, error) {
	out := make([]Movement, 0, len(movs))
	for _, m := range movs {
		if m.Qty.Sign() >= 0 || m.BatchID != 0 {
			out = append(out, m)
			continue
		}
		rows, err := s.repo.BatchBalances(ctx, q, tenantID, m.LocationID, m.ProductID, m.VariantID)
		if err != nil {
			return nil, err
		}
		remaining := m.Qty.Abs()
		for _, b := range rows {
			if !remaining.IsPositive() {
				break
			}
			take := decimal.Min(remaining, b.Qty)
			split := m
			split.BatchID = b.BatchID
			split.Qty = take.Neg()
			out = append(out, split)
			remaining = remaining.Sub(take)
		}
		if remaining.IsPositive() {
			m.Qty = remaining.Neg()
			out = append(out, m)
		}
	}
	return out, nil
}

func setOpt(dst **string, v string) {
	if v != "" {
		*dst = &v
	}
}

// ---- receive & wastage ---------------------------------------------------------

// Receive posts 'receive' movements (incoming stock, optional cost/batch).
func (s *Service) Receive(ctx context.Context, actor identity.Actor, branchPub string, items []LineRequest, note string) error {
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return err
	}
	lines, err := s.resolveLines(ctx, s.conn, actor.TenantID, branchID, items)
	if err != nil {
		return err
	}
	return s.post(ctx, actor.TenantID, actor.ID, MoveReceive, "receive", "", "", note, lines, +1)
}

// Wastage posts 'waste' movements (stock leaving without a sale). The
// reason is reported on each movement row; the allowed set is small so
// reports can group by cause.
func (s *Service) Wastage(ctx context.Context, actor identity.Actor, branchPub, reason string, items []LineRequest, note string) error {
	if !slices.Contains(WastageReasons, reason) {
		return ErrBadReason
	}
	branchID, err := s.branch(ctx, actor, branchPub)
	if err != nil {
		return err
	}
	lines, err := s.resolveLines(ctx, s.conn, actor.TenantID, branchID, items)
	if err != nil {
		return err
	}
	return s.post(ctx, actor.TenantID, actor.ID, MoveWaste, "wastage", "", reason, note, lines, -1)
}

// ---- reads -------------------------------------------------------------------

// Balances returns current stock per the filter.
func (s *Service) Balances(ctx context.Context, actor identity.Actor, f BalanceFilter, pg httpx.Page) ([]Balance, int, error) {
	if f.BranchID != 0 && !actor.CanAccessBranch(f.BranchID) {
		return nil, 0, ErrBranchAccess
	}
	return s.repo.Balances(ctx, actor.TenantID, f, pg)
}

// Movements returns the ledger, newest first.
func (s *Service) Movements(ctx context.Context, actor identity.Actor, f MovementFilter, pg httpx.Page) ([]Movement, int, error) {
	if f.BranchID != 0 && !actor.CanAccessBranch(f.BranchID) {
		return nil, 0, ErrBranchAccess
	}
	return s.repo.Movements(ctx, actor.TenantID, f, pg)
}

// Batches lists a product's batches.
func (s *Service) Batches(ctx context.Context, tenantID, productID uint64) ([]Batch, error) {
	return s.repo.Batches(ctx, tenantID, productID)
}

// Expiring lists soon-to-expire stock.
func (s *Service) Expiring(ctx context.Context, tenantID uint64, days int) ([]ExpiringRow, error) {
	return s.repo.Expiring(ctx, tenantID, days)
}

// Valuations returns stock value per branch at weighted average cost.
func (s *Service) Valuations(ctx context.Context, tenantID uint64) ([]Valuation, error) {
	return s.repo.Valuations(ctx, tenantID)
}

// SetReorder saves a low-stock threshold.
func (s *Service) SetReorder(ctx context.Context, tenantID uint64, rl ReorderLevel) error {
	return s.repo.SetReorder(ctx, tenantID, rl)
}

// Reorders lists thresholds with current stock for a branch.
func (s *Service) Reorders(ctx context.Context, tenantID, branchID uint64) ([]ReorderRow, error) {
	return s.repo.ReorderRows(ctx, tenantID, branchID)
}

// DeleteReorder clears a threshold.
func (s *Service) DeleteReorder(ctx context.Context, tenantID, branchID, productID, variantID uint64) error {
	return s.repo.DeleteReorder(ctx, tenantID, branchID, productID, variantID)
}

// ---- shared helpers -------------------------------------------------------------

// branch resolves a public id and enforces the actor's branch access.
func (s *Service) branch(ctx context.Context, actor identity.Actor, pub string) (uint64, error) {
	id, err := s.branches.BranchByPublicID(ctx, actor.TenantID, pub)
	if err != nil {
		return 0, err
	}
	if !actor.CanAccessBranch(id) {
		return 0, ErrBranchAccess
	}
	return id, nil
}
