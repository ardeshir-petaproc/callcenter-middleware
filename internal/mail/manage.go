package mail

import (
	"context"
	"log/slog"
	"time"

	"callcenter-middleware/internal/store"
)

// SetAIAccess allows or withdraws the AI's use of everything imported from
// a connector, now and for future imports.
func (s *Service) SetAIAccess(ctx context.Context, id int64, enabled bool) (int, error) {
	c, err := s.Store.Mailboxes.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	c.AIEnabled = enabled
	if err := s.Store.Mailboxes.Update(ctx, c); err != nil {
		return 0, err
	}
	all, err := s.Store.KB.List(ctx)
	if err != nil {
		return 0, err
	}
	changed := 0
	for i := range all {
		if src := &all[i]; src.ConnectorID == id && src.AIEnabled != enabled {
			src.AIEnabled = enabled
			if err := s.Store.KB.Update(ctx, src); err != nil {
				return changed, err
			}
			changed++
		}
	}
	return changed, nil
}

// Disconnect deletes a connector and its stored credentials. With purge,
// every message and knowledge source imported from it is deleted too;
// tickets stay, as they are customer records.
func (s *Service) Disconnect(ctx context.Context, id int64, purge bool) error {
	if _, err := s.Store.Mailboxes.Get(ctx, id); err != nil {
		return err
	}
	if err := s.Store.Mailboxes.Delete(ctx, id); err != nil {
		return err
	}
	if !purge {
		return nil
	}
	all, err := s.Store.KB.List(ctx)
	if err != nil {
		return err
	}
	for _, src := range all {
		if src.ConnectorID == id {
			if err := s.KB.Delete(ctx, src.ID); err != nil {
				return err
			}
		}
	}
	return s.Store.PurgeEmails(ctx, id)
}

// Run polls enabled connectors that are due until ctx ends.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.pollDue(ctx)
		}
	}
}

func (s *Service) pollDue(ctx context.Context) {
	all, err := s.Store.Mailboxes.List(store.System(ctx))
	if err != nil {
		slog.Warn("List email connectors failed", "err", err)
		return
	}
	now := s.now()
	for _, c := range all {
		if !c.Enabled {
			continue
		}
		interval := time.Duration(max(c.PollMinutes, 1)) * time.Minute
		if last, err := time.Parse(time.RFC3339, c.LastSync); err == nil && now.Sub(last) < interval {
			continue
		}
		if n, err := s.Sync(store.ForTenant(ctx, c.TenantID), c.ID); err != nil {
			slog.Warn("Email sync failed", "connector", c.ID, "tenant", c.TenantID, "err", err)
		} else if n > 0 {
			slog.Info("Email imported", "connector", c.ID, "tenant", c.TenantID, "count", n)
		}
	}
}
