package ai

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/coder/websocket"
)

type ctlRecorder struct {
	mu          sync.Mutex
	transcripts []string
	ended       chan string
}

func (c *ctlRecorder) AITransfer(callID, queue, reason, summary string)                    {}
func (c *ctlRecorder) AIRoute(callID string, target domain.Action, reason, summary string) {}
func (c *ctlRecorder) AIEndCall(callID, summary string)                                    { c.ended <- summary }
func (c *ctlRecorder) AICallback(callID, number, notes string)                             {}
func (c *ctlRecorder) AIFailed(callID string, err error)                                   {}
func (c *ctlRecorder) AITranscript(callID, speaker, text string) {
	c.mu.Lock()
	c.transcripts = append(c.transcripts, speaker+":"+text)
	c.mu.Unlock()
}

// fakeRealtime emulates the OpenAI Realtime WebSocket API.
func fakeRealtime(t *testing.T, got chan<- map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" || r.URL.Query().Get("model") != "gpt-realtime" {
			t.Errorf("bad auth/model: %v %v", r.Header, r.URL)
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		sentAudio := false
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var ev map[string]any
			_ = json.Unmarshal(msg, &ev)
			got <- ev
			if ev["type"] == "input_audio_buffer.append" && !sentAudio {
				sentAudio = true
				reply := func(v map[string]any) { b, _ := json.Marshal(v); _ = conn.Write(ctx, websocket.MessageText, b) }
				reply(map[string]any{"type": "conversation.item.input_audio_transcription.completed", "transcript": "hello"})
				audio := base64.StdEncoding.EncodeToString(make([]byte, 320))
				reply(map[string]any{"type": "response.output_audio.delta", "item_id": "it1", "delta": audio})
				reply(map[string]any{"type": "response.output_audio_transcript.done", "transcript": "hi there"})
				reply(map[string]any{"type": "response.function_call_arguments.done", "call_id": "fc1",
					"name": "end_call", "arguments": `{"summary":"done"}`})
			}
		}
	}))
}

func TestRealtimeBridge_EndToEnd(t *testing.T) {
	events := make(chan map[string]any, 200)
	srv := fakeRealtime(t, events)
	defer srv.Close()
	st, _ := store.Open(":memory:")
	defer st.Close()

	cfg := &config.Config{OpenAIAPIKey: "sk-test", OpenAIRealtimeURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
		OpenAIRealtimeModel: "gpt-realtime", OpenAIVoice: "marin", OpenAITranscribeModel: "whisper-1",
		MediaBindHost: "127.0.0.1", MediaAdvertiseHost: "127.0.0.1", MediaPortMin: 41000, MediaPortMax: 41010}
	m := NewManager(cfg, &Directory{Store: st})
	ctl := &ctlRecorder{ended: make(chan string, 1)}
	profile := domain.AIProfile{Name: "t", Tools: []string{domain.ToolEndCall}, Enabled: true}
	host, err := m.Start(core.AISpec{CallID: "call-1", CallerNumber: "555", Profile: profile}, ctl)
	if err != nil {
		t.Fatal(err)
	}

	first := <-events
	if first["type"] != "session.update" {
		t.Fatalf("expected session.update first, got %v", first["type"])
	}
	sess := first["session"].(map[string]any)
	format := sess["audio"].(map[string]any)["input"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "audio/pcmu" || len(sess["tools"].([]any)) != 1 {
		t.Fatalf("unexpected session %v", sess)
	}

	// Asterisk side: send RTP to the advertised port and expect RTP back.
	asterisk, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer asterisk.Close()
	dst, _ := net.ResolveUDPAddr("udp", host)
	pkt := make([]byte, 12+160)
	pkt[0] = 0x80
	for i := 0; i < 6; i++ {
		binary.BigEndian.PutUint16(pkt[2:], uint16(i))
		_, _ = asterisk.WriteToUDP(pkt, dst)
	}
	_ = asterisk.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 400)
	n, _, err := asterisk.ReadFromUDP(buf)
	if err != nil || n != 172 || buf[1] != 0 {
		t.Fatalf("expected PCMU RTP frame back, n=%d err=%v", n, err)
	}

	select {
	case s := <-ctl.ended:
		if s != "done" {
			t.Fatalf("unexpected summary %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("end_call not delivered")
	}
	ctl.mu.Lock()
	tr := strings.Join(ctl.transcripts, "|")
	ctl.mu.Unlock()
	if tr != "caller:hello|ai:hi there" {
		t.Fatalf("unexpected transcripts %q", tr)
	}
	m.Stop("call-1")
	if m.Active() != 0 {
		t.Fatal("expected no active sessions")
	}
}
