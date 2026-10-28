package core

import "callcenter-middleware/internal/domain"

// QueueAlert is published when a queue crosses (or recovers from) its
// configured waiting-calls or wait-time threshold.
type QueueAlert struct {
	QueueID        int64  `json:"queue_id"`
	Name           string `json:"name"`
	Level          string `json:"level"` // "alert" or "clear"
	Waiting        int    `json:"waiting"`
	LongestWaitSec int    `json:"longest_wait_sec"`
	Available      int    `json:"available"`
}

func (c *Core) checkAlerts() {
	for _, v := range c.queueViews(0) {
		q := c.cache.queues[v.ID]
		if q == nil || (q.AlertWaiting <= 0 && q.AlertWaitSec <= 0) {
			delete(c.alerts, v.ID)
			continue
		}
		breach := (q.AlertWaiting > 0 && v.Waiting >= q.AlertWaiting) ||
			(q.AlertWaitSec > 0 && v.LongestWaitSec >= q.AlertWaitSec)
		if breach == c.alerts[v.ID] {
			continue
		}
		c.alerts[v.ID] = breach
		level := "clear"
		if breach {
			level = "alert"
		}
		c.publish(q.TenantID, domain.EvQueueAlert, QueueAlert{QueueID: v.ID, Name: v.Name, Level: level,
			Waiting: v.Waiting, LongestWaitSec: v.LongestWaitSec, Available: v.Available})
	}
}

// LevelString returns the alert level (helps consumers holding an `any`).
func (a QueueAlert) LevelString() string { return a.Level }

// SetPublisherForTest swaps the event publisher (tests only).
func (c *Core) SetPublisherForTest(p domain.Publisher) {
	_ = c.do(func() error { c.Publisher = p; return nil })
}
