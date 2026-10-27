// Package store persists call center configuration and call records in SQLite.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"callcenter-middleware/internal/domain"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

var memSeq atomic.Int64

// Store wraps the SQLite database and exposes typed repositories.
type Store struct {
	db *sql.DB

	Queues       *Collection[domain.Queue]
	Flows        *Collection[domain.IVRFlow]
	Schedules    *Collection[domain.Schedule]
	AIProfiles   *Collection[domain.AIProfile]
	Dispositions *Collection[domain.Disposition]
	PauseReasons *Collection[domain.PauseReason]
	Webhooks     *Collection[domain.Webhook]
	Contacts     *Collection[domain.Contact]
}

// Open opens (and migrates) the database at path. Use ":memory:" for tests.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:mem%d?mode=memory&cache=shared", memSeq.Add(1))
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
		dsn = "file:" + path
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	s.Queues = newCollection(db, "queue", func(x *domain.Queue) *int64 { return &x.ID })
	s.Flows = newCollection(db, "flow", func(x *domain.IVRFlow) *int64 { return &x.ID })
	s.Schedules = newCollection(db, "schedule", func(x *domain.Schedule) *int64 { return &x.ID })
	s.AIProfiles = newCollection(db, "ai_profile", func(x *domain.AIProfile) *int64 { return &x.ID })
	s.Dispositions = newCollection(db, "disposition", func(x *domain.Disposition) *int64 { return &x.ID })
	s.PauseReasons = newCollection(db, "pause_reason", func(x *domain.PauseReason) *int64 { return &x.ID })
	s.Webhooks = newCollection(db, "webhook", func(x *domain.Webhook) *int64 { return &x.ID })
	s.Contacts = newCollection(db, "contact", func(x *domain.Contact) *int64 { return &x.ID })
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// Ping checks database health.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) migrate(ctx context.Context) error {
	for i, stmt := range migrations {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	return nil
}

func ms(t time.Time) int64 { return t.UnixMilli() }

func msPtr(t *time.Time) sql.NullInt64 {
	if t == nil || t.IsZero() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.UnixMilli(), Valid: true}
}

func fromMS(v int64) time.Time { return time.UnixMilli(v).UTC() }

func fromNullMS(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.UnixMilli(v.Int64).UTC()
	return &t
}

// ListWebhooks satisfies notify.WebhookSource.
func (s *Store) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	return s.Webhooks.List(ctx)
}
