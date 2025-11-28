package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"prometheus/internal/api/health"
	telegramapi "prometheus/internal/api/telegram"
	"prometheus/internal/metrics"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ServerConfig contains configuration for HTTP server
type ServerConfig struct {
	Port             int
	ServiceName      string
	Version          string
	TelegramWebhook  *telegramapi.WebhookHandler // Optional Telegram webhook handler
}

// Server wraps HTTP server with lifecycle management
type Server struct {
	httpServer *http.Server
	log        *logger.Logger
}

// NewServer creates and configures HTTP server with all routes
func NewServer(cfg ServerConfig, healthHandler *health.Handler, log *logger.Logger) *Server {
	mux := http.NewServeMux()

	// Health check endpoints (Kubernetes probes)
	mux.HandleFunc("/health", healthHandler.HandleHealth)
	mux.HandleFunc("/ready", healthHandler.HandleReadiness)
	mux.HandleFunc("/live", healthHandler.HandleLiveness)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", metrics.Handler())

	// Telegram webhook endpoint (if configured)
	if cfg.TelegramWebhook != nil {
		mux.HandleFunc("/telegram/webhook", cfg.TelegramWebhook.ServeHTTP)
		mux.HandleFunc("/telegram/health", cfg.TelegramWebhook.HealthCheck)
		log.Info("✓ Telegram webhook registered at /telegram/webhook")
	}

	// Root endpoint (service info)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"service":"%s","version":"%s","status":"running"}`,
			cfg.ServiceName, cfg.Version)
	})

	port := 8080
	if cfg.Port > 0 {
		port = cfg.Port
	}

	log.Infof("HTTP server configured on port %d", port)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		log:        log,
	}
}

// Start begins listening for HTTP requests
// Blocks until server is stopped or encounters an error
func (s *Server) Start() error {
	s.log.Infof("Starting HTTP server on %s", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "http server failed")
	}

	return nil
}

// Shutdown gracefully stops the HTTP server
// Waits for active connections to complete within timeout
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Stopping HTTP server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "http server shutdown failed")
	}

	s.log.Info("✓ HTTP server stopped")
	return nil
}
