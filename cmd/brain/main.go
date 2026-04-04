package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/api"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/config"
	"github.com/tonypk/ai-management-brain/internal/events"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/roles"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
	"github.com/tonypk/ai-management-brain/internal/service"
)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize all services
	svc, err := setupServices(cfg, logger)
	if err != nil {
		slog.Error("failed to setup services", "error", err)
		os.Exit(1)
	}
	defer svc.pool.Close()
	defer svc.rdb.Close()

	// Register Telegram raw text handler for report collection, mentor chat, and group @mentions
	registerTelegramTextHandler(svc)

	// Create unified message handler for non-Telegram channels (Slack, Lark)
	unifiedHandler := createUnifiedHandler(svc)

	// Wire unified handler to Slack and Lark adapters
	if svc.slackAdapter != nil {
		svc.slackAdapter.SetMessageHandler(unifiedHandler.HandleMessage)
	}
	if svc.larkAdapter != nil {
		svc.larkAdapter.SetMessageHandler(unifiedHandler.HandleMessage)
	}

	// Subscribe to events for memory extraction
	registerEventSubscribers(svc)

	// Start bot polling in background
	go svc.tgBot.Start()

	// Start event bus listener in background
	go func() {
		if err := svc.eventBus.Listen(ctx); err != nil && err != context.Canceled {
			slog.Error("event bus stopped", "error", err)
		}
	}()

	// Create scheduler callbacks wired to real operations (dynamic mentor)
	callbacks := createSchedulerCallbacks(svc)

	// Create scheduler
	sched, err := scheduler.New(cfg.Timezone, svc.redisClient, callbacks)
	if err != nil {
		slog.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}
	svc.sched = sched

	// Register all scheduled jobs
	registerSchedulerJobs(svc, sched)

	// Create AI Role Manager (requires LLM + scheduler)
	if cfg.AnthropicKey != "" {
		llmClient, _ := brain.NewAnthropicClient(cfg.AnthropicKey)
		bossSender := roles.NewBossSender(svc.channelSender, channel.TypeTelegram, fmt.Sprintf("%d", cfg.BossTelegramID))
		svc.roleManager = roles.NewManager(roles.ManagerConfig{
			Scheduler:     sched,
			EventBus:      svc.eventBus,
			EngineFactory: svc.engineFactory,
			LLM:           llmClient,
			Chaser:        svc.chaser,
			Summarizer:    &roles.SummarizerAdapter{S: svc.summarizer},
			AlertChecker:  &roles.AlertCheckerAdapter{A: svc.alertChecker},
			ActionExec:    &roles.ActionExecAdapter{A: svc.actionExecutor},
			ReportDB:      &roles.ReportDBAdapter{DB: svc.reportDB},
			Queries:       svc.queries,
			Sender:        bossSender,
		})

		// Load existing AI roles for current tenant
		if tenant, err := svc.botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID); err == nil {
			if err := svc.roleManager.LoadExistingForTenant(ctx, tenant.ID); err != nil {
				slog.Error("load existing AI roles", "error", err)
			}
		}
		slog.Info("AI role manager initialized")
	}

	// Action service (write operations for OpenClaw MCP)
	svc.actionSvc = service.NewActionService(service.ActionServiceConfig{
		Queries:    svc.queries,
		Collector:  svc.collector,
		Chaser:     svc.chaser,
		Summarizer: svc.summarizer,
		Sender:     svc.channelSender,
		Factory:    svc.engineFactory,
		ReportDB:   svc.reportDB,
		Timezone:   svc.loc,
	})

	// API router (includes REST API + health check)
	gin.SetMode(gin.ReleaseMode)
	router := api.NewRouter(api.RouterConfig{
		Queries:   svc.queries,
		JWTSecret: cfg.JWTSecret,
		Redis:     svc.rdb,
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
		OrgWizard:         svc.orgWizard,
		OrgEngine:         svc.orgEngine,
		RoleManager:       svc.roleManager,
		SignalAdapter:     svc.signalAdapter,
		MemoryEngine:      svc.memEngine,
		MemoryStore:       svc.memStore,
		ChannelRouter:     svc.channelRouter,
		SlackAdapter:      svc.slackAdapter,
		LarkAdapter:       svc.larkAdapter,
		Scheduler:         sched,
		SeatService:       svc.seatSvc,
		ActionService:     svc.actionSvc,
		Recommender:       svc.recommender,
		Dispatcher:        svc.dispatcher,
		RecFeedback:       svc.recFeedback,
		ContextService:    svc.contextService,
		ExecPlanner:       svc.execPlanner,
		IncentiveEngine:   svc.incentiveEngine,
		ConsultingEngine:  svc.consultingEngine,
		WorldModelService: svc.wmService,
	})

	// Health check (public, outside /api/v1)
	router.GET("/healthz", func(c *gin.Context) {
		status := "ok"
		dbStatus := "ok"
		redisStatus := "ok"

		if err := svc.pool.Ping(c.Request.Context()); err != nil {
			dbStatus = fmt.Sprintf("error: %v", err)
			status = "degraded"
		}
		if err := svc.rdb.Ping(c.Request.Context()).Err(); err != nil {
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
	if svc.signalAdapter != nil {
		go func() {
			if err := svc.signalAdapter.Start(ctx); err != nil && err != context.Canceled {
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

	svc.tgBot.Stop()
	if svc.signalAdapter != nil {
		svc.signalAdapter.Stop()
	}
	if svc.slackAdapter != nil {
		svc.slackAdapter.Stop()
	}
	if svc.larkAdapter != nil {
		svc.larkAdapter.Stop()
	}
	if err := sched.Stop(); err != nil {
		slog.Error("scheduler stop error", "error", err)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	slog.Info("AI Management Brain stopped")
}

// registerEventSubscribers registers all event bus subscribers.
func registerEventSubscribers(svc *services) {
	// Subscribe to events for memory extraction
	if svc.memEngine != nil {
		svc.eventBus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
			var payload events.ReportSubmittedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			reportID, answersJSON, err := svc.reportDB.GetLatestReportByEmployee(ctx, payload.EmployeeID, payload.ReportDate)
			if err != nil {
				slog.Warn("fetch report for memory extraction failed", "error", err)
				return nil // non-fatal
			}
			if err := svc.memEngine.ExtractFromReport(ctx, memory.ReportInput{
				TenantID:   event.TenantID,
				EmployeeID: payload.EmployeeID,
				ReportID:   reportID,
				Content:    answersJSON,
			}); err != nil {
				return err
			}
			// Trigger memory-based recommendation evaluation
			if svc.recommender != nil {
				var tenantUUID pgtype.UUID
				_ = tenantUUID.Scan(event.TenantID)
				var empUUID pgtype.UUID
				_ = empUUID.Scan(payload.EmployeeID)
				_ = svc.recommender.RealtimeEvaluate(ctx, tenantUUID, "memory_extraction_complete", payload.EmployeeName, empUUID, nil)
			}
			return nil
		})

		svc.eventBus.Subscribe(events.ChaseCompleted, func(ctx context.Context, event events.Event) error {
			var payload events.ChaseCompletedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			return svc.memEngine.ExtractFromChase(ctx, memory.ChaseInput{
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

	// Brain Layer v2: StateEngine event subscriber — extract communication events from reports
	if svc.stateEngine != nil {
		svc.eventBus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
			var payload events.ReportSubmittedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			reportID, answersJSON, err := svc.reportDB.GetLatestReportByEmployee(ctx, payload.EmployeeID, payload.ReportDate)
			if err != nil {
				slog.Warn("state_engine: fetch report for event extraction failed", "error", err)
				return nil
			}
			var answers map[string]string
			if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
				slog.Warn("state_engine: parse answers failed", "error", err)
				return nil
			}
			var tenantUUID pgtype.UUID
			_ = tenantUUID.Scan(event.TenantID)
			var reportUUID pgtype.UUID
			_ = reportUUID.Scan(reportID)
			_, err = svc.stateEngine.ExtractEventsFromReport(ctx, tenantUUID, reportUUID, payload.EmployeeName, answers)
			if err != nil {
				slog.Warn("state_engine: extract events failed", "error", err)
			}
			return nil
		})
		slog.Info("state engine event subscriber registered")
	}
}
