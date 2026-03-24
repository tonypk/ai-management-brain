package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"

	"github.com/tonypk/ai-management-brain/internal/api"
	"github.com/tonypk/ai-management-brain/internal/bot"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/config"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/events"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/report"
	"github.com/tonypk/ai-management-brain/internal/roles"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
	"github.com/tonypk/ai-management-brain/internal/seats"
)

// engineForTenant returns the appropriate engine for a tenant (blended or single mentor).
func engineForTenant(factory *brain.EngineFactory, tenant *bot.Tenant, cultureCode string) (*brain.Engine, error) {
	if len(tenant.MentorBlend) > 0 {
		var blend brain.BlendConfig
		if err := json.Unmarshal(tenant.MentorBlend, &blend); err == nil && blend.PrimaryID != "" && blend.SecondaryID != "" {
			return factory.ForBlend(blend.PrimaryID, blend.SecondaryID, blend.Weight, cultureCode)
		}
	}
	return factory.ForTenant(tenant.MentorID, cultureCode)
}

// redactHandler wraps slog.Handler to mask sensitive fields.
type redactHandler struct {
	slog.Handler
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) error {
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		key := strings.ToLower(a.Key)
		if key == "api_key" || key == "bot_token" || key == "password" ||
			key == "encryption_key" || key == "token" || key == "secret" {
			attrs = append(attrs, slog.String(a.Key, "***REDACTED***"))
		} else {
			attrs = append(attrs, a)
		}
		return true
	})
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	for _, a := range attrs {
		newRecord.AddAttrs(a)
	}
	return h.Handler.Handle(ctx, newRecord)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &redactHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{Handler: h.Handler.WithGroup(name)}
}

// schedulerCallbacks wires scheduler to report/chase/summary.
type schedulerCallbacks struct {
	remindFn  func(ctx context.Context) error
	chaseFn   func(ctx context.Context) error
	summaryFn func(ctx context.Context) error
}

func (s *schedulerCallbacks) Remind(ctx context.Context) error  { return s.remindFn(ctx) }
func (s *schedulerCallbacks) Chase(ctx context.Context) error   { return s.chaseFn(ctx) }
func (s *schedulerCallbacks) Summary(ctx context.Context) error { return s.summaryFn(ctx) }

// redisWrapper adapts go-redis to our RedisClient interface.
type redisWrapper struct {
	client *redis.Client
}

func (r *redisWrapper) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisWrapper) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisWrapper) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *redisWrapper) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *redisWrapper) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// fetchBossContext gathers team data for boss chat from the database.
func fetchBossContext(ctx context.Context, queries *sqlc.Queries, tenantID string, loc *time.Location) brain.BossContext {
	uid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return brain.BossContext{}
	}

	latestSummary := ""
	if summary, err := queries.GetLatestSummary(ctx, uid); err == nil {
		latestSummary = summary.Content
	}

	today := time.Now().In(loc).Format("2006-01-02")
	todayDate, _ := time.Parse("2006-01-02", today)
	pgDate := pgtype.Date{Time: todayDate, Valid: true}
	submitted, _ := queries.CountReportsByTenantDate(ctx, sqlc.CountReportsByTenantDateParams{
		TenantID:   uid,
		ReportDate: pgDate,
	})

	emps, _ := queries.ListActiveEmployees(ctx, uid)
	roster := make([]brain.RosterEntry, 0, len(emps))
	for _, e := range emps {
		roster = append(roster, brain.RosterEntry{
			ID:       formatPgUUID(e.ID),
			Name:     e.Name,
			JobTitle: e.JobTitle,
			Role:     e.Role,
			IsActive: e.IsActive,
		})
	}

	return brain.BossContext{
		LatestSummary:  latestSummary,
		SubmittedCount: int(submitted),
		TotalEmployees: len(emps),
		EmployeeRoster: roster,
	}
}

// parseUUIDForChat parses a UUID string into pgtype.UUID.
func parseUUIDForChat(s string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(s); err != nil {
		return uid, err
	}
	return uid, nil
}

// formatPgUUID formats a pgtype.UUID as a hex string.
func formatPgUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// groupDBAdapter adapts sqlc.Queries to bot.GroupQuerier.
type groupDBAdapter struct {
	q *sqlc.Queries
}

func (a *groupDBAdapter) CreateGroupChat(ctx context.Context, tenantID, platform, platformChatID, name, groupType string) (bot.GroupChat, error) {
	tid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return bot.GroupChat{}, fmt.Errorf("parse tenant ID: %w", err)
	}
	gc, err := a.q.CreateGroupChat(ctx, sqlc.CreateGroupChatParams{
		TenantID:       tid,
		Platform:       platform,
		PlatformChatID: platformChatID,
		Name:           name,
		GroupType:      groupType,
	})
	if err != nil {
		return bot.GroupChat{}, err
	}
	return bot.GroupChat{
		ID:       formatPgUUID(gc.ID),
		TenantID: formatPgUUID(gc.TenantID),
		Name:     gc.Name,
	}, nil
}

func (a *groupDBAdapter) GetGroupChatByPlatformID(ctx context.Context, platform, platformChatID string) (bot.GroupChat, error) {
	gc, err := a.q.GetGroupChatByPlatformID(ctx, sqlc.GetGroupChatByPlatformIDParams{
		Platform:       platform,
		PlatformChatID: platformChatID,
	})
	if err != nil {
		return bot.GroupChat{}, err
	}
	return bot.GroupChat{
		ID:       formatPgUUID(gc.ID),
		TenantID: formatPgUUID(gc.TenantID),
		Name:     gc.Name,
	}, nil
}

// seatServiceAdapter bridges seats.SeatService to bot.SeatServicer.
type seatServiceAdapter struct {
	svc *seats.SeatService
}

func (a *seatServiceAdapter) SetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64, seatType string) error {
	return a.svc.SetActiveSeat(ctx, tenantID, telegramUserID, seatType)
}

