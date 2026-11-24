package daraja

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// STKCallback is the payload Daraja posts to the STK CallBackURL.
type STKCallback struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string `json:"Name"`
					Value any    `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// Meta extracts a named CallbackMetadata item (missing on failed pushes).
func (c *STKCallback) Meta(name string) (string, bool) {
	for _, it := range c.Body.StkCallback.CallbackMetadata.Item {
		if it.Name != name {
			continue
		}
		switch v := it.Value.(type) {
		case string:
			return v, true
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), true
		default:
			b, _ := json.Marshal(v)
			return string(b), true
		}
	}
	return "", false
}

// TransDate parses the YYYYMMDDhhmmss TransactionDate metadata value.
func (c *STKCallback) TransDate() time.Time {
	s, ok := c.Meta("TransactionDate")
	if !ok || len(s) < 14 {
		return time.Time{}
	}
	t, err := time.ParseInLocation("20060102150405", s[:14], eat)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// C2BPayment is the validation/confirmation payload for till-initiated
// (SIM Toolkit / app) payments.
type C2BPayment struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

// Time parses TransTime (YYYYMMDDHHmmss, EAT).
func (p *C2BPayment) Time() time.Time {
	t, err := time.ParseInLocation("20060102150405", p.TransTime, eat)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// Payer is the customer's display name.
func (p *C2BPayment) Payer() string {
	name := p.FirstName
	if p.MiddleName != "" {
		name += " " + p.MiddleName
	}
	if p.LastName != "" {
		name += " " + p.LastName
	}
	return name
}

// ReversalResult is the payload Daraja posts to a reversal ResultURL.
type ReversalResult struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
	} `json:"Result"`
}

// Ack is the JSON a webhook answers with — ResultCode 0/"0" accepts.
var Ack = map[string]any{"ResultCode": "0", "ResultDesc": "Accepted"}

// Reject refuses a C2B validation request.
func Reject(desc string) map[string]any {
	return map[string]any{"ResultCode": "C2B00016", "ResultDesc": desc}
}

// EAT is Africa/Nairobi — Daraja timestamps are local Kenyan time.
var eat = func() *time.Location {
	l, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		return time.FixedZone("EAT", 3*3600)
	}
	return l
}()

// String is a small helper for building callback event summaries.
func (c *STKCallback) String() string {
	cb := c.Body.StkCallback
	return fmt.Sprintf("%s (code %d)", cb.ResultDesc, cb.ResultCode)
}
