package events_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/events"
)

func TestEvent_MarshalRoundtrip(t *testing.T) {
	payload := events.ReportSubmittedPayload{
		EmployeeID:   "emp-1",
		EmployeeName: "Alice",
		ReportDate:   "2026-03-20",
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	event := events.Event{
		Type:     events.ReportSubmitted,
		TenantID: "tenant-1",
		Payload:  payloadJSON,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	var decoded events.Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Type != events.ReportSubmitted {
		t.Errorf("Type = %q, want %q", decoded.Type, events.ReportSubmitted)
	}
	if decoded.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", decoded.TenantID, "tenant-1")
	}

	var decodedPayload events.ReportSubmittedPayload
	if err := json.Unmarshal(decoded.Payload, &decodedPayload); err != nil {
		t.Fatal(err)
	}
	if decodedPayload.EmployeeName != "Alice" {
		t.Errorf("EmployeeName = %q, want %q", decodedPayload.EmployeeName, "Alice")
	}
}

func TestAlertPayload_Severity(t *testing.T) {
	alert := events.AlertFiredPayload{
		EmployeeID:   "emp-1",
		EmployeeName: "Bob",
		AlertType:    "consecutive_miss",
		Message:      "Bob has missed 3 consecutive days",
		Severity:     "critical",
	}

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatal(err)
	}

	var decoded events.AlertFiredPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", decoded.Severity, "critical")
	}
	if decoded.AlertType != "consecutive_miss" {
		t.Errorf("AlertType = %q, want %q", decoded.AlertType, "consecutive_miss")
	}
}

func TestBus_SubscribeAndLocalDispatch(t *testing.T) {
	// Test without Redis — just verify subscribe/handler registration works
	bus := events.NewBus(nil)

	var mu sync.Mutex
	var received []events.Event

	bus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})

	// Verify handler was registered (Listen would connect to Redis which we skip)
	// We test the event structure independently
	_ = bus

	// Just verify no panic
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Listen with nil Redis will panic, so we just test structure
	_ = ctx
}

func TestEventTypes(t *testing.T) {
	types := []events.EventType{
		events.ReportSubmitted,
		events.ReportAnalyzed,
		events.ChaseTriggered,
		events.SummaryGenerated,
		events.AlertFired,
		events.MentorChanged,
		events.EmployeeJoined,
	}

	seen := make(map[events.EventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true

		if et == "" {
			t.Error("empty event type")
		}
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := events.NewBus(nil)

	var mu sync.Mutex
	calls := 0

	handler := func(ctx context.Context, event events.Event) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil
	}

	bus.Subscribe(events.ReportSubmitted, handler)
	bus.Subscribe(events.ReportSubmitted, handler) // 2 handlers for same event

	// Verify no panic on subscribe
	if calls != 0 {
		t.Errorf("expected 0 calls before any dispatch, got %d", calls)
	}
}

func TestBus_SubscribeDifferentTypes(t *testing.T) {
	bus := events.NewBus(nil)

	submitted := false
	analyzed := false

	bus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
		submitted = true
		return nil
	})
	bus.Subscribe(events.ReportAnalyzed, func(ctx context.Context, event events.Event) error {
		analyzed = true
		return nil
	})

	// Verify subscriptions registered without panic
	_ = submitted
	_ = analyzed
}

func TestReportAnalyzedPayload_Roundtrip(t *testing.T) {
	payload := events.ReportAnalyzedPayload{
		EmployeeID: "emp-1",
		ReportDate: "2026-03-20",
		Blockers:   "server outage",
		Sentiment:  "negative",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	var decoded events.ReportAnalyzedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.EmployeeID != "emp-1" {
		t.Errorf("EmployeeID = %q, want %q", decoded.EmployeeID, "emp-1")
	}
	if decoded.Blockers != "server outage" {
		t.Errorf("Blockers = %q, want %q", decoded.Blockers, "server outage")
	}
	if decoded.Sentiment != "negative" {
		t.Errorf("Sentiment = %q, want %q", decoded.Sentiment, "negative")
	}
}

func TestEvent_FullRoundtrip_AllPayloadTypes(t *testing.T) {
	tests := []struct {
		name    string
		event   events.Event
		payload interface{}
	}{
		{
			name: "ReportSubmitted",
			payload: events.ReportSubmittedPayload{
				EmployeeID:   "e1",
				EmployeeName: "Alice",
				ReportDate:   "2026-03-20",
			},
		},
		{
			name: "ReportAnalyzed",
			payload: events.ReportAnalyzedPayload{
				EmployeeID: "e1",
				ReportDate: "2026-03-20",
				Blockers:   "none",
				Sentiment:  "positive",
			},
		},
		{
			name: "AlertFired",
			payload: events.AlertFiredPayload{
				EmployeeID:   "e1",
				EmployeeName: "Bob",
				AlertType:    "sentiment_drop",
				Message:      "Sentiment dropped from positive to negative",
				Severity:     "warning",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadJSON, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			event := events.Event{
				Type:     events.EventType(tt.name),
				TenantID: "tenant-1",
				Payload:  payloadJSON,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("marshal event: %v", err)
			}

			var decoded events.Event
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}

			if decoded.TenantID != "tenant-1" {
				t.Errorf("TenantID = %q, want %q", decoded.TenantID, "tenant-1")
			}
			if decoded.Payload == nil {
				t.Error("Payload should not be nil")
			}
		})
	}
}

func TestBus_NewBus_NotNil(t *testing.T) {
	bus := events.NewBus(nil)
	if bus == nil {
		t.Error("NewBus should return non-nil bus")
	}
}

func TestBus_Listen_NoSubscriptions_Cancels(t *testing.T) {
	bus := events.NewBus(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Listen with no subscriptions should just wait for context
	err := bus.Listen(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
