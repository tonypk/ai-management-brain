package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/config"
	"github.com/tonypk/ai-management-brain/internal/report"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

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

	// Load brain engine (default mentor + culture)
	engine, err := brain.NewEngine("inamori", "default")
	if err != nil {
		slog.Error("failed to create brain engine", "error", err)
		os.Exit(1)
	}
	slog.Info("brain engine loaded", "mentor", "inamori", "culture", "default")

	// Create LLM client (optional)
	var llmService *brain.LLMService
	if cfg.AnthropicKey != "" {
		llmClient, err := brain.NewAnthropicClient(cfg.AnthropicKey)
		if err != nil {
			slog.Error("failed to create Anthropic client", "error", err)
			os.Exit(1)
		}
		llmService = brain.NewLLMService(llmClient)
		slog.Info("Anthropic LLM client initialized")
	} else {
		slog.Warn("ANTHROPIC_API_KEY not set — AI features disabled, using template fallbacks")
	}

	// Create report collector
	collector := report.NewCollector(redisClient, engine.GetCheckinQuestions())
	_ = collector // Will be wired to bot handler in full integration

	// Create chaser and summarizer (nil DB adapter for Phase 1)
	chaser := report.NewChaser(nil, llmService, nil, engine)
	summarizer := report.NewSummarizer(nil, llmService, engine)

	// Create scheduler callbacks wired to chaser/summarizer
	callbacks := &schedulerCallbacks{
		remindFn: func(ctx context.Context) error {
			slog.Info("remind job triggered")
			return nil
		},
		chaseFn: func(ctx context.Context) error {
			slog.Info("chase job triggered")
			_ = chaser // chaser available for wiring
			return nil
		},
		summaryFn: func(ctx context.Context) error {
			slog.Info("summary job triggered")
			_ = summarizer // summarizer available for wiring
			return nil
		},
	}

	// Create scheduler
	sched, err := scheduler.New(cfg.Timezone, redisClient, callbacks)
	if err != nil {
		slog.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}

	// Health check HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

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
		slog.Info("health check server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
		}
	}()

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

	if err := sched.Stop(); err != nil {
		slog.Error("scheduler stop error", "error", err)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	slog.Info("AI Management Brain stopped")
}
