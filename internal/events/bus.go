// Package events provides a Redis pub/sub-based event bus for internal communication.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// EventType identifies the kind of event.
type EventType string

const (
	ReportSubmitted  EventType = "report.submitted"
	ReportAnalyzed   EventType = "report.analyzed"
	ChaseTriggered   EventType = "chase.triggered"
	SummaryGenerated EventType = "summary.generated"
	AlertFired       EventType = "alert.fired"
	MentorChanged    EventType = "mentor.changed"
	EmployeeJoined   EventType = "employee.joined"
)

// Event is the envelope for all events.
type Event struct {
	Type     EventType       `json:"type"`
	TenantID string          `json:"tenant_id"`
	Payload  json.RawMessage `json:"payload"`
}

// ReportSubmittedPayload is sent when an employee submits a report.
type ReportSubmittedPayload struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ReportDate   string `json:"report_date"`
}

// ReportAnalyzedPayload is sent after blocker/sentiment analysis completes.
type ReportAnalyzedPayload struct {
	EmployeeID string `json:"employee_id"`
	ReportDate string `json:"report_date"`
	Blockers   string `json:"blockers"`
	Sentiment  string `json:"sentiment"`
}

// AlertFiredPayload is sent when an anomaly is detected.
type AlertFiredPayload struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	AlertType    string `json:"alert_type"` // "consecutive_miss", "sentiment_drop", "blocker_surge"
	Message      string `json:"message"`
	Severity     string `json:"severity"` // "warning", "critical"
}

// Handler processes events.
type Handler func(ctx context.Context, event Event) error

// Bus is a Redis pub/sub event bus.
type Bus struct {
	rdb      *redis.Client
	prefix   string
	handlers map[EventType][]Handler
	mu       sync.RWMutex
}

// NewBus creates a new event bus backed by Redis pub/sub.
func NewBus(rdb *redis.Client) *Bus {
	return &Bus{
		rdb:      rdb,
		prefix:   "brain:events:",
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for a specific event type.
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish sends an event to all subscribers via Redis pub/sub.
func (b *Bus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	channel := b.prefix + string(event.Type)
	if err := b.rdb.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("publish event %s: %w", event.Type, err)
	}

	slog.Debug("event published", "type", event.Type, "tenant_id", event.TenantID)
	return nil
}

// PublishPayload is a convenience method that marshals the payload automatically.
func (b *Bus) PublishPayload(ctx context.Context, eventType EventType, tenantID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return b.Publish(ctx, Event{
		Type:     eventType,
		TenantID: tenantID,
		Payload:  data,
	})
}

// Listen starts listening for events on all subscribed channels.
// It blocks until ctx is cancelled.
func (b *Bus) Listen(ctx context.Context) error {
	b.mu.RLock()
	channels := make([]string, 0, len(b.handlers))
	for eventType := range b.handlers {
		channels = append(channels, b.prefix+string(eventType))
	}
	b.mu.RUnlock()

	if len(channels) == 0 {
		slog.Info("event bus: no subscriptions, skipping listen")
		<-ctx.Done()
		return ctx.Err()
	}

	sub := b.rdb.Subscribe(ctx, channels...)
	defer sub.Close()

	slog.Info("event bus listening", "channels", len(channels))

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			b.dispatch(ctx, msg)
		}
	}
}

func (b *Bus) dispatch(ctx context.Context, msg *redis.Message) {
	var event Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		slog.Error("event bus: unmarshal failed", "channel", msg.Channel, "error", err)
		return
	}

	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			slog.Error("event handler error", "type", event.Type, "error", err)
		}
	}
}
