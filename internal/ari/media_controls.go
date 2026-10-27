package ari

import (
	"context"
	"fmt"
	"net/url"
)

// StartMOH plays music on hold to a channel.
func (c *Client) StartMOH(ctx context.Context, channelID, class string) error {
	path := fmt.Sprintf("/channels/%s/moh", url.PathEscape(channelID))
	if class != "" {
		path += "?mohClass=" + url.QueryEscape(class)
	}
	return c.do(ctx, "moh_start", "POST", path)
}

// StopMOH stops music on hold.
func (c *Client) StopMOH(ctx context.Context, channelID string) error {
	return c.do(ctx, "moh_stop", "DELETE", fmt.Sprintf("/channels/%s/moh", url.PathEscape(channelID)))
}

// StartRinging indicates ringing to a channel.
func (c *Client) StartRinging(ctx context.Context, channelID string) error {
	return c.do(ctx, "ring_start", "POST", fmt.Sprintf("/channels/%s/ring", url.PathEscape(channelID)))
}

// StopRinging stops ringing indication.
func (c *Client) StopRinging(ctx context.Context, channelID string) error {
	return c.do(ctx, "ring_stop", "DELETE", fmt.Sprintf("/channels/%s/ring", url.PathEscape(channelID)))
}

// SetMute mutes or unmutes audio coming from a channel.
func (c *Client) SetMute(ctx context.Context, channelID string, mute bool) error {
	method := "DELETE"
	if mute {
		method = "POST"
	}
	return c.do(ctx, "mute", method, fmt.Sprintf("/channels/%s/mute?direction=in", url.PathEscape(channelID)))
}
