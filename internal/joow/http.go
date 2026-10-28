package joow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTP talks to the Joow platform API. Expected contract (see
// docs/JOOW_INTEGRATION.md):
//
//	GET  /v1/me                  Authorization: Bearer <session token> -> Account
//	GET  /v1/accounts?q=&limit=  X-Api-Key                             -> {"accounts": [Account]}
//	GET  /v1/accounts/{id}       X-Api-Key                             -> Account
//	POST /v1/notifications       X-Api-Key  {"account_id", ...Notification}
type HTTP struct {
	base   string
	apiKey string
	client *http.Client
}

// NewHTTP creates the production provider.
func NewHTTP(baseURL, apiKey string) *HTTP {
	return &HTTP{base: strings.TrimRight(baseURL, "/"), apiKey: apiKey, client: &http.Client{Timeout: 10 * time.Second}}
}

// Name implements Provider.
func (h *HTTP) Name() string { return "joow" }

func (h *HTTP) do(ctx context.Context, method, path, bearer string, body, out any) error {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.base+path, rd)
	if err != nil {
		return err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	} else {
		req.Header.Set("X-Api-Key", h.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized && bearer != "":
		return ErrInvalidToken
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode >= 300:
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("joow %s %s: status %d: %s", method, path, resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// VerifyToken implements Provider.
func (h *HTTP) VerifyToken(ctx context.Context, token string) (*Account, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}
	var a Account
	if err := h.do(ctx, http.MethodGet, "/v1/me", token, nil, &a); err != nil {
		return nil, err
	}
	if a.ID == "" {
		return nil, ErrInvalidToken
	}
	return &a, nil
}

// Search implements Provider.
func (h *HTTP) Search(ctx context.Context, query string, limit int) ([]Account, error) {
	var res struct {
		Accounts []Account `json:"accounts"`
	}
	q := url.Values{"q": {query}, "limit": {fmt.Sprint(limit)}}
	err := h.do(ctx, http.MethodGet, "/v1/accounts?"+q.Encode(), "", nil, &res)
	return res.Accounts, err
}

// Get implements Provider.
func (h *HTTP) Get(ctx context.Context, id string) (*Account, error) {
	var a Account
	if err := h.do(ctx, http.MethodGet, "/v1/accounts/"+url.PathEscape(id), "", nil, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Notify implements Provider.
func (h *HTTP) Notify(ctx context.Context, accountID string, n Notification) error {
	body := map[string]any{"account_id": accountID, "kind": n.Kind, "title": n.Title, "body": n.Body,
		"url": n.URL, "data": n.Data}
	return h.do(ctx, http.MethodPost, "/v1/notifications", "", body, nil)
}
