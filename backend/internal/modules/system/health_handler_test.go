package system

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func serve(h *Handler, path string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	Routes(r, h)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestReadyReportsDependencyFailure(t *testing.T) {
	h := NewHandler("test", map[string]Pinger{"database": fakePinger{errors.New("down")}})
	if rec := serve(h, "/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d", rec.Code)
	}
	if rec := serve(h, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("healthz must not depend on DB, got %d", rec.Code)
	}
}

func TestReadyOK(t *testing.T) {
	h := NewHandler("test", map[string]Pinger{"database": fakePinger{}})
	if rec := serve(h, "/readyz"); rec.Code != http.StatusOK {
		t.Fatalf("readyz status = %d", rec.Code)
	}
}
