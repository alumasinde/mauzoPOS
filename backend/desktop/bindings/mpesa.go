package bindings

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

// MpesaPush sends a customer PIN prompt (POST /pos/mpesa/stk). The cashier
// public id proves the caller's pos.sale.create on the server.
func (a *App) MpesaPush(phone, amount, accountRef string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("POST", "/api/v1/pos/mpesa/stk", map[string]string{
		"cashier_id": a.cashierID(), "phone": phone, "amount": amount,
		"account_ref": accountRef,
	}, &out)
	return out, err
}

// MpesaStatus polls one push's state; the server asks Daraja when the
// callback is overdue.
func (a *App) MpesaStatus(id string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/mpesa/"+urlQuery(id)+"/status", nil, &out)
	return out, err
}

// MpesaRecent lists unallocated collections — feed hydration on (re)connect.
func (a *App) MpesaRecent() (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/mpesa/recent", nil, &out)
	return out, err
}

// MpesaAllocate links a feed collection to a sale receipt.
func (a *App) MpesaAllocate(id, saleRef string) error {
	return a.request("POST", "/api/v1/pos/mpesa/"+urlQuery(id)+"/allocate",
		map[string]string{"cashier_id": a.cashierID(), "sale_ref": saleRef}, nil)
}

// ---- live feed --------------------------------------------------------------
// A goroutine holds the SSE stream and re-emits each collection as a Wails
// event ("mpesa:feed"). Reconnects with backoff until stopped; the cashier's
// token never leaves the Go side.

var feedState = struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	active bool
}{}

// MpesaFeedStart begins streaming confirmed collections to the UI.
func (a *App) MpesaFeedStart() {
	feedState.mu.Lock()
	defer feedState.mu.Unlock()
	if feedState.active || a.token == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	feedState.cancel, feedState.active = cancel, true
	go a.pumpFeed(ctx)
}

// MpesaFeedStop closes the stream.
func (a *App) MpesaFeedStop() {
	feedState.mu.Lock()
	defer feedState.mu.Unlock()
	if feedState.cancel != nil {
		feedState.cancel()
		feedState.cancel, feedState.active = nil, false
	}
}

// pumpFeed maintains the SSE connection with bounded backoff.
func (a *App) pumpFeed(ctx context.Context) {
	defer func() {
		feedState.mu.Lock()
		feedState.active = false
		feedState.mu.Unlock()
	}()
	client := &http.Client{Timeout: 0} //nolint:gosec // SSE is intentionally unbounded
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := a.readFeed(ctx, client); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// readFeed consumes one SSE connection until it drops.
func (a *App) readFeed(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		a.settings.APIBaseURL+"/api/v1/pos/mpesa/feed", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Accept", "text/event-stream")
	res, err := client.Do(req)
	if err != nil {
		return errors.New("feed unreachable")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return errors.New("feed rejected")
	}
	sc := bufio.NewScanner(res.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		wailsrt.EventsEmit(a.ctx, "mpesa:feed", json.RawMessage(payload))
	}
	return sc.Err()
}
