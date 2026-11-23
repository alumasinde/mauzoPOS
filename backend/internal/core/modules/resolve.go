package modules

import (
	"fmt"
	"slices"
)

// ErrUnknown is returned for codes that are not in the registry.
type ErrUnknown struct{ Code string }

func (e ErrUnknown) Error() string { return fmt.Sprintf("modules: unknown module %q", e.Code) }

// Closure returns codes plus every transitive dependency, in catalogue order.
func Closure(codes []string) ([]string, error) {
	want := map[string]bool{}
	var add func(string) error
	add = func(c string) error {
		m, ok := byCode[c]
		if !ok {
			return ErrUnknown{c}
		}
		if want[c] {
			return nil
		}
		want[c] = true
		for _, d := range m.Depends {
			if err := add(d); err != nil {
				return err
			}
		}
		return nil
	}
	for _, c := range codes {
		if err := add(c); err != nil {
			return nil, err
		}
	}
	return inOrder(want), nil
}

// Dependents returns the enabled modules that (transitively) depend on code,
// in catalogue order. Disabling code requires disabling these too.
func Dependents(code string, enabled []string) []string {
	on := map[string]bool{}
	for _, c := range enabled {
		on[c] = true
	}
	hit := map[string]bool{}
	changed := true
	for changed {
		changed = false
		for c := range on {
			if hit[c] || c == code {
				continue
			}
			for _, d := range byCode[c].Depends {
				if d == code || hit[d] {
					hit[c] = true
					changed = true
					break
				}
			}
		}
	}
	return inOrder(hit)
}

// MissingDeps returns the direct and transitive dependencies of code that are
// not in enabled.
func MissingDeps(code string, enabled []string) ([]string, error) {
	all, err := Closure([]string{code})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range all {
		if c != code && !slices.Contains(enabled, c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func inOrder(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for _, c := range order {
		if set[c] {
			out = append(out, c)
		}
	}
	return out
}
