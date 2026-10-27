package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"callcenter-middleware/internal/domain"
)

// WebhookNotifier pushes handoff events to remote endpoints.
type WebhookNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewWebhookNotifier creates a screen-pop notifier.
func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// NotifyQueueHandoff sends POST request with handoff JSON payload.
func (n *WebhookNotifier) NotifyQueueHandoff(ctx context.Context, payload domain.HandoffNotification) error {
	if n.webhookURL == "" {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notif: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create notif req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post notif: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
