package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"
)

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

func knowledge(cfg *config.Config, st *store.Store) *kb.Service {
	return &kb.Service{Store: st, Box: box(cfg), Dir: cfg.DataDir}
}
