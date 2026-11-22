package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1 MiB; uploads use dedicated endpoints

// Decode reads a JSON body into dst, rejecting unknown fields, trailing data and
// oversized payloads. Returned errors are *Error and safe to send to clients.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrBadRequest.WithDetails(map[string]any{"body": "must contain a single JSON object"})
	}
	return nil
}

func decodeError(err error) *Error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxErr *http.MaxBytesError
	msg := "malformed JSON"
	switch {
	case errors.Is(err, io.EOF):
		msg = "request body is empty"
	case errors.As(err, &syntaxErr):
		msg = fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		msg = fmt.Sprintf("field %q has the wrong type", typeErr.Field)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		msg = "unknown field " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	case errors.As(err, &maxErr):
		msg = "request body is too large"
	}
	return ErrBadRequest.WithDetails(map[string]any{"body": msg})
}
