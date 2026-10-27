package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/config"
)

// OpenAI is a minimal REST client for the endpoints the middleware uses.
type OpenAI struct {
	cfg  *config.Config
	http *http.Client
}

// NewOpenAI creates a REST client.
func NewOpenAI(cfg *config.Config) *OpenAI {
	return &OpenAI{cfg: cfg, http: &http.Client{Timeout: 120 * time.Second}}
}

func (o *OpenAI) do(ctx context.Context, path, contentType string, body io.Reader, out any) error {
	if o.cfg.OpenAIAPIKey == "" {
		return ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(o.cfg.OpenAIBaseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", contentType)
	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("openai %s: status %d: %s", path, resp.StatusCode, truncate(string(raw), 300))
	}
	return json.Unmarshal(raw, out)
}

func (o *OpenAI) postJSON(ctx context.Context, path string, in, out any) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return o.do(ctx, path, "application/json", bytes.NewReader(raw), out)
}

// ChatJSON asks a chat model for a JSON object response.
func (o *OpenAI) ChatJSON(ctx context.Context, system, user string, out any) error {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err := o.postJSON(ctx, "/chat/completions", map[string]any{
		"model":           o.cfg.OpenAISummaryModel,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}, &resp)
	if err != nil {
		return err
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("openai: empty completion")
	}
	return json.Unmarshal([]byte(resp.Choices[0].Message.Content), out)
}

// Transcribe converts an audio file to text.
func (o *OpenAI) Transcribe(ctx context.Context, filename string, audio io.Reader) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", o.cfg.OpenAITranscribeModel)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, audio); err != nil {
		return "", err
	}
	w.Close()
	var resp struct {
		Text string `json:"text"`
	}
	if err := o.do(ctx, "/audio/transcriptions", w.FormDataContentType(), &buf, &resp); err != nil {
		return "", err
	}
	return resp.Text, nil
}

// ClientSecret mints a short-lived Realtime credential for a browser
// session configured with the given session object.
func (o *OpenAI) ClientSecret(ctx context.Context, session map[string]any) (map[string]any, error) {
	var out map[string]any
	err := o.postJSON(ctx, "/realtime/client_secrets", map[string]any{
		"expires_after": map[string]any{"anchor": "created_at", "seconds": 600},
		"session":       session,
	}, &out)
	return out, err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
