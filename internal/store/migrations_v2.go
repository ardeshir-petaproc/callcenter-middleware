package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// versioned migrations run once each, in order, after the idempotent base
// schema. Version 1 is the original single-tenant schema.
var versioned = []struct {
	version int
	stmts   []string
}{
	{2, multiTenantV2},
	{3, joowV3},
}

func (s *Store) migrateVersions(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var current int
	err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		current = 1
		if _, err := s.db.ExecContext(ctx, `INSERT INTO schema_version (version) VALUES (1)`); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for _, m := range versioned {
		if m.version <= current {
			continue
		}
		// Table rebuilds must not cascade deletes through foreign keys.
		if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
			return err
		}
		for i, stmt := range m.stmts {
			var args []any
			if strings.Contains(stmt, "?") {
				args = append(args, time.Now().UnixMilli())
			}
			if _, err := s.db.ExecContext(ctx, stmt, args...); err != nil {
				return fmt.Errorf("migration v%d step %d: %w", m.version, i, err)
			}
		}
		if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE schema_version SET version = ?`, m.version); err != nil {
			return err
		}
		current = m.version
	}
	return nil
}

// Every statement receives the current time in ms as its only (optional) arg.
var multiTenantV2 = []string{
	`CREATE TABLE IF NOT EXISTS tenants (
		id INTEGER PRIMARY KEY AUTOINCREMENT, slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active', plan TEXT NOT NULL DEFAULT '', settings TEXT NOT NULL DEFAULT '{}',
		created_at INTEGER NOT NULL)`,
	`INSERT OR IGNORE INTO tenants (id, slug, name, status, created_at) VALUES (1, 'default', 'Default organization', 'active', ?)`,
	`ALTER TABLE entities ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`CREATE INDEX IF NOT EXISTS idx_entities_tenant ON entities(kind, tenant_id)`,
	`CREATE TABLE users_v2 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL,
		extension TEXT NOT NULL DEFAULT '',
		endpoint TEXT NOT NULL DEFAULT '',
		skills TEXT NOT NULL DEFAULT '[]',
		enabled INTEGER NOT NULL DEFAULT 1,
		password_hash TEXT NOT NULL,
		joow_account_id TEXT NOT NULL DEFAULT '',
		custom_role_id INTEGER NOT NULL DEFAULT 0,
		platform_admin INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		profile TEXT NOT NULL DEFAULT '{}',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		UNIQUE (tenant_id, username))`,
	`INSERT INTO users_v2 (id, tenant_id, username, display_name, email, role, extension, endpoint, skills,
		enabled, password_hash, platform_admin, created_at, updated_at)
		SELECT id, 1, username, display_name, email, role, extension, endpoint, skills, enabled, password_hash,
		CASE WHEN id = (SELECT MIN(id) FROM users WHERE role = 'admin') THEN 1 ELSE 0 END, created_at, updated_at
		FROM users`,
	`DROP TABLE users`,
	`ALTER TABLE users_v2 RENAME TO users`,
	`CREATE INDEX IF NOT EXISTS idx_users_joow ON users(joow_account_id)`,
	`ALTER TABLE queue_members ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE calls ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`CREATE INDEX IF NOT EXISTS idx_calls_tenant ON calls(tenant_id, started_at)`,
	`ALTER TABLE callbacks ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE voicemails ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE audit_log ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE agent_events ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE leads ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 1`,
	`CREATE TABLE IF NOT EXISTS tenant_settings (
		tenant_id INTEGER NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL, PRIMARY KEY (tenant_id, key))`,
	`INSERT OR IGNORE INTO tenant_settings (tenant_id, key, value) SELECT 1, key, value FROM settings`,
	`CREATE TABLE IF NOT EXISTS revisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id INTEGER NOT NULL, entity TEXT NOT NULL,
		entity_id INTEGER NOT NULL, action TEXT NOT NULL, data TEXT NOT NULL, changed_by INTEGER NOT NULL DEFAULT 0,
		changed_by_name TEXT NOT NULL DEFAULT '', at INTEGER NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS idx_revisions_entity ON revisions(tenant_id, entity, entity_id)`,
}

var joowV3 = []string{
	`CREATE TABLE IF NOT EXISTS joow_accounts (
		id TEXT PRIMARY KEY, name TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
		phone TEXT NOT NULL DEFAULT '', avatar_url TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL)`,
}
