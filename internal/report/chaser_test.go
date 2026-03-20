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
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "inamori")
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
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "inamori")

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
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "dalio")

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
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "inamori")

	if len(sender.sentMessages) != 1 {
		t.Errorf("should send fallback message, got %d messages", len(sender.sentMessages))
	}
}

func TestChaser_PerEmployeeCulture(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "philippines"},
			{ID: "e2", Name: "Budi", TelegramID: 222, CultureCode: "indonesia"},
			{ID: "e3", Name: "Kumar", TelegramID: 333, CultureCode: "srilanka"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{response: "chase msg"}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "dalio")
	if err != nil {
		t.Fatalf("chase: %v", err)
	}

	// All 3 should get private_message due to culture override (all cultures have NeverNameInGroup)
	if len(sender.sentMessages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(sender.sentMessages))
	}
	for _, log := range db.createdLogs {
		if log.Action != "private_message" {
			t.Errorf("expected private_message for all cultures, got %q for %s", log.Action, log.EmployeeID)
		}
	}
}

func TestChaser_NoEmployeesWithoutReport(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: nil,
		lastChaseStep:          0,
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, nil, sender, factory)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "inamori")
	if err != nil {
		t.Fatalf("chase: %v", err)
	}
	if len(sender.sentMessages) != 0 {
		t.Errorf("expected 0 messages with no employees, got %d", len(sender.sentMessages))
	}
}

func TestChaser_NilLLM_UsesFallback(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "default"},
		},
		lastChaseStep: 0,
	}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	// nil LLM → should use fallback message
	chaser := report.NewChaser(db, nil, sender, factory)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20", "inamori")
	if err != nil {
		t.Fatalf("chase: %v", err)
	}

	if len(sender.sentMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sender.sentMessages))
	}
	msg := sender.sentMessages[0].Message
	if msg == "" {
		t.Error("expected non-empty fallback message")
	}
}

func TestChaser_ChaseLogContainsCorrectFields(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Bob", TelegramID: 222, CultureCode: "default"},
		},
		lastChaseStep: 1, // step 1 already done
	}
	llm := &mockLLM{response: "step 2 reminder"}
	sender := &mockSender{}
	factory := brain.NewEngineFactory()

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, factory)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-21", "grove")

	if len(db.createdLogs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(db.createdLogs))
	}
	log := db.createdLogs[0]
	if log.TenantID != "tenant-1" {
		t.Errorf("log.TenantID = %q", log.TenantID)
	}
	if log.EmployeeID != "e1" {
		t.Errorf("log.EmployeeID = %q", log.EmployeeID)
	}
	if log.ReportDate != "2026-03-21" {
		t.Errorf("log.ReportDate = %q", log.ReportDate)
	}
	if log.Step != 2 {
		t.Errorf("log.Step = %d, want 2", log.Step)
	}
}
