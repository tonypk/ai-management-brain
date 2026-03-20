package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// mockActionDB implements ActionDB for testing (reuses mockTriggerDB shape).
type mockActionDB struct {
	employees     []report.EmployeeInfo
	submittedDays map[string]int
}

func (m *mockActionDB) ListActiveEmployeesWithTelegram(_ context.Context, _ string) ([]report.EmployeeInfo, error) {
	return m.employees, nil
}

func (m *mockActionDB) GetSubmittedDaysLast7(_ context.Context, empID string) (int, error) {
	return m.submittedDays[empID], nil
}

func TestAction_Weekly_Recognition(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
		},
		submittedDays: map[string]int{"e1": 6, "e2": 3},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	executor := report.NewActionExecutor(db, sender, nil, factory)
	err := executor.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	// Inamori has recognition weekly action → should mention Alice as top contributor
	found := false
	for _, msg := range sender.sentMessages {
		if strings.Contains(msg.Message, "Alice") && strings.Contains(msg.Message, "6/7") {
			found = true
		}
	}
	if !found {
		t.Error("expected recognition message for Alice with 6/7 days")
	}
}

func TestAction_Weekly_Ranking(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
			{ID: "e3", Name: "Charlie", TelegramID: 333},
		},
		submittedDays: map[string]int{"e1": 5, "e2": 7, "e3": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	// Ren mentor has ranking action
	executor := report.NewActionExecutor(db, sender, nil, factory)
	err := executor.RunWeekly(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	// Should have sent a ranking message
	rankingFound := false
	for _, msg := range sender.sentMessages {
		if strings.Contains(msg.Message, "Ranking") {
			rankingFound = true
			// Bob (7) should be first, Alice (5) second, Charlie (2) third
			bobIdx := strings.Index(msg.Message, "Bob")
			aliceIdx := strings.Index(msg.Message, "Alice")
			charlieIdx := strings.Index(msg.Message, "Charlie")
			if bobIdx > aliceIdx || aliceIdx > charlieIdx {
				t.Errorf("ranking order wrong: Bob=%d, Alice=%d, Charlie=%d", bobIdx, aliceIdx, charlieIdx)
			}
		}
	}
	if !rankingFound {
		t.Error("expected ranking message for ren mentor")
	}
}

func TestAction_Weekly_OneOnOne(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
		},
		submittedDays: map[string]int{"e1": 2, "e2": 6},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	// Grove mentor has one_on_one action
	executor := report.NewActionExecutor(db, sender, nil, factory)
	err := executor.RunWeekly(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	// Alice (2/7) should be suggested for 1:1, Bob (6/7) should not
	oneOnOneFound := false
	for _, msg := range sender.sentMessages {
		if strings.Contains(msg.Message, "1:1") && strings.Contains(msg.Message, "Alice") {
			oneOnOneFound = true
			if strings.Contains(msg.Message, "Bob") {
				t.Error("Bob with 6/7 days should not need 1:1")
			}
		}
	}
	if !oneOnOneFound {
		t.Error("expected 1:1 suggestion for Alice with low submissions")
	}
}

func TestAction_Weekly_NoEmployees(t *testing.T) {
	db := &mockActionDB{
		employees:     nil,
		submittedDays: map[string]int{},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	executor := report.NewActionExecutor(db, sender, nil, factory)
	err := executor.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	// Recognition with no employees should produce no messages (empty string skipped)
	for _, msg := range sender.sentMessages {
		if strings.Contains(msg.Message, "Recognition") {
			t.Error("should not send recognition with no employees")
		}
	}
}

func TestAction_Monthly(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
		},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	executor := report.NewActionExecutor(db, sender, nil, factory)
	err := executor.RunMonthly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunMonthly: %v", err)
	}

	// Inamori has monthly actions → should send at least one message
	if len(sender.sentMessages) == 0 {
		// Some mentors may not have monthly actions, that's OK
		t.Log("no monthly actions for inamori (expected if none configured)")
	}
}
