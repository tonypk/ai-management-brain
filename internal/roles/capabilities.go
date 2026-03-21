package roles

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// capabilityRunner dispatches capability execution for a role agent.
type capabilityRunner struct {
	agent *RoleAgent
	deps  *AgentDeps
}

func newCapabilityRunner(agent *RoleAgent, deps *AgentDeps) *capabilityRunner {
	return &capabilityRunner{agent: agent, deps: deps}
}

// Run dispatches to the named capability implementation.
func (r *capabilityRunner) Run(ctx context.Context, name string) error {
	switch name {
	case "daily_status_check":
		return r.dailyStatusCheck(ctx)
	case "chase_missing_reports":
		return r.chaseMissingReports(ctx)
	case "weekly_summary":
		return r.weeklySummary(ctx)
	case "detect_anomalies":
		return r.detectAnomalies(ctx)
	default:
		return fmt.Errorf("unknown capability: %s", name)
	}
}

// dailyStatusCheck generates a summary and adds a COO perspective.
func (r *capabilityRunner) dailyStatusCheck(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")

	result, err := r.deps.Summarizer.Generate(ctx, r.agent.TenantID, today, r.agent.engine)
	if err != nil {
		return fmt.Errorf("generate summary: %w", err)
	}

	// Build COO commentary via LLM
	commentary := r.generateCommentary(ctx, "daily status check",
		fmt.Sprintf("Submission rate: %.0f%%, Blockers: %d\n\n%s",
			result.SubmissionRate*100, result.BlockersCount, result.Content))

	msg := r.agent.Brand(fmt.Sprintf("Daily Status (%s)\nSubmission rate: %.0f%%\n\n%s",
		today, result.SubmissionRate*100, commentary))

	return r.deps.Sender.SendToBoss(msg)
}

// chaseMissingReports runs the chaser and sends a branded summary.
func (r *capabilityRunner) chaseMissingReports(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")

	if err := r.deps.Chaser.ChaseAll(ctx, r.agent.TenantID, today, r.agent.MentorID); err != nil {
		return fmt.Errorf("chase all: %w", err)
	}

	msg := r.agent.Brand(fmt.Sprintf("Chase report complete for %s. Non-submitters have been reminded.", today))
	return r.deps.Sender.SendToBoss(msg)
}

// weeklySummary generates a weekly operations insight.
func (r *capabilityRunner) weeklySummary(ctx context.Context) error {
	if err := r.deps.ActionExec.RunWeekly(ctx, r.agent.TenantID, r.agent.MentorID, r.deps.Sender.bossChatID); err != nil {
		return fmt.Errorf("run weekly: %w", err)
	}

	commentary := r.generateCommentary(ctx, "weekly operations review",
		"Generate a brief weekly operations summary highlighting team health, process improvements needed, and key metrics trends.")

	msg := r.agent.Brand(fmt.Sprintf("Weekly Operations Review\n\n%s", commentary))
	return r.deps.Sender.SendToBoss(msg)
}

// detectAnomalies handles alert.fired events by creating suggestions for critical issues.
func (r *capabilityRunner) detectAnomalies(ctx context.Context) error {
	alerts, err := r.deps.AlertChecker.CheckAll(ctx, r.agent.TenantID, r.deps.Sender.bossChatID)
	if err != nil {
		return fmt.Errorf("check alerts: %w", err)
	}

	// Only create suggestions for critical alerts
	for _, alert := range alerts {
		if alert.Severity != "critical" {
			continue
		}

		suggestion := r.generateCommentary(ctx, "anomaly response",
			fmt.Sprintf("Critical alert: %s\nEmployee: %s\nType: %s\n\nProvide a specific, actionable recommendation for handling this situation.",
				alert.Message, alert.EmployeeName, alert.AlertType))

		if err := r.deps.Queries.CreateAISuggestion(ctx, CreateSuggestionParams{
			TenantID:    r.agent.TenantID,
			RoleID:      r.agent.RoleID,
			RoleTitle:   r.agent.Title,
			Capability:  "detect_anomalies",
			Title:       fmt.Sprintf("Action needed: %s — %s", alert.AlertType, alert.EmployeeName),
			Content:     suggestion,
			ContextData: []byte("{}"),
		}); err != nil {
			slog.Error("create ai suggestion", "role", r.agent.RoleID, "error", err)
		}
	}

	return nil
}

// generateCommentary uses LLM to generate role-specific commentary, with fallback.
func (r *capabilityRunner) generateCommentary(ctx context.Context, task, context string) string {
	if r.deps.LLM == nil {
		return context
	}

	prompt := fmt.Sprintf("Task: %s\nContext:\n%s\n\nProvide a brief, actionable commentary (3-5 sentences).", task, context)
	result, err := r.deps.LLM.Chat(ctx, r.agent.SystemPrompt(), prompt)
	if err != nil {
		slog.Warn("LLM commentary failed, using context", "role", r.agent.RoleID, "error", err)
		return context
	}
	return result
}
