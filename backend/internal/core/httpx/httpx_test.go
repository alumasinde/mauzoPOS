package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRejectsUnknownFields(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a","tenant_id":5}`))
	err := Decode(httptest.NewRecorder(), r, &dst)
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "bad_request" {
		t.Fatalf("expected bad_request, got %v", err)
	}
}

func TestDecodeRejectsTrailingData(t *testing.T) {
	var dst map[string]any
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}{"b":2}`))
	if err := Decode(httptest.NewRecorder(), r, &dst); err == nil {
		t.Fatal("expected error for trailing JSON")
	}
}

func TestWriteErrorHidesInternalErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("db password leaked"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatal("internal error message leaked to client")
	}
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "internal_error" {
		t.Fatalf("code = %q", body.Error.Code)
	}
}

func TestParsePageClamps(t *testing.T) {
	p := ParsePage(httptest.NewRequest(http.MethodGet, "/?page=-3&per_page=9999&q=+milk+", nil))
	if p.Page != 1 || p.PerPage != maxPerPage || p.Query != "milk" {
		t.Fatalf("unexpected page: %+v", p)
	}
	if p.Offset() != 0 {
		t.Fatalf("offset = %d", p.Offset())
	}
}
