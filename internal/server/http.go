// Package server exposes the REST API, realtime WebSocket, health checks,
// metrics and the embedded control panel.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"callcenter-middleware/internal/ai"
	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/events"
	"callcenter-middleware/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Deps are the collaborators used by HTTP handlers.
type Deps struct {
	Config   *config.Config
	Store    *store.Store
	Core     *core.Core
	Hub      *events.Hub
	Issuer   *auth.Issuer
	Stream   domain.EventSource
	OpenAI   *ai.OpenAI
	Fetcher  ai.RecordingFetcher
	Started  time.Time
	StaticFS http.FileSystem
}

// Server manages the HTTP server lifecycle.
type Server struct {
	d          Deps
	httpServer *http.Server
	stream     domain.EventSource
	wsClients  atomic.Int64
}

// NewServer sets up the chi router and HTTP server.
func NewServer(port string, d Deps) *Server {
	s := &Server{d: d, stream: d.Stream}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.cors)
	s.registerRoutes(r)

	s.httpServer = &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s
}

// Handler returns the root handler (used by tests).
func (s *Server) Handler() http.Handler { return s.httpServer.Handler }

// Start listens on the configured address.
func (s *Server) Start() error {
	slog.Info("Starting HTTP server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
