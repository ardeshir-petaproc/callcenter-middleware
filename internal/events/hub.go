// Package events fans realtime call center events out to subscribers.
package events

import (
	"sync"
	"time"

	"callcenter-middleware/internal/domain"
)

// Hub is an in-process publish/subscribe broker. Slow subscribers drop
// events rather than block the call core.
type Hub struct {
	mu     sync.RWMutex
	nextID int
	subs   map[int]chan domain.Event
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[int]chan domain.Event)}
}

// Publish delivers ev to every subscriber without blocking.
func (h *Hub) Publish(ev domain.Event) {
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Subscribe registers a new subscriber and returns its channel and a cancel func.
func (h *Hub) Subscribe(buffer int) (<-chan domain.Event, func()) {
	ch := make(chan domain.Event, buffer)
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.subs[id] = ch
	h.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, id)
			h.mu.Unlock()
			close(ch)
		})
	}
}

// Count returns the number of active subscribers.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
