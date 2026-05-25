// Package handlers contains the HTTP handlers and middleware for pingapp.
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const reqIDKey ctxKey = iota

// Register binds the application's endpoints to mux.
func Register(mux *http.ServeMux, version string) {
	mux.HandleFunc("GET /ping", ping)
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /version", versionHandler(version))
	// /chaos always returns 500. Used by the Day 4 alert-fire demo
	// (PrometheusRule on 5xx error rate). Cheap to leave in place; the
	// blast radius is one extra response code in the metric histogram.
	mux.HandleFunc("GET /chaos", chaos)
}

func ping(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func chaos(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte("intentional 500 for alert testing"))
}

func versionHandler(v string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": v})
	}
}

// Middleware assigns a request id and emits one structured access log line per request.
func Middleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)

		ctx := context.WithValue(r.Context(), reqIDKey, id)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rec, r.WithContext(ctx))

		log.LogAttrs(ctx, slog.LevelInfo, "request",
			slog.String("request_id", id),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote", r.RemoteAddr),
		)
	})
}

// RequestID returns the request id stored in ctx by Middleware (empty if absent).
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(reqIDKey).(string)
	return v
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// newRequestID returns a canonical-form UUID v4 sourced from crypto/rand.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should never fail; fall back to a time-based stamp.
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102T150405.000000000")))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}
