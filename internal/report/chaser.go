package report

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/events"
)

// EmployeeInfo holds basic info about an employee for chase operations.
type EmployeeInfo struct {
	ID               string
	Name             string
	TelegramID       int64
	SignalPhone      string
	SlackID          string
	LarkID           string
	PreferredChannel string // default "telegram"
	CultureCode      string
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

// EventBus sends events for chase operations.
type EventBus interface {
	PublishPayload(ctx context.Context, eventType events.EventType, tenantID string, payload any) error
}

// Chaser handles chasing employees who haven't submitted reports.
type Chaser struct {
	db       ChaserDB
	llm      *brain.LLMService
	sender   channel.Sender
	factory  *brain.EngineFactory
	eventBus EventBus
}

// NewChaser creates a new chaser with an EngineFactory for per-employee culture support.
func NewChaser(db ChaserDB, llm *brain.LLMService, sender channel.Sender, factory *brain.EngineFactory) *Chaser {
	return &Chaser{db: db, llm: llm, sender: sender, factory: factory, eventBus: nil}
}

// SetEventBus sets the event bus for emitting chase events.
func (c *Chaser) SetEventBus(bus EventBus) {
	c.eventBus = bus
}

// ChaseAll chases all employees without reports for the given date.
// mentorID is the tenant's active mentor; each employee's culture is used individually.
func (c *Chaser) ChaseAll(ctx context.Context, tenantID, date, mentorID string) error {
	employees, err := c.db.ListEmployeesWithoutReport(ctx, tenantID, date)
	if err != nil {
		return fmt.Errorf("list employees without report: %w", err)
	}

	for _, emp := range employees {
		// Resolve the employee's channel
		chType, chID := resolveEmployeeChannel(emp)
		if chType == "" {
			slog.Warn("employee has no channel configured", "employee", emp.Name)
			continue
		}

		// Create engine per employee culture
		engine, err := c.factory.ForTenant(mentorID, emp.CultureCode)
		if err != nil {
			slog.Error("create engine for chase", "employee_id", emp.ID, "mentor", mentorID, "culture", emp.CultureCode, "error", err)
			continue
		}

		lastStep, err := c.db.GetLastChaseStep(ctx, emp.ID, date)
		if err != nil {
			slog.Error("get last chase step", "employee_id", emp.ID, "error", err)
			continue
		}

		nextStep := lastStep + 1
		step := engine.GetEffectiveChaseStep(nextStep)

		if step.Action == "skip_today" {
			slog.Info("skip chase (max steps reached)", "employee_id", emp.ID)
			continue
		}

		// Generate message via LLM (if available) or use template fallback
		var msg string
		if c.llm != nil {
			systemPrompt := engine.BuildSystemPrompt()
			msg, err = c.llm.GenerateChaseMessage(ctx, systemPrompt, emp.Name, step.Tone)
			if err != nil {
				slog.Warn("LLM failed, using fallback", "error", err, "employee", emp.Name)
				msg = fmt.Sprintf("Hi %s, this is a reminder to submit your daily report.", emp.Name)
			}
		} else {
			msg = fmt.Sprintf("Hi %s, this is a reminder to submit your daily report.", emp.Name)
		}

		// Send message via channel-agnostic sender
		if err := c.sender.Send(ctx, chType, chID, msg); err != nil {
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

		// TODO: emit ChaseCompleted event from chase handler
		// Requires modifying CreateChaseLog to return the created log ID
	}

	return nil
}

// resolveEmployeeChannel resolves the preferred channel type and user ID for an EmployeeInfo.
// This is used by chaser, triggers, actions, and alerts to determine where to send messages.
func resolveEmployeeChannel(emp EmployeeInfo) (channel.Type, string) {
	return channel.ResolveChannel(channel.ResolveEmployee{
		TelegramID:       toPgInt8(emp.TelegramID),
		SignalPhone:      toPgText(emp.SignalPhone),
		SlackID:          toPgText(emp.SlackID),
		LarkID:           toPgText(emp.LarkID),
		PreferredChannel: emp.PreferredChannel,
	})
}

// toPgInt8 converts an int64 to pgtype.Int8 (valid if non-zero).
func toPgInt8(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: v != 0}
}

// toPgText converts a string to pgtype.Text (valid if non-empty).
func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
