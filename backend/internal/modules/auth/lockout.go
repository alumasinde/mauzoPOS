package auth

import (
	"sync"
	"time"
)

// lockout is an in-memory failed-attempt limiter keyed by actor type +
// identity. It intentionally lives in-process; Redis can replace it when
// several API replicas run.
type lockout struct {
	mu       sync.Mutex
	maxFails int
	window   time.Duration
	entries  map[string]*lockoutEntry
}

type lockoutEntry struct {
	fails       int
	firstFailAt time.Time
	lockedUntil time.Time
}

func newLockout(maxFails int, window time.Duration) *lockout {
	return &lockout{maxFails: maxFails, window: window, entries: map[string]*lockoutEntry{}}
}

// lockedUntil returns the remaining lock time (zero when not locked).
func (l *lockout) lockedUntil(key string) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return time.Time{}
	}
	return e.lockedUntil
}

// fail records a failed attempt and locks when the threshold is crossed.
func (l *lockout) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[key]
	if !ok || now.Sub(e.firstFailAt) > l.window {
		e = &lockoutEntry{firstFailAt: now}
		l.entries[key] = e
	}
	e.fails++
	if e.fails >= l.maxFails {
		e.lockedUntil = now.Add(l.window)
		e.fails = 0
		e.firstFailAt = now
	}
}

// succeed clears the counter.
func (l *lockout) succeed(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}
