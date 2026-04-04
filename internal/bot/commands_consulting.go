package bot

import (
	"context"
	"fmt"
	"strings"
)

// resolveTenantID looks up the tenant ID for the boss that sent a command.
func (h *CommandHandler) resolveTenantID(c BotContext) (string, error) {
	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return "", err
	}
	return tenant.ID, nil
}

// HandleConsult manages AI consulting engagements from Telegram.
func (h *CommandHandler) HandleConsult(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Only the boss can start consulting engagements.")
	}
	if h.consulting == nil {
		return c.Send("Consulting engine not available (requires ANTHROPIC_API_KEY).")
	}

	text := strings.TrimSpace(c.Text())

	// Sub-commands: /consult, /consult list, /consult approve <id>, /consult reject <id>, /consult execute <id>, /consult answer <id> <text>
	if text == "" || text == "help" {
		return c.Send("AI Management Consulting\n\n" +
			"/consult <problem> — Start a new consulting engagement\n" +
			"/consult list — List active engagements\n" +
			"/consult answer <id> <text> — Answer a diagnostic question\n" +
			"/consult approve <id> — Approve all actions in engagement\n" +
			"/consult reject <id> — Reject all actions in engagement\n" +
			"/consult execute <id> — Execute approved actions\n")
	}

	ctx := context.Background()

	// Parse sub-commands
	parts := strings.SplitN(text, " ", 3)
	switch parts[0] {
	case "list":
		tenantID, err := h.resolveTenantID(c)
		if err != nil {
			return c.Send("Could not resolve tenant.")
		}
		result, err := h.consulting.ListActiveEngagements(ctx, tenantID)
		if err != nil {
			return c.Send(fmt.Sprintf("Error: %v", err))
		}
		return c.Send(result)

	case "answer":
		if len(parts) < 3 {
			return c.Send("Usage: /consult answer <engagement-id> <your answer>")
		}
		engID := parts[1]
		answer := parts[2]
		nextQ, planText, done, err := h.consulting.AnswerQuestion(ctx, engID, answer)
		if err != nil {
			return c.Send(fmt.Sprintf("Error: %v", err))
		}
		if done {
			msg := "Analysis complete! Here's the plan:\n\n" + planText + "\n\nUse /consult approve " + engID + " to approve actions, or /consult reject " + engID + " to reject."
			return c.Send(msg)
		}
		return c.Send("Next question:\n\n" + nextQ)

	case "approve":
		if len(parts) < 2 {
			return c.Send("Usage: /consult approve <engagement-id>")
		}
		result, err := h.consulting.ReviewActions(ctx, parts[1], true)
		if err != nil {
			return c.Send(fmt.Sprintf("Error: %v", err))
		}
		return c.Send(result)

	case "reject":
		if len(parts) < 2 {
			return c.Send("Usage: /consult reject <engagement-id>")
		}
		result, err := h.consulting.ReviewActions(ctx, parts[1], false)
		if err != nil {
			return c.Send(fmt.Sprintf("Error: %v", err))
		}
		return c.Send(result)

	case "execute":
		if len(parts) < 2 {
			return c.Send("Usage: /consult execute <engagement-id>")
		}
		result, err := h.consulting.ExecuteApproved(ctx, parts[1])
		if err != nil {
			return c.Send(fmt.Sprintf("Error: %v", err))
		}
		return c.Send(result)

	default:
		// Default: start a new engagement with the full text as the problem
		tenantID, err := h.resolveTenantID(c)
		if err != nil {
			return c.Send("Could not resolve tenant.")
		}
		engID, firstQuestion, err := h.consulting.StartEngagement(ctx, tenantID, text, "", "")
		if err != nil {
			return c.Send(fmt.Sprintf("Error starting engagement: %v", err))
		}
		return c.Send(fmt.Sprintf("Consulting engagement started (ID: %s)\n\nFirst question:\n%s\n\nReply with: /consult answer %s <your answer>", engID[:8], firstQuestion, engID[:8]))
	}
}
