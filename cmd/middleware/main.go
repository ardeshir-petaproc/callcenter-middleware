package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"callcenter-middleware/internal/ai"
	"callcenter-middleware/internal/ari"
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/ivr"
	"callcenter-middleware/internal/metrics"
	"callcenter-middleware/internal/notify"
	"callcenter-middleware/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.LoadFromEnv()

	promMetrics := metrics.NewPrometheusMetrics()
	notifier := notify.NewWebhookNotifier(cfg.WebhookURL)
	_ = ai.NewNoopBridge()

	ariClient := ari.NewClient(cfg.ARIURL, cfg.ARIUsername, cfg.ARIPassword, promMetrics)
	stream := ari.NewStream(cfg.ARIWSURL, cfg.ARIApp, cfg.ARIUsername, cfg.ARIPassword, promMetrics)
	engine := ivr.NewEngine(cfg, ariClient, notifier, promMetrics)
	httpServer := server.NewServer(cfg.HTTPPort, stream)

	eventChan := make(chan domain.TelephonyEvent, 128)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := stream.Subscribe(ctx, eventChan); err != nil && err != context.Canceled {
			slog.Error("ARI WebSocket stream exited", "err", err)
		}
	}()

	go func() {
		for ev := range eventChan {
			engine.ProcessEvent(ctx, ev)
		}
	}()

	go func() {
		if err := httpServer.Start(); err != nil {
			slog.Error("HTTP server failed", "err", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 1. Drain active calls fail-open to queue
	engine.DrainAndRelease(shutdownCtx)

	// 2. Stop ARI stream
	cancel()

	// 3. Stop HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "err", err)
	}

	slog.Info("Clean shutdown complete")
}
