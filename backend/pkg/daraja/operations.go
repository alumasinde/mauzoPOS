package daraja

import (
	"context"
	"encoding/json"
	"fmt"
)

// STKPushRequest is the M-Pesa Express processrequest body.
type STKPushRequest struct {
	ShortCode      string // Pay Bill / Till the payment goes to
	Passkey        string // portal-issued STK password key
	TxType         string // TypePayBill or TypeTill
	Amount         int64  // whole KES
	Payer          string // 254XXXXXXXXX — debited phone
	Receiver       string // usually == ShortCode
	CallbackURL    string
	AccountRef     string // ≤ 12 chars — shown in the prompt
	TransactionRef string // ≤ 13 chars
}

// STKPushResponse acknowledges a queued prompt.
type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// STKPush initiates a customer PIN prompt.
func (c *Client) STKPush(ctx context.Context, r STKPushRequest) (*STKPushResponse, error) {
	pw, ts := c.password(r.ShortCode, r.Passkey)
	receiver := r.Receiver
	if receiver == "" {
		receiver = r.ShortCode
	}
	body := map[string]any{
		"BusinessShortCode": r.ShortCode,
		"Password":          pw,
		"Timestamp":         ts,
		"TransactionType":   r.TxType,
		"Amount":            r.Amount,
		"PartyA":            r.Payer,
		"PartyB":            receiver,
		"PhoneNumber":       r.Payer,
		"CallBackURL":       r.CallbackURL,
		"AccountReference":  r.AccountRef,
		"TransactionDesc":   r.TransactionRef,
	}
	var out STKPushResponse
	if err := c.post(ctx, "/mpesa/stkpush/v1/processrequest", body, &out); err != nil {
		return nil, err
	}
	if out.ResponseCode != "0" {
		return nil, &Error{Message: out.ResponseDescription}
	}
	return &out, nil
}

// STKQueryResponse is the status of a previously initiated push.
type STKQueryResponse struct {
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResultCode          any    `json:"ResultCode"` // string or int per Daraja build
	ResultDesc          string `json:"ResultDesc"`
}

// ResultInt normalises ResultCode (Daraja sends both "0" and 0).
func (r *STKQueryResponse) ResultInt() int {
	var n int
	switch v := r.ResultCode.(type) {
	case float64:
		n = int(v)
	case string:
		fmt.Sscanf(v, "%d", &n) //nolint:errcheck
	case json.Number:
		i, _ := v.Int64()
		n = int(i)
	}
	return n
}

// STKQuery asks Daraja for the outcome of a push — the fallback when the
// callback is delayed or never arrives.
func (c *Client) STKQuery(ctx context.Context, shortCode, passkey, checkoutID string) (*STKQueryResponse, error) {
	pw, ts := c.password(shortCode, passkey)
	body := map[string]any{
		"BusinessShortCode": shortCode,
		"Password":          pw,
		"Timestamp":         ts,
		"CheckoutRequestID": checkoutID,
	}
	var out STKQueryResponse
	if err := c.post(ctx, "/mpesa/stkpushquery/v1/query", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// C2BRegisterResponse acknowledges URL registration.
type C2BRegisterResponse struct {
	OriginatorConversationID string `json:"OriginatorCoversationID"` // Daraja's own typo
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// RegisterC2B registers validation/confirmation URLs on the shortcode.
// ResponseType is "Completed" or "Cancelled" — what Daraja does when the
// validation URL cannot answer.
func (c *Client) RegisterC2B(ctx context.Context, shortCode, responseType, confirmationURL, validationURL string) (*C2BRegisterResponse, error) {
	body := map[string]any{
		"ShortCode":       shortCode,
		"ResponseType":    responseType,
		"ConfirmationURL": confirmationURL,
		"ValidationURL":   validationURL,
	}
	var out C2BRegisterResponse
	if err := c.post(ctx, "/mpesa/c2b/v2/registerurl", body, &out); err != nil {
		return nil, err
	}
	if out.ResponseCode != "0" {
		return nil, &Error{Message: out.ResponseDescription}
	}
	return &out, nil
}

// SimulateC2B fires a fake customer payment — sandbox only.
func (c *Client) SimulateC2B(ctx context.Context, shortCode, commandID, billRef string, amount int64, msisdn string) (*C2BRegisterResponse, error) {
	body := map[string]any{
		"ShortCode":     shortCode,
		"CommandID":     commandID,
		"Amount":        amount,
		"Msisdn":        msisdn,
		"BillRefNumber": billRef,
	}
	var out C2BRegisterResponse
	if err := c.post(ctx, "/mpesa/c2b/v1/simulate", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReversalRequest reverses a completed transaction (needs initiator creds).
type ReversalRequest struct {
	Initiator          string
	SecurityCredential string // RSA-encrypted initiator password
	TransactionID      string // TransID / MpesaReceiptNumber to reverse
	Amount             int64
	ReceiverParty      string // the shortcode that received the money
	ResultURL          string
	QueueTimeoutURL    string
	Remarks            string
	Occasion           string
}

// ReversalResponse acknowledges a queued reversal.
type ReversalResponse struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ConversationID           string `json:"ConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// Reverse submits a reversal request.
func (c *Client) Reverse(ctx context.Context, r ReversalRequest) (*ReversalResponse, error) {
	body := map[string]any{
		"Initiator":              r.Initiator,
		"SecurityCredential":     r.SecurityCredential,
		"CommandID":              "TransactionReversal",
		"SenderIdentifierType":   "4",
		"RecieverIdentifierType": "11", //nolint:misspell // Daraja's API key is spelled "Reciever"
		"Amount":                 r.Amount,
		"PartyA":                 r.ReceiverParty,
		"ReceiverParty":          r.ReceiverParty,
		"Remarks":                r.Remarks,
		"QueueTimeOutURL":        r.QueueTimeoutURL,
		"ResultURL":              r.ResultURL,
		"Occasion":               r.Occasion,
		"TransactionID":          r.TransactionID,
	}
	var out ReversalResponse
	if err := c.post(ctx, "/mpesa/reversal/v1/request", body, &out); err != nil {
		return nil, err
	}
	if out.ResponseCode != "0" {
		return nil, &Error{Message: out.ResponseDescription}
	}
	return &out, nil
}
