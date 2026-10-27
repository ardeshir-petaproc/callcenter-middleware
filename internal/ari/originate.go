package ari

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"callcenter-middleware/internal/domain"
)

// Originate places a new channel that enters the Stasis app once answered.
func (c *Client) Originate(ctx context.Context, req domain.OriginateRequest) error {
	q := url.Values{}
	q.Set("endpoint", req.Endpoint)
	q.Set("app", c.app)
	if req.AppArgs != "" {
		q.Set("appArgs", req.AppArgs)
	}
	if req.ChannelID != "" {
		q.Set("channelId", req.ChannelID)
	}
	if req.CallerID != "" {
		q.Set("callerId", req.CallerID)
	}
	if req.TimeoutSec > 0 {
		q.Set("timeout", strconv.Itoa(req.TimeoutSec))
	}
	var body any
	if len(req.Variables) > 0 {
		body = map[string]any{"variables": req.Variables}
	}
	resp, err := c.doRequestBody(ctx, "originate", "POST", "/channels?"+q.Encode(), body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ExternalMedia creates an RTP channel that streams audio to ExternalHost.
func (c *Client) ExternalMedia(ctx context.Context, req domain.ExternalMediaRequest) (string, error) {
	q := url.Values{}
	q.Set("app", c.app)
	q.Set("external_host", req.ExternalHost)
	format := req.Format
	if format == "" {
		format = "ulaw"
	}
	q.Set("format", format)
	q.Set("encapsulation", "rtp")
	q.Set("transport", "udp")
	q.Set("direction", "both")
	if req.ChannelID != "" {
		q.Set("channelId", req.ChannelID)
	}
	resp, err := c.doRequest(ctx, "external_media", "POST", "/channels/externalMedia?"+q.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var ch struct {
		Vars map[string]string `json:"channelvars"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ch)
	addr, port := ch.Vars["UNICASTRTP_LOCAL_ADDRESS"], ch.Vars["UNICASTRTP_LOCAL_PORT"]
	if addr == "" || port == "" {
		return "", nil
	}
	return net.JoinHostPort(addr, port), nil
}

// Snoop creates a snoop channel on a target channel for monitoring.
func (c *Client) Snoop(ctx context.Context, req domain.SnoopRequest) error {
	q := url.Values{}
	q.Set("app", c.app)
	q.Set("spy", orDefault(req.Spy, "both"))
	q.Set("whisper", orDefault(req.Whisper, "none"))
	if req.SnoopID != "" {
		q.Set("snoopId", req.SnoopID)
	}
	if req.AppArgs != "" {
		q.Set("appArgs", req.AppArgs)
	}
	path := fmt.Sprintf("/channels/%s/snoop?%s", url.PathEscape(req.ChannelID), q.Encode())
	return c.do(ctx, "snoop", "POST", path)
}

// FetchRecording streams a stored recording file; the caller closes the body.
func (c *Client) FetchRecording(ctx context.Context, name string) (*http.Response, error) {
	return c.doRequest(ctx, "recording_file", "GET", "/recordings/stored/"+url.PathEscape(name)+"/file")
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
