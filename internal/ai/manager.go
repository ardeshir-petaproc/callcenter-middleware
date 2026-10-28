package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/store"
)

// ErrNotConfigured is returned when no OpenAI API key is set.
var ErrNotConfigured = errors.New("OPENAI_API_KEY not configured")

// Manager implements core.AIManager.
type Manager struct {
	cfg  *config.Config
	dir  *Directory
	pool *portPool

	mu       sync.Mutex
	sessions map[string]*Session
}

// NewManager creates an AI session manager.
func NewManager(cfg *config.Config, dir *Directory) *Manager {
	return &Manager{
		cfg: cfg, dir: dir, pool: newPortPool(cfg.MediaPortMin, cfg.MediaPortMax),
		sessions: map[string]*Session{},
	}
}

// Start allocates an RTP port, starts the realtime session in the
// background and returns the host:port Asterisk should stream to.
func (m *Manager) Start(spec core.AISpec, ctl core.AIControl) (string, error) {
	if m.cfg.OpenAIAPIKey == "" {
		return "", ErrNotConfigured
	}
	var rtp *RTPEndpoint
	port, err := m.pool.acquire(func(p int) error {
		r, err := ListenRTP(m.cfg.MediaBindHost, p)
		if err == nil {
			rtp = r
		}
		return err
	})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(store.ForTenant(context.Background(), spec.TenantID))
	s := &Session{spec: spec, ctl: ctl, cfg: m.cfg, dir: m.dir, rtp: rtp, cancel: cancel,
		queues: m.dir.QueueNames(ctx), topics: m.dir.TopicNames(ctx)}
	m.mu.Lock()
	m.sessions[spec.CallID] = s
	m.mu.Unlock()

	go func() {
		err := s.run(ctx)
		cancel()
		rtp.Close()
		m.pool.release(port)
		m.mu.Lock()
		if m.sessions[spec.CallID] == s {
			delete(m.sessions, spec.CallID)
		}
		m.mu.Unlock()
		if !s.stopped.Load() {
			if err == nil {
				err = errors.New("realtime session closed")
			}
			slog.Warn("AI session ended unexpectedly", "call_id", spec.CallID, "err", err)
			ctl.AIFailed(spec.CallID, err)
		}
	}()
	return fmt.Sprintf("%s:%d", m.cfg.MediaAdvertiseHost, port), nil
}

// Stop ends the AI session for a call.
func (m *Manager) Stop(callID string) {
	m.mu.Lock()
	s := m.sessions[callID]
	delete(m.sessions, callID)
	m.mu.Unlock()
	if s != nil {
		s.stopped.Store(true)
		s.cancel()
	}
}

// SetRemote sets the Asterisk RTP address for a session.
func (m *Manager) SetRemote(callID, addr string) {
	m.mu.Lock()
	s := m.sessions[callID]
	m.mu.Unlock()
	if s == nil {
		return
	}
	if ua, err := net.ResolveUDPAddr("udp", addr); err == nil {
		s.rtp.SetRemote(ua)
	}
}

// Active returns the number of running sessions.
func (m *Manager) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}
