package ari

import (
	"context"
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
