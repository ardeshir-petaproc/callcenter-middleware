package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/events"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/mail"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

// services are the business services shared by the API and background jobs.
type services struct {
	tickets *tickets.Service
	kb      *kb.Service
	mail    *mail.Service
}

func newServices(cfg *config.Config, st *store.Store, hub *events.Hub, idp joow.Provider) *services {
	b := box(cfg)
	t := &tickets.Service{Store: st, Pub: hub, Joow: idp}
	k := &kb.Service{Store: st, Box: b, Dir: cfg.DataDir}
	m := &mail.Service{Store: st, Box: b, Tickets: t, KB: k, Mailbox: mail.IMAP{}}
	t.OnReply = m.Reply
	return &services{tickets: t, kb: k, mail: m}
}

// box returns the encryption box for stored files and secrets. Without
// DATA_ENCRYPTION_KEY a random key is generated once and kept in DATA_DIR.
func box(cfg *config.Config) *secure.Box {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		slog.Error("Create data directory failed", "dir", cfg.DataDir, "err", err)
		os.Exit(1)
	}
	b, err := secure.LoadOrCreate(cfg.DataKey, filepath.Join(cfg.DataDir, ".data-key"))
	if err != nil {
		slog.Error("Load encryption key failed", "err", err)
		os.Exit(1)
	}
	return b
}
