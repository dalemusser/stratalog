package logbrowser

import (
	"sync"
	"time"
)

// LogEvent represents a log entry broadcast to SSE subscribers.
type LogEvent struct {
	ID          string                 `json:"id"`
	Game        string                 `json:"game"`
	PlayerID    string                 `json:"playerId,omitempty"`
	EventType   string                 `json:"eventType,omitempty"`
	ServerTimestamp time.Time              `json:"serverTimestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Hub manages SSE subscribers for real-time log updates.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan LogEvent]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[chan LogEvent]struct{}),
	}
}

// Subscribe adds a new subscriber and returns their channel.
func (h *Hub) Subscribe() chan LogEvent {
	ch := make(chan LogEvent, 16) // buffered to avoid blocking broadcast
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber and closes their channel.
func (h *Hub) Unsubscribe(ch chan LogEvent) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends a log event to all subscribers.
// Non-blocking: slow subscribers are skipped.
func (h *Hub) Broadcast(event LogEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// Skip slow subscribers to avoid blocking
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
