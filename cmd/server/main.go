// Command server runs the pingapp HTTP service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/corleonesado/pingapp/internal/handlers"
	"github.com/corleonesado/pingapp/internal/logger"
	"github.com/corleonesado/pingapp/internal/metrics"
)

// version is injected at build time via -ldflags "-X main.version=<sha>".
var version = "dev"

func main() {
	port := getenv("PORT", "8080")
	level := getenv("LOG_LEVEL", "info")

	log := logger.New(os.Stdout, level)
	slog.SetDefault(log)

	m := metrics.NewDefault()

	chaosEnabled := getenv("ENABLE_CHAOS", "0") == "1"
	if chaosEnabled {
		log.Warn("chaos endpoint enabled — /chaos will return 500", "endpoint", "/chaos")
	}

	appMux := http.NewServeMux()
	handlers.Register(appMux, version, handlers.Options{EnableChaos: chaosEnabled})

	// Stable route labels — anything else collapses to "unknown" so a noisy
	// 404 scanner can't blow up the cardinality of the metric series.
	routeOf := func(r *http.Request) string {
		switch r.URL.Path {
		case "/ping", "/healthz", "/version", "/chaos":
			return r.URL.Path
		default:
			return "unknown"
		}
	}

	// /metrics is exposed outside the metrics middleware so scrapes don't
	// instrument themselves.
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", m.Handler())
	mux.Handle("/", m.Middleware(routeOf, appMux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handlers.Middleware(log, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
		close(idleConnsClosed)
	}()

	log.Info("starting HTTP server", "port", port, "version", version, "log_level", level)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}

	<-idleConnsClosed
	log.Info("server stopped cleanly")
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
