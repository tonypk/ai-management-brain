package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// BotContext abstracts telebot.Context for testability.
type BotContext interface {
	SenderID() int64
	Text() string
	Send(msg string) error
}

// CreateTenantParams holds parameters for tenant creation.
type CreateTenantParams struct {
	Name       string
	BossChatID int64
	MentorID   string
	Timezone   string
}

// CreateEmployeeParams holds parameters for employee creation.
type CreateEmployeeParams struct {
	TenantID    string
	Name        string
	CultureCode string
	InviteCode  string
}

// CommandQuerier defines DB operations for command handlers.
type CommandQuerier interface {
	GetTenantByBossChatID(ctx context.Context, bossChatID int64) (*Tenant, error)
	CreateTenant(ctx context.Context, params CreateTenantParams) (*Tenant, error)
	ListEmployeesByTenant(ctx context.Context, tenantID string) ([]Employee, error)
	CreateEmployee(ctx context.Context, params CreateEmployeeParams) (*Employee, error)
	GetEmployeeByInviteCode(ctx context.Context, code string) (*Employee, error)
	UpdateEmployeeTelegramID(ctx context.Context, employeeID string, telegramID int64) error
	UpdateTenantMentor(ctx context.Context, tenantID, mentorID string) error
}

// CommandHandler handles bot commands.
type CommandHandler struct {
	db             CommandQuerier
	bossChatID     int64
	DiagnosticsFn  func() string // set externally to provide diagnostics info
}

// NewCommandHandler creates a new CommandHandler. The second and third arguments
// are reserved for future dependencies (e.g. brain engine, claude client) and
// are ignored for now.
func NewCommandHandler(db CommandQuerier, _ interface{}, _ interface{}, bossChatID int64) *CommandHandler {
	return &CommandHandler{db: db, bossChatID: bossChatID}
}

// HandleStart initialises the boss's team. If no tenant exists it is created automatically.
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

	return c.Send(fmt.Sprintf(
		"Welcome to AI Management Brain!\n\nYour team '%s' is set up.\n\nUse:\n/addemployee <name> <culture> — Add team member\n/status — View team status\n/mentor <id> — Switch mentor\n/help — Show all commands",
		tenant.Name,
	))
}

// HandleStatus shows the boss a summary of the team and their activity.
func (h *CommandHandler) HandleStatus(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	employees, err := h.db.ListEmployeesByTenant(context.Background(), tenant.ID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

	if len(employees) == 0 {
		return c.Send("No employees yet. Use /addemployee <name> <culture> to add.")
	}

	var sb strings.Builder
	sb.WriteString("Team Status:\n\n")
	for _, emp := range employees {
		status := "not linked"
		if emp.TelegramID > 0 {
			status = "active"
		}
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", emp.Name, status))
	}
	return c.Send(sb.String())
}

// HandleHelp sends the list of available commands. Anyone can call this.
func (h *CommandHandler) HandleHelp(c BotContext) error {
	help := `Available Commands:

/start — Initialize your team
/status — View team & report status
/addemployee <name> <culture> — Add team member (e.g., /addemployee Alice ph)
/join <code> — Link your Telegram (for employees)
/mentor <id> — Switch mentor (inamori, dalio)
/diagnostics — System status
/help — Show this message`
	return c.Send(help)
}

// HandleAddEmployee adds a new employee to the boss's team and generates an invite code.
func (h *CommandHandler) HandleAddEmployee(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 3 {
		return c.Send("Usage: /addemployee <name> <culture>\nExample: /addemployee Alice ph")
	}

	name := parts[1]
	culture := parts[2]
	inviteCode := generateInviteCode()

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	emp, err := h.db.CreateEmployee(context.Background(), CreateEmployeeParams{
		TenantID:    tenant.ID,
		Name:        name,
		CultureCode: culture,
		InviteCode:  inviteCode,
	})
	if err != nil {
		return fmt.Errorf("create employee: %w", err)
	}

	return c.Send(fmt.Sprintf(
		"Employee '%s' added!\nInvite code: %s\n\nShare this code. They can join with: /join %s",
		emp.Name, inviteCode, inviteCode,
	))
}

// HandleJoin links a Telegram user to an employee record via invite code.
func (h *CommandHandler) HandleJoin(c BotContext) error {
	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /join <invite_code>")
	}

	code := parts[1]
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

// HandleMentor switches the active mentor for the boss's team.
func (h *CommandHandler) HandleMentor(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /mentor <id>\nAvailable: inamori, dalio")
	}

	mentorID := parts[1]
	validMentors := map[string]bool{"inamori": true, "dalio": true}
	if !validMentors[mentorID] {
		return c.Send("Unknown mentor. Available: inamori, dalio")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	if err := h.db.UpdateTenantMentor(context.Background(), tenant.ID, mentorID); err != nil {
		return fmt.Errorf("update mentor: %w", err)
	}

	return c.Send(fmt.Sprintf("Mentor switched to '%s'!", mentorID))
}

// HandleDiagnostics shows system diagnostics to the boss.
func (h *CommandHandler) HandleDiagnostics(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	info := "System Diagnostics:\n\nStatus: Running"
	if h.DiagnosticsFn != nil {
		info = h.DiagnosticsFn()
	}
	return c.Send(info)
}

// generateInviteCode creates a short random uppercase hex string.
func generateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
