// Package settings holds terminal configuration. Phase 0 reads environment
// variables; Phase 6 moves terminal identity and API URL into the local SQLite
// database written during activation.
package settings

import (
	"os"
	"strings"
	"time"
)

// Settings holds the terminal's runtime configuration.
type Settings struct {
	APIBaseURL  string
	HTTPTimeout time.Duration
}

// Load reads settings from environment variables with local defaults.
func Load() Settings {
	base := strings.TrimRight(os.Getenv("MAUZO_API_URL"), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return Settings{APIBaseURL: base, HTTPTimeout: 10 * time.Second}
}
