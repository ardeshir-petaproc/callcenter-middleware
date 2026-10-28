package domain

// AIProfile configures an OpenAI realtime voice agent.
type AIProfile struct {
	ID             int64    `json:"id"`
	TenantID       int64    `json:"tenant_id"`
	Name           string   `json:"name"`
	Model          string   `json:"model"`
	Voice          string   `json:"voice"`
	Language       string   `json:"language"`
	Instructions   string   `json:"instructions"`
	Greeting       string   `json:"greeting"`
	Temperature    float64  `json:"temperature"`
	Tools          []string `json:"tools"`
	HandoffQueueID int64    `json:"handoff_queue_id"`
	MaxDurationSec int      `json:"max_duration_sec"`
	KnowledgeBase  string   `json:"knowledge_base"`
	Enabled        bool     `json:"enabled"`
}
