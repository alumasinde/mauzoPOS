package mpesa

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/pkg/daraja"
)

// maxSTKPushesPerDay guards against a buggy till hammering Daraja.
const maxSTKPushesPerDay = 400

// stkQueryAfter / stkGiveUpAfter pace the status-query fallback: Daraja's
// callback is the primary path; we only ask when the customer has had time.
const (
	stkQueryAfter  = 20 * time.Second
	stkQueryEvery  = 15 * time.Second
	stkGiveUpAfter = 130 * time.Second
)

// Auditor writes audit-log rows for sensitive actions.
type Auditor interface {
	Log(ctx context.Context, e audit.Entry)
}

// Secrets is the decrypted credential view used internally — never serialized.
type Secrets struct {
	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	Initiator          string
	SecurityCredential string
}

// Service owns M-Pesa business logic.
type Service struct {
	conn      *sqlx.DB
	repo      *Repo
	box       *secretbox
	audit     Auditor
	sales     SalesDep
	base      string // PUBLIC_BASE_URL for callback URLs
	feed      *broker
	newClient func(base, key, secret string) *daraja.Client // test seam
}

// NewService builds the service. credKey is MPESA_CRED_KEY (dev fallback
// allowed); publicBaseURL builds callback URLs.
func NewService(conn *sqlx.DB, repo *Repo, credKey, publicBaseURL string, a Auditor) (*Service, error) {
	box, err := newSecretbox(credKey)
	if err != nil {
		return nil, err
	}
	s := &Service{conn: conn, repo: repo, box: box, audit: a, base: publicBaseURL, feed: newBroker()}
	s.newClient = func(base, key, secret string) *daraja.Client {
		return daraja.New(base, key, secret, nil)
	}
	return s, nil
}

