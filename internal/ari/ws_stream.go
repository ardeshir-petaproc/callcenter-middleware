package ari

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"callcenter-middleware/internal/domain"
	"github.com/coder/websocket"
)

// Stream maintains a persistent ARI WebSocket event stream.
type Stream struct {
	url       string
	app       string
	username  string
	password  string
	metrics   domain.MetricsRecorder
	mu        sync.RWMutex
	connected bool
}

// NewStream creates a new ARI WebSocket stream manager.
func NewStream(url, app, user, pass string, metrics domain.MetricsRecorder) *Stream {
	return &Stream{
		url:      url,
		app:      app,
		username: user,
		password: pass,
		metrics:  metrics,
	}
}

// IsConnected returns the current connection status.
func (s *Stream) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

func (s *Stream) setConnected(val bool) {
	s.mu.Lock()
	s.connected = val
	s.mu.Unlock()
	if s.metrics != nil {
		s.metrics.SetWSConnected(val)
	}
}

// Subscribe runs the WebSocket loop with exponential reconnect backoff.
func (s *Stream) Subscribe(ctx context.Context, out chan<- domain.TelephonyEvent) error {
	backoff := 100 * time.Millisecond
	maxBackoff := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		connURL := fmt.Sprintf("%s?app=%s&subscribeAll=false", s.url, s.app)
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(s.username+":"+s.password))
		opts := &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{auth}}}

		conn, _, err := websocket.Dial(ctx, connURL, opts)
		if err != nil {
			s.setConnected(false)
			slog.Warn("ARI WebSocket dial failed, retrying", "err", err, "backoff", backoff)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		s.setConnected(true)
		backoff = 100 * time.Millisecond
		slog.Info("Connected to Asterisk ARI WebSocket", "url", connURL)

		s.readLoop(ctx, conn, out)
		s.setConnected(false)
		conn.Close(websocket.StatusNormalClosure, "disconnecting")
	}
}

func (s *Stream) readLoop(ctx context.Context, conn *websocket.Conn, out chan<- domain.TelephonyEvent) {
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			slog.Warn("ARI WebSocket read error", "err", err)
			return
		}
		var raw RawEvent
		if err := json.Unmarshal(msg, &raw); err != nil {
			continue
		}
		if ev, ok := normalizeEvent(&raw); ok {
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}
}

func normalizeEvent(raw *RawEvent) (domain.TelephonyEvent, bool) {
	now := time.Now()
	switch raw.Type {
	case "StasisStart":
		if raw.Channel != nil {
			return domain.TelephonyEvent{
				Type:         domain.EventCallStarted,
				ChannelID:    raw.Channel.ID,
				CallerNumber: raw.Channel.Caller.Number,
				CallerName:   raw.Channel.Caller.Name,
				Timestamp:    now,
			}, true
		}
	case "PlaybackFinished":
		if raw.Playback != nil {
			return domain.TelephonyEvent{
				Type:       domain.EventPlaybackFinished,
				PlaybackID: raw.Playback.ID,
				Timestamp:  now,
			}, true
		}
	case "ChannelDtmfReceived":
		if raw.Channel != nil {
			return domain.TelephonyEvent{
				Type:      domain.EventDTMFReceived,
				ChannelID: raw.Channel.ID,
				Digit:     raw.Digit,
				Timestamp: now,
			}, true
		}
	case "StasisEnd", "ChannelHangupRequest":
		if raw.Channel != nil {
			return domain.TelephonyEvent{
				Type:      domain.EventCallEnded,
				ChannelID: raw.Channel.ID,
				Timestamp: now,
			}, true
		}
	}
	return domain.TelephonyEvent{}, false
}
