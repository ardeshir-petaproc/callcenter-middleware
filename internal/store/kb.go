package store

import (
	"context"
	"strings"
	"unicode"

	"callcenter-middleware/internal/domain"
)

// IndexSource replaces the full-text index of a knowledge source.
func (s *Store) IndexSource(ctx context.Context, tenant, sourceID int64, chunks []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM kb_fts WHERE source_id = ? AND tenant_id = ?`, sourceID, tenant); err != nil {
		return err
	}
	for i, c := range chunks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO kb_fts (content, source_id, tenant_id, chunk) VALUES (?,?,?,?)`,
			c, sourceID, tenant, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DropIndex removes a source from the index.
func (s *Store) DropIndex(ctx context.Context, tenant, sourceID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM kb_fts WHERE source_id = ? AND tenant_id = ?`, sourceID, tenant)
	return err
}

// ftsQuery turns free text into a safe FTS5 query (any word, quoted).
func ftsQuery(q string) string {
	words := strings.FieldsFunc(q, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	var parts []string
	for _, w := range words {
		if len([]rune(w)) < 2 {
			continue
		}
		parts = append(parts, `"`+w+`"`)
		if len(parts) == 12 {
			break
		}
	}
	return strings.Join(parts, " OR ")
}

// SearchKB returns the best matching chunks of a tenant's knowledge base.
// Permission and validity filtering happens in the kb package.
func (s *Store) SearchKB(ctx context.Context, tenant int64, query string, limit int) ([]domain.KBHit, error) {
	match := ftsQuery(query)
	if match == "" {
		return []domain.KBHit{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source_id, chunk, snippet(kb_fts, 0, '[', ']', '…', 24), content,
		bm25(kb_fts) FROM kb_fts WHERE kb_fts MATCH ? AND tenant_id = ? ORDER BY bm25(kb_fts) LIMIT ?`,
		match, tenant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.KBHit{}
	for rows.Next() {
		var h domain.KBHit
		if err := rows.Scan(&h.SourceID, &h.Chunk, &h.Snippet, &h.Text, &h.Score); err != nil {
			return nil, err
		}
		h.Score = -h.Score // bm25: lower is better
		out = append(out, h)
	}
	return out, rows.Err()
}