func (a *seatServiceAdapter) GetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) string {
	return a.svc.GetActiveSeat(ctx, tenantID, telegramUserID)
}

func (a *seatServiceAdapter) ClearActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) error {
	return a.svc.ClearActiveSeat(ctx, tenantID, telegramUserID)
}

func (a *seatServiceAdapter) Chat(ctx context.Context, tenantID, seatType, cultureCode, userMessage string) (string, error) {
	return a.svc.Chat(ctx, tenantID, seatType, cultureCode, userMessage)
}

func (a *seatServiceAdapter) BoardDiscuss(ctx context.Context, tenantID, cultureCode, topic string) ([]bot.SeatBoardResponse, string, error) {
	responses, synthesis, err := a.svc.BoardDiscuss(ctx, tenantID, cultureCode, topic)
	if err != nil {
		return nil, "", err
	}
	result := make([]bot.SeatBoardResponse, len(responses))
	for i, r := range responses {
		result[i] = bot.SeatBoardResponse{
			SeatType:  r.SeatType,
			Title:     r.Title,
			PersonaID: r.PersonaID,
			Content:   r.Content,
		}
	}
	return result, synthesis, nil
}

// runMigrations applies database migrations idempotently.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationSQL := `
CREATE TABLE IF NOT EXISTS tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,
    bot_token     TEXT,
    boss_chat_id  BIGINT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    telegram_id   BIGINT UNIQUE,
    culture_code  TEXT NOT NULL DEFAULT 'default',
    role          TEXT NOT NULL DEFAULT 'member',
    invite_code   TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,
    blockers      TEXT,
    sentiment     TEXT,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);
CREATE TABLE IF NOT EXISTS chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,
    message       TEXT NOT NULL,
    chased_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary_date    DATE NOT NULL,
    content         TEXT NOT NULL,
    submission_rate FLOAT NOT NULL DEFAULT 0,
    blockers_count  INT NOT NULL DEFAULT 0,
    key_metrics     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);
CREATE INDEX IF NOT EXISTS idx_employees_tenant ON employees(tenant_id);
CREATE INDEX IF NOT EXISTS idx_employees_telegram ON employees(telegram_id);
CREATE INDEX IF NOT EXISTS idx_reports_tenant_date ON reports(tenant_id, report_date);
CREATE INDEX IF NOT EXISTS idx_reports_employee_date ON reports(employee_id, report_date);
CREATE INDEX IF NOT EXISTS idx_chase_logs_tenant_date ON chase_logs(tenant_id, report_date);
CREATE INDEX IF NOT EXISTS idx_chase_logs_employee ON chase_logs(employee_id, report_date);
CREATE INDEX IF NOT EXISTS idx_summaries_tenant_date ON summaries(tenant_id, summary_date);
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'boss',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
`
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return err
	}

	// Migration 000002: API Keys + tenant plan column
	migration002 := `
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free';
CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    prefix       TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT 'default',
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    is_active    BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id) WHERE is_active = true;
`
	if _, err := pool.Exec(ctx, migration002); err != nil {
		return err
	}

	// Migration 000003: Organizations + Wizard Sessions
	migration003 := `
CREATE TABLE IF NOT EXISTS organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    industry        TEXT NOT NULL,
    size            INT NOT NULL,
    stage           TEXT NOT NULL,
    business_model  TEXT,
    region          TEXT,
    mentor_id       TEXT NOT NULL,
    management_plan JSONB NOT NULL DEFAULT '{}',
    plan_version    INT NOT NULL DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'draft',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS wizard_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    mentor_id       TEXT NOT NULL,
    current_step    TEXT NOT NULL DEFAULT 'start',
    conversation    JSONB NOT NULL DEFAULT '[]',
    company_profile JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_organizations_tenant ON organizations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wizard_sessions_tenant ON wizard_sessions(tenant_id);
`
	if _, err := pool.Exec(ctx, migration003); err != nil {
		return err
	}

	// Migration 000004: AI Role Instances + Suggestions
	migration004 := `
CREATE TABLE IF NOT EXISTS ai_role_instances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role_id     TEXT NOT NULL,
    title       TEXT NOT NULL,
    mentor_id   TEXT NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, role_id)
);
CREATE TABLE IF NOT EXISTS ai_suggestions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    role_id      TEXT NOT NULL,
    role_title   TEXT NOT NULL,
    capability   TEXT NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    context_data JSONB NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'pending',
    reviewed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ai_role_instances_tenant ON ai_role_instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_suggestions_tenant_status ON ai_suggestions(tenant_id, status);
