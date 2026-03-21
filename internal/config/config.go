package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL    string
	RedisURL       string
	EncryptionKey  []byte // 32 bytes for AES-256
	JWTSecret      []byte // 32 bytes for HMAC-SHA256
	TelegramToken  string
	BossTelegramID int64
	AnthropicKey   string // optional, for system-level use
	Timezone       string
	LogLevel       string
	Port           string

	// Google OAuth (optional)
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	// Stripe billing (optional)
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceIDPro    string
	StripePriceIDEnt    string

	// Signal (optional)
	SignalPhone  string // registered phone number, e.g. "+639123456789"
	SignalAPIURL string // signal-cli-rest-api URL, e.g. "http://signal-cli:8080"

	// Embedding (free HuggingFace Inference API)
	EmbeddingModel string
	EmbeddingBatch int

	// Memory Engine
	MemoryMaxRecall              int
	MemoryMaxTokens              int
	MemoryShortTermDays          int
	MemoryConsolidationThreshold float64
	MemoryMaxPerTenant           int
}

func Load() (*Config, error) {
	cfg := &Config{
		Timezone: getEnv("TIMEZONE", "Asia/Singapore"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("PORT", "8080"),
	}

	// Validate timezone
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return nil, fmt.Errorf("invalid TIMEZONE %q: %w", cfg.Timezone, err)
	}

	var missing []string

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	cfg.RedisURL = os.Getenv("REDIS_URL")
	if cfg.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}

	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		missing = append(missing, "ENCRYPTION_KEY")
	} else {
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("ENCRYPTION_KEY must be 64 hex chars (32 bytes)")
		}
		cfg.EncryptionKey = key
	}

	jwtHex := os.Getenv("JWT_SECRET")
	if jwtHex == "" {
		missing = append(missing, "JWT_SECRET")
	} else {
		secret, err := hex.DecodeString(jwtHex)
		if err != nil || len(secret) != 32 {
			return nil, fmt.Errorf("JWT_SECRET must be 64 hex chars (32 bytes)")
		}
		cfg.JWTSecret = secret
	}

	cfg.TelegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if cfg.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_BOT_TOKEN")
	}

	bossID := os.Getenv("BOSS_TELEGRAM_ID")
	if bossID == "" {
		missing = append(missing, "BOSS_TELEGRAM_ID")
	} else {
		id, err := strconv.ParseInt(bossID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("BOSS_TELEGRAM_ID must be a number: %w", err)
		}
		cfg.BossTelegramID = id
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	cfg.AnthropicKey = os.Getenv("ANTHROPIC_API_KEY")

	// Google OAuth (optional)
	cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	cfg.GoogleRedirectURI = getEnv("GOOGLE_REDIRECT_URI", "http://localhost:8080/auth/callback")

	// Stripe billing (optional)
	cfg.StripeSecretKey = os.Getenv("STRIPE_SECRET_KEY")
	cfg.StripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	cfg.StripePriceIDPro = os.Getenv("STRIPE_PRICE_ID_PRO")
	cfg.StripePriceIDEnt = os.Getenv("STRIPE_PRICE_ID_ENT")

	// Signal (optional)
	cfg.SignalPhone = os.Getenv("SIGNAL_PHONE")
	cfg.SignalAPIURL = getEnv("SIGNAL_API_URL", "http://signal-cli:8080")

	// Embedding model (free HuggingFace Inference API)
	cfg.EmbeddingModel = getEnv("EMBEDDING_MODEL", "sentence-transformers/all-MiniLM-L6-v2")
	cfg.EmbeddingBatch = getEnvInt("EMBEDDING_BATCH_SIZE", 32)

	// Memory Engine defaults
	cfg.MemoryMaxRecall = getEnvInt("MEMORY_MAX_RECALL", 5)
	cfg.MemoryMaxTokens = getEnvInt("MEMORY_MAX_TOKENS", 800)
	cfg.MemoryShortTermDays = getEnvInt("MEMORY_SHORT_TERM_DAYS", 30)
	cfg.MemoryConsolidationThreshold = getEnvFloat("MEMORY_CONSOLIDATION_THRESHOLD", 0.85)
	cfg.MemoryMaxPerTenant = getEnvInt("MEMORY_MAX_PER_TENANT", 20000)

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
