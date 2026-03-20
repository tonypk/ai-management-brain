package config_test

import (
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/config"
)

func TestLoad_RequiredFields(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_SECRET", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:ABC")
	t.Setenv("BOSS_TELEGRAM_ID", "999")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("got DatabaseURL=%q", cfg.DatabaseURL)
	}
	if cfg.BossTelegramID != 999 {
		t.Errorf("got BossTelegramID=%d", cfg.BossTelegramID)
	}
	if cfg.Timezone != "Asia/Singapore" {
		t.Errorf("default timezone should be Asia/Singapore, got %q", cfg.Timezone)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "ENCRYPTION_KEY", "JWT_SECRET", "TELEGRAM_BOT_TOKEN", "BOSS_TELEGRAM_ID"} {
		t.Setenv(key, "")
	}
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoad_InvalidTimezone(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_SECRET", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:ABC")
	t.Setenv("BOSS_TELEGRAM_ID", "999")
	t.Setenv("TIMEZONE", "Invalid/Zone")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestLoad_ValidTimezone(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_SECRET", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:ABC")
	t.Setenv("BOSS_TELEGRAM_ID", "999")
	t.Setenv("TIMEZONE", "America/New_York")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cfg.Timezone != "America/New_York" {
		t.Errorf("timezone = %q", cfg.Timezone)
	}
	_ = time.Now()
}
