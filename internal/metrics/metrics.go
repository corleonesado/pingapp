// Package metrics defines and exposes the application's Prometheus metrics.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry is the application's metrics registry. Tests can construct a
// fresh one; production uses prometheus.DefaultRegisterer via NewDefault.
type Registry struct {
	reg            prometheus.Registerer
	gatherer       prometheus.Gatherer
	requestsTotal  *prometheus.CounterVec
	requestSeconds *prometheus.HistogramVec
	inFlight       prometheus.Gauge
}

// NewDefault returns a Registry backed by the global Prometheus registry.
func NewDefault() *Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return New(r, r)
}

// New builds a Registry against the given registerer + gatherer. Useful for tests.
func New(reg prometheus.Registerer, g prometheus.Gatherer) *Registry {
	rt := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed, partitioned by method, path, and status.",
	}, []string{"method", "path", "status"})

	rs := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency partitioned by method and path.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	inf := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	})

	reg.MustRegister(rt, rs, inf)

	return &Registry{
		reg:            reg,
		gatherer:       g,
		requestsTotal:  rt,
		requestSeconds: rs,
		inFlight:       inf,
	}
}

// Handler returns the /metrics HTTP handler.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.gatherer, promhttp.HandlerOpts{})
}

// Middleware wraps next with request-count, duration, and in-flight metrics.
// `routeOf` resolves the actual request to a stable label value (e.g. "/ping"
// rather than "/ping?cachebust=…"), avoiding label cardinality explosions.
func (r *Registry) Middleware(routeOf func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.inFlight.Inc()
		defer r.inFlight.Dec()

		path := routeOf(req)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, req)
		elapsed := time.Since(start).Seconds()

		r.requestsTotal.WithLabelValues(req.Method, path, strconv.Itoa(rec.status)).Inc()
		r.requestSeconds.WithLabelValues(req.Method, path).Observe(elapsed)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
