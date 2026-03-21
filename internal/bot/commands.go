package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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
	UpdateTenantBlend(ctx context.Context, tenantID, mentorID string, blendJSON []byte) error
	UpdateEmployeeCulture(ctx context.Context, employeeID, cultureCode string) error
	GetEmployeeProfile(ctx context.Context, employeeID string) (*EmployeeProfile, error)
}

// EmployeeProfile holds profile data for display.
type EmployeeProfile struct {
	SubmittedLast7  int
	SubmittedLast30 int
	CurrentStreak   int
	SentimentTrend  string // e.g., "positive", "neutral", "mixed"
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
/addemployee <name> <culture> — Add team member
/join <code> — Link your Telegram (for employees)
/mentor <id> — Switch mentor (inamori, dalio, grove, ren)
/blend <primary> <weight> <secondary> — Blend mentors (e.g., /blend inamori 70 dalio)
/culture <name> <code> — Set employee culture
/profile <name> — View employee profile & stats
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

// mentorDescriptions provides human-readable info for each mentor.
var mentorDescriptions = map[string]string{
	"inamori":     "稻盛和夫 (Kyocera) — 阿米巴经营，敬天爱人，利他哲学",
	"dalio":       "Ray Dalio (Bridgewater) — 极度透明，原则驱动，数据决策",
	"grove":       "Andy Grove (Intel) — OKR驱动，高产出管理，建设性对抗",
	"ren":         "任正非 (华为) — 狼性文化，自我批判，以奋斗者为本",
	"son":         "孙正义 (SoftBank) — 300年愿景，时间机器理论",
	"jobs":        "Steve Jobs (Apple) — 追求极简，现实扭曲力场",
	"bezos":       "Jeff Bezos (Amazon) — Day 1心态，客户至上",
	"ma":          "马云 (阿里巴巴) — 拥抱变化，客户第一，团队合作",
	"musk":        "Elon Musk (Tesla/SpaceX) — 第一性原理，极致紧迫感，10倍思维",
	"buffett":     "沃伦·巴菲特 (Berkshire) — 长期主义，安全边际，复利思维",
	"zhangyiming": "张一鸣 (字节跳动) — 延迟满足，信息平权，Context not Control",
	"leijun":      "雷军 (小米) — 极致性价比，参与感，专注口碑快",
	"caodewang":   "曹德旺 (福耀玻璃) — 实业精神，成本控制，品质第一",
	"chushijian":  "褚时健 (褚橙) — 极致专注，品质至上，逆境重生",
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

// HandleProfile shows an employee's submission profile to the boss.
func (h *CommandHandler) HandleProfile(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /profile <employee_name>\nExample: /profile Alice")
	}

	empName := parts[1]

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	employees, err := h.db.ListEmployeesByTenant(context.Background(), tenant.ID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

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

	profile, err := h.db.GetEmployeeProfile(context.Background(), found.ID)
	if err != nil {
		return c.Send(fmt.Sprintf("Could not load profile for %s.", found.Name))
	}

	status := "not linked"
	if found.TelegramID > 0 {
		status = "active"
	}

	return c.Send(fmt.Sprintf(
		"Employee Profile: %s\n\n"+
			"Status: %s\n"+
			"Culture: %s\n\n"+
			"Last 7 days: %d/7 submitted\n"+
			"Last 30 days: %d submitted\n"+
			"Current streak: %d days\n"+
			"Sentiment trend: %s",
		found.Name, status, found.CultureCode,
		profile.SubmittedLast7, profile.SubmittedLast30,
		profile.CurrentStreak, profile.SentimentTrend,
	))
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

// BlendConfig matches brain.BlendConfig for JSON serialization.
type BlendConfig struct {
	PrimaryID   string  `json:"primary_id"`
	SecondaryID string  `json:"secondary_id"`
	Weight      float64 `json:"weight"`
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

// generateInviteCode creates a short random uppercase hex string.
func generateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
