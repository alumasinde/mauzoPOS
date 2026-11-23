// Package settings provides the tenant_settings key/value store and a small
// JSON-schema validator. Modules register a schema per settings key so writes
// are validated the same way everywhere (onboarding, back-office, POS sync).
package settings

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	mu      sync.RWMutex
	schemas = map[string]json.RawMessage{}
)

// ErrUnknownKey is returned when a setting has no registered schema — unlisted
// keys are rejected rather than silently stored.
var ErrUnknownKey = fmt.Errorf("settings: unknown key")

// RegisterSchema binds a JSON schema to a settings key. It panics on a
// duplicate key or invalid JSON — a programming error caught at boot.
func RegisterSchema(key string, schema any) {
	b, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("settings: marshal schema %q: %v", key, err))
	}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := schemas[key]; dup {
		panic(fmt.Sprintf("settings: duplicate schema %q", key))
	}
	schemas[key] = b
}

// Validate checks raw JSON against the schema registered for key.
func Validate(key string, raw json.RawMessage) error {
	mu.RLock()
	schema, ok := schemas[key]
	mu.RUnlock()
	if !ok {
		return ErrUnknownKey
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("settings %q: invalid JSON: %w", key, err)
	}
	var sch map[string]any
	if err := json.Unmarshal(schema, &sch); err != nil {
		return fmt.Errorf("settings %q: bad schema: %w", key, err)
	}
	return check(doc, sch, key)
}

// ValidateSchema checks raw JSON against an ad-hoc schema (e.g. a module's
// config schema). The schema is round-tripped through JSON so Go literals
// (ints, []string) compare the same way as stored schemas.
func ValidateSchema(schema map[string]any, raw json.RawMessage, path string) error {
	b, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("%s: bad schema: %w", path, err)
	}
	var sch map[string]any
	if err := json.Unmarshal(b, &sch); err != nil {
		return fmt.Errorf("%s: bad schema: %w", path, err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return check(doc, sch, path)
}

// check validates doc against a schema subset: type, required, properties,
// additionalProperties, enum, items, min/max(imum|Length|Items).
func check(doc any, sch map[string]any, path string) error {
	if want, ok := sch["type"].(string); ok && !typeMatches(doc, want) {
		return fmt.Errorf("%s: expected %s", path, want)
	}
	if enum, ok := sch["enum"].([]any); ok && !inEnum(doc, enum) {
		return fmt.Errorf("%s: value not in enum", path)
	}
	switch v := doc.(type) {
	case map[string]any:
		for _, req := range stringSlice(sch["required"]) {
			if _, ok := v[req]; !ok {
				return fmt.Errorf("%s: missing required property %q", path, req)
			}
		}
		props, _ := sch["properties"].(map[string]any)
		allowExtra, _ := sch["additionalProperties"].(bool)
		for k, val := range v {
			sub, ok := props[k]
			if !ok {
				if sch["additionalProperties"] == false && !allowExtra {
					return fmt.Errorf("%s: unexpected property %q", path, k)
				}
				continue
			}
			subSch, _ := sub.(map[string]any)
			if err := check(val, subSch, path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		if items, ok := sch["items"].(map[string]any); ok {
			for i, val := range v {
				if err := check(val, items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	case string:
		if n, ok := sch["minLength"].(float64); ok && len(v) < int(n) {
			return fmt.Errorf("%s: shorter than %d", path, int(n))
		}
		if n, ok := sch["maxLength"].(float64); ok && len(v) > int(n) {
			return fmt.Errorf("%s: longer than %d", path, int(n))
		}
	case float64:
		if n, ok := sch["minimum"].(float64); ok && v < n {
			return fmt.Errorf("%s: below minimum %v", path, n)
		}
		if n, ok := sch["maximum"].(float64); ok && v > n {
			return fmt.Errorf("%s: above maximum %v", path, n)
		}
	}
	return nil
}

func typeMatches(v any, want string) bool {
	switch want {
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "number":
		_, ok := v.(float64)
		return ok
	case "integer":
		f, ok := v.(float64)
		return ok && f == float64(int64(f))
	}
	return false
}

func inEnum(v any, enum []any) bool {
	for _, e := range enum {
		if fmt.Sprint(e) == fmt.Sprint(v) {
			return true
		}
	}
	return false
}

func stringSlice(v any) []string {
	out := []string{}
	arr, ok := v.([]any)
	if !ok {
		return out
	}
	for _, s := range arr {
		if str, ok := s.(string); ok {
			out = append(out, str)
		}
	}
	return out
}
