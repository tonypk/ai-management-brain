package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// Confirmer handles the 4-step plan confirmation flow, formatting each step
// for human review, detecting confirmations, and modifying plans via LLM.
type Confirmer struct {
	llm brain.LLMClient
}

// NewConfirmer creates a new Confirmer with the given LLM client.
func NewConfirmer(llm brain.LLMClient) *Confirmer {
	return &Confirmer{llm: llm}
}

// confirmationWords lists all accepted confirmation phrases (lowercase).
var confirmationWords = []string{
	"ok", "yes", "confirm", "good", "looks good", "lgtm",
	"好", "好的", "可以", "没问题", "确认",
}

// FormatStep renders a single confirmation step as human-readable text.
// Steps are 1-indexed: 1=Mentor+Board, 2=OrgStructure, 3=Policies, 4=Schedule.
func (c *Confirmer) FormatStep(plan *ProposedPlan, step int) string {
	switch step {
	case 1:
		return formatMentorAndBoard(plan)
	case 2:
		return formatOrgStructure(plan)
	case 3:
		return formatPolicies(plan)
	case 4:
		return formatSchedule(plan)
	default:
		return fmt.Sprintf("Unknown step %d", step)
	}
}

// IsConfirmation returns true if the text is a confirmation phrase.
func (c *Confirmer) IsConfirmation(text string) bool {
	normalized := strings.TrimSpace(strings.ToLower(text))
	for _, word := range confirmationWords {
		if normalized == word {
			return true
		}
	}
	return false
}

// HandleModification sends the current plan and a modification request to the LLM,
// returns the updated plan and a formatted description of the changed step.
func (c *Confirmer) HandleModification(ctx context.Context, plan *ProposedPlan, step int, request string) (*ProposedPlan, string, error) {
	systemPrompt := buildModificationPrompt(step)
	userPrompt := fmt.Sprintf("Current plan:\n%s\n\nModification request: %s", toJSON(plan), request)

	resp, err := c.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, "", fmt.Errorf("LLM call failed: %w", err)
	}

	var updated ProposedPlan
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &updated); err != nil {
		return nil, "", fmt.Errorf("failed to parse modified plan: %w", err)
	}

	if err := updated.Validate(); err != nil {
		return nil, "", fmt.Errorf("modified plan is invalid: %w", err)
	}

	formatted := c.FormatStep(&updated, step)
	return &updated, formatted, nil
}

func buildModificationPrompt(step int) string {
	stepNames := map[int]string{
		1: "mentor and board configuration",
		2: "organizational structure",
		3: "policies and framework",
		4: "schedule and timing",
	}
	stepName := stepNames[step]
	if stepName == "" {
		stepName = "plan"
	}

	return fmt.Sprintf(`You are a management systems architect. The boss wants to modify the %s of their management plan.

Apply the requested changes to the plan and return the COMPLETE updated plan as JSON.
Keep all other parts of the plan unchanged.

RESPOND WITH JSON ONLY. No markdown, no explanation.`, stepName)
}

// formatMentorAndBoard renders step 1: mentor selection and board seats.
func formatMentorAndBoard(plan *ProposedPlan) string {
	var sb strings.Builder

	sb.WriteString("== Step 1: Mentor & Board ==\n\n")

	sb.WriteString(fmt.Sprintf("Primary Mentor: %s\n", plan.Mentor.PrimaryID))
	if plan.Mentor.SecondaryID != "" {
		sb.WriteString(fmt.Sprintf("Secondary Mentor: %s (blend weight: %.0f%%)\n",
			plan.Mentor.SecondaryID, plan.Mentor.BlendWeight*100))
	}
	if plan.Mentor.Reasoning != "" {
		sb.WriteString(fmt.Sprintf("Reasoning: %s\n", plan.Mentor.Reasoning))
	}

	sb.WriteString("\nBoard Seats:\n")
	for _, seat := range plan.Board {
		sb.WriteString(fmt.Sprintf("  - %s: %s", strings.ToUpper(seat.SeatType), seat.PersonaID))
		if seat.Reasoning != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", seat.Reasoning))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nReply OK to confirm, or tell me what to change.")
	return sb.String()
}

// formatOrgStructure renders step 2: org design as ASCII tree.
func formatOrgStructure(plan *ProposedPlan) string {
	var sb strings.Builder

	sb.WriteString("== Step 2: Organization Structure ==\n\n")

	tree := buildOrgTree(plan.OrgDesign.Units)
	sb.WriteString(tree)

	if plan.OrgDesign.Reasoning != "" {
		sb.WriteString(fmt.Sprintf("\nReasoning: %s\n", plan.OrgDesign.Reasoning))
	}

	sb.WriteString("\nReply OK to confirm, or tell me what to change.")
	return sb.String()
}

