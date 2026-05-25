package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newMux(version string) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, version)
	return mux
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPing(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	newMux("test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if got := strings.TrimSpace(string(body)); got != "pong" {
		t.Fatalf("body: got %q, want %q", got, "pong")
	}
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newMux("test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
}

func TestVersionReturnsInjectedSHA(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	newMux("abc123").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type: got %q, want application/json", ct)
	}
	var out map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["version"] != "abc123" {
		t.Fatalf("version: got %q, want %q", out["version"], "abc123")
	}
}

func TestMiddlewareAssignsRequestIDWhenAbsent(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	Middleware(discardLogger(), newMux("test")).ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
	// canonical UUID v4: 8-4-4-4-12 hex
	if len(id) != 36 || strings.Count(id, "-") != 4 {
		t.Fatalf("X-Request-ID not canonical UUID: %q", id)
	}
}

func TestChaosReturns500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chaos", nil)
	newMux("test").ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", rec.Code)
	}
}

func TestMiddlewarePropagatesIncomingRequestID(t *testing.T) {
	const incoming = "11111111-2222-4333-8444-555555555555"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", incoming)

	Middleware(discardLogger(), newMux("test")).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != incoming {
		t.Fatalf("X-Request-ID echo: got %q, want %q", got, incoming)
	}
}
