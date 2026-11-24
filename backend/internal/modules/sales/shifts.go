package sales

import (
	"context"
	"time"
)

// Shifts is the seam the shifts module fills: resolving which open shift a
// completed sale's cash belongs to. sales declares it (never imports shifts).
type Shifts interface {
	// ShiftForSale returns the internal shift id covering a sale on a
	// terminal at soldAt — the open shift, else the closed window a late
	// offline upload falls inside. Zero means unattributed.
	ShiftForSale(ctx context.Context, tenantID, branchID, terminalID uint64,
		soldAt time.Time) (uint64, error)
}

// BindShifts wires the resolver after construction.
func (s *Service) BindShifts(v Shifts) { s.shifts = v }

// stampShift attributes the sale to its till's shift — the open one, or the
// closed window a delayed offline upload falls inside.
func (s *Service) stampShift(ctx context.Context, in CompleteInput, sale *Sale) {
	if s.shifts == nil || in.TerminalID == nil {
		return
	}
	if sid, err := s.shifts.ShiftForSale(ctx, in.TenantID, in.BranchID,
		*in.TerminalID, sale.SoldAt); err == nil && sid != 0 {
		sale.ShiftID = &sid
	}
}

// normalizeCompleteInput fills the optional fields the service defaults.
func normalizeCompleteInput(in *CompleteInput) {
	if in.Source == "" {
		in.Source = SourcePOS
	}
	if in.SoldAt.IsZero() {
		in.SoldAt = time.Now()
	}
}
