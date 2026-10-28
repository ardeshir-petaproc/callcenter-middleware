// Package retention deletes recordings, transcripts and email bodies once
// a tenant's retention period has passed. Every purge is audited.
package retention

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// Policy holds retention periods in days; 0 keeps data forever.
type Policy struct {
	RecordingDays  int `json:"recording_days"`
	TranscriptDays int `json:"transcript_days"`
	EmailDays      int `json:"email_days"`
}

var keys = map[string]func(*Policy) *int{
	"retention.recording_days":  func(p *Policy) *int { return &p.RecordingDays },
	"retention.transcript_days": func(p *Policy) *int { return &p.TranscriptDays },
	"retention.email_days":      func(p *Policy) *int { return &p.EmailDays },
}

// Load reads the tenant's policy from its settings.
func Load(ctx context.Context, st *store.Store) Policy {
	var p Policy
	for k, f := range keys {
		*f(&p), _ = strconv.Atoi(st.GetSetting(ctx, k, "0"))
	}
	return p
}

// Save stores the tenant's policy.
func Save(ctx context.Context, st *store.Store, p Policy) error {
	for k, f := range keys {
		if v := *f(&p); v < 0 {
			return fmt.Errorf("%s must not be negative", k)
		}
		if err := st.SetSetting(ctx, k, strconv.Itoa(*f(&p))); err != nil {
			return err
		}
	}
	return nil
}

// Deleter removes stored recording files.
type Deleter interface {
	DeleteRecording(ctx context.Context, name string) error
}

// Result counts what one run purged.
type Result struct {
	Recordings  int   `json:"recordings"`
	Transcripts int64 `json:"transcripts"`
	Emails      int64 `json:"emails"`
	Failed      int   `json:"failed"`
}

// Service applies retention policies.
type Service struct {
	Store   *store.Store
	Deleter Deleter
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func cutoff(now time.Time, days int) time.Time { return now.Add(-time.Duration(days) * 24 * time.Hour) }

// Apply purges expired data for the tenant in ctx.
func (s *Service) Apply(ctx context.Context) (Result, error) {
	p, now, res := Load(ctx, s.Store), s.now(), Result{}
	if p.RecordingDays > 0 && s.Deleter != nil {
		recs, err := s.Store.ExpiredRecordings(ctx, cutoff(now, p.RecordingDays), 500)
		if err != nil {
			return res, err
		}
		for _, r := range recs {
			if err := s.Deleter.DeleteRecording(ctx, r.Name); err != nil && !strings.Contains(err.Error(), "404") {
				res.Failed++
				continue
			}
			if err := s.Store.ClearRecording(ctx, r); err != nil {
				return res, err
			}
			res.Recordings++
		}
	}
	var err error
	if p.TranscriptDays > 0 {
		if res.Transcripts, err = s.Store.PurgeTranscripts(ctx, cutoff(now, p.TranscriptDays)); err != nil {
			return res, err
		}
	}
	if p.EmailDays > 0 {
		if res.Emails, err = s.Store.PurgeEmailBodies(ctx, cutoff(now, p.EmailDays)); err != nil {
			return res, err
		}
	}
	if res.Recordings+int(res.Transcripts+res.Emails) > 0 {
		_ = s.Store.Audit(ctx, domain.AuditEntry{Username: "system", Action: "retention.purge", Entity: "tenant",
			Detail: fmt.Sprintf("recordings=%d transcripts=%d emails=%d", res.Recordings, res.Transcripts, res.Emails)})
	}
	return res, nil
}

// Run applies every tenant's policy periodically until ctx ends.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		tenants, err := s.Store.ListTenants(store.System(ctx))
		if err == nil {
			for _, tn := range tenants {
				if res, err := s.Apply(store.ForTenant(ctx, tn.ID)); err != nil {
					slog.Warn("Retention run failed", "tenant", tn.ID, "err", err)
				} else if res.Recordings+int(res.Transcripts+res.Emails) > 0 {
					slog.Info("Retention purge", "tenant", tn.ID, "recordings", res.Recordings, "transcripts", res.Transcripts)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