// formatPolicies renders step 3: policies and framework.
func formatPolicies(plan *ProposedPlan) string {
	var sb strings.Builder

	sb.WriteString("== Step 3: Policies & Framework ==\n\n")

	sb.WriteString(fmt.Sprintf("Framework: %s\n", plan.Policies.Framework))

	if len(plan.Policies.CheckinQuestions) > 0 {
		sb.WriteString("\nCheck-in Questions:\n")
		for i, q := range plan.Policies.CheckinQuestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, q))
		}
	}

	sb.WriteString(fmt.Sprintf("\nRisk Rules:\n"))
	sb.WriteString(fmt.Sprintf("  - Consecutive misses alert: %d days\n", plan.Policies.RiskRules.ConsecutiveMisses))
	sb.WriteString(fmt.Sprintf("  - Sentiment drop threshold: %.1f\n", plan.Policies.RiskRules.SentimentDropThreshold))
	if len(plan.Policies.RiskRules.UrgentKeywords) > 0 {
		sb.WriteString(fmt.Sprintf("  - Urgent keywords: %s\n", strings.Join(plan.Policies.RiskRules.UrgentKeywords, ", ")))
	}

	sb.WriteString("\nCadence:\n")
	if len(plan.Policies.Cadence.DailyActions) > 0 {
		sb.WriteString(fmt.Sprintf("  Daily: %s\n", strings.Join(plan.Policies.Cadence.DailyActions, ", ")))
	}
	if len(plan.Policies.Cadence.WeeklyActions) > 0 {
		sb.WriteString(fmt.Sprintf("  Weekly (%s): %s\n", plan.Policies.Cadence.WeeklyDay, strings.Join(plan.Policies.Cadence.WeeklyActions, ", ")))
	}
	if len(plan.Policies.Cadence.MonthlyActions) > 0 {
		sb.WriteString(fmt.Sprintf("  Monthly (day %d): %s\n", plan.Policies.Cadence.MonthlyDay, strings.Join(plan.Policies.Cadence.MonthlyActions, ", ")))
	}

	if plan.Policies.Reasoning != "" {
		sb.WriteString(fmt.Sprintf("\nReasoning: %s\n", plan.Policies.Reasoning))
	}

	sb.WriteString("\nReply OK to confirm, or tell me what to change.")
	return sb.String()
}

// formatSchedule renders step 4: schedule and timing.
func formatSchedule(plan *ProposedPlan) string {
	var sb strings.Builder

	sb.WriteString("== Step 4: Schedule ==\n\n")

	sb.WriteString(fmt.Sprintf("Timezone: %s\n\n", plan.Schedule.Timezone))

	schedules := []struct {
		name string
		cron string
	}{
		{"Check-in Reminder", plan.Schedule.Checkin},
		{"Chase (follow-up)", plan.Schedule.Chase},
		{"Daily Summary", plan.Schedule.Summary},
		{"Morning Briefing", plan.Schedule.Briefing},
		{"Signal Scan", plan.Schedule.SignalScan},
	}

	for _, s := range schedules {
		if s.cron != "" {
			sb.WriteString(fmt.Sprintf("  - %s: %s (%s)\n", s.name, s.cron, describeCron(s.cron)))
		}
	}

	sb.WriteString("\nReply OK to confirm, or tell me what to change.")
	return sb.String()
}

// buildOrgTree renders OrgUnitPlan slice as an ASCII tree.
func buildOrgTree(units []OrgUnitPlan) string {
	if len(units) == 0 {
		return "(no units defined)\n"
	}

	// Index children by parent_ref_id.
	childrenOf := make(map[string][]OrgUnitPlan)
	for _, u := range units {
		childrenOf[u.ParentRefID] = append(childrenOf[u.ParentRefID], u)
	}

	// Find roots (empty parent_ref_id).
	roots := childrenOf[""]

	var sb strings.Builder
	for i, root := range roots {
		isLast := i == len(roots)-1
		renderTreeNode(&sb, root, childrenOf, "", isLast, true)
	}
	return sb.String()
}

// renderTreeNode recursively renders one node and its children.
func renderTreeNode(sb *strings.Builder, unit OrgUnitPlan, childrenOf map[string][]OrgUnitPlan, prefix string, isLast bool, isRoot bool) {
	// Build connector.
	connector := ""
	if !isRoot {
		if isLast {
			connector = prefix + "\u2514\u2500\u2500 "
		} else {
			connector = prefix + "\u251c\u2500\u2500 "
		}
	}

	// Format the node line.
	label := unit.Name
	if unit.HeadRole != "" {
		label += fmt.Sprintf(" (%s)", unit.HeadRole)
	}

	if isRoot {
		sb.WriteString(label + "\n")
	} else {
		sb.WriteString(connector + label + "\n")
	}

	// Recurse into children.
	children := childrenOf[unit.RefID]
	childPrefix := prefix
	if !isRoot {
		if isLast {
			childPrefix = prefix + "    "
		} else {
			childPrefix = prefix + "\u2502   "
		}
	}

	for i, child := range children {
		childIsLast := i == len(children)-1
		renderTreeNode(sb, child, childrenOf, childPrefix, childIsLast, false)
	}
}

// describeCron provides a human-readable description of common cron patterns.
func describeCron(cron string) string {
	parts := strings.Fields(cron)
	if len(parts) < 5 {
		return cron
	}

	minute := parts[0]
	hour := parts[1]
	dayOfWeek := parts[4]

	// Build time string.
	timeStr := ""
	if hour != "*" && minute != "*" && !strings.Contains(hour, "/") && !strings.Contains(minute, "/") {
		timeStr = fmt.Sprintf("%s:%s", zeroPad(hour), zeroPad(minute))
	}

	// Day-of-week mapping.
	dayDesc := describeDaysOfWeek(dayOfWeek)

	// Handle interval patterns like */30.
	if strings.HasPrefix(minute, "*/") {
		interval := strings.TrimPrefix(minute, "*/")
		if strings.Contains(hour, "-") {
			return fmt.Sprintf("every %s min, %s, %s", interval, hour, dayDesc)
		}
		return fmt.Sprintf("every %s min, %s", interval, dayDesc)
	}

	if timeStr != "" && dayDesc != "" {
		return fmt.Sprintf("%s, %s", timeStr, dayDesc)
	}
	if timeStr != "" {
		return timeStr
	}
	return cron
}

// describeDaysOfWeek converts cron day-of-week to readable text.
func describeDaysOfWeek(dow string) string {
	if dow == "*" {
		return "every day"
	}
	if dow == "1-5" {
		return "Mon-Fri"
	}
	if dow == "0-6" || dow == "0-7" {
		return "every day"
	}
	return dow
}

// zeroPad adds a leading zero to single-digit strings.
func zeroPad(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
