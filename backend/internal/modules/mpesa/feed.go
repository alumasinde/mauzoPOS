package mpesa

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// feedEvent is one message pushed to subscribed tills.
type feedEvent struct {
	Kind string       `json:"kind"` // collection | reversal | stk
	Txn  *Transaction `json:"txn"`
}

// subscriber is one open till stream.
type subscriber struct {
	ch       chan feedEvent
	branchID uint64
}

// broker fans feed events out to SSE subscribers. A till only receives
// payments for its own branch, or for short codes not tied to a branch, so
// a payment made at one shop never pops up at another.
type broker struct {
	mu   sync.Mutex
	subs map[uint64]map[*subscriber]struct{}
}

func newBroker() *broker { return &broker{subs: map[uint64]map[*subscriber]struct{}{}} }

func (b *broker) publish(tenantID uint64, ev feedEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.subs[tenantID] {
		if ev.Txn != nil && ev.Txn.BranchID != nil && *ev.Txn.BranchID != sub.branchID {
			continue
		}
		select {
		case sub.ch <- ev:
		default: // slow consumer — drop rather than stall the callback
		}
	}
}

func (b *broker) subscribe(tenantID, branchID uint64) (chan feedEvent, func()) {
	sub := &subscriber{ch: make(chan feedEvent, 16), branchID: branchID}
	b.mu.Lock()
	if b.subs[tenantID] == nil {
		b.subs[tenantID] = map[*subscriber]struct{}{}
	}
	b.subs[tenantID][sub] = struct{}{}
	b.mu.Unlock()
	return sub.ch, func() {
		b.mu.Lock()
		delete(b.subs[tenantID], sub)
		b.mu.Unlock()
	}
}

// ServeFeed streams a branch's events to a POS device over SSE.
func (s *Service) ServeFeed(w http.ResponseWriter, r *http.Request, tenantID, branchID uint64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, unsub := s.feed.subscribe(tenantID, branchID)
	defer unsub()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			if ev.Txn != nil {
				v := tillView(*ev.Txn)
				ev.Txn = &v
			}
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

// publishSTK tells the till that fired a prompt how it ended — instantly,
// instead of waiting for its next status poll. A success is also a
// collection, so it lands in the branch's "received" list if the cashier
// closed the payment screen meanwhile.
func (s *Service) publishSTK(tenantID uint64, t *Transaction) {
	s.feed.publish(tenantID, feedEvent{Kind: "stk", Txn: t})
	if t.Status == StSuccess {
		s.feed.publish(tenantID, feedEvent{Kind: "collection", Txn: t})
	}
}
