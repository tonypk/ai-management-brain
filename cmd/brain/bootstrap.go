package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/bot"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/config"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/events"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/onboarding"
	"github.com/tonypk/ai-management-brain/internal/report"
	"github.com/tonypk/ai-management-brain/internal/roles"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
	"github.com/tonypk/ai-management-brain/internal/seats"
	"github.com/tonypk/ai-management-brain/internal/service"
	"github.com/tonypk/ai-management-brain/internal/worldmodel"
)

// services holds all initialized services and dependencies used throughout the application.
type services struct {
	cfg         *config.Config
	loc         *time.Location
	pool        *pgxpool.Pool
	rdb         *redis.Client
	redisClient *redisWrapper

	engineFactory *brain.EngineFactory
	llmService    *brain.LLMService
	chatService   *brain.ChatService
	onboardingSvc *onboarding.Service

	queries  *sqlc.Queries
	botDB    *bot.DBAdapter
	reportDB *report.DBAdapter

	memEngine *memory.MemoryEngine
	memStore  *memory.MemoryStore

	seatSvc         *seats.SeatService
	contextService  *brain.ContextService
	stateEngine     *brain.StateEngine
	execPlanner     *brain.ExecutionPlanner
	incentiveEngine *brain.IncentiveEngine
	recommender     *brain.Recommender

	consultingEngine *brain.ConsultingEngine

	collector *report.Collector

	tgAdapter     *channel.TelegramAdapter
	signalAdapter *channel.SignalAdapter
	slackAdapter  *channel.SlackAdapter
	larkAdapter   *channel.LarkAdapter
	channelRouter *channel.Router
	channelSender *channel.RouterSender

	dispatcher  *brain.Dispatcher
	recFeedback *brain.RecommendationFeedback

	eventBus *events.Bus

	chaser         *report.Chaser
	summarizer     *report.Summarizer
	triggerChecker *report.TriggerChecker
	actionExecutor *report.ActionExecutor
	alertChecker   *report.AlertChecker
	analyzer       *report.Analyzer

	wmExtractor *worldmodel.Extractor
	wmService   *worldmodel.Service
	wmDecay     *worldmodel.DecayRunner
	wmInsights  *worldmodel.InsightsGenerator

	cmdHandler *bot.CommandHandler
	tgBot      *bot.Bot

	orgWizard *brain.OrgWizard
	orgEngine *brain.OrgEngine

	roleManager *roles.Manager

	actionSvc *service.ActionService

	sched *scheduler.Scheduler
}

