package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"callcenter-middleware/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server manages the HTTP server lifecycle.
type Server struct {
	httpServer *http.Server
	stream     domain.EventSource
}

// NewServer sets up the chi router and HTTP server.
func NewServer(port string, stream domain.EventSource) *Server {
	s := &Server{stream: stream}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	s.registerRoutes(r)

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

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
