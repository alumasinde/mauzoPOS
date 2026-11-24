package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// feedEvent is one message pushed to subscribed tills.
type feedEvent struct {
	Kind string       `json:"kind"` // collection | reversal
	Txn  *Transaction `json:"txn"`
}

// broker fans feed events out to SSE subscribers, keyed by tenant. Tills get a
// cheap always-open stream so a confirmation lands on screen in <1s.
type broker struct {
	mu   sync.Mutex
	subs map[uint64]map[chan feedEvent]struct{}
}

func newBroker() *broker { return &broker{subs: map[uint64]map[chan feedEvent]struct{}{}} }

func (b *broker) publish(tenantID uint64, ev feedEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[tenantID] {
		select {
		case ch <- ev:
		default: // slow consumer — drop rather than stall the callback
		}
	}
}

func (b *broker) subscribe(tenantID uint64) (chan feedEvent, func()) {
	ch := make(chan feedEvent, 16)
	b.mu.Lock()
	if b.subs[tenantID] == nil {
		b.subs[tenantID] = map[chan feedEvent]struct{}{}
	}
	b.subs[tenantID][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs[tenantID], ch)
		b.mu.Unlock()
	}
}

// ServeFeed streams events to a POS device over SSE.
func (s *Service) ServeFeed(w http.ResponseWriter, r *http.Request, tenantID uint64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ch, unsub := s.feed.subscribe(tenantID)
	defer unsub()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			writeSSE(w, flusher, ev)
		case <-ping.C:
			_, _ = w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, f http.Flusher, ev feedEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
	f.Flush()
}

// FeedSince returns recent unallocated successes for initial feed hydration —
// a till that reconnects re-reads these rather than trusting the stream.
func (s *Service) FeedSince(ctx context.Context, tenantID uint64) ([]Transaction, error) {
	return s.repo.Unallocated(ctx, tenantID, time.Now().Add(-pendingFeedDays))
}
