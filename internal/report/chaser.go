package report

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// EmployeeInfo holds basic info about an employee for chase operations.
type EmployeeInfo struct {
	ID          string
	Name        string
	TelegramID  int64
	CultureCode string
}

// ChaseLogEntry holds data for one chase log record.
type ChaseLogEntry struct {
	TenantID   string
	EmployeeID string
	ReportDate string
	Step       int
	Action     string
	Message    string
}

// ChaserDB defines the database operations needed by the chaser.
type ChaserDB interface {
	ListEmployeesWithoutReport(ctx context.Context, tenantID, date string) ([]EmployeeInfo, error)
	GetLastChaseStep(ctx context.Context, employeeID, date string) (int, error)
	CreateChaseLog(ctx context.Context, entry ChaseLogEntry) error
}

// MessageSender sends messages to users.
type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

// Chaser handles chasing employees who haven't submitted reports.
type Chaser struct {
	db     ChaserDB
	llm    *brain.LLMService
	sender MessageSender
	engine *brain.Engine
}

// NewChaser creates a new chaser.
func NewChaser(db ChaserDB, llm *brain.LLMService, sender MessageSender, engine *brain.Engine) *Chaser {
	return &Chaser{db: db, llm: llm, sender: sender, engine: engine}
}

// ChaseAll chases all employees without reports for the given date.
func (c *Chaser) ChaseAll(ctx context.Context, tenantID, date string) error {
	employees, err := c.db.ListEmployeesWithoutReport(ctx, tenantID, date)
	if err != nil {
		return fmt.Errorf("list employees without report: %w", err)
	}

	for _, emp := range employees {
		lastStep, err := c.db.GetLastChaseStep(ctx, emp.ID, date)
		if err != nil {
			slog.Error("get last chase step", "employee_id", emp.ID, "error", err)
			continue
		}

		nextStep := lastStep + 1
		step := c.engine.GetEffectiveChaseStep(nextStep)

		if step.Action == "skip_today" {
			slog.Info("skip chase (max steps reached)", "employee_id", emp.ID)
			continue
		}

		// Build system prompt from engine
		systemPrompt := c.engine.BuildSystemPrompt()

		// Generate message via LLM
		msg, err := c.llm.GenerateChaseMessage(ctx, systemPrompt, emp.Name, step.Tone)
		if err != nil {
			// Fallback to template
			slog.Warn("LLM failed, using fallback", "error", err, "employee", emp.Name)
			msg = fmt.Sprintf("Hi %s, this is a reminder to submit your daily report.", emp.Name)
		}

		// Send message
		if err := c.sender.SendMessage(emp.TelegramID, msg); err != nil {
			slog.Error("send chase message", "employee_id", emp.ID, "error", err)
			continue
		}

		// Log chase
		if err := c.db.CreateChaseLog(ctx, ChaseLogEntry{
			TenantID:   tenantID,
			EmployeeID: emp.ID,
			ReportDate: date,
			Step:       nextStep,
			Action:     step.Action,
			Message:    msg,
		}); err != nil {
			slog.Error("create chase log", "employee_id", emp.ID, "error", err)
		}
	}

	return nil
}
