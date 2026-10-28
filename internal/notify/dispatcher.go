package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"callcenter-middleware/internal/domain"
)

// WebhookSource lists the configured webhook subscriptions.
type WebhookSource interface {
	ListWebhooks(ctx context.Context) ([]domain.Webhook, error)
}

// Dispatcher delivers realtime events to configured webhooks with an
// HMAC-SHA256 signature in X-Callcenter-Signature and bounded retries.
type Dispatcher struct {
	src    WebhookSource
	client *http.Client
	mu     sync.RWMutex
	hooks  []domain.Webhook
	// Open decrypts sealed webhook secrets ("enc:..."), see secure.Box.
	Open func(tenant int64, sealed string) (string, error)
}

// NewDispatcher creates a webhook dispatcher.
func NewDispatcher(src WebhookSource) *Dispatcher {
	return &Dispatcher{src: src, client: &http.Client{Timeout: 5 * time.Second}}
}

// Reload refreshes the cached webhook list.
func (d *Dispatcher) Reload(ctx context.Context) {
	hooks, err := d.src.ListWebhooks(ctx)
	if err != nil {
		slog.Warn("Webhook reload failed", "err", err)
		return
	}
	d.mu.Lock()
	d.hooks = hooks
	d.mu.Unlock()
}

// Run consumes events until the channel closes.
func (d *Dispatcher) Run(ctx context.Context, events <-chan domain.Event) {
	d.Reload(ctx)
	for ev := range events {
		if ev.Type == domain.EvConfigChanged {
			d.Reload(ctx)
		}
		d.mu.RLock()
		hooks := d.hooks
		d.mu.RUnlock()
		for _, h := range hooks {
			// Webhooks only ever receive their own tenant's events.
			if h.Enabled && h.TenantID == ev.TenantID && matches(h.Events, ev.Type) {
				go d.deliver(ctx, h, ev)
			}
		}
	}
}

func matches(patterns []string, evType string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if p == "*" || p == evType {
			return true
		}
	}
	return false
}

func (d *Dispatcher) deliver(ctx context.Context, h domain.Webhook, ev domain.Event) {
	body, err := json.Marshal(ev)
	if err != nil {
		return
	}
	backoff := time.Second
	for attempt := 1; attempt <= 3; attempt++ {
		if err = d.post(ctx, h, body); err == nil {
			return
		}
		select {
		case <-time.After(backoff):
			backoff *= 3
		case <-ctx.Done():
			return
		}
	}
	slog.Warn("Webhook delivery failed", "url", h.URL, "event", ev.Type, "err", err)
}

func (d *Dispatcher) post(ctx context.Context, h domain.Webhook, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	secret := h.Secret
	if d.Open != nil && strings.HasPrefix(secret, "enc:") {
		if secret, err = d.Open(h.TenantID, secret); err != nil {
			return err
		}
	}
	if secret != "" {
		req.Header.Set("X-Callcenter-Signature", "sha256="+Sign(secret, body))
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// Sign returns the hex HMAC-SHA256 of body using secret.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
