package report_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

type mockTriggerDB struct {
	employees     []report.EmployeeInfo
	missedDays    map[string]int // employeeID → missed days
	submittedDays map[string]int // employeeID → submitted days
}

func (m *mockTriggerDB) ListActiveEmployeesWithTelegram(_ context.Context, _ string) ([]report.EmployeeInfo, error) {
	return m.employees, nil
}

func (m *mockTriggerDB) GetMissedDaysLast7(_ context.Context, empID string) (int, error) {
	return m.missedDays[empID], nil
}

func (m *mockTriggerDB) GetSubmittedDaysLast7(_ context.Context, empID string) (int, error) {
	return m.submittedDays[empID], nil
}

func TestTrigger_ConsecutiveMiss3Days(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
			{ID: "e2", Name: "Bob", TelegramID: 222, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4, "e2": 1}, // Alice missed 4, Bob missed 1
		submittedDays: map[string]int{"e1": 3, "e2": 6},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	// Inamori has consecutive_miss_3days trigger → should fire for Alice (4 missed)
	missTriggered := false
	for _, r := range results {
		if r.EmployeeName == "Alice" && r.Event == "consecutive_miss_3days" {
			missTriggered = true
		}
	}
	if !missTriggered {
		t.Error("expected trigger for Alice with 4 missed days")
	}

	// Bob should NOT trigger (only 1 missed day)
	for _, r := range results {
		if r.EmployeeName == "Bob" && r.Event == "consecutive_miss_3days" {
			t.Error("Bob should not trigger with 1 missed day")
		}
	}

	// Should have sent message to boss
	if len(sender.sentMessages) == 0 {
		t.Error("expected boss notification")
	}
}

func TestTrigger_ExceptionalPerformance(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Star", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 0},
		submittedDays: map[string]int{"e1": 7},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	// Ren has exceptional_performance trigger
	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	perfTriggered := false
	for _, r := range results {
		if r.Event == "exceptional_performance" {
			perfTriggered = true
		}
	}
	if !perfTriggered {
		t.Error("expected exceptional_performance trigger for Star with 7/7 days")
	}
}

func TestTrigger_NoTriggersWhenAllGood(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Normal", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 1},
		submittedDays: map[string]int{"e1": 4},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no triggers, got %d", len(results))
	}
}
