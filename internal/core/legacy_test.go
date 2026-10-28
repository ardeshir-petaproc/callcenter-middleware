package core_test

import (
	"context"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

func TestLegacyFlow_DTMFBargeInRoutesToDialplanQueue(t *testing.T) {
	h := newHarness(t, nil)
	h.reload()
	h.incoming("ch-100", "09120000000", "100")

	if !h.sw.has("answer ch-100") || !h.sw.has("play ch-100 sound:welcome") {
		t.Fatalf("expected answer and greeting, got %v", h.sw.ops)
	}
	h.dtmf("ch-100", "2")
	if !h.sw.has("stopplay") {
		t.Errorf("expected playback stopped for barge-in, got %v", h.sw.ops)
	}
	if !h.sw.has("continue ch-100 ext-queues 601") {
		t.Fatalf("expected continued to queue 601, got %v", h.sw.ops)
	}
	calls, _, _ := h.st.ListCalls(tctx(), domain.CallFilter{})
	if len(calls) != 1 || calls[0].Outcome != domain.OutcomeDialplan || calls[0].IVRPath != "2" {
		t.Fatalf("unexpected CDR %+v", calls)
	}
}

func TestLegacyFlow_TimeoutRetriesThenFallsBack(t *testing.T) {
	h := newHarness(t, nil)
	h.reload()
	h.incoming("ch-1", "1", "100")
	// Greeting finishes three times (initial + 2 retries) without input.
	for i := 0; i < 3; i++ {
		h.finishPlayback()
		h.advance(6 * time.Second)
	}
	if got := h.sw.count("play ch-1 sound:welcome"); got != 3 {
		t.Fatalf("expected 3 greetings, got %d (%v)", got, h.sw.ops)
	}
	if !h.sw.has("continue ch-1 ext-queues 600") {
		t.Fatalf("expected fallback to default queue, got %v", h.sw.ops)
	}
}

func TestLegacyFlow_InvalidDigitRepeatsGreeting(t *testing.T) {
	h := newHarness(t, nil)
	h.reload()
	h.incoming("ch-1", "1", "100")
	h.dtmf("ch-1", "9")
	if got := h.sw.count("play ch-1 sound:welcome"); got != 2 {
		t.Fatalf("expected greeting replayed, got %d", got)
	}
}

func TestDrain_ReleasesCallsToDefaultQueue(t *testing.T) {
	h := newHarness(t, nil)
	h.reload()
	h.incoming("drain-ch-1", "09121111111", "100")

	ctx, cancel := context.WithTimeout(tctx(), 2*time.Second)
	defer cancel()
	h.core.DrainAndRelease(ctx)

	if !h.sw.has("continue drain-ch-1 ext-queues 600") {
		t.Fatalf("expected drained call released to queue 600, got %v", h.sw.ops)
	}
	if len(h.core.Snapshot(0).Calls) != 0 {
		t.Fatal("expected no live calls after drain")
	}
}

// finishPlayback completes the most recent playback on any channel.
func (h *harness) finishPlayback() {
	h.sw.mu.Lock()
	var id string
	for i := len(h.sw.ops) - 1; i >= 0; i-- {
		var ch, uri string
		if n, _ := sscanf3(h.sw.ops[i], &ch, &uri, &id); n == 3 {
			break
		}
		id = ""
	}
	h.sw.mu.Unlock()
	if id != "" {
		h.event(domain.TelephonyEvent{Type: domain.EventPlaybackFinished, PlaybackID: id})
	}
}
