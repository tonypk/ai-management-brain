// Package service provides business logic services that bridge API handlers
// to internal subsystems (collector, chaser, summarizer, channel sender).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// CheckinResult holds the outcome of a triggered check-in.
type CheckinResult struct {
	SentTo  []string `json:"sent_to"`
	Skipped []string `json:"skipped"`
}

// ChaseResult holds the outcome of a triggered chase.
type ChaseResult struct {
	Chased  []string `json:"chased"`
	Skipped []string `json:"skipped"`
}

// SummaryResult holds the outcome of a triggered summary.
type SummaryResult struct {
	Summary        string  `json:"summary"`
	SubmissionRate float64 `json:"submission_rate"`
	SentTo         string  `json:"sent_to"`
}

// MessageResult holds the outcome of sending an arbitrary message.
type MessageResult struct {
	SentTo  string `json:"sent_to"`
	Channel string `json:"channel"`
}

// ActionService exposes write operations for the OpenClaw MCP layer.
type ActionService struct {
	queries    *sqlc.Queries
	collector  *report.Collector
	chaser     *report.Chaser
	summarizer *report.Summarizer
	sender     channel.Sender
	factory    *brain.EngineFactory
	reportDB   *report.DBAdapter
	timezone   *time.Location
}

// ActionServiceConfig holds dependencies for ActionService.
type ActionServiceConfig struct {
	Queries    *sqlc.Queries
	Collector  *report.Collector
	Chaser     *report.Chaser
	Summarizer *report.Summarizer
	Sender     channel.Sender
	Factory    *brain.EngineFactory
	ReportDB   *report.DBAdapter
	Timezone   *time.Location
}

// NewActionService creates a new ActionService with all dependencies.
func NewActionService(cfg ActionServiceConfig) *ActionService {
	return &ActionService{
		queries:    cfg.Queries,
		collector:  cfg.Collector,
		chaser:     cfg.Chaser,
		summarizer: cfg.Summarizer,
		sender:     cfg.Sender,
		factory:    cfg.Factory,
		reportDB:   cfg.ReportDB,
		timezone:   cfg.Timezone,
	}
}

// TriggerCheckin sends check-in questions to all or a specific employee.
// If employeeName is empty, it sends to all active employees who haven't submitted today.
func (s *ActionService) TriggerCheckin(ctx context.Context, tenantID pgtype.UUID, employeeName string) (*CheckinResult, error) {
	tenant, err := s.queries.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	engine, err := s.factory.ForTenant(tenant.MentorID, "default")
	if err != nil {
		return nil, fmt.Errorf("load engine: %w", err)
	}
	questions := engine.GetCheckinQuestions()

	employees, err := s.queries.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	today := pgtype.Date{Time: time.Now().In(s.timezone).Truncate(24 * time.Hour), Valid: true}

	result := &CheckinResult{}

	for _, emp := range employees {
		// Filter by name if specified
		if employeeName != "" && !fuzzyNameMatch(emp.Name, employeeName) {
			continue
		}

		// Check if already submitted today
		count, err := s.queries.CountReportsByEmployeeDate(ctx, sqlc.CountReportsByEmployeeDateParams{
			EmployeeID: emp.ID,
			ReportDate: today,
		})
		if err == nil && count > 0 {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (already submitted)", emp.Name))
			continue
		}

		// Collector works with employee ID string
		empID := formatUUID(emp.ID)
		_, firstQ, err := s.collector.StartWithQuestions(ctx, empID, questions)
		if err != nil {
			slog.Error("action: start checkin", "employee", emp.Name, "error", err)
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (error: %s)", emp.Name, err.Error()))
			continue
		}

		// Send via channel
		chType, chID := resolveChannel(emp)
		if chType == "" {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (no channel)", emp.Name))
			continue
		}

		msg := fmt.Sprintf("Good morning %s! Time for your daily check-in.\n\n%s", emp.Name, firstQ)
		if err := s.sender.Send(ctx, chType, chID, msg); err != nil {
			slog.Error("action: send checkin", "employee", emp.Name, "error", err)
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (send failed)", emp.Name))
			continue
		}

		result.SentTo = append(result.SentTo, emp.Name)
	}

	if employeeName != "" && len(result.SentTo) == 0 && len(result.Skipped) == 0 {
		return nil, fmt.Errorf("no employee found matching '%s'", employeeName)
	}

	return result, nil
}

