package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"

	"github.com/coder/websocket"
)

// rtEvent is the subset of Realtime server events the bridge consumes.
type rtEvent struct {
	Type       string `json:"type"`
	Delta      string `json:"delta"`
	ItemID     string `json:"item_id"`
	Transcript string `json:"transcript"`
	CallID     string `json:"call_id"`
	Name       string `json:"name"`
	Arguments  string `json:"arguments"`
	Error      *struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Session is one phone call bridged to an OpenAI realtime conversation.
type Session struct {
	spec    core.AISpec
	ctl     core.AIControl
	cfg     *config.Config
	dir     *Directory
	rtp     *RTPEndpoint
	queues  []string
	conn    *websocket.Conn
	writeMu sync.Mutex
	cancel  context.CancelFunc
	stopped atomic.Bool
	ending  atomic.Bool

	mu        sync.Mutex
	inBuf     []byte
	itemID    string
	itemStart int64
}

// run connects to OpenAI and pumps audio until the context ends.
func (s *Session) run(ctx context.Context) error {
	u, err := url.Parse(s.cfg.OpenAIRealtimeURL)
	if err != nil {
		return err
	}
	q := u.Query()
	model := s.spec.Profile.Model
	if model == "" {
		model = s.cfg.OpenAIRealtimeModel
	}
	q.Set("model", model)
	u.RawQuery = q.Encode()
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, _, err := websocket.Dial(dialCtx, u.String(), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": []string{"Bearer " + s.cfg.OpenAIAPIKey}},
	})
	cancel()
	if err != nil {
		return fmt.Errorf("realtime dial: %w", err)
	}
	conn.SetReadLimit(4 << 20)
	s.conn = conn
	defer conn.Close(websocket.StatusNormalClosure, "call ended")

	sc := SessionContext{CallerNumber: s.spec.CallerNumber, CallerName: s.spec.CallerName, Queues: s.queues, Phone: true}
	if err := s.send(ctx, map[string]any{"type": "session.update",
		"session": BuildSession(s.cfg, s.spec.Profile, sc)}); err != nil {
		return err
	}
	greet := "Greet the caller briefly and ask how you can help."
	if g := s.spec.Profile.Greeting; g != "" {
		greet = "Greet the caller by saying: " + g
	}
	if err := s.send(ctx, map[string]any{"type": "response.create",
		"response": map[string]any{"instructions": greet}}); err != nil {
		return err
	}
	go s.rtp.WriteLoop(ctx)
	go func() {
		if err := s.rtp.ReadLoop(ctx, func(b []byte) { s.onCallerAudio(ctx, b) }); err != nil {
			slog.Warn("RTP read loop ended", "call_id", s.spec.CallID, "err", err)
		}
	}()
	return s.readLoop(ctx)
}

func (s *Session) send(ctx context.Context, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.conn.Write(wctx, websocket.MessageText, raw)
}

// onCallerAudio batches 100 ms of caller audio per append event.
func (s *Session) onCallerAudio(ctx context.Context, b []byte) {
	s.mu.Lock()
	s.inBuf = append(s.inBuf, b...)
	if len(s.inBuf) < 800 {
		s.mu.Unlock()
		return
	}
	chunk := base64.StdEncoding.EncodeToString(s.inBuf)
	s.inBuf = s.inBuf[:0]
	s.mu.Unlock()
	_ = s.send(ctx, map[string]any{"type": "input_audio_buffer.append", "audio": chunk})
}

func (s *Session) readLoop(ctx context.Context) error {
	for {
		_, msg, err := s.conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("realtime read: %w", err)
		}
		var ev rtEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}
		if err := s.handle(ctx, ev); err != nil {
			return err
		}
	}
}
