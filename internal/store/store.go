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
	Campaigns    *Collection[domain.Campaign]
	Departments  *Collection[domain.Department]
	JobProfiles  *Collection[domain.JobProfile]
	Numbers      *Collection[domain.PhoneNumber]
	Roles        *Collection[domain.CustomRole]
	Invitations  *Collection[domain.Invitation]
	SLAPolicies  *Collection[domain.SLAPolicy]
	KB           *Collection[domain.KnowledgeSource]
	Mailboxes    *Collection[domain.EmailConnector]
	Topics       *Collection[domain.Topic]
	Rules        *Collection[domain.RoutingRule]
	Scorecards   *Collection[domain.QAScorecard]
	Evaluations  *Collection[domain.QAEvaluation]
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
	s.Queues = newCollection(db, "queue", func(x *domain.Queue) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Flows = newCollection(db, "flow", func(x *domain.IVRFlow) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Schedules = newCollection(db, "schedule", func(x *domain.Schedule) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.AIProfiles = newCollection(db, "ai_profile", func(x *domain.AIProfile) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Dispositions = newCollection(db, "disposition", func(x *domain.Disposition) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.PauseReasons = newCollection(db, "pause_reason", func(x *domain.PauseReason) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Webhooks = newCollection(db, "webhook", func(x *domain.Webhook) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Contacts = newCollection(db, "contact", func(x *domain.Contact) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Campaigns = newCollection(db, "campaign", func(x *domain.Campaign) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Departments = newCollection(db, "department", func(x *domain.Department) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.JobProfiles = newCollection(db, "job_profile", func(x *domain.JobProfile) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Numbers = newCollection(db, "phone_number", func(x *domain.PhoneNumber) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Roles = newCollection(db, "custom_role", func(x *domain.CustomRole) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.KB = newCollection(db, "kb_source", func(x *domain.KnowledgeSource) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.SLAPolicies = newCollection(db, "sla_policy", func(x *domain.SLAPolicy) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Invitations = newCollection(db, "invitation", func(x *domain.Invitation) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Mailboxes = newCollection(db, "email_connector", func(x *domain.EmailConnector) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Topics = newCollection(db, "topic", func(x *domain.Topic) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Rules = newCollection(db, "routing_rule", func(x *domain.RoutingRule) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Scorecards = newCollection(db, "qa_scorecard", func(x *domain.QAScorecard) (*int64, *int64) { return &x.ID, &x.TenantID })
	s.Evaluations = newCollection(db, "qa_evaluation", func(x *domain.QAEvaluation) (*int64, *int64) { return &x.ID, &x.TenantID })
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
	// Additive column migrations; "duplicate column" means already applied.
	for _, stmt := range columnMigrations {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("column migration %q: %w", stmt, err)
		}
	}
	return s.migrateVersions(ctx)
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

// ListWebhooks satisfies notify.WebhookSource (all tenants; the dispatcher
// matches each event to its own tenant's hooks).
func (s *Store) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	return s.Webhooks.List(System(ctx))
}

// DB exposes the handle for adapters that keep their own tables.
func (s *Store) DB() *sql.DB { return s.db }
