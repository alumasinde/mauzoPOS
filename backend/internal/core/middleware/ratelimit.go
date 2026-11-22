package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"mauzo/internal/core/httpx"
)

// RateLimiter is an in-memory per-client token bucket. It is per process; when
// we run several API replicas it will be swapped for a Redis-backed limiter
// behind the same middleware signature.
type RateLimiter struct {
	rps   rate.Limit
	burst int

	mu      sync.Mutex
	clients map[string]*client
}

type client struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter builds a per-client token-bucket rate limiter.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{rps: rate.Limit(rps), burst: burst, clients: map[string]*client{}}
}

func (rl *RateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	c, ok := rl.clients[key]
	if !ok {
		c = &client{lim: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clients[key] = c
	}
	c.lastSeen = now
	return c.lim.AllowN(now, 1)
}

// Cleanup drops clients idle for longer than idle. Call periodically.
func (rl *RateLimiter) Cleanup(idle time.Duration) {
	cutoff := time.Now().Add(-idle)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, c := range rl.clients {
		if c.lastSeen.Before(cutoff) {
			delete(rl.clients, k)
		}
	}
}

// Handler limits by remote IP. X-Forwarded-For is deliberately ignored until a
// trusted reverse proxy is configured in production.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if !rl.allow(host, time.Now()) {
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, r, httpx.ErrTooManyRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
