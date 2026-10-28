package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/events"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/mail"
	"callcenter-middleware/internal/messages"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

// services are the business services shared by the API and background jobs.
type services struct {
	tickets  *tickets.Service
	kb       *kb.Service
	mail     *mail.Service
	messages *messages.Service
}

func newServices(cfg *config.Config, st *store.Store, hub *events.Hub, idp joow.Provider) *services {
	b := box(cfg, st)
	t := &tickets.Service{Store: st, Pub: hub, Joow: idp}
	k := &kb.Service{Store: st, Box: b, Dir: cfg.DataDir}
	m := &mail.Service{Store: st, Box: b, Tickets: t, KB: k, Mailbox: mail.IMAP{}}
	msgs := &messages.Service{Store: st, Tickets: t, Joow: idp}
	t.OnReply = func(ctx context.Context, tk *domain.Ticket, c *domain.TicketComment) {
		m.Reply(ctx, tk, c)
		msgs.Reply(ctx, tk, c)
	}
	return &services{tickets: t, kb: k, mail: m, messages: msgs}
}

// box returns the encryption box for stored files and secrets. Without
// DATA_ENCRYPTION_KEY a random key is generated once and kept in DATA_DIR.
// If encrypted data already exists but no key is available (e.g. a restore
// without the key file), startup fails instead of creating a new key that
// could never decrypt it.
func box(cfg *config.Config, st *store.Store) *secure.Box {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		slog.Error("Create data directory failed", "dir", cfg.DataDir, "err", err)
		os.Exit(1)
	}
	keyFile := filepath.Join(cfg.DataDir, ".data-key")
	if _, err := os.Stat(keyFile); cfg.DataKey == "" && os.IsNotExist(err) {
		_, kbErr := os.Stat(filepath.Join(cfg.DataDir, "kb"))
		if kbErr == nil || st.HasSealedData(store.System(context.Background())) {
			slog.Error("Encrypted data exists but no key: set DATA_ENCRYPTION_KEY or restore the key file", "path", keyFile)
			os.Exit(1)
		}
	}
	b, err := secure.LoadOrCreate(cfg.DataKey, keyFile)
	if err != nil {
		slog.Error("Load encryption key failed", "err", err)
		os.Exit(1)
	}
	return b
}

// joowProvider selects the Joow integration: the real Joow API or the
// self-contained local provider used for development and demos.
// Misconfiguration is fatal rather than silently falling back to the local
// provider.
func joowProvider(cfg *config.Config, st *store.Store) joow.Provider {
	switch cfg.JoowMode {
	case "http":
		if cfg.JoowAPIURL == "" || cfg.JoowAPIKey == "" {
			slog.Error("JOOW_MODE=http requires JOOW_API_URL and JOOW_API_KEY")
			os.Exit(1)
		}
		slog.Info("Joow integration: HTTP", "url", cfg.JoowAPIURL)
		return joow.NewHTTP(cfg.JoowAPIURL, cfg.JoowAPIKey)
	case "local":
		slog.Warn("Joow integration: local development provider (not for production)")
		return joow.NewLocal(st.DB(), cfg.JWTSecret)
	}
	slog.Error("JOOW_MODE must be local or http", "value", cfg.JoowMode)
	os.Exit(1)
	return nil
}

// devJoowTokens decides whether the unauthenticated /dev/joow-token endpoint
// (mints a Joow sign-in token for any local account) is exposed: only in
// local mode and only for demos or when explicitly enabled.
func devJoowTokens(cfg *config.Config) bool {
	on := cfg.JoowMode == "local" && (cfg.SeedDemo || cfg.JoowDevTokens)
	if on {
		slog.Warn("Dev Joow sign-in tokens enabled: anyone can sign in as a local Joow account")
	}
	return on
}
