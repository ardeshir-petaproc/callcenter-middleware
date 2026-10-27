package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"callcenter-middleware/internal/domain"
	"github.com/coder/websocket"
)

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	dbOK := s.d.Store == nil || s.d.Store.Ping(r.Context()) == nil
	if s.stream != nil && s.stream.IsConnected() && dbOK {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "ari_ws": true, "db": dbOK})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unready",
		"ari_ws": s.stream != nil && s.stream.IsConnected(), "db": dbOK})
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, s.d.Core.Snapshot())
	return nil
}

func (s *Server) systemInfo(w http.ResponseWriter, r *http.Request) error {
	cfg := s.d.Config
	writeJSON(w, http.StatusOK, map[string]any{
		"version":           Version,
		"ari_connected":     s.stream != nil && s.stream.IsConnected(),
		"ai_enabled":        cfg.AIEnabled(),
		"db_ok":             s.d.Store.Ping(r.Context()) == nil,
		"uptime_sec":        int(time.Since(s.d.Started).Seconds()),
		"started_at":        s.d.Started.UTC(),
		"ws_clients":        s.wsClients.Load(),
		"default_queue":     cfg.DefaultQueue,
		"queue_context":     cfg.QueueContext,
		"outbound_template": cfg.OutboundDialTemplate,
	})
	return nil
}

// handleWS streams realtime events: a snapshot on connect, every change as
// it happens, and a fresh snapshot every 10 seconds for self-healing UIs.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")
	s.wsClients.Add(1)
	defer s.wsClients.Add(-1)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	events, unsubscribe := s.d.Hub.Subscribe(256)
	defer unsubscribe()

	// Reader: detect client close and answer pings.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()
	send := func(ev domain.Event) bool {
		raw, err := json.Marshal(ev)
		if err != nil {
			return true
		}
		wctx, c := context.WithTimeout(ctx, 10*time.Second)
		defer c()
		return conn.Write(wctx, websocket.MessageText, raw) == nil
	}
	snapshot := func() bool {
		return send(domain.Event{Type: "snapshot", Data: s.d.Core.Snapshot(), At: time.Now().UTC()})
	}
	if !snapshot() {
		return
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok || !send(ev) {
				return
			}
		case <-ticker.C:
			if !snapshot() {
				slog.Debug("WebSocket client gone")
				return
			}
		}
	}
}
