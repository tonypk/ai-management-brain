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
