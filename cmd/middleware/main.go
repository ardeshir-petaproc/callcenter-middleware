package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // embedded zone database for schedules in minimal images

	"callcenter-middleware/internal/ai"
	"callcenter-middleware/internal/ari"
	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/events"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/metrics"
	"callcenter-middleware/internal/notify"
	"callcenter-middleware/internal/seed"
	"callcenter-middleware/internal/server"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
	"callcenter-middleware/internal/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.LoadFromEnv()
	if cfg.JWTGenerated {
		slog.Warn("JWT_SECRET not set: generated an ephemeral secret, sessions reset on restart")
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.Error("Open database failed", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}
	defer st.Close()
	idp := joowProvider(cfg, st)
	bootstrap(cfg, st, idp)

	promMetrics := metrics.NewPrometheusMetrics()
	hub := events.NewHub()
	ariClient := ari.NewClient(cfg.ARIURL, cfg.ARIUsername, cfg.ARIPassword, cfg.ARIApp, promMetrics)
	stream := ari.NewStream(cfg.ARIWSURL, cfg.ARIApp, cfg.ARIUsername, cfg.ARIPassword, promMetrics)
	openai := ai.NewOpenAI(cfg)
	analyzer := &ai.Analyzer{Store: st, OpenAI: openai, Fetcher: ariClient, Pub: hub}

	callCore := core.New(core.Deps{
		Config: cfg, Switch: ariClient, Store: st, Publisher: hub, Metrics: promMetrics,
		Notifier: notify.NewWebhookNotifier(cfg.WebhookURL),
		AI:       ai.NewManager(cfg, &ai.Directory{Store: st}), Analyzer: analyzer,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventChan := make(chan domain.TelephonyEvent, 512)
	coreCtx, stopCore := context.WithCancel(context.Background())
	go callCore.Run(coreCtx, eventChan)
	if err := callCore.Reload(ctx); err != nil {
		slog.Error("Load configuration failed", "err", err)
		os.Exit(1)
	}

	go func() {
		if err := stream.Subscribe(ctx, eventChan); err != nil && err != context.Canceled {
			slog.Error("ARI WebSocket stream exited", "err", err)
		}
	}()
	startBackground(ctx, st, hub, analyzer)

	httpServer := server.NewServer(cfg.HTTPPort, server.Deps{
		Config: cfg, Store: st, Core: callCore, Hub: hub, Issuer: auth.NewIssuer(cfg.JWTSecret, cfg.TokenTTL),
		Stream: stream, OpenAI: openai, Fetcher: ariClient, Started: time.Now(),
		StaticFS: server.StaticFS(web.Assets()), Joow: idp, Tickets: &tickets.Service{Store: st, Pub: hub, Joow: idp}, DevJoowTokens: cfg.JoowMode != "http",
	})
	go func() {
		if err := httpServer.Start(); err != nil {
			slog.Error("HTTP server failed", "err", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.DrainTimeout+5*time.Second)
	defer shutdownCancel()

	// 1. Drain calls that are not with a human fail-open to the default queue.
	callCore.DrainAndRelease(shutdownCtx)
	// 2. Stop ARI stream and the call core.
	cancel()
	stopCore()
	// 3. Stop HTTP server.
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "err", err)
	}
	slog.Info("Clean shutdown complete")
}

func bootstrap(cfg *config.Config, st *store.Store, idp joow.Provider) {
	root := context.Background()
	ctx := store.ForTenant(root, core.DefaultTenant)
	if err := seed.EnsureAdmin(ctx, st, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		slog.Error("Bootstrap admin failed", "err", err)
		os.Exit(1)
	}
	if err := seed.EnsureDefaults(ctx, st); err != nil {
		slog.Warn("Seed defaults failed", "err", err)
	}
	if !cfg.SeedDemo {
		return
	}
	if err := seed.Demo(ctx, st); err != nil {
		slog.Warn("Seed demo data failed", "err", err)
	}
	if err := seed.DemoSecondTenant(root, st); err != nil {
		slog.Warn("Seed second demo tenant failed", "err", err)
	}
	if local, ok := idp.(*joow.Local); ok {
		if err := seed.DemoJoowAccounts(root, local.Upsert); err != nil {
			slog.Warn("Seed Joow accounts failed", "err", err)
		}
	}
}

// joowProvider selects the Joow integration: the real Joow API or the
// self-contained local provider used for development and demos.
func joowProvider(cfg *config.Config, st *store.Store) joow.Provider {
	if cfg.JoowMode == "http" && cfg.JoowAPIURL != "" {
		slog.Info("Joow integration: HTTP", "url", cfg.JoowAPIURL)
		return joow.NewHTTP(cfg.JoowAPIURL, cfg.JoowAPIKey)
	}
	slog.Warn("Joow integration: local development provider")
	return joow.NewLocal(st.DB(), cfg.JWTSecret)
}

// startBackground runs webhook delivery and voicemail transcription.
func startBackground(ctx context.Context, st *store.Store, hub *events.Hub, analyzer *ai.Analyzer) {
	hookEvents, _ := hub.Subscribe(1024)
	go notify.NewDispatcher(st).Run(ctx, hookEvents)

	vmEvents, _ := hub.Subscribe(64)
	go func() {
		for ev := range vmEvents {
			if vm, ok := ev.Data.(*domain.Voicemail); ok && ev.Type == domain.EvVoicemail {
				go analyzer.TranscribeVoicemail(ctx, *vm)
			}
		}
	}()
}
