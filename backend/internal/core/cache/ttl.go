// Package cache provides a small in-process TTL cache used for per-request
// authorization data (role permissions, enabled modules, branch access).
//
// It is per API process: a write on this process invalidates immediately;
// other replicas converge within the TTL. Swap for Redis pub/sub
// invalidation when running several replicas makes that window matter.
package cache

import (
	"sync"
	"time"
)

// TTL is a concurrency-safe map with per-entry expiry.
type TTL[K comparable, V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	entries map[K]entry[V]
	now     func() time.Time
}

type entry[V any] struct {
	v   V
	exp time.Time
}

// New builds a cache. max bounds memory; when full, expired entries are
// swept and, if still full, the cache is reset.
func New[K comparable, V any](ttl time.Duration, limit int) *TTL[K, V] {
	return &TTL[K, V]{ttl: ttl, max: limit, entries: map[K]entry[V]{}, now: time.Now}
}

// Get returns a live value.
func (c *TTL[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[k]
	if !ok || c.now().After(e.exp) {
		var zero V
		return zero, false
	}
	return e.v, true
}

// Set stores v for the cache TTL.
func (c *TTL[K, V]) Set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.entries) >= c.max {
		for key, e := range c.entries {
			if now.After(e.exp) {
				delete(c.entries, key)
			}
		}
		if len(c.entries) >= c.max {
			c.entries = map[K]entry[V]{}
		}
	}
	c.entries[k] = entry[V]{v: v, exp: now.Add(c.ttl)}
}

// Delete invalidates one key.
func (c *TTL[K, V]) Delete(k K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, k)
}

// Clear invalidates everything (e.g. after a platform-wide module change).
func (c *TTL[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[K]entry[V]{}
}

// GetOrLoad returns the cached value or calls load and caches its result.
// Errors are not cached.
func (c *TTL[K, V]) GetOrLoad(k K, load func() (V, error)) (V, error) {
	if v, ok := c.Get(k); ok {
		return v, nil
	}
	v, err := load()
	if err != nil {
		return v, err
	}
	c.Set(k, v)
	return v, nil
}