// setupServices initializes all services and returns them in a services struct.
// The caller is responsible for closing pool and rdb.
func setupServices(cfg *config.Config, logger *slog.Logger) (*services, error) {
	svc := &services{cfg: cfg}

	// Load timezone
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	svc.loc = loc

	ctx := contextForSetup()

	// Connect PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	svc.pool = pool

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	slog.Info("PostgreSQL connected")

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("database migrations applied")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}
	rdb := redis.NewClient(opt)
	svc.rdb = rdb

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	slog.Info("Redis connected")

	svc.redisClient = &redisWrapper{client: rdb}

	// Load industry templates
	if err := brain.LoadIndustries(); err != nil {
		slog.Warn("failed to load industry templates", "error", err)
	}

	// Create engine factory (dynamic mentor+culture per tenant)
	svc.engineFactory = brain.NewEngineFactory()

	// Verify default mentor loads
	if _, err := svc.engineFactory.ForTenant("inamori", "default"); err != nil {
		return nil, fmt.Errorf("load default engine: %w", err)
	}
	slog.Info("engine factory ready", "mentors", "inamori,dalio,grove,ren")

	// Create LLM client (optional)
	if cfg.AnthropicKey != "" {
		llmClient, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			return nil, fmt.Errorf("create Anthropic client: %w", err)
		}
		svc.llmService = brain.NewLLMService(llmClient)
		svc.orgEngine = brain.NewOrgEngine(llmClient)
		svc.orgWizard = brain.NewOrgWizard(llmClient)

		// Create ChatService (uses same llmClient)
		svc.chatService = brain.NewChatService(brain.ChatServiceConfig{
			LLM:           llmClient,
			Redis:         &redisWrapper{client: rdb},
			EngineFactory: svc.engineFactory,
			BossTgID:      cfg.BossTelegramID,
		})

		slog.Info("Anthropic LLM client initialized (org engine + chat ready)")
	} else {
		slog.Warn("ANTHROPIC_API_KEY not set — AI features disabled, using template fallbacks")
	}

	// Create sqlc queries and adapters
	svc.queries = sqlc.New(pool)
	svc.botDB = bot.NewDBAdapter(svc.queries)
	svc.reportDB = report.NewDBAdapter(svc.queries)

	// Initialize memory engine (requires ANTHROPIC_API_KEY; uses free HuggingFace embeddings)
	if cfg.AnthropicKey != "" {
		memLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			return nil, fmt.Errorf("create memory LLM client: %w", err)
		}

		embedder := memory.NewHuggingFaceEmbedder(cfg.EmbeddingModel, cfg.EmbeddingBatch)
		svc.memStore = memory.NewMemoryStore(svc.queries, pool)
		extractor := memory.NewExtractor(memLLM, embedder)
		retriever := memory.NewRetriever(svc.memStore, embedder, cfg.MemoryMaxRecall, cfg.MemoryMaxTokens)
		consolidator := memory.NewConsolidator(svc.memStore, memLLM, embedder, cfg.MemoryConsolidationThreshold)
		profiler := memory.NewProfileBuilder(svc.memStore, memLLM, embedder)
		svc.memEngine = memory.NewMemoryEngine(svc.memStore, embedder, retriever, extractor, consolidator, profiler)

		// Inject memory engine into brain engine factory and chat service
		svc.engineFactory.SetMemoryEngine(svc.memEngine)
		if svc.chatService != nil {
			svc.chatService.SetMemoryEngine(svc.memEngine)
		}

		slog.Info("memory engine enabled", "embedding_model", cfg.EmbeddingModel)
	} else {
		slog.Info("memory engine disabled (no ANTHROPIC_API_KEY)")
	}

	// Seat service (C-Suite)
	if cfg.AnthropicKey != "" {
		seatLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			return nil, fmt.Errorf("create seat LLM client: %w", err)
		}
		svc.seatSvc = seats.NewSeatService(seats.SeatServiceConfig{
			DB:            svc.queries,
			EngineFactory: svc.engineFactory,
			Memory:        svc.memEngine,
			LLM:           seatLLM,
			Redis:         svc.redisClient,
		})
		slog.Info("seat service initialized (C-Suite)")
	}

	// Onboarding service (requires LLM)
	if cfg.AnthropicKey != "" {
		onboardLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			return nil, fmt.Errorf("create onboarding LLM client: %w", err)
		}
		svc.onboardingSvc = onboarding.NewService(svc.queries, rdb, onboardLLM, onboardLLM, onboardLLM, onboardLLM)
		slog.Info("onboarding service initialized")
	}

	// Brain Layer v2: Context Service + State Engine + Execution Planner + Incentive Engine + Recommender
	svc.contextService = brain.NewContextService(svc.queries)
	if svc.memStore != nil {
		svc.contextService.SetMemoryReader(svc.memStore)
	}
	if cfg.AnthropicKey != "" {
		stateLLM, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			return nil, fmt.Errorf("create state engine LLM client: %w", err)
		}
		svc.stateEngine = brain.NewStateEngine(stateLLM, svc.queries)
		svc.execPlanner = brain.NewExecutionPlanner(stateLLM, svc.queries, svc.contextService)
		svc.incentiveEngine = brain.NewIncentiveEngine(stateLLM, svc.queries, svc.contextService)
		svc.recommender = brain.NewRecommender(stateLLM, svc.queries, svc.contextService)
		if svc.memStore != nil {
			memEval := brain.NewMemoryPatternEvaluator(svc.memStore)
			svc.recommender.SetMemoryEvaluator(memEval)
		}
		slog.Info("brain layer v2 engines initialized (state + context + planner + incentive + recommender)")
	}

	// Create consulting engine (requires LLM + dispatcher, wired after dispatcher is created below)
	// consultingEngine is wired later after dispatcher is available

	// Create report collector with default questions (overridden per-remind)
	defaultEngine, _ := svc.engineFactory.ForTenant("inamori", "default")
	svc.collector = report.NewCollector(svc.redisClient, defaultEngine.GetCheckinQuestions())

	// Create Telegram channel adapter (Phase 4: multi-channel foundation)
	svc.tgAdapter, err = channel.NewTelegramAdapter(channel.TelegramConfig{
		Token: cfg.TelegramToken,
	})
	if err != nil {
		return nil, fmt.Errorf("create telegram adapter: %w", err)
	}
	slog.Info("telegram channel adapter created")

	// Signal channel adapter (optional)
	if cfg.SignalPhone != "" {
		svc.signalAdapter = channel.NewSignalAdapter(channel.SignalConfig{
			APIURL:      cfg.SignalAPIURL,
			PhoneNumber: cfg.SignalPhone,
			WebhookURL:  "http://brain:8080/api/v1/signal/webhook",
		})
		slog.Info("signal channel adapter created", "phone", cfg.SignalPhone)
	}

	// Create bot wrapper from the adapter's underlying telebot (for command registration)
	svc.tgBot = bot.NewBotFromTelebot(svc.tgAdapter.Bot(), cfg.BossTelegramID, svc.botDB)

	// Slack channel adapter (optional)
	if cfg.SlackBotToken != "" {
		var slackErr error
		svc.slackAdapter, slackErr = channel.NewSlackAdapter(channel.SlackConfig{
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
	if cfg.LarkAppID != "" && cfg.LarkAppSecret != "" {
		var larkErr error
		svc.larkAdapter, larkErr = channel.NewLarkAdapter(channel.LarkConfig{
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
	svc.channelRouter = channel.NewRouter()
	svc.channelRouter.Register(svc.tgAdapter)
	if svc.signalAdapter != nil {
		svc.channelRouter.Register(svc.signalAdapter)
	}
	if svc.slackAdapter != nil {
		svc.channelRouter.Register(svc.slackAdapter)
	}
	if svc.larkAdapter != nil {
		svc.channelRouter.Register(svc.larkAdapter)
	}
	svc.channelSender = channel.NewRouterSender(svc.channelRouter)

	// Create recommendation dispatcher and feedback loop
	if svc.recommender != nil {
		svc.dispatcher = brain.NewDispatcher(svc.queries, svc.channelSender)
		if svc.memStore != nil {
			svc.recFeedback = brain.NewRecommendationFeedback(svc.memStore)
		}
		slog.Info("recommendation dispatcher initialized")
	}

	// Wire consulting engine now that dispatcher and contextService are available
	if cfg.AnthropicKey != "" && svc.dispatcher != nil {
		consultingLLM, consultingLLMErr := brain.NewAnthropicClient(cfg.AnthropicKey)
		if consultingLLMErr != nil {
			return nil, fmt.Errorf("create consulting engine LLM client: %w", consultingLLMErr)
		}
		svc.consultingEngine = brain.NewConsultingEngine(consultingLLM, svc.contextService, svc.dispatcher, svc.queries, svc.memStore)
		slog.Info("consulting engine enabled")
	}

	// Create event bus
	svc.eventBus = events.NewBus(rdb)

	// Create chaser, summarizer, trigger checker, action executor, and analyzer
	// All use channel.Sender for channel-agnostic messaging
	svc.chaser = report.NewChaser(svc.reportDB, svc.llmService, svc.channelSender, svc.engineFactory)
	svc.chaser.SetEventBus(svc.eventBus)
	svc.summarizer = report.NewSummarizer(svc.reportDB, svc.llmService)
	svc.triggerChecker = report.NewTriggerChecker(svc.reportDB, svc.channelSender, svc.engineFactory)
	if svc.recommender != nil {
		svc.triggerChecker.SetRecommender(svc.recommender)
	}
	svc.actionExecutor = report.NewActionExecutor(svc.reportDB, svc.channelSender, svc.llmService, svc.engineFactory)
	svc.alertChecker = report.NewAlertChecker(svc.reportDB, svc.channelSender)
	svc.analyzer = report.NewAnalyzer(svc.reportDB, svc.llmService)

	// World Model components
	svc.wmExtractor = worldmodel.NewExtractor(svc.llmService, svc.queries)
	svc.wmService = worldmodel.NewService(svc.queries)
	svc.wmDecay = worldmodel.NewDecayRunner(svc.queries)
	svc.wmInsights = worldmodel.NewInsightsGenerator(svc.queries, svc.llmService)

	// Wire World Model context to summarizer
	svc.summarizer.SetWorldModelContextFn(svc.wmService.ForSummaryContext)
	if svc.recommender != nil {
		svc.recommender.SetWorldModelService(svc.wmService)
	}

	// Create command handler and register commands
	svc.cmdHandler = bot.NewCommandHandler(svc.botDB, nil, nil, cfg.BossTelegramID)
	svc.cmdHandler.SetGroupDB(&groupDBAdapter{q: svc.queries})
	if svc.seatSvc != nil {
		svc.cmdHandler.SetSeatService(&seatServiceAdapter{svc: svc.seatSvc})
	}
	if svc.onboardingSvc != nil {
		svc.cmdHandler.SetOnboardingService(&onboardingAdapter{svc: svc.onboardingSvc})
	}
	if svc.consultingEngine != nil {
		svc.cmdHandler.SetConsultingService(&consultingBotAdapter{engine: svc.consultingEngine, queries: svc.queries})
	}

	// Wire diagnostics to show scheduler info + current mentor
	startTime := time.Now()
	svc.cmdHandler.DiagnosticsFn = func() string {
		diagCtx := context.Background()
		uptime := time.Since(startTime).Round(time.Second)
		aiStatus := "disabled (no API key)"
		if cfg.AnthropicKey != "" {
			aiStatus = "enabled"
		}

		// Look up current mentor
		mentorID := "unknown"
		if tenant, err := svc.botDB.GetTenantByBossChatID(diagCtx, cfg.BossTelegramID); err == nil {
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
			if v, err := rdb.Get(diagCtx, "scheduler:last_run:"+key).Result(); err == nil && v != "" {
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

	svc.tgBot.RegisterCommands(svc.cmdHandler)

	return svc, nil
}

// contextForSetup returns a background context for service initialization.
func contextForSetup() context.Context {
	return context.Background()
}
