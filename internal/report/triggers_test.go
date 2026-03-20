package report_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

type mockTriggerDB struct {
	employees     []report.EmployeeInfo
	missedDays    map[string]int // employeeID -> missed days
	submittedDays map[string]int // employeeID -> submitted days
	listErr       error          // error from ListActiveEmployeesWithTelegram
	missedDaysErr map[string]error
	submittedErr  map[string]error
}

func (m *mockTriggerDB) ListActiveEmployeesWithTelegram(_ context.Context, _ string) ([]report.EmployeeInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.employees, nil
}

func (m *mockTriggerDB) GetMissedDaysLast7(_ context.Context, empID string) (int, error) {
	if m.missedDaysErr != nil {
		if err, ok := m.missedDaysErr[empID]; ok {
			return 0, err
		}
	}
	return m.missedDays[empID], nil
}

func (m *mockTriggerDB) GetSubmittedDaysLast7(_ context.Context, empID string) (int, error) {
	if m.submittedErr != nil {
		if err, ok := m.submittedErr[empID]; ok {
			return 0, err
		}
	}
	return m.submittedDays[empID], nil
}

func TestTrigger_ConsecutiveMiss3Days(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
			{ID: "e2", Name: "Bob", TelegramID: 222, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4, "e2": 1},
		submittedDays: map[string]int{"e1": 3, "e2": 6},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	missTriggered := false
	for _, r := range results {
		if r.EmployeeName == "Alice" && r.Event == "consecutive_miss_3days" {
			missTriggered = true
		}
	}
	if !missTriggered {
		t.Error("expected trigger for Alice with 4 missed days")
	}

	for _, r := range results {
		if r.EmployeeName == "Bob" && r.Event == "consecutive_miss_3days" {
			t.Error("Bob should not trigger with 1 missed day")
		}
	}

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

// --- New tests for increased coverage ---

func TestTrigger_SentimentDropEvent_ReturnsFalse(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 0},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	for _, r := range results {
		if r.Event == "sentiment_drop" {
			t.Error("sentiment_drop should not fire (not yet implemented)")
		}
	}
}

func TestTrigger_BlockerUnresolved_ReturnsFalse(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Charlie", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 5},
		submittedDays: map[string]int{"e1": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	for _, r := range results {
		if r.Event == "blocker_unresolved" {
			t.Error("blocker_unresolved should not fire (not yet implemented)")
		}
	}
}

func TestTrigger_OutputDecline_UsesConsecutiveMissLogic(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 3},
		submittedDays: map[string]int{"e1": 4},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	declineTriggered := false
	for _, r := range results {
		if r.Event == "output_decline_3days" {
			declineTriggered = true
			if r.Action != "suggest_one_on_one" {
				t.Errorf("expected action suggest_one_on_one, got %q", r.Action)
			}
		}
	}
	if !declineTriggered {
		t.Error("expected output_decline_3days trigger with 3 missed days")
	}
}

func TestTrigger_ConsecutiveLowOutput_UsesConsecutiveMissLogic(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "SlowWorker", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 5},
		submittedDays: map[string]int{"e1": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	lowOutputTriggered := false
	for _, r := range results {
		if r.Event == "consecutive_low_output" {
			lowOutputTriggered = true
			if r.Action != "performance_warning" {
				t.Errorf("expected action performance_warning, got %q", r.Action)
			}
		}
	}
	if !lowOutputTriggered {
		t.Error("expected consecutive_low_output trigger for SlowWorker with 5 missed days")
	}
}

func TestTrigger_ExceptionalPerformance_ExactBoundary(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "HighPerf", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 1},
		submittedDays: map[string]int{"e1": 6},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	triggered := false
	for _, r := range results {
		if r.Event == "exceptional_performance" {
			triggered = true
		}
	}
	if !triggered {
		t.Error("expected exceptional_performance to trigger with exactly 6 submitted days")
	}
}

func TestTrigger_ExceptionalPerformance_BelowThreshold(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "AlmostStar", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 2},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	for _, r := range results {
		if r.Event == "exceptional_performance" {
			t.Error("exceptional_performance should NOT trigger with 5 submitted days")
		}
	}
}

