package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// mockLLM implements brain.LLMClient for testing
type mockLLM struct {
	response string
	err      error
	calls    int
}

func (m *mockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	m.calls++
	return m.response, m.err
}

// mockChaserDB implements the DB interface for chaser
type mockChaserDB struct {
	employeesWithoutReport []report.EmployeeInfo
	lastChaseStep          int
	createdLogs            []report.ChaseLogEntry
}

func (m *mockChaserDB) ListEmployeesWithoutReport(ctx context.Context, tenantID string, date string) ([]report.EmployeeInfo, error) {
	return m.employeesWithoutReport, nil
}

func (m *mockChaserDB) GetLastChaseStep(ctx context.Context, employeeID string, date string) (int, error) {
	return m.lastChaseStep, nil
}

func (m *mockChaserDB) CreateChaseLog(ctx context.Context, entry report.ChaseLogEntry) error {
	m.createdLogs = append(m.createdLogs, entry)
	return nil
}

// mockSender implements the MessageSender interface
type mockSender struct {
	sentMessages []sentMessage
}

type sentMessage struct {
	ChatID  int64
	Message string
}

func (m *mockSender) SendMessage(chatID int64, text string) error {
	m.sentMessages = append(m.sentMessages, sentMessage{chatID, text})
	return nil
}

func TestChaser_ChasesEmployeesWithoutReport(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "philippines"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{response: "Hi Alice, gentle reminder!"}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "philippines")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("chase: %v", err)
	}
	if len(sender.sentMessages) != 1 {
		t.Errorf("expected 1 message sent, got %d", len(sender.sentMessages))
	}
	if len(db.createdLogs) != 1 {
		t.Errorf("expected 1 chase log, got %d", len(db.createdLogs))
	}
}

func TestChaser_SkipsTodayWhenMaxSteps(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Bob", TelegramID: 222, CultureCode: "singapore"},
		},
		lastChaseStep: 99,
	}
	llm := &mockLLM{response: "reminder"}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "singapore")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	if len(sender.sentMessages) != 0 {
		t.Error("should not send message when skip_today")
	}
}

func TestChaser_CultureOverrideApplied(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Carlos", TelegramID: 333, CultureCode: "philippines"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{response: "reminder"}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("dalio", "philippines")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	if len(db.createdLogs) != 1 {
		t.Fatal("expected 1 log")
	}
	if db.createdLogs[0].Action != "private_message" {
		t.Errorf("action should be private_message (culture override), got %q", db.createdLogs[0].Action)
	}
}

func TestChaser_LLMFallback(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Dan", TelegramID: 444, CultureCode: "default"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{err: errors.New("api down")}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "default")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	if len(sender.sentMessages) != 1 {
		t.Errorf("should send fallback message, got %d messages", len(sender.sentMessages))
	}
}
