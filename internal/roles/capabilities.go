package roles

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

func init() {
	ActionRegistry["chase_missing"] = executeChaseMissing
	ActionRegistry["daily_summary"] = executeDailySummary
	ActionRegistry["weekly_summary"] = executeWeeklySummary
	ActionRegistry["check_alerts"] = executeCheckAlerts
	ActionRegistry["create_suggestion"] = executeCreateSuggestion
	ActionRegistry["send_branded_msg"] = executeSendBrandedMsg
}

// executeDailySummary generates a summary and adds role-specific commentary.
func executeDailySummary(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	today := time.Now().Format("2006-01-02")

	result, err := deps.Summarizer.Generate(ctx, agent.TenantID, today, agent.engine)
	if err != nil {
		return fmt.Errorf("generate summary: %w", err)
	}

	commentary := generateCommentary(ctx, agent, deps, "daily status check",
		fmt.Sprintf("Submission rate: %.0f%%, Blockers: %d\n\n%s",
			result.SubmissionRate*100, result.BlockersCount, result.Content))

	msg := agent.Brand(fmt.Sprintf("Daily Status (%s)\nSubmission rate: %.0f%%\n\n%s",
		today, result.SubmissionRate*100, commentary))

	return deps.Sender.SendToBoss(msg)
}

// executeChaseMissing runs the chaser and sends a branded summary.
func executeChaseMissing(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	today := time.Now().Format("2006-01-02")

	if err := deps.Chaser.ChaseAll(ctx, agent.TenantID, today, agent.MentorID); err != nil {
		return fmt.Errorf("chase all: %w", err)
	}

	msg := agent.Brand(fmt.Sprintf("Chase report complete for %s. Non-submitters have been reminded.", today))
	return deps.Sender.SendToBoss(msg)
}

// executeWeeklySummary generates a weekly operations insight.
func executeWeeklySummary(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	if err := deps.ActionExec.RunWeekly(ctx, agent.TenantID, agent.MentorID, deps.Sender.bossChatID); err != nil {
		return fmt.Errorf("run weekly: %w", err)
	}

	commentary := generateCommentary(ctx, agent, deps, "weekly operations review",
		"Generate a brief weekly operations summary highlighting team health, process improvements needed, and key metrics trends.")

	msg := agent.Brand(fmt.Sprintf("Weekly Operations Review\n\n%s", commentary))
	return deps.Sender.SendToBoss(msg)
}

// executeCheckAlerts handles alert checking by creating suggestions for critical issues.
func executeCheckAlerts(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	alerts, err := deps.AlertChecker.CheckAll(ctx, agent.TenantID, deps.Sender.bossChatID)
	if err != nil {
		return fmt.Errorf("check alerts: %w", err)
	}

	for _, alert := range alerts {
		if alert.Severity != "critical" {
			continue
		}

		suggestion := generateCommentary(ctx, agent, deps, "anomaly response",
			fmt.Sprintf("Critical alert: %s\nEmployee: %s\nType: %s\n\nProvide a specific, actionable recommendation for handling this situation.",
				alert.Message, alert.EmployeeName, alert.AlertType))

		if err := deps.Queries.CreateAISuggestion(ctx, CreateSuggestionParams{
			TenantID:    agent.TenantID,
			RoleID:      agent.RoleID,
			RoleTitle:   agent.Title,
			Capability:  "check_alerts",
			Title:       fmt.Sprintf("Action needed: %s — %s", alert.AlertType, alert.EmployeeName),
			Content:     suggestion,
			ContextData: []byte("{}"),
		}); err != nil {
			slog.Error("create ai suggestion", "role", agent.RoleID, "error", err)
		}
	}

	return nil
}

// executeCreateSuggestion generates a generic strategic suggestion for boss approval.
func executeCreateSuggestion(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	scope := agent.config.Scope
	commentary := generateCommentary(ctx, agent, deps, "strategic suggestion",
		fmt.Sprintf("Based on your role scope (%s), provide a specific, actionable suggestion for improving operations.", scope))

	if err := deps.Queries.CreateAISuggestion(ctx, CreateSuggestionParams{
		TenantID:    agent.TenantID,
		RoleID:      agent.RoleID,
		RoleTitle:   agent.Title,
		Capability:  "create_suggestion",
		Title:       fmt.Sprintf("Suggestion from %s", agent.Title),
		Content:     commentary,
		ContextData: []byte("{}"),
	}); err != nil {
		return fmt.Errorf("create suggestion: %w", err)
	}

	return nil
}

// executeSendBrandedMsg sends a simple branded status message to boss.
func executeSendBrandedMsg(ctx context.Context, agent *RoleAgent, deps *AgentDeps) error {
	commentary := generateCommentary(ctx, agent, deps, "status update",
		fmt.Sprintf("You are the %s. Provide a brief status update relevant to your scope: %s", agent.Title, agent.config.Scope))

	msg := agent.Brand(commentary)
	return deps.Sender.SendToBoss(msg)
}

// generateCommentary uses LLM to generate role-specific commentary, with fallback.
func generateCommentary(ctx context.Context, agent *RoleAgent, deps *AgentDeps, task, context string) string {
	if deps.LLM == nil {
		return context
	}

	prompt := fmt.Sprintf("Task: %s\nContext:\n%s\n\nProvide a brief, actionable commentary (3-5 sentences).", task, context)
	result, err := deps.LLM.Chat(ctx, agent.SystemPrompt(), prompt)
	if err != nil {
		slog.Warn("LLM commentary failed, using context", "role", agent.RoleID, "error", err)
		return context
	}
	return result
}
