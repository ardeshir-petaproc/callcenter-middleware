package ivr_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/ivr"
)

type mockSwitch struct {
	mu           sync.Mutex
	answered     []string
	played       []string
	stopped      []string
	continued    []string
	hungup       []string
}

func (m *mockSwitch) AnswerChannel(ctx context.Context, ch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.answered = append(m.answered, ch)
	return nil
}

func (m *mockSwitch) PlayMedia(ctx context.Context, ch, uri, playID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.played = append(m.played, playID)
	return nil
}

func (m *mockSwitch) StopPlayback(ctx context.Context, playID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, playID)
	return nil
}

func (m *mockSwitch) ContinueToQueue(ctx context.Context, ch, ctxt, queue string, prio int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.continued = append(m.continued, queue)
	return nil
}

func (m *mockSwitch) HangupChannel(ctx context.Context, ch, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hungup = append(m.hungup, ch)
	return nil
}

func TestIVRFlow_DTMFRouting(t *testing.T) {
	cfg := &config.Config{
		DefaultQueue:  "600",
		QueueContext:  "ext-queues",
		IVRTimeout:    1 * time.Second,
		MaxRetries:    2,
		GreetingMedia: "sound:welcome",
		QueueMappings: map[string]string{"1": "600", "2": "601"},
	}
	sw := &mockSwitch{}
	engine := ivr.NewEngine(cfg, sw, nil, nil)
	ctx := context.Background()

	// 1. Channel answers
	engine.ProcessEvent(ctx, domain.TelephonyEvent{
		Type:         domain.EventCallStarted,
		ChannelID:    "ch-100",
		CallerNumber: "09120000000",
		Timestamp:    time.Now(),
	})

	if len(sw.answered) != 1 || sw.answered[0] != "ch-100" {
		t.Fatalf("expected ch-100 answered, got %+v", sw.answered)
	}
	if len(sw.played) != 1 {
		t.Fatalf("expected media played, got %+v", sw.played)
	}

	// 2. DTMF Barge-in: Digit 2 -> Queue 601
	engine.ProcessEvent(ctx, domain.TelephonyEvent{
		Type:      domain.EventDTMFReceived,
		ChannelID: "ch-100",
		Digit:     "2",
		Timestamp: time.Now(),
	})

	if len(sw.stopped) != 1 {
		t.Errorf("expected playback stopped for barge-in, got %+v", sw.stopped)
	}
	if len(sw.continued) != 1 || sw.continued[0] != "601" {
		t.Fatalf("expected continued to queue 601, got %+v", sw.continued)
	}
}

func TestIVRDrain_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		DefaultQueue:  "600",
		QueueContext:  "ext-queues",
		IVRTimeout:    1 * time.Second,
		MaxRetries:    2,
		GreetingMedia: "sound:welcome",
		QueueMappings: map[string]string{"1": "600"},
	}
	sw := &mockSwitch{}
	engine := ivr.NewEngine(cfg, sw, nil, nil)
	ctx := context.Background()

	engine.ProcessEvent(ctx, domain.TelephonyEvent{
		Type:         domain.EventCallStarted,
		ChannelID:    "drain-ch-1",
		CallerNumber: "09121111111",
		Timestamp:    time.Now(),
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	engine.DrainAndRelease(shutdownCtx)

	if len(sw.continued) != 1 || sw.continued[0] != "600" {
		t.Fatalf("expected drained call released to queue 600, got %+v", sw.continued)
	}
}
