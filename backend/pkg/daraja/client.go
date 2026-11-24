// Package daraja is a thin, production-safe client for the Safaricom Daraja
// API (M-Pesa Express / C2B / Reversal). It owns OAuth token caching — a
// token is reused until sixty seconds before expiry — and builds the
// timestamped STK password internally so callers never hand-roll base64.
package daraja

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Base URLs per environment.
const (
	BaseSandbox    = "https://sandbox.safaricom.co.ke"
	BaseProduction = "https://api.safaricom.co.ke"
)

// STK transaction types.
const (
	TypePayBill = "CustomerPayBillOnline"  // Pay Bill shortcode
	TypeTill    = "CustomerBuyGoodsOnline" // Buy Goods till
)

// ResultCodes maps well-known Daraja result codes to UI-friendly text.
var ResultCodes = map[int]string{
	0:    "Success",
	1:    "Insufficient funds",
	1031: "Incorrect mobile number",
	1032: "Cancelled by customer",
	1037: "Customer unreachable — prompt timed out",
	2001: "Wrong M-Pesa PIN",
}

// Error is a non-2xx Daraja response; Message carries the server's own text.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("daraja: %s", e.Message) }

// Client calls one Daraja environment with one app's credentials.
type Client struct {
	base   string
	key    string
	secret string
	http   *http.Client
	clock  func() time.Time // test hook

	mu        sync.Mutex
	token     string
	tokenExpr time.Time
}

// New builds a client for the given base URL (BaseSandbox or BaseProduction)
// and consumer credentials. httpClient may be nil for the default.
func New(base, consumerKey, consumerSecret string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{base: base, key: consumerKey, secret: consumerSecret, http: httpClient, clock: time.Now}
}

// Token returns a cached OAuth access token, refreshing sixty seconds early.
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && c.clock().Before(c.tokenExpr) {
		tok := c.token
		c.mu.Unlock()
		return tok, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.base+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(url.QueryEscape(c.key), url.QueryEscape(c.secret))

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := c.do(req, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", &Error{Message: "oauth returned no access_token"}
	}
	var secs int64
	fmt.Sscanf(out.ExpiresIn, "%d", &secs) //nolint:errcheck // best-effort; default below
	if secs <= 0 {
		secs = 3500
	}
	c.mu.Lock()
	c.token = out.AccessToken
	c.tokenExpr = c.clock().Add(time.Duration(secs-60) * time.Second)
	c.mu.Unlock()
	return out.AccessToken, nil
}

// do executes req, decoding 2xx bodies into out and wrapping the rest.
func (c *Client) do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode/100 != 2 {
		var e struct {
			ErrorCode    string `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
			ResultDesc   string `json:"ResultDesc"`
			ResultCode   any    `json:"ResultCode"`
		}
		msg := res.Status
		if json.Unmarshal(body, &e) == nil {
			switch {
			case e.ErrorMessage != "":
				msg = e.ErrorMessage
			case e.ResultDesc != "":
				msg = e.ResultDesc
			}
		}
		return &Error{Code: res.StatusCode, Message: msg}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// post sends an authenticated JSON POST.
func (c *Client) post(ctx context.Context, path string, in, out any) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	buf, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

// password builds base64(shortcode + passkey + timestamp) and its timestamp.
func (c *Client) password(shortCode, passkey string) (pw, ts string) {
	ts = c.clock().Format("20060102150405")
	return base64.StdEncoding.EncodeToString([]byte(shortCode + passkey + ts)), ts
}
