package bindings

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mauzo/desktop/settings"
)

func TestCheckServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
	}))
	defer srv.Close()

	app := NewApp("test", settings.Settings{APIBaseURL: srv.URL, HTTPTimeout: time.Second})
	st := app.CheckServer()
	if !st.Reachable || st.Status != "ok" || st.Version != "1.0.0" || st.Error != "" {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestCheckServerUnreachable(t *testing.T) {
	app := NewApp("test", settings.Settings{APIBaseURL: "http://127.0.0.1:1", HTTPTimeout: time.Second})
	if st := app.CheckServer(); st.Reachable || st.Error == "" {
		t.Fatalf("expected unreachable, got %+v", st)
	}
}
