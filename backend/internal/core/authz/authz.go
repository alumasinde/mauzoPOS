// Package authz is the permission registry. Every module declares its
// permissions (with metadata) in its permissions.go at init; the registry is
// synced to the permissions table at boot, and route guards panic on unknown
// strings so typos fail at startup rather than silently denying.
package authz

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Scope separates the two auth contexts. Platform permissions are never
// grantable to tenant roles and vice versa.
type Scope string

// Permission scopes.
const (
	ScopePlatform Scope = "platform"
	ScopeTenant   Scope = "tenant"
)

// Permission is one registered permission.
type Permission struct {
	// Code is module.resource.action, e.g. "sales.refund.create".
	Code string
	// Module is the registry module that must be enabled for the tenant before
	// the permission takes effect. Empty means always available (core).
	Module string
	// Description is shown in the role editor.
	Description string
	// Sensitive actions are always audited and, at the POS, need a supervisor
	// PIN when the cashier lacks the permission.
	Sensitive bool
}

// Scope derives the auth context from the code prefix.
func (p Permission) Scope() Scope {
	if strings.HasPrefix(p.Code, "platform.") {
		return ScopePlatform
	}
	return ScopeTenant
}

var (
	mu       sync.RWMutex
	registry = map[string]Permission{}
)

// Define registers permissions. It panics on duplicates or malformed codes —
// programming errors caught at boot.
func Define(perms ...Permission) {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range perms {
		if strings.Count(p.Code, ".") < 1 || strings.ContainsAny(p.Code, " */") {
			panic(fmt.Sprintf("authz: malformed permission %q", p.Code))
		}
		if _, dup := registry[p.Code]; dup {
			panic(fmt.Sprintf("authz: duplicate permission %q", p.Code))
		}
		registry[p.Code] = p
	}
}

// Registered reports whether code is a known permission.
func Registered(code string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[code]
	return ok
}

// Lookup returns a permission's metadata.
func Lookup(code string) (Permission, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[code]
	return p, ok
}

// All returns every registered permission sorted by code.
func All() []Permission {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Permission, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Permission) int { return strings.Compare(a.Code, b.Code) })
	return out
}

// InScope returns the codes of every permission in scope s, sorted.
func InScope(s Scope) []string {
	var out []string
	for _, p := range All() {
		if p.Scope() == s {
			out = append(out, p.Code)
		}
	}
	return out
}

// complete is set once every module's permissions are linked
// (modules/registerall). Syncing the registry to the database is refused
// before that, because a partial registry would delete real grants.
var complete bool

// MarkComplete records that the registry is complete. Called by
// modules/registerall — and by test fixtures, which sync only the test DB.
func MarkComplete() {
	mu.Lock()
	defer mu.Unlock()
	complete = true
}

// IsComplete reports whether MarkComplete was called.
func IsComplete() bool {
	mu.RLock()
	defer mu.RUnlock()
	return complete
}