// TriggerChase chases employees who haven't submitted today's report.
// If employeeName is empty, it chases all non-submitters.
func (s *ActionService) TriggerChase(ctx context.Context, tenantID pgtype.UUID, employeeName string) (*ChaseResult, error) {
	tenant, err := s.queries.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	today := time.Now().In(s.timezone).Format("2006-01-02")
	tenantIDStr := formatUUID(tenantID)

	if employeeName == "" {
		// Chase all — delegate to existing chaser
		if err := s.chaser.ChaseAll(ctx, tenantIDStr, today, tenant.MentorID); err != nil {
			return nil, fmt.Errorf("chase all: %w", err)
		}

		// Build result from employees without report
		emps, _ := s.reportDB.ListEmployeesWithoutReport(ctx, tenantIDStr, today)
		result := &ChaseResult{}
		for _, emp := range emps {
			result.Chased = append(result.Chased, emp.Name)
		}
		return result, nil
	}

	// Chase specific employee
	employees, err := s.queries.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	todayDate := pgtype.Date{Time: time.Now().In(s.timezone).Truncate(24 * time.Hour), Valid: true}
	result := &ChaseResult{}

	for _, emp := range employees {
		if !fuzzyNameMatch(emp.Name, employeeName) {
			continue
		}

		// Check if already submitted
		count, _ := s.queries.CountReportsByEmployeeDate(ctx, sqlc.CountReportsByEmployeeDateParams{
			EmployeeID: emp.ID,
			ReportDate: todayDate,
		})
		if count > 0 {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (already submitted)", emp.Name))
			continue
		}

		chType, chID := resolveChannel(emp)
		if chType == "" {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (no channel)", emp.Name))
			continue
		}

		msg := fmt.Sprintf("Hi %s, this is a reminder to submit your daily report.", emp.Name)
		if err := s.sender.Send(ctx, chType, chID, msg); err != nil {
			slog.Error("action: chase send", "employee", emp.Name, "error", err)
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (send failed)", emp.Name))
			continue
		}

		result.Chased = append(result.Chased, emp.Name)
	}

	if len(result.Chased) == 0 && len(result.Skipped) == 0 {
		return nil, fmt.Errorf("no employee found matching '%s'", employeeName)
	}

	return result, nil
}

// TriggerSummary generates today's team summary and sends it to the boss.
func (s *ActionService) TriggerSummary(ctx context.Context, tenantID pgtype.UUID) (*SummaryResult, error) {
	tenant, err := s.queries.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	engine, err := s.factory.ForTenant(tenant.MentorID, "default")
	if err != nil {
		return nil, fmt.Errorf("load engine: %w", err)
	}

	today := time.Now().In(s.timezone).Format("2006-01-02")
	tenantIDStr := formatUUID(tenantID)

	summaryResult, err := s.summarizer.Generate(ctx, tenantIDStr, today, engine)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
	}

	// Send to boss via channel sender
	header := fmt.Sprintf("Daily Summary (%s)\nMentor: %s\nSubmission rate: %.0f%%\n\n",
		today, tenant.MentorID, summaryResult.SubmissionRate*100)
	fullMsg := header + summaryResult.Content

	bossID := fmt.Sprintf("%d", tenant.BossChatID)
	if err := s.sender.Send(ctx, channel.TypeTelegram, bossID, fullMsg); err != nil {
		slog.Error("action: send summary to boss", "error", err)
	}

	return &SummaryResult{
		Summary:        summaryResult.Content,
		SubmissionRate: summaryResult.SubmissionRate,
		SentTo:         "boss",
	}, nil
}

// SendMessage sends an arbitrary message to an employee via their preferred channel.
func (s *ActionService) SendMessage(ctx context.Context, tenantID pgtype.UUID, employeeName, message string) (*MessageResult, error) {
	emp, err := s.queries.GetEmployeeByNameFuzzy(ctx, sqlc.GetEmployeeByNameFuzzyParams{
		TenantID: tenantID,
		Column2:  pgtype.Text{String: employeeName, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("no employee found matching '%s'", employeeName)
	}

	chType, chID := resolveChannel(emp)
	if chType == "" {
		return nil, fmt.Errorf("employee '%s' has no messaging channel configured", emp.Name)
	}

	if err := s.sender.Send(ctx, chType, chID, message); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return &MessageResult{
		SentTo:  emp.Name,
		Channel: string(chType),
	}, nil
}

// fuzzyNameMatch performs case-insensitive substring matching.
func fuzzyNameMatch(fullName, query string) bool {
	return strings.Contains(strings.ToLower(fullName), strings.ToLower(strings.TrimSpace(query)))
}

// formatUUID converts pgtype.UUID to string.
func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// resolveChannel resolves channel type and ID from a sqlc.Employee.
func resolveChannel(emp sqlc.Employee) (channel.Type, string) {
	return channel.ResolveChannel(channel.ResolveEmployee{
		TelegramID:       emp.TelegramID,
		SignalPhone:      emp.SignalPhone,
		SlackID:          emp.SlackID,
		LarkID:           emp.LarkID,
		PreferredChannel: emp.PreferredChannel,
	})
}
