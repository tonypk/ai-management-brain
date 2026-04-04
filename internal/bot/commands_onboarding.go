package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// HandleStart initialises the boss's team. If no tenant exists it is created automatically.
// If onboarding is enabled and not yet completed, delegates to the onboarding service.
func (h *CommandHandler) HandleStart(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Please contact your manager for access. Use /join <code> if you have an invite code.")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("get tenant: %w", err)
	}

	if tenant == nil {
		tenant, err = h.db.CreateTenant(context.Background(), CreateTenantParams{
			Name:       "My Team",
			BossChatID: c.SenderID(),
			MentorID:   "inamori",
			Timezone:   "Asia/Manila",
		})
		if err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}
		slog.Info("tenant auto-created", "tenant_id", tenant.ID)
	}

	// If onboarding service is set and onboarding not complete, delegate
	if h.onboarding != nil && tenant.OnboardingCompletedAt == nil {
		resp, err := h.onboarding.HandleMessage(
			context.Background(), tenant.ID,
			"telegram", strconv.FormatInt(c.SenderID(), 10), "/start")
		if err != nil {
			slog.Error("onboarding HandleMessage failed", "error", err)
			return c.Send("Something went wrong starting onboarding. Please try again.")
		}
		return c.Send(resp)
	}

	// Original welcome for already-onboarded tenants
	return c.Send(fmt.Sprintf(
		"Welcome to AI Management Brain!\n\nYour team '%s' is set up.\n\nUse:\n/addemployee name | job | resp | country | lang — Add team member\n/status — View team status\n/mentor <id> — Switch mentor\n/help — Show all commands",
		tenant.Name,
	))
}

// HandleJoin links a Telegram user to an employee record via invite code,
// OR registers a group chat when called from a group context.
func (h *CommandHandler) HandleJoin(c BotContext) error {
	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /join <invite_code>")
	}

	code := parts[1]

	// Group chat registration
	chatType := c.ChatType()
	if chatType == "group" || chatType == "supergroup" {
		if h.groupDB == nil {
			return c.Send("Group features not available.")
		}
		// Look up tenant by invite code to get tenant_id
		emp, err := h.db.GetEmployeeByInviteCode(context.Background(), code)
		if err != nil {
			return c.Send("Invalid invite code.")
		}

		chatID := fmt.Sprintf("%d", c.ChatID())
		title := c.ChatTitle()
		if title == "" {
			title = "Unnamed Group"
		}

		// Check if already registered
		existing, err := h.groupDB.GetGroupChatByPlatformID(context.Background(), "telegram", chatID)
		if err == nil && existing.ID != "" {
			return c.Send(fmt.Sprintf("This group '%s' is already registered.", existing.Name))
		}

		gc, err := h.groupDB.CreateGroupChat(context.Background(), emp.TenantID, "telegram", chatID, title, "general")
		if err != nil {
			slog.Error("create group chat", "error", err)
			return c.Send("Failed to register group. Please try again.")
		}
		slog.Info("group chat registered", "group_id", gc.ID, "name", gc.Name, "tenant", gc.TenantID)
		return c.Send(fmt.Sprintf("Group '%s' registered! The mentor will now be active here.\n\nUse the admin dashboard to change the group type.", title))
	}

	// Private chat — existing employee join flow
	emp, err := h.db.GetEmployeeByInviteCode(context.Background(), code)
	if err != nil {
		return c.Send("Invalid invite code.")
	}

	if err := h.db.UpdateEmployeeTelegramID(context.Background(), emp.ID, c.SenderID()); err != nil {
		return fmt.Errorf("update telegram id: %w", err)
	}

	slog.Info("employee joined", "employee_id", emp.ID, "telegram_id", c.SenderID())
	return c.Send(fmt.Sprintf(
		"Welcome %s! You're now linked to the team. You'll receive daily check-in questions.",
		emp.Name,
	))
}

// generateInviteCode creates a short random uppercase hex string.
func generateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