// SetClientFactory overrides Daraja client construction (tests, stub servers).
func (s *Service) SetClientFactory(f func(base, key, secret string) *daraja.Client) {
	s.newClient = f
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// ConfigView is the masked credentials response for the UI.
type ConfigView struct {
	Env                   string `json:"env"`
	ShortCode             string `json:"short_code"`
	ShortCodeKind         string `json:"short_code_kind"`
	ConsumerKey           string `json:"consumer_key"` // masked
	ConsumerSecret        string `json:"consumer_secret"`
	Passkey               string `json:"passkey"`
	InitiatorName         string `json:"initiator_name"`
	SecurityCredential    string `json:"security_credential"`
	CallbackToken         string `json:"-"`
	STKCallbackURL        string `json:"stk_callback_url"`
	C2BConfirmationURL    string `json:"c2b_confirmation_url"`
	C2BValidationURL      string `json:"c2b_validation_url"`
	ReversalResultURL     string `json:"reversal_result_url"`
	C2BRegisteredAt       string `json:"c2b_registered_at,omitempty"`
	HasConsumerSecret     bool   `json:"has_consumer_secret"`
	HasPasskey            bool   `json:"has_passkey"`
	HasSecurityCredential bool   `json:"has_security_credential"`
	// SecretsUnreadable is true when stored secrets cannot be decrypted under
	// the current key — the view still renders so credentials can be re-entered.
	SecretsUnreadable bool `json:"secrets_unreadable"`
}

// Config returns the masked config view (secrets never leave the server).
func (s *Service) Config(ctx context.Context, tenantID uint64) (*ConfigView, error) {
	c, err := s.repo.Credentials(ctx, tenantID)
	if err != nil || c == nil {
		return &ConfigView{}, err
	}
	secs, err := s.secrets(c)
	unreadable := err != nil
	if unreadable {
		secs = &Secrets{}
	}
	v := &ConfigView{
		Env: c.Env, ShortCode: c.ShortCode, ShortCodeKind: c.ShortCodeKind,
		ConsumerKey: mask(secs.ConsumerKey), ConsumerSecret: mask(secs.ConsumerSecret),
		Passkey: mask(secs.Passkey), InitiatorName: c.InitiatorName,
		SecurityCredential:    mask(secs.SecurityCredential),
		HasConsumerSecret:     unreadable && c.ConsumerSecretEnc != nil || secs.ConsumerSecret != "",
		HasPasskey:            unreadable && c.PasskeyEnc != nil || secs.Passkey != "",
		HasSecurityCredential: unreadable && c.SecurityCredentialEnc != nil || secs.SecurityCredential != "",
		SecretsUnreadable:     unreadable,
		STKCallbackURL:        s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/stk",
		C2BConfirmationURL:    s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/c2b/confirmation",
		C2BValidationURL:      s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/c2b/validation",
		ReversalResultURL:     s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/reversal",
	}
	if c.C2BRegisteredAt != nil {
		v.C2BRegisteredAt = c.C2BRegisteredAt.Format(time.RFC3339)
	}
	return v, nil
}

// ConfigInput is the settings form payload. Empty secrets keep the stored
// value — the UI masks them, so we cannot distinguish "unchanged" from
// "cleared" without a sentinel: send "" to keep, "-clear-" to erase.
type ConfigInput struct {
	Env                string `json:"env"`
	ShortCode          string `json:"short_code"`
	ShortCodeKind      string `json:"short_code_kind"`
	ConsumerKey        string `json:"consumer_key"`
	ConsumerSecret     string `json:"consumer_secret"`
	Passkey            string `json:"passkey"`
	InitiatorName      string `json:"initiator_name"`
	SecurityCredential string `json:"security_credential"`
}

// SaveConfig upserts credentials. Masked values ("••••xxxx") are ignored so a
// round-tripped form cannot corrupt stored secrets.
func (s *Service) SaveConfig(ctx context.Context, tenantID uint64, in ConfigInput, actorID uint64) error {
	if in.ShortCodeKind != "till" && in.ShortCodeKind != "paybill" {
		return ErrBadKind
	}
	if in.Env != "sandbox" && in.Env != "production" {
		in.Env = "sandbox"
	}
	c := &Credentials{TenantID: tenantID, Env: in.Env,
		ShortCode: in.ShortCode, ShortCodeKind: in.ShortCodeKind, InitiatorName: in.InitiatorName}
	if old, err := s.repo.Credentials(ctx, tenantID); err == nil && old != nil {
		c.ID, c.CallbackToken, c.C2BRegisteredAt = old.ID, old.CallbackToken, old.C2BRegisteredAt
		// Shortcode change invalidates the URL registration.
		if old.ShortCode != in.ShortCode || old.Env != in.Env {
			c.C2BRegisteredAt = nil
		}
		// Preserve secrets left masked/blank; "-clear-" erases.
		if old.ConsumerKeyEnc != nil {
			c.ConsumerKeyEnc = old.ConsumerKeyEnc
		}
		c.ConsumerSecretEnc, c.PasskeyEnc, c.SecurityCredentialEnc =
			old.ConsumerSecretEnc, old.PasskeyEnc, old.SecurityCredentialEnc
	}
	if c.CallbackToken == "" {
		c.CallbackToken = newCallbackToken()
	}
	var err error
	if clean, ok := freshSecret(in.ConsumerKey); ok {
		if c.ConsumerKeyEnc, err = s.box.seal(clean); err != nil {
			return err
		}
	}
	if clean, ok := freshSecret(in.ConsumerSecret); ok {
		if c.ConsumerSecretEnc, err = s.box.seal(clean); err != nil {
			return err
		}
	}
	if clean, ok := freshSecret(in.Passkey); ok {
		if c.PasskeyEnc, err = s.box.seal(clean); err != nil {
			return err
		}
	}
	if clean, ok := freshSecret(in.SecurityCredential); ok {
		if c.SecurityCredentialEnc, err = s.box.seal(clean); err != nil {
			return err
		}
	}
	if err := s.repo.UpsertCredentials(ctx, c); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, audit.Entry{
			TenantID: &tenantID, ActorID: &actorID, ActorType: "user",
			Action: "mpesa.config.saved", EntityType: "mpesa_credentials", EntityID: fmt.Sprint(c.ID),
		})
	}
	return nil
}

// freshSecret reports whether a form value is a new secret (not a mask).
func freshSecret(v string) (string, bool) {
	if v == "" || len(v) >= 4 && v[:4] == "••••" {
		return "", false
	}
	return v, true
}

