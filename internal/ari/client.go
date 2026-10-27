package ari

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"callcenter-middleware/internal/domain"
)

// Client provides typed HTTP REST access to Asterisk ARI.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	metrics    domain.MetricsRecorder
}

// NewClient creates a new ARI REST client instance.
func NewClient(baseURL, username, password string, metrics domain.MetricsRecorder) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		metrics: metrics,
	}
}

func (c *Client) doRequest(ctx context.Context, op, method, path string) (*http.Response, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create req %s: %w", op, err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	duration := time.Since(start).Seconds()
	if c.metrics != nil {
		c.metrics.RecordARILatency(op, duration)
	}
	if err != nil {
		return nil, fmt.Errorf("execute %s: %w", op, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("op %s failed: status %d body %s", op, resp.StatusCode, string(body))
	}
	return resp, nil
}
