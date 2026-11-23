package auth

import (
	"sync"
	"time"
)

// Memory bounds: once the map reaches pruneAt entries, expired ones are
// dropped; if it is still above maxEntries (e.g. a spray of random
// identities inside one window), unlocked counters are discarded so the map
// can never grow without bound. Active locks are always kept.
const (
	lockoutPruneAt    = 10_000
	lockoutMaxEntries = 100_000
)

// lockout is an in-memory failed-attempt limiter keyed by actor type +
// identity (and by session for MFA). It intentionally lives in-process; Redis
// can replace it when several API replicas run.
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

// lockedUntil returns when the key's lock ends (zero when not locked).
func (l *lockout) lockedUntil(key string) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return time.Time{}
	}
	return e.lockedUntil
}

// fail records a failed attempt. It reports true when this attempt crossed
// the threshold and locked the key.
func (l *lockout) fail(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.entries) >= lockoutPruneAt {
		l.prune(now)
	}
	e, ok := l.entries[key]
	if !ok || now.Sub(e.firstFailAt) > l.window {
		e = &lockoutEntry{firstFailAt: now, lockedUntil: e.lockedUntilOrZero()}
		l.entries[key] = e
	}
	e.fails++
	if e.fails >= l.maxFails {
		e.lockedUntil = now.Add(l.window)
		e.fails = 0
		e.firstFailAt = now
		return true
	}
	return false
}

// succeed clears the counter.
func (l *lockout) succeed(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// prune drops expired entries, then — if still oversized — every entry that
// is not currently locked. Caller holds l.mu.
func (l *lockout) prune(now time.Time) {
	for k, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.firstFailAt) > l.window {
			delete(l.entries, k)
		}
	}
	if len(l.entries) < lockoutMaxEntries {
		return
	}
	for k, e := range l.entries {
		if now.After(e.lockedUntil) {
			delete(l.entries, k)
		}
	}
}

func (e *lockoutEntry) lockedUntilOrZero() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.lockedUntil
}
