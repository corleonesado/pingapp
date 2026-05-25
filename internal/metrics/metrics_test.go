package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func newTestRegistry() *Registry {
	r := prometheus.NewRegistry()
	return New(r, r)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func errHandler(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	})
}

func staticRoute(r *http.Request) string { return r.URL.Path }

func TestMiddlewareCountsRequestsByStatus(t *testing.T) {
	m := newTestRegistry()
	mw := m.Middleware(staticRoute, okHandler())

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	}

	body := scrape(t, m)
	if !strings.Contains(body, `http_requests_total{method="GET",path="/ping",status="200"} 3`) {
		t.Fatalf("missing or wrong http_requests_total counter, got:\n%s", body)
	}
}

func TestMiddlewareRecordsLatencyHistogram(t *testing.T) {
	m := newTestRegistry()
	mw := m.Middleware(staticRoute, okHandler())

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

	body := scrape(t, m)
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",path="/ping"} 1`) {
		t.Fatalf("missing histogram count, got:\n%s", body)
	}
}

func TestMiddlewareCapturesNon2xxStatus(t *testing.T) {
	m := newTestRegistry()
	mw := m.Middleware(staticRoute, errHandler(http.StatusInternalServerError))

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	body := scrape(t, m)
	if !strings.Contains(body, `http_requests_total{method="GET",path="/boom",status="500"} 1`) {
		t.Fatalf("expected status=500 counter, got:\n%s", body)
	}
}

func TestInFlightGaugeReturnsToZero(t *testing.T) {
	m := newTestRegistry()
	mw := m.Middleware(staticRoute, okHandler())

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

	body := scrape(t, m)
	// After the synchronous request returns, the gauge should be back to 0.
	if !strings.Contains(body, "http_requests_in_flight 0") {
		t.Fatalf("expected http_requests_in_flight 0 after request, got:\n%s", body)
	}
}

// scrape hits the registry's /metrics handler and returns the body.
func scrape(t *testing.T, m *Registry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics handler returned %d", rec.Code)
	}
	b, _ := io.ReadAll(rec.Body)
	return string(b)
}
