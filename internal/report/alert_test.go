package report_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/report"
)

// mockAlertDB implements report.AlertDB for testing.
type mockAlertDB struct {
	employees       []report.EmployeeInfo
	missedDays      map[string]int
	sentiments      map[string][]string
	missedDaysErr   error
	sentimentsErr   error
}

func (m *mockAlertDB) ListActiveEmployeesWithTelegram(ctx context.Context, tenantID string) ([]report.EmployeeInfo, error) {
	return m.employees, nil
}

func (m *mockAlertDB) GetConsecutiveMissDays(ctx context.Context, employeeID string) (int, error) {
	if m.missedDaysErr != nil {
		return 0, m.missedDaysErr
	}
	return m.missedDays[employeeID], nil
}

func (m *mockAlertDB) GetRecentSentiments(ctx context.Context, employeeID string, days int) ([]string, error) {
	if m.sentimentsErr != nil {
		return nil, m.sentimentsErr
	}
	return m.sentiments[employeeID], nil
}

// mockAlertSender implements report.AlertSender for testing.
type mockAlertSender struct {
	messages []sentMessage
}

func (m *mockAlertSender) SendMessage(chatID int64, text string) error {
	m.messages = append(m.messages, sentMessage{chatID, text})
	return nil
}

func TestAlertChecker_ConsecutiveMiss_Warning(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
		},
		missedDays: map[string]int{"e1": 3},
		sentiments: map[string][]string{"e1": {"positive", "positive"}},
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].AlertType != "consecutive_miss" {
		t.Errorf("alert type = %q, want consecutive_miss", alerts[0].AlertType)
	}
	if alerts[0].Severity != "warning" {
		t.Errorf("severity = %q, want warning", alerts[0].Severity)
	}
	if len(sender.messages) != 1 {
		t.Errorf("expected 1 message sent to boss, got %d", len(sender.messages))
	}
}

func TestAlertChecker_ConsecutiveMiss_Critical(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Bob", TelegramID: 222},
		},
		missedDays: map[string]int{"e1": 5},
		sentiments: map[string][]string{"e1": {}},
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", alerts[0].Severity)
	}
}

func TestAlertChecker_SentimentDrop(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Carol", TelegramID: 333},
		},
		missedDays: map[string]int{"e1": 0},
		sentiments: map[string][]string{"e1": {"negative", "negative", "negative", "positive"}},
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].AlertType != "sentiment_drop" {
		t.Errorf("alert type = %q, want sentiment_drop", alerts[0].AlertType)
	}
}

func TestAlertChecker_NoAlerts_HealthyEmployee(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Dan", TelegramID: 444},
		},
		missedDays: map[string]int{"e1": 1},
		sentiments: map[string][]string{"e1": {"positive", "neutral", "positive"}},
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for healthy employee, got %d", len(alerts))
	}
	if len(sender.messages) != 0 {
		t.Errorf("expected no messages sent, got %d", len(sender.messages))
	}
}

func TestAlertChecker_MultipleEmployees_MultipleAlerts(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
			{ID: "e3", Name: "Carol", TelegramID: 333},
		},
		missedDays: map[string]int{"e1": 4, "e2": 0, "e3": 6},
		sentiments: map[string][]string{
			"e1": {"positive"},
			"e2": {"negative", "negative", "negative"},
			"e3": {},
		},
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	// e1: consecutive_miss (4 days, warning)
	// e2: sentiment_drop (3 negative)
	// e3: consecutive_miss (6 days, critical)
	if len(alerts) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(alerts))
	}
	if len(sender.messages) != 1 {
		t.Errorf("expected 1 aggregated message to boss, got %d", len(sender.messages))
	}
}

func TestHasSentimentDrop_InsufficientData(t *testing.T) {
	db := &mockAlertDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Eve", TelegramID: 555},
		},
		missedDays: map[string]int{"e1": 0},
		sentiments: map[string][]string{"e1": {"negative", "negative"}}, // only 2, need 3+
	}
	sender := &mockAlertSender{}
	checker := report.NewAlertChecker(db, sender)

	alerts, err := checker.CheckAll(context.Background(), "t1", 999)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts with insufficient sentiment data, got %d", len(alerts))
	}
}
