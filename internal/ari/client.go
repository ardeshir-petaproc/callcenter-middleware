package ari

import (
	"bytes"
	"context"
	"encoding/json"
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
	app        string
	httpClient *http.Client
	metrics    domain.MetricsRecorder
}

// NewClient creates a new ARI REST client instance.
func NewClient(baseURL, username, password, app string, metrics domain.MetricsRecorder) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		app:      app,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		metrics: metrics,
	}
}

func (c *Client) doRequest(ctx context.Context, op, method, path string) (*http.Response, error) {
	return c.doRequestBody(ctx, op, method, path, nil)
}

// do executes a request and discards the response body.
func (c *Client) do(ctx context.Context, op, method, path string) error {
	resp, err := c.doRequest(ctx, op, method, path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) doRequestBody(ctx context.Context, op, method, path string, body any) (*http.Response, error) {
	start := time.Now()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", op, err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create req %s: %w", op, err)
	}
	req.SetBasicAuth(c.username, c.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

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
