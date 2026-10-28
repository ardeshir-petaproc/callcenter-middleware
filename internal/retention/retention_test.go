package retention

import (
	"context"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

type deleter struct{ deleted []string }

func (d *deleter) DeleteRecording(_ context.Context, name string) error {
	d.deleted = append(d.deleted, name)
	return nil
}

func TestApply_PurgesOnlyExpiredDataOfTheTenant(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	acme, other := store.ForTenant(context.Background(), 1), store.ForTenant(context.Background(), 2)
	add := func(ctx context.Context, id string, age time.Duration) {
		c := &domain.Call{ID: id, Direction: "inbound", State: domain.CallEnded, StartedAt: now.Add(-age), Recording: "call-" + id, Tags: []string{}}
		if err := st.SaveCall(ctx, c); err != nil {
			t.Fatal(err)
		}
		_ = st.AddTranscript(ctx, domain.TranscriptEntry{CallID: id, Speaker: "caller", Text: "hi", At: c.StartedAt})
	}
	add(acme, "old", 40*24*time.Hour)
	add(acme, "new", 2*24*time.Hour)
	add(other, "foreign-old", 40*24*time.Hour)
	if err := Save(acme, st, Policy{RecordingDays: 30, TranscriptDays: 30}); err != nil {
		t.Fatal(err)
	}
	d := &deleter{}
	res, err := (&Service{Store: st, Deleter: d, Now: func() time.Time { return now }}).Apply(acme)
	if err != nil {
		t.Fatal(err)
	}
	if res.Recordings != 1 || res.Transcripts != 1 || len(d.deleted) != 1 || d.deleted[0] != "call-old" {
		t.Fatalf("purge: %+v %v", res, d.deleted)
	}
	if c, _ := st.GetCall(acme, "old"); c.Recording != "" {
		t.Fatalf("recording reference kept: %+v", c)
	}
	if c, _ := st.GetCall(acme, "new"); c.Recording == "" {
		t.Fatal("recent recording deleted")
	}
	if c, _ := st.GetCall(other, "foreign-old"); c.Recording == "" {
		t.Fatal("another tenant's recording deleted")
	}
	if tr, _ := st.Transcript(acme, "old"); len(tr) != 0 {
		t.Fatalf("old transcript kept: %+v", tr)
	}
	audit, _ := st.ListAudit(acme, store.AuditFilter{Action: "retention.purge", Limit: 5})
	if len(audit) != 1 {
		t.Fatalf("purge audit: %+v", audit)
	}
}
