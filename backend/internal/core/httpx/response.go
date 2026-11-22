// Package httpx holds HTTP response/request helpers shared by every module.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("httpx: encode response", "err", err)
	}
}

// OK writes {"data": v}.
func OK(w http.ResponseWriter, v any) { JSON(w, http.StatusOK, Envelope{Data: v}) }

// Created writes 201 {"data": v}.
func Created(w http.ResponseWriter, v any) { JSON(w, http.StatusCreated, Envelope{Data: v}) }

// NoContent writes 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// List writes {"data": items, "meta": meta}.
func List(w http.ResponseWriter, items any, meta PageMeta) {
	JSON(w, http.StatusOK, Envelope{Data: items, Meta: &meta})
}

// Envelope is the standard {"data": ..., "meta": ...} response wrapper.
type Envelope struct {
	Data any       `json:"data"`
	Meta *PageMeta `json:"meta,omitempty"`
}
