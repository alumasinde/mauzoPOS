// Package authz is the permission registry. Every module registers its
// permission strings at init; unknown strings fail route wiring at boot.
// Tenant role→permission mapping lands with the roles tables in Phase 3.
package authz

import (
	"fmt"
	"slices"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]struct{}{}
)

// Register adds permission strings to the catalogue. It panics on duplicates —
// that is a programming error caught at boot.
func Register(perms ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range perms {
		if _, dup := registry[p]; dup {
			panic(fmt.Sprintf("authz: duplicate permission %q", p))
		}
		registry[p] = struct{}{}
	}
}

// Registered reports whether p is a known permission string.
func Registered(p string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[p]
	return ok
}

// All returns every registered permission, sorted.
func All() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for p := range registry {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