func TestTrigger_ConsecutiveMiss_ExactBoundary(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Boundary", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 3},
		submittedDays: map[string]int{"e1": 4},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	triggered := false
	for _, r := range results {
		if r.Event == "consecutive_miss_3days" {
			triggered = true
		}
	}
	if !triggered {
		t.Error("expected trigger with exactly 3 missed days (boundary)")
	}
}

func TestTrigger_ConsecutiveMiss_BelowBoundary(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "AlmostMiss", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 2},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	for _, r := range results {
		if r.Event == "consecutive_miss_3days" {
			t.Error("should NOT trigger with 2 missed days")
		}
	}
}

func TestTrigger_NameTemplateReplacement(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "TemplateTest", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 5},
		submittedDays: map[string]int{"e1": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one trigger result")
	}

	for _, r := range results {
		if strings.Contains(r.Message, "{name}") {
			t.Error("message should have {name} replaced with employee name")
		}
		if !strings.Contains(r.Message, "TemplateTest") {
			t.Errorf("message should contain employee name, got: %q", r.Message)
		}
	}
}

func TestTrigger_PublicRecognitionAction(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "StarPerformer", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 0},
		submittedDays: map[string]int{"e1": 7},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	recognitionTriggered := false
	for _, r := range results {
		if r.Action == "public_recognition" {
			recognitionTriggered = true
		}
	}
	if !recognitionTriggered {
		t.Error("expected public_recognition action for exceptional_performance")
	}

	recognitionSent := false
	for _, msg := range sender.sentMessages {
		if msg.ChatID == 999 && strings.Contains(msg.Message, "Recognition") {
			recognitionSent = true
		}
	}
	if !recognitionSent {
		t.Error("expected Recognition message sent to boss")
	}
}

func TestTrigger_PrivateCheckinAction(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "NeedsHelp", TelegramID: 555, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 5},
		submittedDays: map[string]int{"e1": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ma", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	privateCheckinTriggered := false
	for _, r := range results {
		if r.Action == "private_checkin" {
			privateCheckinTriggered = true
		}
	}
	if !privateCheckinTriggered {
		t.Error("expected private_checkin action from ma mentor")
	}

	sentToEmployee := false
	for _, msg := range sender.sentMessages {
		if msg.ChatID == 555 {
			sentToEmployee = true
		}
	}
	if !sentToEmployee {
		t.Error("expected private_checkin message sent to employee directly")
	}
}

func TestTrigger_SuggestOneOnOneAction(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Declining", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4},
		submittedDays: map[string]int{"e1": 3},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	suggestTriggered := false
	for _, r := range results {
		if r.Action == "suggest_one_on_one" {
			suggestTriggered = true
		}
	}
	if !suggestTriggered {
		t.Error("expected suggest_one_on_one action for output_decline_3days")
	}

	bossMsgSent := false
	for _, msg := range sender.sentMessages {
		if msg.ChatID == 999 && strings.Contains(msg.Message, "Trigger Alert") {
			bossMsgSent = true
		}
	}
	if !bossMsgSent {
		t.Error("expected Trigger Alert message to boss for suggest_one_on_one action")
	}
}

func TestTrigger_PerformanceWarningAction(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "LowOutput", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4},
		submittedDays: map[string]int{"e1": 3},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	warningTriggered := false
	for _, r := range results {
		if r.Action == "performance_warning" {
			warningTriggered = true
		}
	}
	if !warningTriggered {
		t.Error("expected performance_warning action for consecutive_low_output")
	}

	bossMsgSent := false
	for _, msg := range sender.sentMessages {
		if msg.ChatID == 999 {
			bossMsgSent = true
		}
	}
	if !bossMsgSent {
		t.Error("expected boss notification for performance_warning")
	}
}

func TestTrigger_CheckAll_InvalidMentor(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	_, err := checker.CheckAll(context.Background(), "tenant-1", "nonexistent_mentor_xyz", 999)
	if err == nil {
		t.Error("expected error for invalid mentor ID")
	}
}

