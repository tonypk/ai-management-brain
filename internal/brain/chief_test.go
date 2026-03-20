package brain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// mockChiefDB implements brain.ChiefDB for testing.
type mockChiefDB struct {
	employees map[string]brain.ChiefEmployee
	statuses  map[string]brain.EmployeeStatus
}

func (m *mockChiefDB) FindEmployeeByName(ctx context.Context, tenantID, name string) (brain.ChiefEmployee, error) {
	emp, ok := m.employees[name]
	if !ok {
		return brain.ChiefEmployee{}, fmt.Errorf("employee %q not found", name)
	}
	return emp, nil
}

func (m *mockChiefDB) GetEmployeeStatus(ctx context.Context, employeeID string) (brain.EmployeeStatus, error) {
	status, ok := m.statuses[employeeID]
	if !ok {
		return brain.EmployeeStatus{}, fmt.Errorf("status not found for %q", employeeID)
	}
	return status, nil
}

// mockChiefSender implements brain.ChiefSender for testing.
type mockChiefSender struct {
	sent []struct {
		ChannelType string
		UserID      string
		Text        string
	}
}

func (m *mockChiefSender) Send(ctx context.Context, channelType, userID, text string) error {
	m.sent = append(m.sent, struct {
		ChannelType string
		UserID      string
		Text        string
	}{channelType, userID, text})
	return nil
}

func TestChief_AskEmployee(t *testing.T) {
	db := &mockChiefDB{
		employees: map[string]brain.ChiefEmployee{
			"john": {ID: "e1", Name: "John", ChannelID: "123", Channel: "telegram"},
		},
	}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "Ask John about project progress")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(sender.sent))
	}
	if sender.sent[0].UserID != "123" {
		t.Errorf("sent to userID = %q, want %q", sender.sent[0].UserID, "123")
	}
	if reply == "" {
		t.Error("expected non-empty reply")
	}
}

func TestChief_CheckStatus(t *testing.T) {
	db := &mockChiefDB{
		employees: map[string]brain.ChiefEmployee{
			"alice": {ID: "e2", Name: "Alice", ChannelID: "456", Channel: "slack"},
		},
		statuses: map[string]brain.EmployeeStatus{
			"e2": {
				EmployeeName:   "Alice",
				SubmittedToday: true,
				MissedDays:     0,
				LastSentiment:  "positive",
			},
		},
	}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "How is Alice doing?")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if reply == "" {
		t.Error("expected non-empty status reply")
	}
}

func TestChief_SwitchMentor_Valid(t *testing.T) {
	db := &mockChiefDB{employees: map[string]brain.ChiefEmployee{}}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "Switch to dalio")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if reply == "" {
		t.Error("expected non-empty reply")
	}
}

func TestChief_SwitchMentor_Invalid(t *testing.T) {
	db := &mockChiefDB{employees: map[string]brain.ChiefEmployee{}}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "Switch to batman")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if reply == "" {
		t.Error("expected error message about unknown mentor")
	}
}

func TestChief_EmployeeNotFound(t *testing.T) {
	db := &mockChiefDB{employees: map[string]brain.ChiefEmployee{}}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "Ask NonExistent about work")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	// Should return a user-friendly "not found" message, not an error
	if reply == "" {
		t.Error("expected non-empty reply about employee not found")
	}
	if len(sender.sent) != 0 {
		t.Errorf("expected 0 messages sent for non-existent employee, got %d", len(sender.sent))
	}
}

func TestChief_GetSummary(t *testing.T) {
	db := &mockChiefDB{employees: map[string]brain.ChiefEmployee{}}
	sender := &mockChiefSender{}
	orch := brain.NewOrchestrator(nil)
	chief := brain.NewChief(orch, db, sender, nil)

	reply, err := chief.HandleCommand(context.Background(), "t1", "summary")
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if reply == "" {
		t.Error("expected non-empty reply")
	}
}
