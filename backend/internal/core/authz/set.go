package authz

import (
	"path"
	"slices"
)

// Set is an immutable set of granted permission codes.
type Set struct {
	all   bool // grants every permission of its scope (tenant owner)
	codes map[string]struct{}
}

// NewSet builds a set from codes.
func NewSet(codes ...string) Set {
	m := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		m[c] = struct{}{}
	}
	return Set{codes: m}
}

// AllOf returns a set granting every permission (used for locked owner roles).
func AllOf() Set { return Set{all: true} }

// Has reports whether code is granted.
func (s Set) Has(code string) bool {
	if s.all {
		return true
	}
	_, ok := s.codes[code]
	return ok
}

// IsAll reports whether the set grants everything.
func (s Set) IsAll() bool { return s.all }

// Contains reports whether every code is granted (used to stop privilege
// escalation: you may only grant what you hold).
func (s Set) Contains(codes []string) bool {
	for _, c := range codes {
		if !s.Has(c) {
			return false
		}
	}
	return true
}

// Codes returns the explicit codes, sorted. For an all-set it returns every
// registered permission of scope sc.
func (s Set) Codes(sc Scope) []string {
	if s.all {
		return InScope(sc)
	}
	out := make([]string, 0, len(s.codes))
	for c := range s.codes {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

// Expand resolves patterns ("catalog.*", "*.view", exact codes) against the
// registered permissions of scope sc. Patterns use path.Match syntax; '*'
// spans dots. Unknown exact codes are dropped, so templates may reference
// permissions that a later phase registers.
func Expand(sc Scope, patterns []string) []string {
	var out []string
	for _, code := range InScope(sc) {
		for _, p := range patterns {
			if ok, _ := path.Match(p, code); ok {
				out = append(out, code)
				break
			}
		}
	}
	return out
}