func TestTrigger_CheckAll_DBErrorOnListEmployees(t *testing.T) {
	db := &mockTriggerDB{
		listErr: errors.New("db connection failed"),
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	_, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err == nil {
		t.Error("expected error when DB fails to list employees")
	}
	if !strings.Contains(err.Error(), "list employees") {
		t.Errorf("error should mention list employees, got: %v", err)
	}
}

func TestTrigger_CheckAll_DBErrorOnEventCheck(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "ErrorEmp", TelegramID: 111, CultureCode: "default"},
		},
		missedDays: map[string]int{},
		missedDaysErr: map[string]error{
			"e1": errors.New("query timeout"),
		},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("should not return top-level error for per-employee DB failure: %v", err)
	}

	for _, r := range results {
		if r.EmployeeName == "ErrorEmp" && r.Event == "consecutive_miss_3days" {
			t.Error("should not trigger for employee with DB error")
		}
	}
}

func TestTrigger_CheckAll_NoEmployees(t *testing.T) {
	db := &mockTriggerDB{
		employees:     []report.EmployeeInfo{},
		missedDays:    map[string]int{},
		submittedDays: map[string]int{},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results with no employees, got %d", len(results))
	}
	if len(sender.sentMessages) != 0 {
		t.Error("should not send messages with no employees")
	}
}

func TestTrigger_MultipleTriggersPerEmployee(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "MultiTrigger", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 5},
		submittedDays: map[string]int{"e1": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	lowOutputCount := 0
	exceptionalCount := 0
	for _, r := range results {
		if r.Event == "consecutive_low_output" {
			lowOutputCount++
		}
		if r.Event == "exceptional_performance" {
			exceptionalCount++
		}
	}
	if lowOutputCount != 1 {
		t.Errorf("expected 1 consecutive_low_output trigger, got %d", lowOutputCount)
	}
	if exceptionalCount != 0 {
		t.Errorf("expected 0 exceptional_performance triggers, got %d", exceptionalCount)
	}
}

func TestTrigger_MultipleEmployees_MultipleResults(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
			{ID: "e2", Name: "Bob", TelegramID: 222, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4, "e2": 5},
		submittedDays: map[string]int{"e1": 3, "e2": 2},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	aliceTriggered := false
	bobTriggered := false
	for _, r := range results {
		if r.EmployeeName == "Alice" && r.Event == "consecutive_miss_3days" {
			aliceTriggered = true
		}
		if r.EmployeeName == "Bob" && r.Event == "consecutive_miss_3days" {
			bobTriggered = true
		}
	}
	if !aliceTriggered || !bobTriggered {
		t.Errorf("expected both employees to trigger: Alice=%v, Bob=%v", aliceTriggered, bobTriggered)
	}

	if len(sender.sentMessages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(sender.sentMessages))
	}
}

func TestTrigger_SubmittedDaysError_SkipsTrigger(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "ErrorStar", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 0},
		submittedDays: map[string]int{},
		submittedErr: map[string]error{
			"e1": errors.New("db read error"),
		},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("should not return top-level error: %v", err)
	}

	for _, r := range results {
		if r.Event == "exceptional_performance" {
			t.Error("should not trigger exceptional_performance with DB error")
		}
	}
}

func TestTrigger_TriggerResult_Fields(t *testing.T) {
	db := &mockTriggerDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
		},
		missedDays:    map[string]int{"e1": 4},
		submittedDays: map[string]int{"e1": 3},
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	checker := report.NewTriggerChecker(db, sender, factory)
	results, err := checker.CheckAll(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	r := results[0]
	if r.EmployeeID != "e1" {
		t.Errorf("expected EmployeeID e1, got %q", r.EmployeeID)
	}
	if r.EmployeeName != "Alice" {
		t.Errorf("expected EmployeeName Alice, got %q", r.EmployeeName)
	}
	if r.Event != "consecutive_miss_3days" {
		t.Errorf("expected Event consecutive_miss_3days, got %q", r.Event)
	}
	if r.Action != "manager_private_chat" {
		t.Errorf("expected Action manager_private_chat, got %q", r.Action)
	}
	if r.Message == "" {
		t.Error("expected non-empty Message")
	}
}
