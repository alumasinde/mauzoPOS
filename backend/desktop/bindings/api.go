package bindings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrOffline is returned when the API cannot be reached — callers then fall
// back to the local database.
var ErrOffline = errors.New("offline")

// apiError surfaces the server's error payload.
type apiError struct {
	msg    string
	status int
}

func (e *apiError) Error() string { return e.msg }

// refused reports a definitive rejection of the request itself (4xx other
// than auth, timeouts and rate limits) — retrying the same body cannot help.
func (e *apiError) refused() bool {
	switch e.status {
	case 401, 403, 408, 429:
		return false
	}
	return e.status >= 400 && e.status < 500
}

// request performs one API call; when the device is activated it attaches the
// bearer token. Network failures map to ErrOffline so callers can degrade to
// the local working set.
func (a *App) request(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(a.ctx, method, a.settings.APIBaseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	res, err := a.http.Do(req)
	if err != nil {
		return ErrOffline
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var env struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Error.Message != "" {
			return &apiError{msg: env.Error.Message, status: res.StatusCode}
		}
		return &apiError{msg: fmt.Sprintf("HTTP %d", res.StatusCode), status: res.StatusCode}
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	return json.Unmarshal(env.Data, out)
}

// online reports whether the API is reachable right now.
func (a *App) online() bool {
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet,
		a.settings.APIBaseURL+"/readyz", nil)
	if err != nil {
		return false
	}
	res, err := a.http.Do(req)
	if err != nil {
		return false
	}
	_ = res.Body.Close()
	return res.StatusCode == http.StatusOK
}

// urlQuery escapes a query parameter.
func urlQuery(v string) string { return url.QueryEscape(v) }
