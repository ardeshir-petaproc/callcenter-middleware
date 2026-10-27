package ari

// RawChannel holds Asterisk channel payload from ARI.
type RawChannel struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	State  string    `json:"state"`
	Caller RawCaller `json:"caller"`
}

// RawCaller holds caller ID details.
type RawCaller struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

// RawPlayback represents playback state in Asterisk ARI.
type RawPlayback struct {
	ID        string `json:"id"`
	MediaURI  string `json:"media_uri"`
	TargetURI string `json:"target_uri"`
	State     string `json:"state"`
}

// RawEvent represents incoming Asterisk ARI WebSocket events.
type RawEvent struct {
	Type      string       `json:"type"`
	Timestamp string       `json:"timestamp"`
	Channel   *RawChannel  `json:"channel,omitempty"`
	Playback  *RawPlayback `json:"playback,omitempty"`
	Digit     string       `json:"digit,omitempty"`
}
