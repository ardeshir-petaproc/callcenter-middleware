package ari

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AnswerChannel sends an answer request to the channel.
func (c *Client) AnswerChannel(ctx context.Context, channelID string) error {
	path := fmt.Sprintf("/channels/%s/answer", url.PathEscape(channelID))
	resp, err := c.doRequest(ctx, "answer", "POST", path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ContinueToQueue releases a channel from Stasis into dialplan context.
func (c *Client) ContinueToQueue(ctx context.Context, channelID, contextName, queueExt string, priority int) error {
	path := fmt.Sprintf("/channels/%s/continue?context=%s&extension=%s&priority=%d",
		url.PathEscape(channelID),
		url.QueryEscape(contextName),
		url.QueryEscape(queueExt),
		priority,
	)
	resp, err := c.doRequest(ctx, "continue", "POST", path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// HangupChannel terminates an active channel with reason.
func (c *Client) HangupChannel(ctx context.Context, channelID, reason string) error {
	path := fmt.Sprintf("/channels/%s?reason=%s", url.PathEscape(channelID), url.QueryEscape(reason))
	resp, err := c.doRequest(ctx, "hangup", "DELETE", path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ListChannels returns the ids of all active channels.
func (c *Client) ListChannels(ctx context.Context) ([]string, error) {
	resp, err := c.doRequest(ctx, "list_channels", "GET", "/channels")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var chans []RawChannel
	if err := json.NewDecoder(resp.Body).Decode(&chans); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(chans))
	for _, ch := range chans {
		ids = append(ids, ch.ID)
	}
	return ids, nil
}