// secrets decrypts the credential row for internal use.
func (s *Service) secrets(c *Credentials) (*Secrets, error) {
	out := &Secrets{Initiator: c.InitiatorName}
	var err error
	if out.ConsumerKey, err = s.box.open(c.ConsumerKeyEnc); err != nil {
		return nil, err
	}
	if out.ConsumerSecret, err = s.box.open(c.ConsumerSecretEnc); err != nil {
		return nil, err
	}
	if out.Passkey, err = s.box.open(c.PasskeyEnc); err != nil {
		return nil, err
	}
	if out.SecurityCredential, err = s.box.open(c.SecurityCredentialEnc); err != nil {
		return nil, err
	}
	return out, nil
}

// clientFor builds a Daraja client from stored credentials.
func (s *Service) clientFor(_ context.Context, c *Credentials) (*daraja.Client, *Secrets, error) {
	secs, err := s.secrets(c)
	if err != nil {
		return nil, nil, err
	}
	if secs.ConsumerKey == "" || secs.ConsumerSecret == "" {
		return nil, nil, ErrNotConfigured
	}
	base := daraja.BaseSandbox
	if c.Env == "production" {
		base = daraja.BaseProduction
	}
	return s.newClient(base, secs.ConsumerKey, secs.ConsumerSecret), secs, nil
}

