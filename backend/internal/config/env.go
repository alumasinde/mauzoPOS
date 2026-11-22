package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func getStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func getList(key string) []string {
	raw := getStr(key, "")
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// envReader collects parse errors so Load can report all bad values at once.
type envReader struct{ err error }

func (e *envReader) fail(key, raw string, err error) {
	e.err = errors.Join(e.err, fmt.Errorf("config: invalid %s=%q: %w", key, raw, err))
}

func (e *envReader) int(key string, def int) int {
	raw := getStr(key, "")
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		e.fail(key, raw, err)
		return def
	}
	return v
}

func (e *envReader) float(key string, def float64) float64 {
	raw := getStr(key, "")
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		e.fail(key, raw, err)
		return def
	}
	return v
}

func (e *envReader) duration(key string, def time.Duration) time.Duration {
	raw := getStr(key, "")
	if raw == "" {
		return def
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		e.fail(key, raw, err)
		return def
	}
	return v
}

// clampInt bounds v so narrowing conversions are provably safe.
func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
