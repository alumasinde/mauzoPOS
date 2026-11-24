// Package httpx holds HTTP response/request helpers shared by every module.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
)

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if env, ok := v.(Envelope); ok {
		env.Data = fixNullSlices(env.Data)
		v = env
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

// fixNullSlices rewrites nil slices to empty ones so list payloads encode as
// [] rather than null — clients index and iterate these fields unconditionally.
// Handles the Data slice itself, map payloads, and slices nested inside DTO
// structs (e.g. detail responses built by append into nil fields).
func fixNullSlices(v any) any {
	rv := reflect.ValueOf(v)
	out := normValue(rv)
	if !out.IsValid() {
		return v
	}
	return out.Interface()
}

// normValue returns rv with nil slices replaced by empty slices, recursively.
// Nil pointers stay nil — a null reference is meaningful, a null list is not.
func normValue(rv reflect.Value) reflect.Value {
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return reflect.MakeSlice(rv.Type(), 0, 0)
		}
		if rv.Type().Elem().Kind() == reflect.Interface {
			out := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out.Index(i).Set(normValue(rv.Index(i)))
			}
			return out
		}
	case reflect.Map:
		if !rv.IsNil() && rv.Type().Key().Kind() == reflect.String &&
			rv.Type().Elem().Kind() == reflect.Interface {
			out := reflect.MakeMapWithSize(rv.Type(), rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				out.SetMapIndex(iter.Key(), normValue(iter.Value()))
			}
			return out
		}
	case reflect.Interface:
		if !rv.IsNil() {
			return reflect.ValueOf(normValue(rv.Elem()).Interface())
		}
	case reflect.Pointer:
		if !rv.IsNil() && rv.Elem().Kind() == reflect.Struct {
			normStruct(rv.Elem())
		}
	case reflect.Struct:
		// Rebuild only plain DTOs — types with no exported fields (time.Time,
		// decimal.Decimal) are returned untouched.
		if hasNoExported(rv.Type()) {
			return rv
		}
		out := reflect.New(rv.Type()).Elem()
		for i := 0; i < rv.NumField(); i++ {
			f := out.Field(i)
			if f.CanSet() {
				f.Set(normValue(rv.Field(i)))
			}
		}
		return out
	}
	return rv
}

// normStruct normalizes slice fields of an addressable struct in place.
func normStruct(rv reflect.Value) {
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if !f.CanSet() {
			continue
		}
		f.Set(normValue(f))
	}
}

// hasNoExported reports whether t has no exported fields.
func hasNoExported(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			return false
		}
	}
	return true
}

// Envelope is the standard {"data": ..., "meta": ...} response wrapper.
type Envelope struct {
	Data any       `json:"data"`
	Meta *PageMeta `json:"meta,omitempty"`
}
