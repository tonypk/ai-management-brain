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
	TelegramToken  string
	BossTelegramID int64
	AnthropicKey   string // optional, for system-level use
	Timezone       string
	LogLevel       string
	Port           string
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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
