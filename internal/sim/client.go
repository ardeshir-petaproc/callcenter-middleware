package sim

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

// client is one connected ARI WebSocket subscriber.
type client struct {
	out chan []byte
}

func (c *client) send(raw []byte) {
	select {
	case c.out <- raw:
	default: // drop when the consumer is too slow
	}
}

func (s *Sim) authorized(r *http.Request) bool {
	if s.opts.Username == "" {
		return true
	}
	u, p, ok := r.BasicAuth()
	return ok && u == s.opts.Username && p == s.opts.Password
}

func (s *Sim) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("app") != s.opts.App {
		http.Error(w, "unknown app", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	c := &client{out: make(chan []byte, 1024)}
	s.mu.Lock()
	s.clients[c] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, c)
		s.mu.Unlock()
	}()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-c.out:
			if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
				return
			}
		}
	}
}
