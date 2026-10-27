package core_test

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"callcenter-middleware/internal/domain"
)

// fakeSwitch records every ARI command issued by the core.
type fakeSwitch struct {
	mu         sync.Mutex
	ops        []string
	originated []domain.OriginateRequest
	failOrig   bool
}

func (f *fakeSwitch) rec(format string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, fmt.Sprintf(format, args...))
	return nil
}

func (f *fakeSwitch) has(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, op := range f.ops {
		if strings.HasPrefix(op, prefix) {
			return true
		}
	}
	return false
}

func (f *fakeSwitch) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, op := range f.ops {
		if strings.HasPrefix(op, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeSwitch) lastOrig() domain.OriginateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.originated) == 0 {
		return domain.OriginateRequest{}
	}
	return f.originated[len(f.originated)-1]
}

func (f *fakeSwitch) AnswerChannel(_ context.Context, ch string) error { return f.rec("answer %s", ch) }
func (f *fakeSwitch) PlayMedia(_ context.Context, ch, uri, id string) error {
	return f.rec("play %s %s %s", ch, uri, id)
}
func (f *fakeSwitch) StopPlayback(_ context.Context, id string) error {
	return f.rec("stopplay %s", id)
}
func (f *fakeSwitch) ContinueToQueue(_ context.Context, ch, cx, ext string, p int) error {
	return f.rec("continue %s %s %s", ch, cx, ext)
}
func (f *fakeSwitch) HangupChannel(_ context.Context, ch, r string) error {
	return f.rec("hangup %s", ch)
}
func (f *fakeSwitch) Originate(_ context.Context, r domain.OriginateRequest) error {
	if f.failOrig {
		return fmt.Errorf("originate failed")
	}
	f.mu.Lock()
	f.originated = append(f.originated, r)
	f.mu.Unlock()
	return f.rec("originate %s %s", r.Endpoint, r.ChannelID)
}
func (f *fakeSwitch) CreateBridge(_ context.Context, id, t string) error {
	return f.rec("bridge %s", id)
}
func (f *fakeSwitch) DestroyBridge(_ context.Context, id string) error {
	return f.rec("unbridge %s", id)
}
func (f *fakeSwitch) AddToBridge(_ context.Context, id string, chans ...string) error {
	return f.rec("addbridge %s %s", id, strings.Join(chans, ","))
}
func (f *fakeSwitch) RemoveFromBridge(_ context.Context, id, ch string) error {
	return f.rec("rmbridge %s %s", id, ch)
}
func (f *fakeSwitch) StartMOH(_ context.Context, ch, class string) error { return f.rec("moh %s", ch) }
func (f *fakeSwitch) StopMOH(_ context.Context, ch string) error         { return f.rec("stopmoh %s", ch) }
func (f *fakeSwitch) StartRinging(_ context.Context, ch string) error    { return f.rec("ring %s", ch) }
func (f *fakeSwitch) StopRinging(_ context.Context, ch string) error     { return f.rec("stopring %s", ch) }
func (f *fakeSwitch) RecordBridge(_ context.Context, id, name string) error {
	return f.rec("recbridge %s %s", id, name)
}
func (f *fakeSwitch) RecordChannel(_ context.Context, ch, name string, max int) error {
	return f.rec("recchan %s %s", ch, name)
}
func (f *fakeSwitch) StopRecording(_ context.Context, name string) error {
	return f.rec("stoprec %s", name)
}
func (f *fakeSwitch) ExternalMedia(_ context.Context, r domain.ExternalMediaRequest) (string, error) {
	return "10.0.0.9:10000", f.rec("extmedia %s %s", r.ChannelID, r.ExternalHost)
}
func (f *fakeSwitch) Snoop(_ context.Context, r domain.SnoopRequest) error {
	return f.rec("snoop %s %s %s", r.ChannelID, r.SnoopID, r.Whisper)
}
func (f *fakeSwitch) ListChannels(_ context.Context) ([]string, error) { return nil, nil }
func (f *fakeSwitch) SetMute(_ context.Context, ch string, m bool) error {
	return f.rec("mute %s %v", ch, m)
}