// RegisterC2B pushes the tenant's webhook URLs to Daraja.
func (s *Service) RegisterC2B(ctx context.Context, tenantID, actorID uint64) error {
	c, err := s.repo.Credentials(ctx, tenantID)
	if err != nil {
		return err
	}
	if c == nil || c.ShortCode == "" {
		return ErrNotConfigured
	}
	cli, _, err := s.clientFor(ctx, c)
	if err != nil {
		return err
	}
	_, err = cli.RegisterC2B(ctx, c.ShortCode, "Completed",
		s.base+"/api/v1/hooks/mpesa/"+c.CallbackToken+"/c2b/confirmation",
		s.base+"/api/v1/hooks/mpesa/"+c.CallbackToken+"/c2b/validation")
	if err != nil {
		return err
	}
	if err := s.repo.SetC2BRegistered(ctx, tenantID, time.Now().UTC()); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, audit.Entry{
			TenantID: &tenantID, ActorID: &actorID, ActorType: "user",
			Action: "mpesa.c2b.registered", EntityType: "mpesa_credentials", EntityID: fmt.Sprint(c.ID),
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// STK push
// ---------------------------------------------------------------------------

var phoneRe = regexp.MustCompile(`^(?:\+?254|0)(7|1)\d{8}$`)

// NormalizePhone maps Kenyan user input to 2547XXXXXXXX; "" when invalid.
func NormalizePhone(p string) string {
	p = regexp.MustCompile(`[\s\-()]`).ReplaceAllString(p, "")
	if !phoneRe.MatchString(p) {
		return ""
	}
	if p[0] == '0' {
		return "254" + p[1:]
	}
	if p[0] == '+' {
		return p[1:]
	}
	return p
}

// InitiateSTK sends a PIN prompt and records the pending transaction.
// The caller supplies a public id so the row is referenceable immediately.
func (s *Service) InitiateSTK(ctx context.Context, tenantID, branchID, byUserID uint64,
	phone, accountRef, note string, amount decimal.Decimal) (*Transaction, error) {
	if amount.LessThan(decimal.NewFromInt(1)) || !amount.Equal(amount.Floor()) {
		return nil, ErrInvalidAmount
	}
	if NormalizePhone(phone) == "" {
		return nil, ErrInvalidPhone
	}
	c, err := s.repo.Credentials(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.ShortCode == "" {
		return nil, ErrNotConfigured
	}
	cli, secs, err := s.clientFor(ctx, c)
	if err != nil {
		return nil, err
	}
	if secs.Passkey == "" {
		return nil, ErrNotConfigured
	}
	if err := s.repo.CheckAndBumpSTKCount(ctx, tenantID, maxSTKPushesPerDay); err != nil {
		return nil, err
	}
	txType := daraja.TypeTill
	if c.ShortCodeKind == "paybill" {
		txType = daraja.TypePayBill
	}
	resp, err := cli.STKPush(ctx, daraja.STKPushRequest{
		ShortCode: c.ShortCode, Passkey: secs.Passkey, TxType: txType,
		Amount: amount.IntPart(), Payer: NormalizePhone(phone),
		CallbackURL: s.base + "/api/v1/hooks/mpesa/" + c.CallbackToken + "/stk",
		AccountRef:  truncate(accountRef, 12), TransactionRef: truncate(note, 13),
	})
	if err != nil {
		return nil, mapDarajaErr(err)
	}
	t := &Transaction{
		TenantID: tenantID, PublicID: security.NewPublicID(), BranchID: opt(branchID),
		Kind: KindSTK, Status: StPending, CheckoutRequestID: strp(resp.CheckoutRequestID),
		MerchantRequestID: strp(resp.MerchantRequestID), Phone: NormalizePhone(phone),
		Amount: amount, AccountRef: accountRef, ShortCode: c.ShortCode, InitiatedBy: opt(byUserID),
	}
	if err := s.repo.InsertTransaction(ctx, t); err != nil {
		return nil, err
	}
	LogEvent(ctx, s.conn, tenantID, &t.ID, "stk_initiated", resp.CheckoutRequestID,
		fmt.Sprintf("prompt sent to %s for %s", t.Phone, amount.StringFixed(0)), nil)
	return t, nil
}

// STKStatus returns the transaction's current state, asking Daraja when the
// callback is overdue — the fallback path the docs recommend.
func (s *Service) STKStatus(ctx context.Context, tenantID uint64, publicID string) (*Transaction, error) {
	t, err := s.repo.ByPublic(ctx, tenantID, publicID)
	if err != nil || t == nil {
		return nil, httpx.ErrNotFound
	}
	if t.Kind != KindSTK || (t.Status != StPending && t.Status != StInitiated) {
		return t, nil
	}
	age := time.Since(t.CreatedAt)
	if age > stkGiveUpAfter {
		if err := s.repo.UpdateStatus(ctx, t.ID, StTimeout, "No result — prompt expired."); err == nil {
			t.Status = StTimeout
		}
		return t, nil
	}
	if age < stkQueryAfter || (t.LastQueriedAt != nil && time.Since(*t.LastQueriedAt) < stkQueryEvery) {
		return t, nil
	}
	return t, s.queryFallback(ctx, t)
}

// queryFallback asks Daraja for the result and records it if terminal.
func (s *Service) queryFallback(ctx context.Context, t *Transaction) error {
	c, err := s.repo.Credentials(ctx, t.TenantID)
	if err != nil || c == nil {
		return err
	}
	cli, secs, err := s.clientFor(ctx, c)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_ = s.repo.TouchQueried(ctx, t.ID, now)
	res, err := cli.STKQuery(ctx, c.ShortCode, secs.Passkey, deref(t.CheckoutRequestID))
	if err != nil {
		LogEvent(ctx, s.conn, t.TenantID, &t.ID, EvStatusQuery, deref(t.CheckoutRequestID), err.Error(), nil)
		return nil // soft-fail; next poll retries
	}
	LogEvent(ctx, s.conn, t.TenantID, &t.ID, EvStatusQuery, deref(t.CheckoutRequestID),
		fmt.Sprintf("query: %s", res.ResultDesc), nil)
	code := res.ResultInt()
	if res.ResponseCode != "0" {
		return nil // Daraja hasn't processed it yet
	}
	return s.applyOutcome(ctx, t, code, res.ResultDesc, "", nil)
}

// applyOutcome maps a Daraja result code onto the transaction row. The M-Pesa
// receipt (transID) may be empty for failures.
func (s *Service) applyOutcome(ctx context.Context, t *Transaction, code int, desc, transID string, raw []byte) error {
	status := StFailed
	switch code {
	case 0:
		status = StSuccess
	case 1032:
		status = StCancelled
	}
	var transPtr *string
	if transID != "" {
		transPtr = &transID
	}
	if err := s.repo.UpdateOutcome(ctx, t.ID, status, transPtr, &code, desc, "", "", nil, raw); err != nil {
		return err
	}
	t.Status = status
	if transID != "" {
		t.TransID = &transID
	}
	s.publishSTK(t.TenantID, t)
	return nil
}

func opt(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}

func strp(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// mapDarajaErr converts client errors into friendly API errors.
func mapDarajaErr(err error) error {
	slog.Warn("daraja call failed", "err", err)
	return httpx.NewError(502, "mpesa_upstream", "M-Pesa is unreachable or rejected the request — check Daraja credentials.")
}
