package store

// conversationsV8 maps chat threads (e.g. Joow conversations) to tickets.
var conversationsV8 = []string{
	`CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id INTEGER NOT NULL, channel TEXT NOT NULL,
		external_id TEXT NOT NULL, ticket_id INTEGER NOT NULL, created_at INTEGER NOT NULL,
		UNIQUE (tenant_id, channel, external_id))`,
	`CREATE INDEX IF NOT EXISTS idx_conversations_ticket ON conversations(ticket_id)`,
}
