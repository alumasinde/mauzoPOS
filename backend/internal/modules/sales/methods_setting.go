package sales

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// enabledMethods returns the payment methods the tenant accepts — its
// payments.methods setting, or DefaultMethods plus credit when unset. A
// switched-off method is refused here, not just hidden on the till.
func (s *Service) enabledMethods(ctx context.Context, tenantID uint64) (map[string]bool, error) {
	list, err := s.repo.PaymentMethodsSetting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		list = append(append([]string{}, DefaultMethods...), "credit", "bank")
	}
	out := make(map[string]bool, len(list))
	for _, m := range list {
		out[m] = true
	}
	return out, nil
}

// PaymentMethodsSetting reads tenant_settings payments.methods (nil when unset).
func (r *Repo) PaymentMethodsSetting(ctx context.Context, tenantID uint64) ([]string, error) {
	var raw []byte
	err := r.conn.GetContext(ctx, &raw,
		"SELECT value FROM tenant_settings WHERE tenant_id = ? AND `key` = 'payments.methods'", tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, nil // malformed setting: fall back to defaults
	}
	return list, nil
}

// MarkMpesaVerified confirms a sale's unverified M-Pesa line once
// reconciliation links the real transaction to it.
func (s *Service) MarkMpesaVerified(ctx context.Context, tenantID, saleID uint64, transID string) error {
	_, err := s.conn.ExecContext(ctx,
		`UPDATE sale_payments SET status = ?, reference = ?
		 WHERE tenant_id = ? AND sale_id = ? AND method = 'mpesa' AND status = ?
		 ORDER BY id LIMIT 1`, PayConfirmed, transID, tenantID, saleID, PayUnverified)
	return err
}

// checkMethods refuses unknown methods and methods the tenant switched off.
func (s *Service) checkMethods(ctx context.Context, tenantID uint64, in []PaymentInput) error {
	allowed, err := s.enabledMethods(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, pi := range in {
		if _, ok := methods[pi.Method]; !ok {
			return ErrUnknownMethod
		}
		if !allowed[pi.Method] {
			return ErrMethodDisabled
		}
	}
	return nil
}
