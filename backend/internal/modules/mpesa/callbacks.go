package mpesa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/security"
	"mauzo/pkg/daraja"
)

// ---------------------------------------------------------------------------
// Webhook ingestion — every handler stores the raw body first, then processes.
// The callbacks are unsigned; the {token} path segment is the shared secret.
// ---------------------------------------------------------------------------

// HandleSTKCallback processes the STK push result. Idempotent: a duplicate
// delivery for the same checkout id is acked without re-processing.
func (s *Service) HandleSTKCallback(ctx context.Context, tenantID uint64, raw []byte) {
	var cb daraja.STKCallback
	if err := json.Unmarshal(raw, &cb); err != nil {
		slog.Warn("mpesa: undecodable stk callback", "tenant", tenantID, "err", err)
		return
	}
	res := cb.Body.StkCallback
	t, err := s.repo.ByCheckout(ctx, tenantID, res.CheckoutRequestID)
	if err != nil {
		slog.Warn("mpesa: stk callback lookup failed", "tenant", tenantID, "err", err)
		return
	}
	if t == nil {
		slog.Warn("mpesa: stk callback for unknown checkout", "tenant", tenantID,
			"checkout", res.CheckoutRequestID)
		return
	}
	LogEvent(ctx, s.conn, tenantID, &t.ID, EvSTKCallback, res.CheckoutRequestID,
		fmt.Sprintf("stk result %d: %s", res.ResultCode, res.ResultDesc), raw)
	if t.Status != StPending && t.Status != StInitiated {
		return // already resolved (duplicate delivery or the status query won)
	}
	receipt, _ := cb.Meta("MpesaReceiptNumber")
	phone, _ := cb.Meta("PhoneNumber")
	var transTime *time.Time
	if tt := cb.TransDate(); !tt.IsZero() {
		transTime = &tt
	}
	status := StFailed
	switch res.ResultCode {
	case 0:
		status = StSuccess
	case 1032:
		status = StCancelled
	}
	code := res.ResultCode
	var transPtr *string
	if receipt != "" {
		transPtr = &receipt
	}
	if err := s.repo.UpdateOutcome(ctx, t.ID, status, transPtr, &code, res.ResultDesc,
		"", phone, transTime, raw); err != nil {
		slog.Error("mpesa: stk outcome write failed", "tenant", tenantID, "err", err)
		return
	}
	t.Status, t.TransID, t.Phone, t.TransTime = status, transPtr, phone, transTime
	if status == StSuccess {
		s.feed.publish(tenantID, feedEvent{Kind: "collection", Txn: t})
	}
}

// HandleC2BConfirmation processes a confirmed customer payment. Dedupe is by
// TransID — Daraja re-sends confirmations and also fires C2B for STK-originated
// payments, so an existing row just gets its event logged (and merged if the
// STK callback hasn't landed yet).
func (s *Service) HandleC2BConfirmation(ctx context.Context, tenantID uint64, raw []byte) {
	var p daraja.C2BPayment
	if err := json.Unmarshal(raw, &p); err != nil {
		slog.Warn("mpesa: undecodable c2b confirmation", "tenant", tenantID, "err", err)
		return
	}
	amount, err := decimal.NewFromString(p.TransAmount)
	if err != nil || p.TransID == "" {
		slog.Warn("mpesa: malformed c2b confirmation", "tenant", tenantID,
			"trans", p.TransID, "err", err)
		return
	}
	if existing, err := s.repo.ByTransID(ctx, tenantID, p.TransID); err != nil {
		slog.Error("mpesa: c2b dedupe lookup failed", "tenant", tenantID, "err", err)
		return
	} else if existing != nil {
		LogEvent(ctx, s.conn, tenantID, &existing.ID, EvC2BConfirm, p.TransID,
			"duplicate c2b confirmation", raw)
		if existing.Status != StSuccess {
			code := 0
			_ = s.repo.UpdateOutcome(ctx, existing.ID, StSuccess, &p.TransID, &code,
				"confirmed via c2b", p.Payer(), p.MSISDN, nil, raw)
		}
		return
	}
	transTime := p.Time()
	t := &Transaction{
		TenantID: tenantID, PublicID: security.NewPublicID(), Kind: KindC2B,
		Status: StSuccess, TransID: &p.TransID, Phone: p.MSISDN,
		PayerName: p.Payer(), Amount: amount, AccountRef: p.BillRefNumber,
		ShortCode: p.BusinessShortCode, ResultCode: intPtr(0), TransTime: &transTime,
	}
	if err := s.repo.InsertTransaction(ctx, t); err != nil {
		if !errors.Is(err, errDup) {
			slog.Error("mpesa: c2b insert failed", "tenant", tenantID, "err", err)
		}
		return // concurrent duplicate — the unique key already deduped
	}
	LogEvent(ctx, s.conn, tenantID, &t.ID, EvC2BConfirm, p.TransID,
		fmt.Sprintf("c2b %s from %s", amount.StringFixed(0), p.MSISDN), raw)
	s.feed.publish(tenantID, feedEvent{Kind: "collection", Txn: t})
}

