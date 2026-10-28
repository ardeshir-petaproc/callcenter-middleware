package store

// routingV7 keeps the routing decision log (why each request went where).
var routingV7 = []string{
	`CREATE TABLE IF NOT EXISTS routing_decisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id INTEGER NOT NULL, call_id TEXT NOT NULL DEFAULT '',
		at INTEGER NOT NULL, rule_id INTEGER NOT NULL DEFAULT 0, data TEXT NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS idx_routing_decisions ON routing_decisions(tenant_id, at)`,
	`CREATE INDEX IF NOT EXISTS idx_routing_decisions_call ON routing_decisions(call_id)`,
}
