package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// BlendConfig matches brain.BlendConfig for JSON serialization.
type BlendConfig struct {
	PrimaryID   string  `json:"primary_id"`
	SecondaryID string  `json:"secondary_id"`
	Weight      float64 `json:"weight"`
}

// HandleMentor switches the active mentor for the boss's team.
func (h *CommandHandler) HandleMentor(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		var sb strings.Builder
		sb.WriteString("Usage: /mentor <id>\n\nAvailable mentors:\n")
		for id, desc := range mentorDescriptions {
			sb.WriteString(fmt.Sprintf("  %s — %s\n", id, desc))
		}
		return c.Send(sb.String())
	}

	mentorID := strings.ToLower(parts[1])
	if _, ok := mentorDescriptions[mentorID]; !ok {
		var sb strings.Builder
		sb.WriteString("Unknown mentor. Available:\n")
		for id, desc := range mentorDescriptions {
			sb.WriteString(fmt.Sprintf("  %s — %s\n", id, desc))
		}
		return c.Send(sb.String())
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	if err := h.db.UpdateTenantMentor(context.Background(), tenant.ID, mentorID); err != nil {
		return fmt.Errorf("update mentor: %w", err)
	}

	desc := mentorDescriptions[mentorID]
	return c.Send(fmt.Sprintf("Mentor switched to '%s'!\n%s", mentorID, desc))
}

// HandleBlend sets mentor blending for the boss's team.
func (h *CommandHandler) HandleBlend(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())

	// /blend off — disable blending
	if len(parts) >= 2 && strings.ToLower(parts[1]) == "off" {
		tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
		if err != nil {
			return c.Send("No team found. Use /start first.")
		}
		if err := h.db.UpdateTenantBlend(context.Background(), tenant.ID, tenant.MentorID, nil); err != nil {
			return fmt.Errorf("clear blend: %w", err)
		}
		return c.Send(fmt.Sprintf("Mentor blending disabled. Using pure '%s'.", tenant.MentorID))
	}

	// /blend <primary> <weight> <secondary> — e.g., /blend inamori 70 dalio
	if len(parts) < 4 {
		return c.Send("Usage: /blend <primary> <weight%> <secondary>\nExample: /blend inamori 70 dalio\n\nThis blends 70% Inamori + 30% Dalio.\n\nUse /blend off to disable.")
	}

	primaryID := strings.ToLower(parts[1])
	weightStr := parts[2]
	secondaryID := strings.ToLower(parts[3])

	if _, ok := mentorDescriptions[primaryID]; !ok {
		return c.Send(fmt.Sprintf("Unknown primary mentor '%s'.", primaryID))
	}
	if _, ok := mentorDescriptions[secondaryID]; !ok {
		return c.Send(fmt.Sprintf("Unknown secondary mentor '%s'.", secondaryID))
	}
	if primaryID == secondaryID {
		return c.Send("Primary and secondary mentors must be different.")
	}

	weight, err := strconv.Atoi(strings.TrimSuffix(weightStr, "%"))
	if err != nil || weight < 50 || weight > 90 {
		return c.Send("Weight must be between 50-90 (e.g., 70 for 70% primary).")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	blend := BlendConfig{
		PrimaryID:   primaryID,
		SecondaryID: secondaryID,
		Weight:      float64(weight) / 100.0,
	}
	blendJSON, err := json.Marshal(blend)
	if err != nil {
		return fmt.Errorf("marshal blend: %w", err)
	}

	if err := h.db.UpdateTenantBlend(context.Background(), tenant.ID, primaryID, blendJSON); err != nil {
		return fmt.Errorf("save blend: %w", err)
	}

	return c.Send(fmt.Sprintf(
		"Mentor blending enabled!\n\n%d%% %s + %d%% %s\n\nPrimary: %s\nSecondary: %s",
		weight, primaryID, 100-weight, secondaryID,
		mentorDescriptions[primaryID],
		mentorDescriptions[secondaryID],
	))
}

// HandleCulture sets or views employee culture codes.
func (h *CommandHandler) HandleCulture(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 3 {
		return c.Send("Usage: /culture <employee_name> <code>\nAvailable cultures: default, philippines, singapore, indonesia, srilanka\n\nExample: /culture Alice philippines")
	}

	empName := parts[1]
	cultureCode := strings.ToLower(parts[2])

	validCultures := map[string]bool{
		"default": true, "philippines": true, "singapore": true,
		"indonesia": true, "srilanka": true,
	}
	if !validCultures[cultureCode] {
		return c.Send("Unknown culture. Available: default, philippines, singapore, indonesia, srilanka")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	employees, err := h.db.ListEmployeesByTenant(context.Background(), tenant.ID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

	// Find employee by name (case-insensitive)
	var found *Employee
	for i, emp := range employees {
		if strings.EqualFold(emp.Name, empName) {
			found = &employees[i]
			break
		}
	}
	if found == nil {
		return c.Send(fmt.Sprintf("Employee '%s' not found.", empName))
	}

	if err := h.db.UpdateEmployeeCulture(context.Background(), found.ID, cultureCode); err != nil {
		return fmt.Errorf("update culture: %w", err)
	}

	return c.Send(fmt.Sprintf("Culture for %s set to '%s'.", found.Name, cultureCode))
}