// HandleC2BValidation answers the pre-payment check. Confirmation is the
// source of truth — validation only accepts and logs so a flaky validation
// endpoint can never block collections.
func (s *Service) HandleC2BValidation(ctx context.Context, tenantID uint64, raw []byte) {
	var p daraja.C2BPayment
	if err := json.Unmarshal(raw, &p); err == nil {
		LogEvent(ctx, s.conn, tenantID, nil, EvC2BValidation, p.TransID,
			fmt.Sprintf("c2b validation %s from %s", p.TransAmount, p.MSISDN), raw)
	}
}

// HandleReversalResult processes a reversal outcome/timeout callback. On a
// successful reversal the original transaction is marked reversed; its sale
// allocation is kept for audit but the row no longer funds the sale.
func (s *Service) HandleReversalResult(ctx context.Context, tenantID uint64, raw []byte) {
	var res daraja.ReversalResult
	if err := json.Unmarshal(raw, &res); err != nil {
		slog.Warn("mpesa: undecodable reversal result", "tenant", tenantID, "err", err)
		return
	}
	r := res.Result
	v, err := s.repo.ReversalByConversation(ctx, r.OriginatorConversationID)
	if err != nil || v == nil || v.TenantID != tenantID {
		slog.Warn("mpesa: reversal result for unknown request", "tenant", tenantID,
			"conv", r.OriginatorConversationID)
		return
	}
	if v.Status != "pending" {
		return // duplicate delivery
	}
	status := "failed"
	if r.ResultType == 1 {
		status = "timeout" //nolint:goconst
	} else if r.ResultCode == 0 {
		status = "reversed"
	}
	if err := s.repo.UpdateReversal(ctx, v.ID, status, &r.ResultCode, r.ResultDesc); err != nil {
		slog.Error("mpesa: reversal update failed", "tenant", tenantID, "err", err)
		return
	}
	LogEvent(ctx, s.conn, tenantID, &v.TxnID, EvReversalResult, r.OriginatorConversationID,
		fmt.Sprintf("reversal %s: %s", status, r.ResultDesc), raw)
	t, _ := s.repo.ByID(ctx, tenantID, v.TxnID)
	if t == nil || (t.Status != StSuccess && t.Status != StRevPending) {
		return
	}
	next := ""
	if status == "reversed" {
		next = StReversed
	} else if t.Status == StRevPending {
		next = StSuccess // failed/timed-out reversal — collection stands
	}
	if next == "" {
		return
	}
	_ = s.repo.UpdateStatus(ctx, t.ID, next, "Reversal "+status+".")
	t.Status = next
	s.feed.publish(tenantID, feedEvent{Kind: "reversal", Txn: t})
}

// RequestReversal initiates a Daraja reversal for a confirmed collection.
// Needs initiator credentials; records the pending reversal before the call so
// the async result always lands on a row.
func (s *Service) RequestReversal(ctx context.Context, tenantID, txnID, byUserID uint64,
	reason string) error {
	t, err := s.repo.ByID(ctx, tenantID, txnID)
	if err != nil || t == nil {
		return ErrNoTransaction
	}
	if t.Status != StSuccess || t.TransID == nil {
		return ErrNoReversal
	}
	c, err := s.repo.Credentials(ctx, tenantID)
	if err != nil || c == nil {
		return ErrNotConfigured
	}
	cli, secs, err := s.clientFor(ctx, c)
	if err != nil {
		return err
	}
	if secs.Initiator == "" || secs.SecurityCredential == "" {
		return ErrNoInitiator
	}
	if reason == "" {
		reason = "Reversal requested"
	}
	resp, err := cli.Reverse(ctx, daraja.ReversalRequest{
		Initiator: secs.Initiator, SecurityCredential: secs.SecurityCredential,
		TransactionID: *t.TransID, Amount: t.Amount.IntPart(), ReceiverParty: c.ShortCode,
		ResultURL:       s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/reversal",
		QueueTimeoutURL: s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/reversal",
		Remarks:         truncate(reason, 100),
	})
	if err != nil {
		return mapDarajaErr(err)
	}
	v := &Reversal{
		TenantID: tenantID, TxnID: txnID,
		OriginatorConversationID: resp.OriginatorConversationID,
		ConversationID:           resp.ConversationID,
		Amount:                   t.Amount, Reason: reason, RequestedBy: &byUserID,
	}
	if err := s.repo.InsertReversal(ctx, v); err != nil {
		return err
	}
	_ = s.repo.UpdateStatus(ctx, txnID, StRevPending, "Reversal requested.")
	LogEvent(ctx, s.conn, tenantID, &txnID, EvReversalReq, resp.OriginatorConversationID,
		fmt.Sprintf("reversal of %s: %s", *t.TransID, reason), nil)
	if s.audit != nil {
		s.audit.Log(ctx, auditEntry(tenantID, byUserID, "mpesa.reversal.requested", t.PublicID))
	}
	t.Status = StRevPending
	s.feed.publish(tenantID, feedEvent{Kind: "reversal", Txn: t})
	return nil
}

func intPtr(v int) *int { return &v }

// auditEntry builds a terse audit row for mpesa actions.
func auditEntry(tenantID, actorID uint64, action, entityID string) audit.Entry {
	return audit.Entry{
		TenantID: &tenantID, ActorID: &actorID, ActorType: "user",
		Action: action, EntityType: "mpesa_transaction", EntityID: entityID,
	}
}