`
	if _, err := pool.Exec(ctx, migration004); err != nil {
		return err
	}

	// Migration 000005: Memories table with pgvector
	migration005 := `
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    memory_type  VARCHAR(30) NOT NULL,
    memory_tier  VARCHAR(20) NOT NULL DEFAULT 'short_term',
    employee_id  UUID REFERENCES employees(id),
    source_type  VARCHAR(30),
    source_id    UUID,
    content      TEXT NOT NULL,
    summary      TEXT,
    embedding    vector(384),
    importance   FLOAT DEFAULT 0.5,
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    merged_into  UUID REFERENCES memories(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX IF NOT EXISTS idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
`
	if _, err := pool.Exec(ctx, migration005); err != nil {
		return err
	}

	// Migration 000006: see sql/migrations/000006_vector384.up.sql
	migration006 := `ALTER TABLE memories ALTER COLUMN embedding TYPE vector(384);`
	if _, err := pool.Exec(ctx, migration006); err != nil {
		return err
	}

	// Migration 000007: Multi-channel support
	migration007 := `
ALTER TABLE employees ADD COLUMN IF NOT EXISTS signal_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS slack_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS lark_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS preferred_channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_signal ON employees(signal_phone) WHERE signal_phone IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_slack ON employees(slack_id) WHERE slack_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_lark ON employees(lark_id) WHERE lark_id IS NOT NULL;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS slack_bot_token TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS slack_signing_secret TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS lark_app_id TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS lark_app_secret TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS signal_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS enabled_channels TEXT[] NOT NULL DEFAULT '{telegram}';
ALTER TABLE reports ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
ALTER TABLE chase_logs ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
`
	if _, err := pool.Exec(ctx, migration007); err != nil {
		return err
	}

	const migration008 = `
-- 000008: employee profile fields
ALTER TABLE employees ADD COLUMN IF NOT EXISTS job_title       TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS country         TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS language        TEXT NOT NULL DEFAULT '';
`
	if _, err := pool.Exec(ctx, migration008); err != nil {
		return err
	}

	const migration009 = `
-- 000009: group chats
CREATE TABLE IF NOT EXISTS group_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    platform VARCHAR(20) NOT NULL DEFAULT 'telegram',
    platform_chat_id VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    group_type VARCHAR(50) NOT NULL DEFAULT 'general',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(platform, platform_chat_id)
);
CREATE INDEX IF NOT EXISTS idx_group_chats_tenant ON group_chats(tenant_id) WHERE is_active = true;
`
	_, err := pool.Exec(ctx, migration009)
	return err
}

func main() {
	// Set up log redaction
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(&redactHandler{Handler: baseHandler})
	slog.SetDefault(logger)

	slog.Info("AI Management Brain starting...")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Load timezone for date formatting
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("failed to load timezone", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping PostgreSQL", "error", err)
		os.Exit(1)
	}
	slog.Info("PostgreSQL connected")

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to parse Redis URL", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("failed to ping Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("Redis connected")

	redisClient := &redisWrapper{client: rdb}

	// Load industry templates
	if err := brain.LoadIndustries(); err != nil {
		slog.Warn("failed to load industry templates", "error", err)
	}

	// Create engine factory (dynamic mentor+culture per tenant)
	engineFactory := brain.NewEngineFactory()

	// Verify default mentor loads
	if _, err := engineFactory.ForTenant("inamori", "default"); err != nil {
		slog.Error("failed to load default engine", "error", err)
		os.Exit(1)
	}
	slog.Info("engine factory ready", "mentors", "inamori,dalio,grove,ren")

	// Create LLM client (optional)
	var llmService *brain.LLMService
	var orgWizard *brain.OrgWizard
	var orgEngine *brain.OrgEngine
	var chatService *brain.ChatService
	if cfg.AnthropicKey != "" {
		llmClient, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			slog.Error("failed to create Anthropic client", "error", err)
			os.Exit(1)
		}
		llmService = brain.NewLLMService(llmClient)
		orgEngine = brain.NewOrgEngine(llmClient)
		orgWizard = brain.NewOrgWizard(llmClient)

		// Create ChatService (uses same llmClient)
		chatService = brain.NewChatService(brain.ChatServiceConfig{
			LLM:           llmClient,
			Redis:         &redisWrapper{client: rdb},
			EngineFactory: engineFactory,
			BossTgID:      cfg.BossTelegramID,
		})

		slog.Info("Anthropic LLM client initialized (org engine + chat ready)")
	} else {
		slog.Warn("ANTHROPIC_API_KEY not set — AI features disabled, using template fallbacks")
	}

	// Create sqlc queries and adapters
	queries := sqlc.New(pool)
	botDB := bot.NewDBAdapter(queries)
	reportDB := report.NewDBAdapter(queries)

	// Initialize memory engine (requires ANTHROPIC_API_KEY; uses free HuggingFace embeddings)
	var memEngine *memory.MemoryEngine
	var memStore *memory.MemoryStore
	if cfg.AnthropicKey != "" {
		memLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			slog.Error("failed to create memory LLM client", "error", err)
			os.Exit(1)
		}

		embedder := memory.NewHuggingFaceEmbedder(cfg.EmbeddingModel, cfg.EmbeddingBatch)
		memStore = memory.NewMemoryStore(queries, pool)
		extractor := memory.NewExtractor(memLLM, embedder)
		retriever := memory.NewRetriever(memStore, embedder, cfg.MemoryMaxRecall, cfg.MemoryMaxTokens)
		consolidator := memory.NewConsolidator(memStore, memLLM, embedder, cfg.MemoryConsolidationThreshold)
		profiler := memory.NewProfileBuilder(memStore, memLLM, embedder)
		memEngine = memory.NewMemoryEngine(memStore, embedder, retriever, extractor, consolidator, profiler)

		// Inject memory engine into brain engine factory and chat service
		engineFactory.SetMemoryEngine(memEngine)
		if chatService != nil {
			chatService.SetMemoryEngine(memEngine)
		}

		slog.Info("memory engine enabled", "embedding_model", cfg.EmbeddingModel)
	} else {
		slog.Info("memory engine disabled (no ANTHROPIC_API_KEY)")
	}

	// Seat service (C-Suite)
	var seatSvc *seats.SeatService
	if cfg.AnthropicKey != "" {
		seatLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			slog.Error("failed to create seat LLM client", "error", err)
			os.Exit(1)
		}
		seatSvc = seats.NewSeatService(seats.SeatServiceConfig{
			DB:            queries,
			EngineFactory: engineFactory,
			Memory:        memEngine,
			LLM:           seatLLM,
			Redis:         redisClient,
		})
		slog.Info("seat service initialized (C-Suite)")
	}

	// Create report collector with default questions (overridden per-remind)
	defaultEngine, _ := engineFactory.ForTenant("inamori", "default")
	collector := report.NewCollector(redisClient, defaultEngine.GetCheckinQuestions())

	// Create Telegram channel adapter (Phase 4: multi-channel foundation)
	tgAdapter, err := channel.NewTelegramAdapter(channel.TelegramConfig{
		Token: cfg.TelegramToken,
	})
	if err != nil {
		slog.Error("failed to create telegram adapter", "error", err)
		os.Exit(1)
	}
	slog.Info("telegram channel adapter created")

	// Signal channel adapter (optional)
	var signalAdapter *channel.SignalAdapter
	if cfg.SignalPhone != "" {
		signalAdapter = channel.NewSignalAdapter(channel.SignalConfig{
			APIURL:      cfg.SignalAPIURL,
			PhoneNumber: cfg.SignalPhone,
			WebhookURL:  "http://brain:8080/api/v1/signal/webhook",
		})
		slog.Info("signal channel adapter created", "phone", cfg.SignalPhone)
	}

	// Create bot wrapper from the adapter's underlying telebot (for command registration)
	tgBot := bot.NewBotFromTelebot(tgAdapter.Bot(), cfg.BossTelegramID, botDB)

	// Slack channel adapter (optional)
	var slackAdapter *channel.SlackAdapter
	if cfg.SlackBotToken != "" {
		var slackErr error
		slackAdapter, slackErr = channel.NewSlackAdapter(channel.SlackConfig{
			BotToken:      cfg.SlackBotToken,
			SigningSecret: cfg.SlackSigningSecret,
		})
		if slackErr != nil {
			slog.Error("failed to create slack adapter", "error", slackErr)
		} else {
			slog.Info("slack channel adapter created")
		}
	}

	// Lark channel adapter (optional)
	var larkAdapter *channel.LarkAdapter
	if cfg.LarkAppID != "" && cfg.LarkAppSecret != "" {
		var larkErr error
		larkAdapter, larkErr = channel.NewLarkAdapter(channel.LarkConfig{
			AppID:     cfg.LarkAppID,
			AppSecret: cfg.LarkAppSecret,
		})
		if larkErr != nil {
			slog.Error("failed to create lark adapter", "error", larkErr)
		} else {
			slog.Info("lark channel adapter created")
		}
	}

	// Create channel router and register adapters
	channelRouter := channel.NewRouter()
	channelRouter.Register(tgAdapter)
	if signalAdapter != nil {
		channelRouter.Register(signalAdapter)
	}
	if slackAdapter != nil {
		channelRouter.Register(slackAdapter)
	}
	if larkAdapter != nil {
		channelRouter.Register(larkAdapter)
	}
	channelSender := channel.NewRouterSender(channelRouter)

	// Create event bus
	eventBus := events.NewBus(rdb)

	// Create chaser, summarizer, trigger checker, action executor, and analyzer
	// All use channel.Sender for channel-agnostic messaging
	chaser := report.NewChaser(reportDB, llmService, channelSender, engineFactory)
	chaser.SetEventBus(eventBus)
	summarizer := report.NewSummarizer(reportDB, llmService)
	triggerChecker := report.NewTriggerChecker(reportDB, channelSender, engineFactory)
	actionExecutor := report.NewActionExecutor(reportDB, channelSender, llmService, engineFactory)
	alertChecker := report.NewAlertChecker(reportDB, channelSender)
	analyzer := report.NewAnalyzer(reportDB, llmService)

	// Create command handler and register commands
	cmdHandler := bot.NewCommandHandler(botDB, nil, nil, cfg.BossTelegramID)
	cmdHandler.SetGroupDB(&groupDBAdapter{q: queries})
	if seatSvc != nil {
		cmdHandler.SetSeatService(&seatServiceAdapter{svc: seatSvc})
	}

	// Wire diagnostics to show scheduler info + current mentor
	startTime := time.Now()
	cmdHandler.DiagnosticsFn = func() string {
		uptime := time.Since(startTime).Round(time.Second)
		aiStatus := "disabled (no API key)"
		if cfg.AnthropicKey != "" {
			aiStatus = "enabled"
		}

		// Look up current mentor
		mentorID := "unknown"
		if tenant, err := botDB.GetTenantByBossChatID(context.Background(), cfg.BossTelegramID); err == nil {
			mentorID = tenant.MentorID
		}

		// Read last run times from Redis
		lastRuns := map[string]string{
			"remind":          "never",
			"chase":           "never",
			"summary":         "never",
			"weekly_actions":  "never",
			"monthly_actions": "never",
		}
		for key := range lastRuns {
			if v, err := rdb.Get(ctx, "scheduler:last_run:"+key).Result(); err == nil && v != "" {
				lastRuns[key] = v
			}
		}

		return fmt.Sprintf(
			"System Diagnostics\n\n"+
				"Uptime: %s\n"+
				"Timezone: %s\n"+
				"AI Features: %s\n"+
				"Active Mentor: %s\n"+
				"Available Mentors: inamori, dalio, grove, ren\n\n"+
				"Last Remind: %s\n"+
				"Last Chase: %s\n"+
				"Last Summary: %s\n"+
				"Last Weekly Actions: %s\n"+
				"Last Monthly Actions: %s",
			uptime, cfg.Timezone, aiStatus, mentorID,
			lastRuns["remind"], lastRuns["chase"], lastRuns["summary"],
			lastRuns["weekly_actions"], lastRuns["monthly_actions"],
		)
	}

	tgBot.RegisterCommands(cmdHandler)

	// Register raw text handler for report collection, mentor chat, and group @mentions
	tgBot.RegisterRawTextHandler(func(c tele.Context) error {
		ctx := context.Background()
		senderID := c.Sender().ID
		text := c.Text()
		sendReply := func(msg string) error { return c.Send(msg) }

		// === GROUP CHAT HANDLING ===
		chatType := string(c.Chat().Type)
		if chatType == "group" || chatType == "supergroup" {
			// Only respond to @mentions
			botUsername := "@" + c.Bot().Me.Username
			if !strings.Contains(text, botUsername) {
				return nil // ignore non-mention messages
			}

			// Strip the @mention from the text
			cleanText := strings.ReplaceAll(text, botUsername, "")
			cleanText = strings.TrimSpace(cleanText)
			if cleanText == "" {
				return c.Reply("有什么我可以帮你的吗？")
			}

			chatID := fmt.Sprintf("%d", c.Chat().ID)
			gc, err := queries.GetGroupChatByPlatformID(ctx, sqlc.GetGroupChatByPlatformIDParams{
				Platform:       "telegram",
				PlatformChatID: chatID,
			})
			if err != nil {
				slog.Debug("group message from unregistered group", "chat_id", chatID)
				return nil
			}
			if !gc.IsActive {
				return nil
			}

			// Load mentor engine
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				slog.Error("group chat: get tenant", "error", err)
				return nil
			}

			engine, err := engineForTenant(engineFactory, tenant, "default")
			if err != nil {
				slog.Error("group chat: load engine", "error", err)
				return nil
			}

			// Get latest summary for team context
			summaryText := ""
			if summary, err := queries.GetLatestSummary(ctx, gc.TenantID); err == nil {
				summaryText = summary.Content
			}

			// Build group reply prompt
			systemPrompt := brain.BuildGroupReplyPrompt(
				engine.MentorName(),
				gc.GroupType,
				summaryText,
				cleanText,
			)

			if chatService == nil || chatService.LLM() == nil {
				return c.Reply(brain.AIDisabledMessage())
			}

			// Use LLM single-turn Chat
			response, err := chatService.LLM().Chat(ctx, systemPrompt, cleanText)
			if err != nil {
				slog.Error("group reply LLM failed", "error", err, "group", gc.Name)
				return nil
			}

			return c.Reply(response)
		}

		// === PRIVATE CHAT HANDLING ===

		// Check if sender is the boss FIRST (boss may not be in employees table)
		if senderID == cfg.BossTelegramID {
			if chatService == nil {
				return sendReply(brain.AIDisabledMessage())
			}

			// C-Suite seat routing: if boss has an active seat via /talk, route to seat chat
			if seatSvc != nil {
				tenant, err := botDB.GetTenantByBossChatID(ctx, senderID)
				if err == nil {
					activeSeat := seatSvc.GetActiveSeat(ctx, tenant.ID, senderID)
					if activeSeat != "" {
						tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
						reply, seatErr := seatSvc.Chat(ctx, tenant.ID, activeSeat, "default", text)
						if seatErr != nil {
							slog.Error("seat chat failed", "seat", activeSeat, "error", seatErr)
							return sendReply("Seat chat failed. Use /talk off to return to default mode.")
						}
						return sendReply(reply)
					}
				}
			}

			tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
			tenant, err := botDB.GetTenantByBossChatID(ctx, senderID)
			if err != nil {
				slog.Error("boss chat: get tenant", "error", err)
				return sendReply(brain.AIErrorMessage())
			}
			bossCtx := fetchBossContext(ctx, queries, tenant.ID, loc)
			resp, err := chatService.HandleBoss(ctx, tenant.ID, tenant.MentorID, "default", text, bossCtx)
			if err != nil {
				slog.Error("boss chat failed", "error", err)
				return sendReply(brain.AIErrorMessage())
			}
			return sendReply(resp)
		}

		// Look up employee by telegram_id
		emp, err := botDB.GetEmployeeByTelegramID(ctx, senderID)
		if err != nil {
			return nil
		}

		empID := emp.ID
		state := collector.GetState(ctx, empID)
		lower := strings.ToLower(strings.TrimSpace(text))

		switch state {
		case report.StateConfirming:
			if lower == "confirm" {
				answers := collector.GetAnswers(ctx, empID)
				cState, msg, err := collector.Confirm(ctx, empID)
				if err != nil {
					slog.Error("confirm report", "employee_id", empID, "error", err)
					return sendReply("Error confirming report. Please try again.")
				}
				if cState == report.StateComplete && answers != nil {
					today := time.Now().In(loc).Format("2006-01-02")
					if err := reportDB.CreateReport(ctx, emp.TenantID, empID, today, answers); err != nil {
						slog.Error("save report", "employee_id", empID, "error", err)
						return sendReply("Report confirmed but failed to save. Please contact your manager.")
					}
					slog.Info("report saved", "employee_id", empID, "date", today)

					// Publish report submitted event
					_ = eventBus.PublishPayload(ctx, events.ReportSubmitted, emp.TenantID, events.ReportSubmittedPayload{
						EmployeeID:   empID,
						EmployeeName: emp.Name,
						ReportDate:   today,
						Channel:      "telegram",
					})

					// Run async blocker/sentiment analysis
					go func(eid, tid, date string) {
						if err := analyzer.Analyze(context.Background(), eid, date); err != nil {
							slog.Error("report analysis failed", "employee_id", eid, "error", err)
						}
					}(empID, emp.TenantID, today)
				}
				return sendReply(msg)
			}
			if lower == "edit" {
				_, firstQ, err := collector.Start(ctx, empID)
				if err != nil {
					return sendReply("Error restarting. Please try again.")
				}
				return sendReply("Let's start over.\n\n" + firstQ)
			}
			return sendReply("Please reply 'confirm' to submit or 'edit' to start over.")

		case report.StateCollecting:
			cState, nextMsg, err := collector.HandleAnswer(ctx, empID, text)
			if err != nil {
				slog.Error("handle answer", "employee_id", empID, "error", err)
				return sendReply("Error processing your answer. Please try again.")
			}
			_ = cState
			if nextMsg != "" {
				return sendReply(nextMsg)
			}

		default:
			// Mentor chat — idle state
			if chatService == nil {
				return nil
			}
			tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				slog.Warn("mentor chat: get tenant", "error", err)
				return nil
			}
			resp, err := chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
				EmployeeID:  empID,
				TenantID:    emp.TenantID,
				Name:        emp.Name,
				MentorID:    tenant.MentorID,
				CultureCode: emp.CultureCode,
				Text:        text,
			})
			if err != nil {
				slog.Error("mentor chat failed", "employee_id", empID, "error", err)
				return nil
			}
			if resp != "" {
				return sendReply(resp)
			}
		}

		return nil
	})

	// Create unified message handler for non-Telegram channels (Slack, Lark)
	unifiedHandler := channel.NewUnifiedHandler(channel.UnifiedHandlerConfig{
		Queries: queries,
		Sender:  channelSender,
		OnText: func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (string, error) {
			state := collector.GetState(ctx, employeeID)
			lower := strings.ToLower(strings.TrimSpace(text))

			switch state {
			case report.StateConfirming:
				if lower == "confirm" {
					answers := collector.GetAnswers(ctx, employeeID)
					cState, msg, err := collector.Confirm(ctx, employeeID)
					if err != nil {
						return "Error confirming report. Please try again.", nil
					}
					if cState == report.StateComplete && answers != nil {
						today := time.Now().In(loc).Format("2006-01-02")
						if err := reportDB.CreateReport(ctx, tenantID, employeeID, today, answers); err != nil {
							return "Report confirmed but failed to save.", nil
						}
						_ = eventBus.PublishPayload(ctx, events.ReportSubmitted, tenantID, events.ReportSubmittedPayload{
							EmployeeID:   employeeID,
							EmployeeName: "",
							ReportDate:   today,
							Channel:      channelType,
						})
						go func() {
							if err := analyzer.Analyze(context.Background(), employeeID, today); err != nil {
								slog.Error("report analysis failed", "employee_id", employeeID, "error", err)
							}
						}()
					}
					return msg, nil
				}
				if lower == "edit" {
					_, firstQ, err := collector.Start(ctx, employeeID)
					if err != nil {
						return "Error restarting. Please try again.", nil
					}
					return "Let's start over.\n\n" + firstQ, nil
				}
				return "Please reply 'confirm' to submit or 'edit' to start over.", nil

			case report.StateCollecting:
				_, nextMsg, err := collector.HandleAnswer(ctx, employeeID, text)
				if err != nil {
					return "Error processing your answer. Please try again.", nil
				}
				return nextMsg, nil

			default:
				// Mentor chat — idle state
				if chatService == nil {
					return "", nil
				}
				tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
				if err != nil {
					return "", nil
				}
				resp, err := chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
					EmployeeID:       employeeID,
					TenantID:         tenantID,
					Name:             empName,
					JobTitle:         empJobTitle,
					Responsibilities: empResponsibilities,
					Country:          empCountry,
					Language:         empLanguage,
					MentorID:         tenant.MentorID,
					CultureCode:      empCultureCode,
					Text:             text,
				})
				if err != nil {
					slog.Error("unified mentor chat failed", "employee_id", employeeID, "error", err)
					return "", nil
				}
				return resp, nil
			}
		},
	})

	// Wire unified handler to Slack and Lark adapters
	if slackAdapter != nil {
		slackAdapter.SetMessageHandler(unifiedHandler.HandleMessage)
	}
	if larkAdapter != nil {
		larkAdapter.SetMessageHandler(unifiedHandler.HandleMessage)
	}

	// Subscribe to events for memory extraction
	if memEngine != nil {
		eventBus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
			var payload events.ReportSubmittedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			reportID, answersJSON, err := reportDB.GetLatestReportByEmployee(ctx, payload.EmployeeID, payload.ReportDate)
			if err != nil {
				slog.Warn("fetch report for memory extraction failed", "error", err)
				return nil // non-fatal
			}
			return memEngine.ExtractFromReport(ctx, memory.ReportInput{
				TenantID:   event.TenantID,
				EmployeeID: payload.EmployeeID,
				ReportID:   reportID,
				Content:    answersJSON,
			})
		})

		eventBus.Subscribe(events.ChaseCompleted, func(ctx context.Context, event events.Event) error {
			var payload events.ChaseCompletedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			return memEngine.ExtractFromChase(ctx, memory.ChaseInput{
				TenantID:   event.TenantID,
				EmployeeID: payload.EmployeeID,
				ChaseLogID: payload.ChaseLogID,
				Step:       payload.Step,
				Action:     payload.Action,
				Message:    payload.Message,
			})
		})

		slog.Info("memory event subscribers registered")
	}

	// Start bot polling in background
	go tgBot.Start()

	// Start event bus listener in background
	go func() {
		if err := eventBus.Listen(ctx); err != nil && err != context.Canceled {
			slog.Error("event bus stopped", "error", err)
		}
	}()

	// Create scheduler callbacks wired to real operations (dynamic mentor)
	callbacks := &schedulerCallbacks{
		remindFn: func(ctx context.Context) error {
			slog.Info("remind job: sending check-in questions")
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			// Get mentor's questions (supports blending)
			engine, err := engineForTenant(engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine for remind: %w", err)
			}
			questions := engine.GetCheckinQuestions()

			emps, err := reportDB.ListActiveEmployees(ctx, tenant.ID)
			if err != nil {
				return fmt.Errorf("list employees: %w", err)
			}
			if len(emps) == 0 {
				slog.Info("remind job: no employees to remind")
				return nil
			}
			for _, emp := range emps {
				_, firstQ, err := collector.StartWithQuestions(ctx, emp.ID, questions)
				if err != nil {
					slog.Error("start collection", "employee_id", emp.ID, "error", err)
					continue
				}
				msg := fmt.Sprintf("Good morning %s! Time for your daily check-in.\n\n%s", emp.Name, firstQ)
				if err := tgBot.SendMessage(emp.TelegramID, msg); err != nil {
					slog.Error("send remind", "employee_id", emp.ID, "error", err)
				}
			}
			slog.Info("remind job: completed", "employees_reminded", len(emps), "mentor", tenant.MentorID)
			return nil
		},
		chaseFn: func(ctx context.Context) error {
			slog.Info("chase job: chasing non-submitters")
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}
			today := time.Now().In(loc).Format("2006-01-02")
			return chaser.ChaseAll(ctx, tenant.ID, today, tenant.MentorID)
		},
		summaryFn: func(ctx context.Context) error {
			slog.Info("summary job: generating daily summary")
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}
			engine, err := engineForTenant(engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine for summary: %w", err)
			}
			today := time.Now().In(loc).Format("2006-01-02")
			result, err := summarizer.Generate(ctx, tenant.ID, today, engine)
			if err != nil {
				return fmt.Errorf("generate summary: %w", err)
			}
			header := fmt.Sprintf("Daily Summary (%s)\nMentor: %s\nSubmission rate: %.0f%%\n\n", today, tenant.MentorID, result.SubmissionRate*100)
			if err := tgBot.SendMessage(cfg.BossTelegramID, header+result.Content); err != nil {
				return fmt.Errorf("send summary to boss: %w", err)
			}
			slog.Info("summary job: completed", "submission_rate", result.SubmissionRate, "mentor", tenant.MentorID)

			// Run trigger rules after summary
			bossEmp := report.EmployeeInfo{
				ID: "boss", Name: "Boss",
				TelegramID:       cfg.BossTelegramID,
				PreferredChannel: "telegram",
			}
			triggerResults, err := triggerChecker.CheckAll(ctx, tenant.ID, tenant.MentorID, bossEmp)
			if err != nil {
				slog.Error("trigger check failed", "error", err)
			} else if len(triggerResults) > 0 {
				slog.Info("triggers fired", "count", len(triggerResults))
			}

			return nil
		},
	}

	// Create scheduler
	sched, err := scheduler.New(cfg.Timezone, redisClient, callbacks)
	if err != nil {
		slog.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}

	// Boss employee info for proactive actions (channel-agnostic)
	bossEmployeeInfo := report.EmployeeInfo{
		ID: "boss", Name: "Boss",
		TelegramID:       cfg.BossTelegramID,
		PreferredChannel: "telegram",
	}

	// Register proactive action jobs
	if err := sched.AddJob("weekly_actions", "0 18 * * 5", func(ctx context.Context) error {
		slog.Info("weekly actions job: running proactive actions")
		tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
		if err != nil {
			return fmt.Errorf("get tenant: %w", err)
		}
		return actionExecutor.RunWeekly(ctx, tenant.ID, tenant.MentorID, bossEmployeeInfo)
	}); err != nil {
		slog.Error("failed to register weekly actions job", "error", err)
		os.Exit(1)
	}

	if err := sched.AddJob("monthly_actions", "0 18 1 * *", func(ctx context.Context) error {
		slog.Info("monthly actions job: running proactive actions")
		tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
		if err != nil {
			return fmt.Errorf("get tenant: %w", err)
		}
		return actionExecutor.RunMonthly(ctx, tenant.ID, tenant.MentorID, bossEmployeeInfo)
	}); err != nil {
		slog.Error("failed to register monthly actions job", "error", err)
		os.Exit(1)
	}
	slog.Info("proactive action jobs registered", "weekly", "Friday 18:00", "monthly", "1st 18:00")

	// Memory consolidation jobs
	if memEngine != nil {
		if err := sched.AddJob("memory-clean", "0 2 * * *", func(ctx context.Context) error {
			slog.Info("memory-clean job: cleaning expired memories")
			return memEngine.RunConsolidation(ctx, memory.ConsolidationClean)
		}); err != nil {
			slog.Error("failed to register memory-clean job", "error", err)
		}

		if err := sched.AddJob("memory-consolidate", "0 3 * * 0", func(ctx context.Context) error {
			slog.Info("memory-consolidate job: merging short-term memories")
			return memEngine.RunConsolidation(ctx, memory.ConsolidationMerge)
		}); err != nil {
			slog.Error("failed to register memory-consolidate job", "error", err)
		}

		if err := sched.AddJob("memory-profiles", "0 4 1 * *", func(ctx context.Context) error {
			slog.Info("memory-profiles job: rebuilding employee profiles")
			return memEngine.RunConsolidation(ctx, memory.ConsolidationRebuild)
		}); err != nil {
			slog.Error("failed to register memory-profiles job", "error", err)
		}

		slog.Info("memory consolidation jobs registered",
			"clean", "daily 02:00",
			"consolidate", "weekly Sunday 03:00",
			"profiles", "monthly 1st 04:00",
		)
	}

	// Group mentor autonomous posting job
	if chatService != nil && chatService.LLM() != nil {
		if err := sched.AddJob("group_mentor", "0 10 * * *", func(ctx context.Context) error {
			slog.Info("group_mentor job: running autonomous posting decisions")
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			var tenantUUID pgtype.UUID
			if err := tenantUUID.Scan(tenant.ID); err != nil {
				return fmt.Errorf("parse tenant UUID: %w", err)
			}

			groups, err := queries.ListActiveGroupChatsByTenant(ctx, tenantUUID)
			if err != nil {
				return fmt.Errorf("list active groups: %w", err)
			}
			if len(groups) == 0 {
				slog.Info("group_mentor job: no active groups")
				return nil
			}

			engine, err := engineForTenant(engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine: %w", err)
			}

			// Collect team data
			today := time.Now().In(loc)
			weekday := today.Weekday().String()
			todayDate := pgtype.Date{Time: today.Truncate(24 * time.Hour), Valid: true}

			submissionRate := "N/A"
			emps, _ := queries.ListActiveEmployees(ctx, tenantUUID)
			if len(emps) > 0 {
				submitted, _ := queries.CountReportsByTenantDate(ctx, sqlc.CountReportsByTenantDateParams{
					TenantID:   tenantUUID,
					ReportDate: todayDate,
				})
				pct := float64(submitted) / float64(len(emps)) * 100
				submissionRate = fmt.Sprintf("%.0f%% (%d/%d)", pct, submitted, len(emps))
			}

			summaryText := ""
			if summary, err := queries.GetLatestSummary(ctx, tenantUUID); err == nil {
				summaryText = summary.Content
				if len(summaryText) > 500 {
					summaryText = summaryText[:500] + "..."
				}
			}

			llmClient := chatService.LLM()

			for _, gc := range groups {
				groupID := formatPgUUID(gc.ID)

				// Anti-spam: check Redis for last post time
				antiSpamKey := fmt.Sprintf("group:last_post:%s", groupID)
				if _, err := redisClient.Get(ctx, antiSpamKey); err == nil {
					slog.Debug("group_mentor: skipping (posted recently)", "group", gc.Name)
					continue
				}

				// Build decision prompt
				prompt := brain.BuildGroupDecisionPrompt(
					engine.MentorName(),
					gc.GroupType,
					brain.GroupTeamData{
						SubmissionRate: submissionRate,
						LatestSummary:  summaryText,
						Weekday:        weekday,
					},
				)

				response, err := llmClient.Chat(ctx, prompt, "Decide whether to post.")
				if err != nil {
					slog.Error("group_mentor: LLM decision failed", "group", gc.Name, "error", err)
					continue
				}

				if brain.IsSkipDecision(response) {
					slog.Debug("group_mentor: AI decided SKIP", "group", gc.Name)
					continue
				}

				// Send message to group
				chatID, _ := strconv.ParseInt(gc.PlatformChatID, 10, 64)
				if chatID == 0 {
					slog.Error("group_mentor: invalid chat ID", "platform_chat_id", gc.PlatformChatID)
					continue
				}

				if err := tgBot.SendMessage(chatID, response); err != nil {
					slog.Error("group_mentor: send failed", "group", gc.Name, "error", err)
					continue
				}

				// Set anti-spam key (24h TTL)
				_ = redisClient.Set(ctx, antiSpamKey, "1", 24*time.Hour)
				slog.Info("group_mentor: posted to group", "group", gc.Name, "message_len", len(response))
			}

			return nil
		}); err != nil {
			slog.Error("failed to register group_mentor job", "error", err)
		} else {
			slog.Info("group_mentor job registered", "schedule", "daily 10:00")
		}
	}

	// Create AI Role Manager (requires LLM + scheduler)
	var roleManager *roles.Manager
	if cfg.AnthropicKey != "" {
		llmClient, _ := brain.NewAnthropicClient(cfg.AnthropicKey)
		bossSender := roles.NewBossSender(channelSender, channel.TypeTelegram, fmt.Sprintf("%d", cfg.BossTelegramID))
		roleManager = roles.NewManager(roles.ManagerConfig{
			Scheduler:     sched,
			EventBus:      eventBus,
			EngineFactory: engineFactory,
			LLM:           llmClient,
			Chaser:        chaser,
			Summarizer:    &roles.SummarizerAdapter{S: summarizer},
			AlertChecker:  &roles.AlertCheckerAdapter{A: alertChecker},
			ActionExec:    &roles.ActionExecAdapter{A: actionExecutor},
			ReportDB:      &roles.ReportDBAdapter{DB: reportDB},
			Queries:       queries,
			Sender:        bossSender,
		})

		// Load existing AI roles for current tenant
		if tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID); err == nil {
			if err := roleManager.LoadExistingForTenant(ctx, tenant.ID); err != nil {
				slog.Error("load existing AI roles", "error", err)
			}
		}
		slog.Info("AI role manager initialized")
	}

	// API router (includes REST API + health check)
	gin.SetMode(gin.ReleaseMode)
	router := api.NewRouter(api.RouterConfig{
		Queries:     queries,
		JWTSecret:   cfg.JWTSecret,
		Redis:       rdb,
		OAuth: api.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURI:  cfg.GoogleRedirectURI,
		},
		Billing: api.BillingConfig{
			SecretKey:     cfg.StripeSecretKey,
			WebhookSecret: cfg.StripeWebhookSecret,
			ProPriceID:    cfg.StripePriceIDPro,
			EntPriceID:    cfg.StripePriceIDEnt,
		},
		OrgWizard:   orgWizard,
		OrgEngine:   orgEngine,
		RoleManager:    roleManager,
		SignalAdapter:  signalAdapter,
		MemoryEngine:  memEngine,
		MemoryStore:   memStore,
		ChannelRouter: channelRouter,
		SlackAdapter:  slackAdapter,
		LarkAdapter:   larkAdapter,
		Scheduler:     sched,
		SeatService:   seatSvc,
	})

	// Health check (public, outside /api/v1)
	router.GET("/healthz", func(c *gin.Context) {
		status := "ok"
		dbStatus := "ok"
		redisStatus := "ok"

		if err := pool.Ping(c.Request.Context()); err != nil {
			dbStatus = fmt.Sprintf("error: %v", err)
			status = "degraded"
		}
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
			status = "degraded"
		}

		code := http.StatusOK
		if status == "degraded" {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, gin.H{
			"status": status,
			"db":     dbStatus,
			"redis":  redisStatus,
		})
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	go func() {
		slog.Info("API server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API server error", "error", err)
		}
	}()

	// Start Signal adapter (if configured)
	if signalAdapter != nil {
		go func() {
			if err := signalAdapter.Start(ctx); err != nil && err != context.Canceled {
				slog.Error("signal adapter error", "error", err)
			}
		}()
	}

	// Start scheduler
	sched.Start(ctx)

	slog.Info("AI Management Brain is running",
		"timezone", cfg.Timezone,
		"port", cfg.Port,
	)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	slog.Info("shutdown signal received", "signal", sig)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	tgBot.Stop()
	if signalAdapter != nil {
		signalAdapter.Stop()
	}
	if slackAdapter != nil {
		slackAdapter.Stop()
	}
	if larkAdapter != nil {
		larkAdapter.Stop()
	}
	if err := sched.Stop(); err != nil {
		slog.Error("scheduler stop error", "error", err)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	slog.Info("AI Management Brain stopped")
}
