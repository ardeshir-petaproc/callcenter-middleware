package domain

import "time"

// Campaign statuses.
const (
	CampaignDraft     = "draft"
	CampaignRunning   = "running"
	CampaignPaused    = "paused"
	CampaignCompleted = "completed"
)

// Campaign is an outbound progressive-dialing campaign. Agents of QueueID
// who are available are rung first and then connected to the next lead.
type Campaign struct {
	ID           int64  `json:"id"`
	TenantID     int64  `json:"tenant_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	QueueID      int64  `json:"queue_id"`
	Status       string `json:"status"`
	MaxAttempts  int    `json:"max_attempts"`
	RetryMinutes int    `json:"retry_minutes"`
	ScheduleID   int64  `json:"schedule_id"`
	Script       string `json:"script"`
}

// Lead statuses.
const (
	LeadPending  = "pending"
	LeadDialing  = "dialing"
	LeadAnswered = "answered"
	LeadFailed   = "failed"
	LeadDone     = "done"
)

// Lead is one number to call in a campaign.
type Lead struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	CampaignID  int64     `json:"campaign_id"`
	Number      string    `json:"number"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	NextAttempt time.Time `json:"next_attempt"`
	CallID      string    `json:"call_id"`
	Notes       string    `json:"notes"`
	UpdatedAt   time.Time `json:"updated_at"`
}
