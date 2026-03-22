package report

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
)

// ActionDB defines DB operations for proactive actions.
type ActionDB interface {
	ListActiveEmployees(ctx context.Context, tenantID string) ([]EmployeeInfo, error)
	GetSubmittedDaysLast7(ctx context.Context, employeeID string) (int, error)
}

// ActionExecutor runs proactive actions from mentor config.
type ActionExecutor struct {
	db      ActionDB
	sender  channel.Sender
	llm     *brain.LLMService
	factory *brain.EngineFactory
}

// NewActionExecutor creates a new action executor.
func NewActionExecutor(db ActionDB, sender channel.Sender, llm *brain.LLMService, factory *brain.EngineFactory) *ActionExecutor {
	return &ActionExecutor{db: db, sender: sender, llm: llm, factory: factory}
}

// RunWeekly executes weekly proactive actions for the tenant.
func (a *ActionExecutor) RunWeekly(ctx context.Context, tenantID, mentorID string, bossInfo EmployeeInfo) error {
	engine, err := a.factory.ForTenant(mentorID, "default")
	if err != nil {
		return fmt.Errorf("load engine: %w", err)
	}

	actions := engine.GetWeeklyActions()
	if len(actions) == 0 {
		return nil
	}

	employees, err := a.db.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

	// Resolve boss channel once for all action messages
	bossChType, bossChID := resolveEmployeeChannel(bossInfo)
	if bossChType == "" {
		return fmt.Errorf("boss has no channel configured")
	}

	for _, action := range actions {
		msg, err := a.generateActionMessage(ctx, action.Type, action.Desc, employees, engine)
		if err != nil {
			slog.Error("generate action message", "type", action.Type, "error", err)
			continue
		}
		if msg == "" {
			continue
		}

		if err := a.sender.Send(ctx, bossChType, bossChID, msg); err != nil {
			slog.Error("send weekly action", "type", action.Type, "error", err)
		} else {
			slog.Info("weekly action sent", "type", action.Type)
		}
	}

	return nil
}

// RunMonthly executes monthly proactive actions for the tenant.
func (a *ActionExecutor) RunMonthly(ctx context.Context, tenantID, mentorID string, bossInfo EmployeeInfo) error {
	engine, err := a.factory.ForTenant(mentorID, "default")
	if err != nil {
		return fmt.Errorf("load engine: %w", err)
	}

	actions := engine.GetMonthlyActions()
	if len(actions) == 0 {
		return nil
	}

	// Resolve boss channel once for all action messages
	bossChType, bossChID := resolveEmployeeChannel(bossInfo)
	if bossChType == "" {
		return fmt.Errorf("boss has no channel configured")
	}

	for _, action := range actions {
		msg := fmt.Sprintf("📊 Monthly Action: %s\n\n%s", action.Type, action.Desc)
		if err := a.sender.Send(ctx, bossChType, bossChID, msg); err != nil {
			slog.Error("send monthly action", "type", action.Type, "error", err)
		}
	}

	return nil
}

// generateActionMessage creates the message for a specific action type.
func (a *ActionExecutor) generateActionMessage(ctx context.Context, actionType, desc string, employees []EmployeeInfo, engine *brain.Engine) (string, error) {
	switch actionType {
	case "recognition":
		return a.generateRecognition(ctx, employees)
	case "ranking":
		return a.generateRanking(ctx, employees)
	case "self_criticism":
		return fmt.Sprintf("📋 Weekly Action: %s\n\n%s\n\nConsider scheduling a team self-reflection session.", actionType, desc), nil
	case "team_pulse":
		return fmt.Sprintf("📋 Weekly Action: %s\n\n%s\n\nConsider sending a quick pulse survey to the team.", actionType, desc), nil
	case "okr_review":
		return fmt.Sprintf("📋 Weekly Action: %s\n\n%s\n\nTime to review OKR progress with the team.", actionType, desc), nil
	case "one_on_one":
		return a.generateOneOnOneSuggestion(ctx, employees)
	default:
		return fmt.Sprintf("📋 Weekly Action: %s\n\n%s", actionType, desc), nil
	}
}

// generateRecognition finds the top contributor (most submissions this week).
func (a *ActionExecutor) generateRecognition(ctx context.Context, employees []EmployeeInfo) (string, error) {
	if len(employees) == 0 {
		return "", nil
	}

	var topName string
	var topDays int

	for _, emp := range employees {
		days, err := a.db.GetSubmittedDaysLast7(ctx, emp.ID)
		if err != nil {
			continue
		}
		if days > topDays {
			topDays = days
			topName = emp.Name
		}
	}

	if topName == "" || topDays == 0 {
		return "", nil
	}

	return fmt.Sprintf("🌟 Weekly Recognition\n\nTop contributor this week: %s (%d/7 days submitted)\n\nConsider publicly recognizing their consistency and dedication.", topName, topDays), nil
}

// generateRanking creates a weekly submission ranking.
func (a *ActionExecutor) generateRanking(ctx context.Context, employees []EmployeeInfo) (string, error) {
	if len(employees) == 0 {
		return "", nil
	}

	type empScore struct {
		Name string
		Days int
	}
	var scores []empScore
	for _, emp := range employees {
		days, _ := a.db.GetSubmittedDaysLast7(ctx, emp.ID)
		scores = append(scores, empScore{emp.Name, days})
	}

	// Sort by days (simple bubble sort for small lists)
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Days > scores[i].Days {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("📊 Weekly Submission Ranking\n\n")
	for i, s := range scores {
		medal := ""
		if i == 0 {
			medal = "🥇 "
		} else if i == 1 {
			medal = "🥈 "
		} else if i == 2 {
			medal = "🥉 "
		}
		sb.WriteString(fmt.Sprintf("%s%s — %d/7 days\n", medal, s.Name, s.Days))
	}

	return sb.String(), nil
}

// generateOneOnOneSuggestion identifies employees who might need a 1:1.
func (a *ActionExecutor) generateOneOnOneSuggestion(ctx context.Context, employees []EmployeeInfo) (string, error) {
	var needsAttention []string
	for _, emp := range employees {
		days, _ := a.db.GetSubmittedDaysLast7(ctx, emp.ID)
		if days <= 3 {
			needsAttention = append(needsAttention, emp.Name)
		}
	}

	if len(needsAttention) == 0 {
		return "📋 Weekly 1:1 Suggestion\n\nAll team members are performing well. No urgent 1:1s needed.", nil
	}

	return fmt.Sprintf("📋 Weekly 1:1 Suggestion\n\nConsider scheduling 1:1s with:\n- %s\n\nThey had lower submission rates this week.",
		strings.Join(needsAttention, "\n- ")), nil
}
