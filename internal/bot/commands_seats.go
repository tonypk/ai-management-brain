package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// HandleTalk switches the boss into a specific C-Suite seat for direct chat.
func (h *CommandHandler) HandleTalk(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied. Only the boss can use /talk.")
	}
	if h.seatSvc == nil {
		return c.Send("C-Suite features are not enabled.")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /talk <seat_type>\nExample: /talk cmo\nUse /talk off to return to default mode.\nUse /team to see available seats.")
	}

	seatType := strings.ToLower(parts[1])

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}
	tenantID := tenant.ID

	if seatType == "off" {
		if err := h.seatSvc.ClearActiveSeat(context.Background(), tenantID, c.SenderID()); err != nil {
			return c.Send("Failed to exit seat mode.")
		}
		return c.Send("Returned to default mode.")
	}

	// Verify the seat exists for this tenant
	_, err = h.db.GetSeatByTenantAndType(context.Background(), tenantID, seatType)
	if err != nil {
		return c.Send(fmt.Sprintf("No %q seat assigned. Use /assign %s <persona> to create one, or /team to see current seats.", seatType, seatType))
	}

	if err := h.seatSvc.SetActiveSeat(context.Background(), tenantID, c.SenderID(), seatType); err != nil {
		return c.Send("Failed to switch seat.")
	}

	return c.Send(fmt.Sprintf("Switched to %s. Send messages to chat with this role.\nUse /talk off to return to default mode.", strings.ToUpper(seatType)))
}

// HandleBoard triggers a multi-seat board discussion on a given topic.
func (h *CommandHandler) HandleBoard(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied. Only the boss can use /board.")
	}
	if h.seatSvc == nil {
		return c.Send("C-Suite features are not enabled.")
	}

	text := c.Text()
	// Remove /board prefix
	topic := strings.TrimSpace(strings.TrimPrefix(text, "/board"))
	if topic == "" {
		return c.Send("Usage: /board <topic>\nExample: /board Should we enter the Southeast Asian market?")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}
	tenantID := tenant.ID

	_ = c.Send("Board discussion starting... This may take a moment.")

	responses, synthesis, err := h.seatSvc.BoardDiscuss(context.Background(), tenantID, "default", topic)
	if err != nil {
		return c.Send(fmt.Sprintf("Board discussion failed: %s", err.Error()))
	}

	// Format and send results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Board Discussion\nTopic: %s\n\n", topic))

	for _, r := range responses {
		sb.WriteString(fmt.Sprintf("[%s]\n%s\n\n", r.Title, r.Content))
	}

	sb.WriteString(fmt.Sprintf("Synthesis\n%s", synthesis))

	return c.Send(sb.String())
}

// HandleTeam lists all C-Suite seats assigned to the boss's tenant.
func (h *CommandHandler) HandleTeam(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied.")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}
	tenantID := tenant.ID

	seatsList, err := h.db.ListSeatsByTenantID(context.Background(), tenantID)
	if err != nil || len(seatsList) == 0 {
		return c.Send("No C-Suite seats assigned yet.\nUse /assign <seat_type> <persona> to get started.\nExample: /assign ceo inamori")
	}

	var sb strings.Builder
	sb.WriteString("Your C-Suite Team\n\n")
	for _, s := range seatsList {
		status := "active"
		if !s.IsActive {
			status = "inactive"
		}
		desc := ""
		if d, ok := mentorDescriptions[s.PersonaID]; ok {
			desc = " - " + d
		}
		sb.WriteString(fmt.Sprintf("[%s] %s (%s): %s%s\n", status, s.Title, s.SeatType, s.PersonaID, desc))
	}
	sb.WriteString("\n/talk <seat> to chat | /board <topic> to discuss")

	return c.Send(sb.String())
}

// HandleAssign assigns a mentor persona to a C-Suite seat for the boss's tenant.
func (h *CommandHandler) HandleAssign(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied.")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 3 {
		return c.Send("Usage: /assign <seat_type> <persona_id>\nExample: /assign cmo trout\n\nAvailable personas:\n" + listPersonas())
	}

	seatType := strings.ToLower(parts[1])
	personaID := strings.ToLower(parts[2])

	if !brain.ValidMentors[personaID] {
		return c.Send(fmt.Sprintf("Unknown persona %q.\n\nAvailable:\n%s", personaID, listPersonas()))
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}
	tenantID := tenant.ID

	// Create or update the seat
	title := defaultTitleForSeatType(seatType)
	err = h.db.UpsertSeat(context.Background(), tenantID, seatType, title, personaID, "")
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to assign seat: %s", err.Error()))
	}

	desc := mentorDescriptions[personaID]
	return c.Send(fmt.Sprintf("Assigned %s to %s seat.\n%s\n\nUse /talk %s to start chatting.", personaID, strings.ToUpper(seatType), desc, seatType))
}

// listPersonas returns a formatted list of all available mentor personas.
func listPersonas() string {
	var sb strings.Builder
	for id, desc := range mentorDescriptions {
		sb.WriteString(fmt.Sprintf("  %s - %s\n", id, desc))
	}
	return sb.String()
}

// defaultTitleForSeatType maps seat types to human-readable titles.
func defaultTitleForSeatType(seatType string) string {
	defaults := map[string]string{
		"ceo":  "Chief Executive Officer",
		"cfo":  "Chief Financial Officer",
		"cmo":  "Chief Marketing Officer",
		"cto":  "Chief Technology Officer",
		"chro": "Chief Human Resources Officer",
		"coo":  "Chief Operations Officer",
	}
	if t, ok := defaults[seatType]; ok {
		return t
	}
	return seatType
}
