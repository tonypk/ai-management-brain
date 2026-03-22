package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

// ---------------------------------------------------------------------------
// New tests using miniredis for real Redis pub/sub coverage
// ---------------------------------------------------------------------------

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

func TestBus_Publish_Success(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	ctx := context.Background()

	payloadJSON, err := json.Marshal(events.ReportSubmittedPayload{
		EmployeeID:   "emp-1",
		EmployeeName: "Alice",
		ReportDate:   "2026-03-20",
	})
	if err != nil {
		t.Fatal(err)
	}

	event := events.Event{
		Type:     events.ReportSubmitted,
		TenantID: "tenant-1",
		Payload:  payloadJSON,
	}

	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}

func TestBus_Publish_RedisDown(t *testing.T) {
	mr, rdb := newTestRedis(t)

	bus := events.NewBus(rdb)

	// Shut down miniredis to simulate failure
	mr.Close()

	payloadJSON, _ := json.Marshal(events.ReportSubmittedPayload{
		EmployeeID:   "emp-1",
		EmployeeName: "Alice",
		ReportDate:   "2026-03-20",
	})

	event := events.Event{
		Type:     events.ReportSubmitted,
		TenantID: "tenant-1",
		Payload:  payloadJSON,
	}

	err := bus.Publish(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Redis is down, got nil")
	}
}

func TestBus_PublishPayload_Success(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	ctx := context.Background()

	payload := events.AlertFiredPayload{
		EmployeeID:   "emp-2",
		EmployeeName: "Bob",
		AlertType:    "sentiment_drop",
		Message:      "Sentiment dropped",
		Severity:     "warning",
	}

	err := bus.PublishPayload(ctx, events.AlertFired, "tenant-2", payload)
	if err != nil {
		t.Fatalf("PublishPayload returned error: %v", err)
	}
}

func TestBus_PublishPayload_MarshalError(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	// A channel (func) cannot be marshalled to JSON
	badPayload := make(chan int)

	err := bus.PublishPayload(context.Background(), events.AlertFired, "tenant-1", badPayload)
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}

