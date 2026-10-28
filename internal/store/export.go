package store

import (
	"context"
	"encoding/json"
)

// EachEntity streams every configuration document of the tenant in ctx,
// grouped by kind. Secrets inside documents are the caller's concern.
func (s *Store) EachEntity(ctx context.Context, fn func(kind string, id int64, data json.RawMessage) error) error {
	sc, err := scopeOf(ctx)
	if err != nil || sc.system {
		return ErrNoTenant
	}
	rows, err := s.db.QueryContext(ctx, `SELECT kind, id, data FROM entities WHERE tenant_id = ? ORDER BY kind, id`, sc.tenant)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, data string
		var id int64
		if err := rows.Scan(&kind, &id, &data); err != nil {
			return err
		}
		if err := fn(kind, id, json.RawMessage(data)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// HasSealedData reports whether any data encrypted with the master key
// exists (sealed secrets or email bodies), across all tenants.
func (s *Store) HasSealedData(ctx context.Context) bool {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM entities WHERE data LIKE '%"enc:%'
		UNION ALL SELECT 1 FROM emails WHERE body LIKE 'enc:%' LIMIT 1`).Scan(&one)
	return err == nil
}
