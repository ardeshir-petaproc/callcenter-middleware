package ari

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CreateBridge creates a bridge with a caller-chosen id.
func (c *Client) CreateBridge(ctx context.Context, bridgeID, bridgeType string) error {
	if bridgeType == "" {
		bridgeType = "mixing"
	}
	path := fmt.Sprintf("/bridges/%s?type=%s", url.PathEscape(bridgeID), url.QueryEscape(bridgeType))
	return c.do(ctx, "bridge_create", "POST", path)
}

// DestroyBridge removes a bridge; channels in it are left in Stasis.
func (c *Client) DestroyBridge(ctx context.Context, bridgeID string) error {
	return c.do(ctx, "bridge_destroy", "DELETE", "/bridges/"+url.PathEscape(bridgeID))
}

// AddToBridge adds one or more channels to a bridge.
func (c *Client) AddToBridge(ctx context.Context, bridgeID string, channelIDs ...string) error {
	path := fmt.Sprintf("/bridges/%s/addChannel?channel=%s",
		url.PathEscape(bridgeID), url.QueryEscape(strings.Join(channelIDs, ",")))
	return c.do(ctx, "bridge_add", "POST", path)
}

// RemoveFromBridge removes a channel from a bridge.
func (c *Client) RemoveFromBridge(ctx context.Context, bridgeID, channelID string) error {
	path := fmt.Sprintf("/bridges/%s/removeChannel?channel=%s", url.PathEscape(bridgeID), url.QueryEscape(channelID))
	return c.do(ctx, "bridge_remove", "POST", path)
}

// RecordBridge starts a mixed recording of a bridge.
func (c *Client) RecordBridge(ctx context.Context, bridgeID, name string) error {
	path := fmt.Sprintf("/bridges/%s/record?name=%s&format=wav&ifExists=overwrite",
		url.PathEscape(bridgeID), url.QueryEscape(name))
	return c.do(ctx, "bridge_record", "POST", path)
}

// RecordChannel records a channel (voicemail); '#' ends the recording.
func (c *Client) RecordChannel(ctx context.Context, channelID, name string, maxSec int) error {
	path := fmt.Sprintf("/channels/%s/record?name=%s&format=wav&maxDurationSeconds=%d&maxSilenceSeconds=10&beep=true&terminateOn=%s&ifExists=overwrite",
		url.PathEscape(channelID), url.QueryEscape(name), maxSec, url.QueryEscape("#"))
	return c.do(ctx, "channel_record", "POST", path)
}

// StopRecording stops a live recording and stores it.
func (c *Client) StopRecording(ctx context.Context, name string) error {
	return c.do(ctx, "record_stop", "POST", "/recordings/live/"+url.PathEscape(name)+"/stop")
}