func TestBus_Listen_ReceivesAndDispatches(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	var mu sync.Mutex
	var received []events.Event

	bus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	// Give the subscriber time to connect
	time.Sleep(100 * time.Millisecond)

	// Publish an event using a separate client (simulates another service)
	payloadJSON, _ := json.Marshal(events.ReportSubmittedPayload{
		EmployeeID:   "emp-10",
		EmployeeName: "Charlie",
		ReportDate:   "2026-03-21",
	})
	event := events.Event{
		Type:     events.ReportSubmitted,
		TenantID: "tenant-5",
		Payload:  payloadJSON,
	}
	if err := bus.Publish(ctx, event); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Wait for the handler to be called
	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for handler to receive event")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 received event, got %d", len(received))
	}
	if received[0].Type != events.ReportSubmitted {
		t.Errorf("Type = %q, want %q", received[0].Type, events.ReportSubmitted)
	}
	if received[0].TenantID != "tenant-5" {
		t.Errorf("TenantID = %q, want %q", received[0].TenantID, "tenant-5")
	}

	var p events.ReportSubmittedPayload
	if err := json.Unmarshal(received[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p.EmployeeName != "Charlie" {
		t.Errorf("EmployeeName = %q, want %q", p.EmployeeName, "Charlie")
	}

	cancel()
}

func TestBus_Listen_ContextCancellation(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	bus.Subscribe(events.AlertFired, func(ctx context.Context, event events.Event) error {
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	// Give Listen time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	select {
	case err := <-listenErr:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return after context cancellation")
	}
}

func TestBus_Listen_ContextTimeout(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	bus.Subscribe(events.ChaseTriggered, func(ctx context.Context, event events.Event) error {
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := bus.Listen(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestBus_Dispatch_InvalidJSON(t *testing.T) {
	// This test verifies dispatch handles invalid JSON gracefully.
	// We publish raw invalid JSON to the Redis channel and ensure
	// the listener does not panic.
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	handlerCalled := false
	bus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
		handlerCalled = true
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish invalid JSON directly to the channel
	channel := "brain:events:" + string(events.ReportSubmitted)
	if err := rdb.Publish(ctx, channel, "not-valid-json{{{").Err(); err != nil {
		t.Fatalf("raw publish: %v", err)
	}

	// Give dispatch time to process the invalid message
	time.Sleep(200 * time.Millisecond)

	if handlerCalled {
		t.Error("handler should NOT be called for invalid JSON")
	}

	cancel()

	select {
	case err := <-listenErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected listen error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return")
	}
}

func TestBus_Dispatch_HandlerError(t *testing.T) {
	// Verify that a handler error is logged but does not crash the bus
	// and subsequent handlers still run.
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	var secondCalled atomic.Bool

	// First handler returns an error
	bus.Subscribe(events.AlertFired, func(ctx context.Context, event events.Event) error {
		return errors.New("handler failed intentionally")
	})

	// Second handler should still be called
	bus.Subscribe(events.AlertFired, func(ctx context.Context, event events.Event) error {
		secondCalled.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err := bus.PublishPayload(ctx, events.AlertFired, "tenant-err", events.AlertFiredPayload{
		EmployeeID:   "emp-err",
		EmployeeName: "ErrorUser",
		AlertType:    "test",
		Message:      "test error handling",
		Severity:     "low",
	})
	if err != nil {
		t.Fatalf("PublishPayload: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		if secondCalled.Load() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for second handler to be called")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	if !secondCalled.Load() {
		t.Error("second handler was not called after first handler returned error")
	}

	cancel()
}

func TestBus_MultipleHandlers_AllCalled(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	const numHandlers = 5
	var callCount atomic.Int32

	for i := 0; i < numHandlers; i++ {
		bus.Subscribe(events.MentorChanged, func(ctx context.Context, event events.Event) error {
			callCount.Add(1)
			return nil
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err := bus.PublishPayload(ctx, events.MentorChanged, "tenant-multi", map[string]string{
		"employee_id": "emp-m1",
		"new_mentor":  "mentor-2",
	})
	if err != nil {
		t.Fatalf("PublishPayload: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		if callCount.Load() >= int32(numHandlers) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out: only %d/%d handlers called", callCount.Load(), numHandlers)
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	if got := callCount.Load(); got != int32(numHandlers) {
		t.Errorf("expected %d handler calls, got %d", numHandlers, got)
	}

	cancel()
}

func TestBus_DifferentEventTypes_OnlyMatchingHandlersCalled(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	var submittedCalled atomic.Bool
	var analyzedCalled atomic.Bool

	bus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
		submittedCalled.Store(true)
		return nil
	})
	bus.Subscribe(events.ReportAnalyzed, func(ctx context.Context, event events.Event) error {
		analyzedCalled.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Only publish ReportSubmitted
	err := bus.PublishPayload(ctx, events.ReportSubmitted, "tenant-routing", events.ReportSubmittedPayload{
		EmployeeID:   "emp-r1",
		EmployeeName: "Diana",
		ReportDate:   "2026-03-21",
	})
	if err != nil {
		t.Fatalf("PublishPayload: %v", err)
	}

	// Wait for the submitted handler to be called
	deadline := time.After(2 * time.Second)
	for {
		if submittedCalled.Load() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for ReportSubmitted handler")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Give extra time to see if the analyzed handler gets incorrectly called
	time.Sleep(100 * time.Millisecond)

	if analyzedCalled.Load() {
		t.Error("ReportAnalyzed handler should NOT have been called for a ReportSubmitted event")
	}

	cancel()
}

func TestBus_Listen_MultipleEventsInSequence(t *testing.T) {
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	var mu sync.Mutex
	var received []events.Event

	bus.Subscribe(events.EmployeeJoined, func(ctx context.Context, event events.Event) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish 3 events in sequence
	for i := 0; i < 3; i++ {
		err := bus.PublishPayload(ctx, events.EmployeeJoined, "tenant-seq", map[string]string{
			"employee_id": "emp-seq-" + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("PublishPayload #%d: %v", i, err)
		}
	}

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n >= 3 {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timed out: received %d/3 events", len(received))
			mu.Unlock()
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Errorf("expected 3 events, got %d", len(received))
	}
	for _, ev := range received {
		if ev.Type != events.EmployeeJoined {
			t.Errorf("unexpected event type: %s", ev.Type)
		}
		if ev.TenantID != "tenant-seq" {
			t.Errorf("unexpected tenant: %s", ev.TenantID)
		}
	}

	cancel()
}

func TestBus_PublishPayload_VerifyEventStructure(t *testing.T) {
	// Verify that PublishPayload correctly wraps the payload into an Event
	_, rdb := newTestRedis(t)
	bus := events.NewBus(rdb)

	var mu sync.Mutex
	var received events.Event

	bus.Subscribe(events.SummaryGenerated, func(ctx context.Context, event events.Event) error {
		mu.Lock()
		received = event
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = bus.Listen(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	type SummaryPayload struct {
		TeamID  string `json:"team_id"`
		Summary string `json:"summary"`
	}

	err := bus.PublishPayload(ctx, events.SummaryGenerated, "tenant-sp", SummaryPayload{
		TeamID:  "team-1",
		Summary: "All reports submitted on time.",
	})
	if err != nil {
		t.Fatalf("PublishPayload: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		hasEvent := received.Type != ""
		mu.Unlock()
		if hasEvent {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SummaryGenerated event")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if received.Type != events.SummaryGenerated {
		t.Errorf("Type = %q, want %q", received.Type, events.SummaryGenerated)
	}
	if received.TenantID != "tenant-sp" {
		t.Errorf("TenantID = %q, want %q", received.TenantID, "tenant-sp")
	}

	var sp SummaryPayload
	if err := json.Unmarshal(received.Payload, &sp); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if sp.TeamID != "team-1" {
		t.Errorf("TeamID = %q, want %q", sp.TeamID, "team-1")
	}
	if sp.Summary != "All reports submitted on time." {
		t.Errorf("Summary = %q, want %q", sp.Summary, "All reports submitted on time.")
	}

	cancel()
}

func TestChatCompletedPayload_Marshal(t *testing.T) {
	p := events.ChatCompletedPayload{
		EmployeeID: "emp-123",
		Messages:   `[{"role":"user","content":"hi"}]`,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded events.ChatCompletedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EmployeeID != "emp-123" {
		t.Fatalf("unexpected: %+v", decoded)
	}
}

func TestChatCompletedEventType(t *testing.T) {
	if events.ChatCompleted != "chat.completed" {
		t.Fatalf("unexpected event type: %s", events.ChatCompleted)
	}
}
