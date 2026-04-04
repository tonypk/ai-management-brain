package bot

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// BotContext abstracts telebot.Context for testability.
type BotContext interface {
	SenderID() int64
	Text() string
	Send(msg string) error
	ChatID() int64
	ChatType() string  // "private", "group", "supergroup"
	ChatTitle() string // group name (empty for private chats)
	Reply(msg string) error
}

// OnboardingHandler defines the onboarding service interface for bot routing.
type OnboardingHandler interface {
	HandleMessage(ctx context.Context, tenantID string, channelType, userID, text string) (string, error)
}

// GroupQuerier defines DB operations for group chat management.
type GroupQuerier interface {
	CreateGroupChat(ctx context.Context, tenantID, platform, platformChatID, name, groupType string) (GroupChat, error)
	GetGroupChatByPlatformID(ctx context.Context, platform, platformChatID string) (GroupChat, error)
}

// GroupChat holds basic group chat info for bot use.
type GroupChat struct {
	ID, TenantID, Name string
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
	TenantID, Name, CultureCode, InviteCode string
	JobTitle, Responsibilities               string
	Country, Language                        string
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
	GetSeatByTenantAndType(ctx context.Context, tenantID string, seatType string) (SeatInfo, error)
	ListSeatsByTenantID(ctx context.Context, tenantID string) ([]SeatInfo, error)
	UpsertSeat(ctx context.Context, tenantID, seatType, title, personaID, scope string) error
}

// EmployeeProfile holds profile data for display.
type EmployeeProfile struct {
	SubmittedLast7  int
	SubmittedLast30 int
	CurrentStreak   int
	SentimentTrend  string // e.g., "positive", "neutral", "mixed"
}

// ConsultingServicer abstracts the consulting engine for bot command use.
type ConsultingServicer interface {
	StartEngagement(ctx context.Context, tenantID, problem, mentorID, cultureCode string) (engagementID, firstQuestion string, err error)
	AnswerQuestion(ctx context.Context, engagementID, answer string) (nextQuestion, planText string, done bool, err error)
	ReviewActions(ctx context.Context, engagementID string, approved bool) (string, error)
	ExecuteApproved(ctx context.Context, engagementID string) (string, error)
	ListActiveEngagements(ctx context.Context, tenantID string) (string, error)
}

// SeatBoardResponse holds one seat's contribution in a board discussion.
type SeatBoardResponse struct {
	SeatType, Title, PersonaID, Content string
}

// SeatServicer is the interface CommandHandler needs from seats.SeatService.
type SeatServicer interface {
	SetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64, seatType string) error
	GetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) string
	ClearActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) error
	Chat(ctx context.Context, tenantID, seatType, cultureCode, userMessage string) (string, error)
	BoardDiscuss(ctx context.Context, tenantID, cultureCode, topic string) ([]SeatBoardResponse, string, error)
}

// SeatInfo holds seat data for the bot package (avoids importing seats package).
type SeatInfo struct {
	ID, SeatType, Title, PersonaID, Scope string
	IsActive                              bool
}

// CommandHandler handles bot commands.
type CommandHandler struct {
	db            CommandQuerier
	groupDB       GroupQuerier
	seatSvc       SeatServicer
	onboarding    OnboardingHandler
	consulting    ConsultingServicer
	bossChatID    int64
	DiagnosticsFn func() string
}

// SetGroupDB injects the group querier for group chat features.
func (h *CommandHandler) SetGroupDB(gdb GroupQuerier) { h.groupDB = gdb }

// SetSeatService injects the seat service for C-Suite features.
func (h *CommandHandler) SetSeatService(svc SeatServicer) { h.seatSvc = svc }

// SetOnboardingService injects the onboarding service for chat-native onboarding.
func (h *CommandHandler) SetOnboardingService(svc OnboardingHandler) { h.onboarding = svc }

// SetConsultingService injects the consulting engine for /consult commands.
func (h *CommandHandler) SetConsultingService(svc ConsultingServicer) { h.consulting = svc }

// NewCommandHandler creates a new CommandHandler. The second and third arguments
// are reserved for future dependencies (e.g. brain engine, claude client) and
// are ignored for now.
func NewCommandHandler(db CommandQuerier, _ interface{}, _ interface{}, bossChatID int64) *CommandHandler {
	return &CommandHandler{db: db, bossChatID: bossChatID}
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
	"trout":       "Jack Trout (Trout & Partners) — 定位理论，占据用户心智中的独特位置",
	"meyer":       "Erin Meyer (INSEAD) — 文化地图，跨文化沟通与领导力",
}

// HandleHelp sends the list of available commands. Anyone can call this.
func (h *CommandHandler) HandleHelp(c BotContext) error {
	help := `Available Commands:

/start — Initialize your team
/status — View team & report status
/addemployee name | job | resp | country | lang — Add team member
/join <code> — Link your Telegram (for employees)
/mentor <id> — Switch mentor (inamori, dalio, grove, ren)
/blend <primary> <weight> <secondary> — Blend mentors (e.g., /blend inamori 70 dalio)
/culture <name> <code> — Set employee culture
/profile <name> — View employee profile & stats
/talk <seat> — Chat with a C-Suite seat (e.g., /talk ceo)
/board <topic> — Board discussion across all seats
/team — View your C-Suite seats
/assign <seat> <persona> — Assign persona to seat
/diagnostics — System status
/help — Show this message`
	return c.Send(help)
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
