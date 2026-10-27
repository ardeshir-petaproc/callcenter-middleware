package ari

import (
	"context"
	"fmt"
	"net/url"
)

// PlayMedia initiates media playback on the specified channel.
func (c *Client) PlayMedia(ctx context.Context, channelID, mediaURI, playbackID string) error {
	path := fmt.Sprintf("/channels/%s/play/%s?media=%s",
		url.PathEscape(channelID),
		url.PathEscape(playbackID),
		url.QueryEscape(mediaURI),
	)
	resp, err := c.doRequest(ctx, "play", "POST", path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// StopPlayback immediately cancels active playback for barge-in.
func (c *Client) StopPlayback(ctx context.Context, playbackID string) error {
	path := fmt.Sprintf("/playbacks/%s", url.PathEscape(playbackID))
	resp, err := c.doRequest(ctx, "stop_playback", "DELETE", path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
