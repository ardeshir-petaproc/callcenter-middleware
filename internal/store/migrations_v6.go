package store

// emailV6 stores messages imported by email connectors. Bodies are sealed
// by the caller before they reach the database.
var emailV6 = []string{
	`CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id INTEGER NOT NULL, connector_id INTEGER NOT NULL,
		uid INTEGER NOT NULL, message_id TEXT NOT NULL DEFAULT '', in_reply_to TEXT NOT NULL DEFAULT '[]',
		from_addr TEXT NOT NULL DEFAULT '', from_name TEXT NOT NULL DEFAULT '', subject TEXT NOT NULL DEFAULT '',
		date INTEGER NOT NULL, body TEXT NOT NULL DEFAULT '', ticket_id INTEGER NOT NULL DEFAULT 0,
		kb_source_id INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, UNIQUE (connector_id, uid))`,
	`CREATE INDEX IF NOT EXISTS idx_emails_msgid ON emails(tenant_id, message_id)`,
}
