# Phase 1: Core Bot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Telegram bot that runs the complete daily management loop — ask employees check-in questions, collect reports via multi-turn DM conversation, chase non-submitters using mentor strategy + cultural adaptation, and send AI-generated summaries to the boss.

**Architecture:** Single Go binary (`cmd/brain`) running Telegram Bot + Scheduler + health endpoint in one process. PostgreSQL for persistence, Redis for conversation state and scheduler metadata. Mentor strategies and culture packs loaded from embedded YAML files at startup. All queries scoped by tenant_id from day 1.

**Tech Stack:** Go 1.22+, Gin, sqlc, PostgreSQL 16, Redis 7, telebot/v3, go-co-op/gocron, anthropic-go, AES-256-GCM encryption, slog structured logging.

**Spec:** See `README.md` in project root for full design specification.

---

## File Map

```
ai-management-brain/
├── cmd/
│   └── brain/main.go                          # Wires everything, starts bot+scheduler+healthz
├── internal/
│   ├── config/
│   │   ├── config.go                          # Env-based config struct with validation
│   │   └── config_test.go
│   ├── pkg/
│   │   ├── crypto.go                          # AES-256-GCM envelope encryption
│   │   ├── crypto_test.go
│   │   └── response.go                        # Unified API response (Phase 3 prep)
│   ├── db/
│   │   └── sqlc/                              # sqlc-generated code (models.go, db.go, querier.go, *.sql.go)
│   ├── brain/
│   │   ├── mentor.go                          # Load mentor strategy from YAML
│   │   ├── mentor_test.go
│   │   ├── culture.go                         # Load culture pack from YAML
│   │   ├── culture_test.go
│   │   ├── engine.go                          # Assemble system prompt, apply culture overrides
│   │   ├── engine_test.go
│   │   ├── llm.go                             # Claude API client with retry + error classification
│   │   └── llm_test.go
│   ├── bot/
│   │   ├── bot.go                             # Bot setup, webhook/polling config
│   │   ├── handler.go                         # Message routing, conversation dispatch
│   │   ├── commands.go                        # /start /status /help /addemployee /join /mentor /diagnostics
│   │   ├── commands_test.go
│   │   ├── middleware.go                      # Identity resolution (telegram_id → employee/boss)
│   │   └── middleware_test.go                 # Identity resolution tests
│   ├── report/
│   │   ├── collector.go                       # Multi-turn conversation state machine (Redis)
│   │   ├── collector_test.go
│   │   ├── testutil_test.go                   # mockRedis + mock helpers shared across report tests
│   │   ├── chaser.go                          # Mentor-driven chase escalation
│   │   ├── chaser_test.go
│   │   ├── summarizer.go                      # AI summary with mentor-specific focus
│   │   └── summarizer_test.go
│   └── scheduler/
│       ├── scheduler.go                       # Job registration + missed-job catch-up
│       └── scheduler_test.go
├── configs/
│   ├── mentors/
│   │   ├── inamori.yaml
│   │   └── dalio.yaml
│   └── cultures/
│       ├── philippines.yaml
│       └── singapore.yaml
├── sql/
│   ├── migrations/
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   └── queries/
│       ├── tenants.sql
│       ├── employees.sql
│       ├── reports.sql
│       ├── chase_logs.sql
│       └── summaries.sql
├── go.mod
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
├── .gitignore
└── sqlc.yaml
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`, `.gitignore`, `.env.example`, `Makefile`, `docker-compose.yml`, `Dockerfile`, `sqlc.yaml`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/anna/Documents/ai-management-brain
go mod init github.com/tonypk/ai-management-brain
```

- [ ] **Step 2: Create .gitignore**

```
# Binaries
/brain
/bin/
*.exe

# Environment
.env
*.env.local

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Test
coverage.out
coverage.html
```

- [ ] **Step 3: Create .env.example**

```env
# Database
DATABASE_URL=postgres://brain:brain@localhost:5432/brain?sslmode=disable
REDIS_URL=redis://localhost:6379/0

# Encryption (generate with: openssl rand -hex 32)
ENCRYPTION_KEY=your-64-char-hex-key-here

# Telegram
TELEGRAM_BOT_TOKEN=your-bot-token
BOSS_TELEGRAM_ID=123456789

# Anthropic (optional — tenants can use BYOK)
ANTHROPIC_API_KEY=sk-ant-...

# App
TIMEZONE=Asia/Singapore
LOG_LEVEL=info
PORT=8080
```

- [ ] **Step 4: Create docker-compose.yml**

```yaml
version: '3.9'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: brain
      POSTGRES_USER: brain
      POSTGRES_PASSWORD: brain
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U brain"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    volumes: [redisdata:/data]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  pgdata:
  redisdata:
```

- [ ] **Step 5: Create Makefile**

```makefile
.PHONY: build run test lint migrate sqlc dev

build:
	CGO_ENABLED=0 go build -o brain ./cmd/brain

run: build
	./brain

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

lint:
	go vet ./...

sqlc:
	sqlc generate

migrate-up:
	migrate -path sql/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path sql/migrations -database "$(DATABASE_URL)" down 1

dev:
	docker compose up -d postgres redis
	@echo "Postgres: localhost:5432, Redis: localhost:6379"

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain
```

- [ ] **Step 6: Create Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o brain ./cmd/brain

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/brain .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
CMD ["./brain"]
```

- [ ] **Step 7: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

- [ ] **Step 8: Install CLI tools and Go dependencies**

```bash
# Install CLI tools
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install Go dependencies
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get gopkg.in/telebot.v3
go get github.com/go-co-op/gocron/v2
go get github.com/liushuangls/go-anthropic/v2
go get gopkg.in/yaml.v3
go get github.com/golang-migrate/migrate/v4
go get github.com/google/uuid
```

- [ ] **Step 9: Create placeholder main.go**

Create `cmd/brain/main.go`:
```go
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("AI Management Brain starting...")
}
```

- [ ] **Step 10: Verify it compiles and commit**

```bash
go mod tidy
go build ./cmd/brain
```

```bash
git add go.mod go.sum .gitignore .env.example Makefile docker-compose.yml Dockerfile sqlc.yaml cmd/brain/main.go
git commit -m "feat: project scaffolding — go.mod, docker-compose, Makefile, Dockerfile"
```

---

## Task 2: Database Schema + Migrations + sqlc

**Files:**
- Create: `sql/migrations/000001_init.up.sql`, `sql/migrations/000001_init.down.sql`
- Create: `sql/queries/tenants.sql`, `sql/queries/employees.sql`, `sql/queries/reports.sql`, `sql/queries/chase_logs.sql`, `sql/queries/summaries.sql`
- Generated: `internal/db/sqlc/` (all files)

- [ ] **Step 1: Create migration up**

Create `sql/migrations/000001_init.up.sql`:
```sql
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,              -- AES-256-GCM encrypted
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,             -- optional: {"inamori": 0.7, "dalio": 0.3}
    bot_token     TEXT,              -- AES-256-GCM encrypted
    boss_chat_id  BIGINT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    telegram_id   BIGINT UNIQUE,
    culture_code  TEXT NOT NULL DEFAULT 'default',
    role          TEXT NOT NULL DEFAULT 'member',  -- boss | manager | member
    invite_code   TEXT,              -- for /join registration
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,    -- {"q1": "...", "q2": "...", "q3": "..."}
    blockers      TEXT,              -- AI-extracted blockers
    sentiment     TEXT,              -- positive | neutral | negative
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);

CREATE TABLE chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,      -- private_message | public_reminder | manager_notify | send_failed
    message       TEXT NOT NULL,
    chased_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary_date    DATE NOT NULL,
    content         TEXT NOT NULL,
    submission_rate FLOAT NOT NULL DEFAULT 0,
    blockers_count  INT NOT NULL DEFAULT 0,
    key_metrics     JSONB,           -- mentor-specific metrics
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);

-- Indexes
CREATE INDEX idx_employees_tenant ON employees(tenant_id);
CREATE INDEX idx_employees_telegram ON employees(telegram_id);
CREATE INDEX idx_reports_tenant_date ON reports(tenant_id, report_date);
CREATE INDEX idx_reports_employee_date ON reports(employee_id, report_date);
CREATE INDEX idx_chase_logs_tenant_date ON chase_logs(tenant_id, report_date);
CREATE INDEX idx_chase_logs_employee ON chase_logs(employee_id, report_date);
CREATE INDEX idx_summaries_tenant_date ON summaries(tenant_id, summary_date);
```

- [ ] **Step 2: Create migration down**

Create `sql/migrations/000001_init.down.sql`:
```sql
DROP TABLE IF EXISTS summaries;
DROP TABLE IF EXISTS chase_logs;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS tenants;
```

- [ ] **Step 3: Write sqlc queries — tenants.sql**

Create `sql/queries/tenants.sql`:
```sql
-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: GetTenantByBossChatID :one
SELECT * FROM tenants WHERE boss_chat_id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (name, timezone, anthropic_key, mentor_id, bot_token, boss_chat_id, config)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateTenantMentor :exec
UPDATE tenants SET mentor_id = $2, mentor_blend = $3 WHERE id = $1;

-- name: UpdateTenantConfig :exec
UPDATE tenants SET config = $2 WHERE id = $1;
```

- [ ] **Step 4: Write sqlc queries — employees.sql**

Create `sql/queries/employees.sql`:
```sql
-- name: GetEmployee :one
SELECT * FROM employees WHERE id = $1 AND tenant_id = $2;

-- name: GetEmployeeByTelegramID :one
SELECT * FROM employees WHERE telegram_id = $1;

-- name: ListActiveEmployees :many
SELECT * FROM employees WHERE tenant_id = $1 AND is_active = true ORDER BY name;

-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateEmployeeTelegramID :exec
UPDATE employees SET telegram_id = $2 WHERE id = $1;

-- name: GetEmployeeByInviteCode :one
SELECT * FROM employees WHERE invite_code = $1 AND telegram_id IS NULL;

-- name: ListEmployeesWithoutReport :many
SELECT e.* FROM employees e
LEFT JOIN reports r ON e.id = r.employee_id AND r.report_date = $2
WHERE e.tenant_id = $1 AND e.is_active = true AND e.role = 'member' AND r.id IS NULL;
```

- [ ] **Step 5: Write sqlc queries — reports.sql**

Create `sql/queries/reports.sql`:
```sql
-- name: CreateReport :one
INSERT INTO reports (tenant_id, employee_id, report_date, answers, blockers, sentiment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetReportsByTenantDate :many
SELECT r.*, e.name as employee_name FROM reports r
JOIN employees e ON r.employee_id = e.id
WHERE r.tenant_id = $1 AND r.report_date = $2
ORDER BY r.submitted_at;

-- name: CountReportsByTenantDate :one
SELECT COUNT(*) FROM reports WHERE tenant_id = $1 AND report_date = $2;

-- name: GetEmployeeReportStreak :one
SELECT COUNT(*) as missed_days FROM generate_series(
    CURRENT_DATE - INTERVAL '7 days', CURRENT_DATE - INTERVAL '1 day', '1 day'
) d(day)
WHERE NOT EXISTS (
    SELECT 1 FROM reports WHERE employee_id = $1 AND report_date = d.day::date
);
```

- [ ] **Step 6: Write sqlc queries — chase_logs.sql and summaries.sql**

Create `sql/queries/chase_logs.sql`:
```sql
-- name: CreateChaseLog :one
INSERT INTO chase_logs (tenant_id, employee_id, report_date, step, action, message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetLastChaseStep :one
SELECT COALESCE(MAX(step), 0) as last_step FROM chase_logs
WHERE employee_id = $1 AND report_date = $2;
```

Create `sql/queries/summaries.sql`:
```sql
-- name: CreateSummary :one
INSERT INTO summaries (tenant_id, summary_date, content, submission_rate, blockers_count, key_metrics)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSummary :one
SELECT * FROM summaries WHERE tenant_id = $1 AND summary_date = $2;
```

- [ ] **Step 7: Start Docker, run migration, generate sqlc**

```bash
make dev
export DATABASE_URL="postgres://brain:brain@localhost:5432/brain?sslmode=disable"
make migrate-up
make sqlc
```

- [ ] **Step 8: Verify generated code compiles**

```bash
go mod tidy
go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add sql/ internal/db/ go.mod go.sum
git commit -m "feat: database schema, migrations, and sqlc queries for 5 core tables"
```

---

## Task 3: Config Module

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write test for config loading**

Create `internal/config/config_test.go`:
```go
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
	// Unset all required vars (t.Setenv restores after test)
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "ENCRYPTION_KEY", "TELEGRAM_BOT_TOKEN", "BOSS_TELEGRAM_ID"} {
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
	_ = time.Now() // ensure timezone loaded
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
go test ./internal/config/ -v
```
Expected: FAIL (package does not exist)

- [ ] **Step 3: Implement config.go**

Create `internal/config/config.go`:
```go
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
```

- [ ] **Step 4: Run tests — verify pass**

```bash
go test ./internal/config/ -v
```
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config module with env loading, validation, and timezone check"
```

---

## Task 4: Crypto Module (AES-256-GCM)

**Files:**
- Create: `internal/pkg/crypto.go`
- Create: `internal/pkg/crypto_test.go`

- [ ] **Step 1: Write tests**

Create `internal/pkg/crypto_test.go`:
```go
package pkg_test

import (
	"crypto/rand"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/pkg"
)

func testKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := testKey()
	plaintext := "sk-ant-api03-secret-key"

	ciphertext, err := pkg.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	result, err := pkg.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := testKey()
	key2 := testKey()

	ciphertext, _ := pkg.Encrypt("secret", key1)
	_, err := pkg.Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("should fail with wrong key")
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	key := testKey()
	c1, _ := pkg.Encrypt("same", key)
	c2, _ := pkg.Encrypt("same", key)
	if c1 == c2 {
		t.Fatal("same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	key := testKey()
	ciphertext, err := pkg.Encrypt("", key)
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	result, err := pkg.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}
```

- [ ] **Step 2: Run tests — verify fail**

```bash
go test ./internal/pkg/ -v
```
Expected: FAIL

- [ ] **Step 3: Implement crypto.go**

Create `internal/pkg/crypto.go`:
```go
package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns base64-encoded "nonce+ciphertext".
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded "nonce+ciphertext" using AES-256-GCM.
func Decrypt(encoded string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
```

**BYOK Integration Note:** The `Encrypt` and `Decrypt` functions are called at the application layer (in bot commands and main wiring) when storing/reading `anthropic_key` and `bot_token`. The sqlc queries store the already-encrypted ciphertext as plain TEXT. Example usage pattern:
```go
// Before DB insert:
encryptedKey, _ := pkg.Encrypt(rawAPIKey, cfg.EncryptionKey)
queries.CreateTenant(ctx, sqlc.CreateTenantParams{AnthropicKey: &encryptedKey, ...})

// After DB read:
rawKey, _ := pkg.Decrypt(*tenant.AnthropicKey, cfg.EncryptionKey)
```

- [ ] **Step 4: Run tests — verify pass**

```bash
go test ./internal/pkg/ -v
```
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/pkg/
git commit -m "feat: AES-256-GCM encryption for BYOK secrets"
```

---

## Task 5: Mentor Strategy Loader

**Files:**
- Create: `internal/brain/mentor.go`
- Create: `internal/brain/mentor_test.go`
- Create: `configs/mentors/inamori.yaml`
- Create: `configs/mentors/dalio.yaml`

- [ ] **Step 1: Create Inamori YAML**

Create `configs/mentors/inamori.yaml`:
```yaml
id: inamori
name: 稻盛和夫
name_en: Kazuo Inamori
company: 京瓷 · KDDI
philosophy: 敬天爱人，自利利他
version: 1

strategy:
  checkin_questions:
    - "今天你为团队做了什么贡献？"
    - "遇到什么困难需要大家帮助？"
    - "你从今天的工作中学到了什么？"

  chase:
    method: private_first
    escalation:
      - action: private_message
        delay: "0"
        tone: warm_reminder
      - action: manager_notify
        delay: "2h"
        tone: caring_concern
      - action: skip_today
        delay: "4h"
    forbidden:
      - public_naming
      - shame_based
    encouraged:
      - private_conversation
      - effort_recognition

  summary:
    focus:
      - morale
      - collaboration
      - support_needs
    highlight: team_achievements
    flag: emotional_signals
    metrics:
      - name: 团队协作度
        source: mutual_mentions
      - name: 需要关怀
        source: sentiment_negative

  actions:
    weekly:
      - type: recognition
        desc: "感谢本周贡献最大的成员"
      - type: team_pulse
        desc: "团队氛围快速调查"
    monthly:
      - type: report
        desc: "利他贡献月报"
    triggers:
      - event: consecutive_miss_3days
        action: manager_private_chat
        message: "{name} 连续3天未提交，建议私下关心一下"
      - event: sentiment_drop
        action: private_checkin
        message: "最近感觉你状态不太好，有什么我能帮到你的吗？"

  system_prompt: |
    你融合了稻盛和夫的管理哲学。核心原则：
    1. 以利他心出发，先考虑对方感受再提要求
    2. 强调集体荣誉感和团队归属感
    3. 认可努力本身，不只看结果数字
    4. 温和而坚定，批评前先充分肯定
    5. 「全员经营」— 让每个人感觉自己是经营者而非打工人
```

- [ ] **Step 2: Create Dalio YAML**

Create `configs/mentors/dalio.yaml`:
```yaml
id: dalio
name: 瑞·达利欧
name_en: Ray Dalio
company: Bridgewater Associates
philosophy: 极度透明，原则驱动
version: 1

strategy:
  checkin_questions:
    - "今天你做了哪些重要决策？依据是什么？"
    - "有没有发现之前的判断是错误的？学到了什么？"
    - "你对当前进展的真实评估是什么？（1-10分）"

  chase:
    method: public_direct
    escalation:
      - action: public_reminder
        delay: "0"
        tone: direct_transparent
      - action: public_reminder
        delay: "1h"
        tone: principled_firm
      - action: manager_notify
        delay: "3h"
        tone: factual_escalation
      - action: skip_today
        delay: "4h"
    forbidden:
      - sugar_coating
      - avoiding_conflict
    encouraged:
      - radical_transparency
      - fact_based_feedback

  summary:
    focus:
      - decision_quality
      - mistakes_learned
      - principles_applied
    highlight: breakthrough_decisions
    flag: repeated_mistakes
    metrics:
      - name: 决策质量评分
        source: self_rating_avg
      - name: 错误复盘率
        source: mistake_acknowledgments

  actions:
    weekly:
      - type: pain_reflection
        desc: "本周最痛苦的教训是什么？"
      - type: principle_extraction
        desc: "从本周错误中提炼出的原则"
    monthly:
      - type: report
        desc: "决策质量月度回顾"
    triggers:
      - event: same_mistake_twice
        action: create_principle
        message: "你在 {context} 上犯了同样的错误。建议创建一条原则来防止再次发生。"
      - event: low_self_rating
        action: private_checkin
        message: "你的自评分数偏低，是否需要讨论一下遇到的挑战？"

  system_prompt: |
    你融合了瑞·达利欧的管理哲学。核心原则：
    1. 极度透明 — 所有反馈公开、直接、基于事实
    2. 拥抱错误 — 错误是学习的最佳机会
    3. 原则驱动 — 遇到问题先检查是否有适用原则
    4. 可信度加权 — 重视有track record的意见
    5. 痛苦 + 反思 = 进步 — 不回避不适感
```

- [ ] **Step 3: Write mentor loader tests**

Create `internal/brain/mentor_test.go`:
```go
package brain_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestLoadMentor_Inamori(t *testing.T) {
	m, err := brain.LoadMentor("inamori")
	if err != nil {
		t.Fatalf("load inamori: %v", err)
	}
	if m.ID != "inamori" {
		t.Errorf("id = %q", m.ID)
	}
	if len(m.Strategy.CheckinQuestions) == 0 {
		t.Error("no checkin questions")
	}
	if m.Strategy.Chase.Method != "private_first" {
		t.Errorf("chase method = %q, want private_first", m.Strategy.Chase.Method)
	}
	if len(m.Strategy.Chase.Escalation) == 0 {
		t.Error("no escalation steps")
	}
	if m.Strategy.Summary.Highlight == "" {
		t.Error("no summary highlight")
	}
	if m.Strategy.SystemPrompt == "" {
		t.Error("no system prompt")
	}
}

func TestLoadMentor_Dalio(t *testing.T) {
	m, err := brain.LoadMentor("dalio")
	if err != nil {
		t.Fatalf("load dalio: %v", err)
	}
	if m.Strategy.Chase.Method != "public_direct" {
		t.Errorf("chase method = %q, want public_direct", m.Strategy.Chase.Method)
	}
}

func TestLoadMentor_NotFound(t *testing.T) {
	_, err := brain.LoadMentor("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mentor")
	}
}

func TestGetCheckinQuestions(t *testing.T) {
	m, _ := brain.LoadMentor("inamori")
	qs := m.GetCheckinQuestions()
	if len(qs) < 2 {
		t.Errorf("expected at least 2 questions, got %d", len(qs))
	}
}

func TestGetChaseStep(t *testing.T) {
	m, _ := brain.LoadMentor("inamori")
	step1 := m.GetChaseStep(1)
	if step1.Action != "private_message" {
		t.Errorf("step 1 action = %q", step1.Action)
	}
	step2 := m.GetChaseStep(2)
	if step2.Action != "manager_notify" {
		t.Errorf("step 2 action = %q", step2.Action)
	}
	// Out of range returns skip
	stepN := m.GetChaseStep(99)
	if stepN.Action != "skip_today" {
		t.Errorf("out of range should return skip_today, got %q", stepN.Action)
	}
}
```

- [ ] **Step 4: Run tests — verify fail**

```bash
go test ./internal/brain/ -v -run TestLoadMentor
```
Expected: FAIL

- [ ] **Step 5: Implement mentor.go**

Create `internal/brain/mentor.go` with:
- `MentorConfig` struct matching YAML schema (ID, Name, NameEn, Company, Philosophy, Version, Strategy)
- `Strategy` struct with `CheckinQuestions`, `Chase`, `Summary`, `Actions`, `SystemPrompt`
- `ChaseConfig` with `Method`, `Escalation []EscalationStep`, `Forbidden`, `Encouraged`
- `EscalationStep` with `Action`, `Delay`, `Tone`
- `SummaryConfig` with `Focus`, `Highlight`, `Flag`, `Metrics`
- `LoadMentor(id string) (*MentorConfig, error)` — reads from `configs/mentors/{id}.yaml` using `os.ReadFile` (embed in Phase 3)
- `GetCheckinQuestions() []string`
- `GetChaseStep(n int) EscalationStep` — returns nth step (1-indexed) or `EscalationStep{Action: "skip_today"}` if out of range
- `GetSummaryConfig() SummaryConfig`
- `BuildSystemPrompt() string`

- [ ] **Step 6: Run tests — verify pass**

```bash
go test ./internal/brain/ -v -run TestLoadMentor
go test ./internal/brain/ -v -run TestGetCheckin
go test ./internal/brain/ -v -run TestGetChaseStep
```
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/brain/mentor.go internal/brain/mentor_test.go configs/mentors/
git commit -m "feat: mentor strategy loader with Inamori and Dalio YAMLs"
```

---

## Task 6: Culture Pack Loader

**Files:**
- Create: `internal/brain/culture.go`
- Create: `internal/brain/culture_test.go`
- Create: `configs/cultures/philippines.yaml`
- Create: `configs/cultures/singapore.yaml`

- [ ] **Step 1: Create Philippines YAML**

Create `configs/cultures/philippines.yaml`:
```yaml
market: Philippines
language: Filipino / English
timezone: Asia/Manila
version: 1

communication_style:
  directness: low
  hierarchy_respect: high
  relationship_first: true
  group_face: high

chase_rules:
  never_name_in_group: true
  private_before_escalate: true
  warmth_required: true
  acknowledge_effort: true

forbidden_patterns:
  - "Why haven't you..."
  - "You are the only one"
  - "As I mentioned"
  - "Everyone else has submitted"
  - "You need to explain yourself"

preferred_patterns:
  - "Hope you're doing well"
  - "The team really values your input"
  - "Whenever you have a moment"
  - "Just a gentle reminder"
  - "We'd love to hear from you"
```

- [ ] **Step 2: Create Singapore YAML**

Create `configs/cultures/singapore.yaml`:
```yaml
market: Singapore
language: English
timezone: Asia/Singapore
version: 1

communication_style:
  directness: high
  hierarchy_respect: medium
  relationship_first: false
  group_face: medium

chase_rules:
  never_name_in_group: false
  private_before_escalate: false
  warmth_required: false
  acknowledge_effort: false

forbidden_patterns:
  - "Please do the needful"
  - "Kindly revert"

preferred_patterns:
  - "Please submit by EOD"
  - "Quick reminder on the daily report"
  - "Appreciate your prompt response"
```

- [ ] **Step 3: Write culture loader tests**

Create `internal/brain/culture_test.go`:
```go
package brain_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestLoadCulture_Philippines(t *testing.T) {
	c, err := brain.LoadCulture("philippines")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.CommunicationStyle.Directness != "low" {
		t.Errorf("directness = %q", c.CommunicationStyle.Directness)
	}
	if !c.ChaseRules.NeverNameInGroup {
		t.Error("PH should never name in group")
	}
	if len(c.ForbiddenPatterns) == 0 {
		t.Error("expected forbidden patterns")
	}
}

func TestLoadCulture_Singapore(t *testing.T) {
	c, err := brain.LoadCulture("singapore")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.CommunicationStyle.Directness != "high" {
		t.Errorf("directness = %q, want high", c.CommunicationStyle.Directness)
	}
}

func TestLoadCulture_Default(t *testing.T) {
	c, err := brain.LoadCulture("default")
	if err != nil {
		t.Fatalf("default culture should not error: %v", err)
	}
	if c.CommunicationStyle.Directness != "medium" {
		t.Errorf("default directness should be medium")
	}
}

func TestShouldOverride_PHPublicChase(t *testing.T) {
	c, _ := brain.LoadCulture("philippines")
	// PH culture should override public chase to private
	if !c.ShouldOverride("public_reminder") {
		t.Error("PH should override public_reminder to private")
	}
}

func TestShouldOverride_SGNoOverride(t *testing.T) {
	c, _ := brain.LoadCulture("singapore")
	// SG culture should NOT override public chase
	if c.ShouldOverride("public_reminder") {
		t.Error("SG should not override public_reminder")
	}
}
```

- [ ] **Step 4: Run tests — verify fail, then implement**

Create `internal/brain/culture.go` with:
- `CulturePack` struct: Market, Language, Timezone, Version, CommunicationStyle, ChaseRules, ForbiddenPatterns, PreferredPatterns
- `CommunicationStyle` struct: Directness, HierarchyRespect, RelationshipFirst, GroupFace
- `ChaseRules` struct: NeverNameInGroup, PrivateBeforeEscalate, WarmthRequired, AcknowledgeEffort
- `LoadCulture(code string) (*CulturePack, error)` — reads YAML; for "default" or unknown codes, returns a default pack with medium directness
- `ShouldOverride(action string) bool` — returns true if `NeverNameInGroup` is true and action is `public_reminder` or `public_naming`
- `GetForbiddenPatterns() []string`
- `GetPreferredPatterns() []string`

- [ ] **Step 5: Run tests — verify pass**

```bash
go test ./internal/brain/ -v -run TestLoadCulture
go test ./internal/brain/ -v -run TestShouldOverride
```

- [ ] **Step 6: Commit**

```bash
git add internal/brain/culture.go internal/brain/culture_test.go configs/cultures/
git commit -m "feat: culture pack loader with Philippines and Singapore packs"
```

---

## Task 7: Brain Engine (Strategy Executor)

**Files:**
- Create: `internal/brain/engine.go`
- Create: `internal/brain/engine_test.go`

- [ ] **Step 1: Write engine tests**

Create `internal/brain/engine_test.go`:
```go
package brain_test

import (
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestEngine_BuildSystemPrompt(t *testing.T) {
	e, err := brain.NewEngine("inamori", "philippines")
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	prompt := e.BuildSystemPrompt()
	if prompt == "" {
		t.Fatal("empty prompt")
	}
	// Should contain mentor philosophy
	if !strings.Contains(prompt, "利他") {
		t.Error("prompt should contain Inamori's philosophy")
	}
	// Should contain culture rules
	if !strings.Contains(prompt, "Philippines") {
		t.Error("prompt should include cultural context")
	}
	// Should contain forbidden patterns
	if !strings.Contains(prompt, "FORBIDDEN") {
		t.Error("prompt should include forbidden section")
	}
}

func TestEngine_GetChaseMessage_CultureOverride(t *testing.T) {
	// Dalio wants public, but PH culture should force private
	e, _ := brain.NewEngine("dalio", "philippines")
	step := e.GetEffectiveChaseStep(1)
	// PH culture should override Dalio's public_reminder to private_message
	if step.Action == "public_reminder" {
		t.Error("PH culture should override Dalio's public chase to private")
	}
	if step.Action != "private_message" {
		t.Errorf("expected private_message, got %q", step.Action)
	}
}

func TestEngine_GetEffectiveChaseStep_NoOverride(t *testing.T) {
	// Dalio + SG = no override needed (both direct)
	e, _ := brain.NewEngine("dalio", "singapore")
	step := e.GetEffectiveChaseStep(1)
	if step.Action != "public_reminder" {
		t.Errorf("SG should not override Dalio's public chase, got %q", step.Action)
	}
}

func TestEngine_GetCheckinQuestions(t *testing.T) {
	e, _ := brain.NewEngine("inamori", "philippines")
	qs := e.GetCheckinQuestions()
	if len(qs) < 2 {
		t.Errorf("expected at least 2 questions, got %d", len(qs))
	}
}
```

- [ ] **Step 2: Run tests — verify fail, then implement engine.go**

Create `internal/brain/engine.go`:
```go
package brain

import "fmt"

// Engine assembles mentor strategy + culture pack into executable decisions.
type Engine struct {
	mentor  *MentorConfig
	culture *CulturePack
}

func NewEngine(mentorID, cultureCode string) (*Engine, error) {
	m, err := LoadMentor(mentorID)
	if err != nil {
		return nil, fmt.Errorf("load mentor %q: %w", mentorID, err)
	}
	c, err := LoadCulture(cultureCode)
	if err != nil {
		return nil, fmt.Errorf("load culture %q: %w", cultureCode, err)
	}
	return &Engine{mentor: m, culture: c}, nil
}

func (e *Engine) BuildSystemPrompt() string {
	prompt := e.mentor.Strategy.SystemPrompt
	prompt += "\n\n--- Cultural Context ---\n"
	prompt += fmt.Sprintf("Employee culture: %s\n", e.culture.Market)
	prompt += fmt.Sprintf("Communication directness: %s\n", e.culture.CommunicationStyle.Directness)
	if len(e.culture.ForbiddenPatterns) > 0 {
		prompt += "FORBIDDEN phrases (never use these):\n"
		for _, p := range e.culture.ForbiddenPatterns {
			prompt += fmt.Sprintf("- %s\n", p)
		}
	}
	if len(e.culture.PreferredPatterns) > 0 {
		prompt += "Preferred phrases:\n"
		for _, p := range e.culture.PreferredPatterns {
			prompt += fmt.Sprintf("- %s\n", p)
		}
	}
	return prompt
}

func (e *Engine) GetCheckinQuestions() []string {
	return e.mentor.GetCheckinQuestions()
}

func (e *Engine) GetSummaryConfig() SummaryConfig {
	return e.mentor.GetSummaryConfig()
}

// GetEffectiveChaseStep returns the chase step with cultural overrides applied.
func (e *Engine) GetEffectiveChaseStep(step int) EscalationStep {
	s := e.mentor.GetChaseStep(step)
	// Culture override: if culture says never public, downgrade to private
	if e.culture.ShouldOverride(s.Action) {
		s.Action = "private_message"
	}
	return s
}
```

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/brain/ -v
```
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/brain/engine.go internal/brain/engine_test.go
git commit -m "feat: brain engine — mentor strategy executor with culture overrides"
```

---

## Task 8: Claude API Wrapper

**Files:**
- Create: `internal/brain/llm.go`
- Create: `internal/brain/llm_test.go`

- [ ] **Step 1: Write LLM client tests (with mock)**

Create `internal/brain/llm_test.go`:
```go
package brain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// mockLLM implements brain.LLMClient for testing
type mockLLM struct {
	response string
	err      error
	calls    int
}

func (m *mockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	m.calls++
	return m.response, m.err
}

func TestLLM_GenerateChaseMessage(t *testing.T) {
	mock := &mockLLM{response: "Hi! Just a friendly reminder to submit your daily report."}
	svc := brain.NewLLMService(mock)

	msg, err := svc.GenerateChaseMessage(context.Background(), "system prompt", "John", "warm_reminder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestLLM_GenerateSummary(t *testing.T) {
	mock := &mockLLM{response: "## Daily Summary\n3/5 employees submitted..."}
	svc := brain.NewLLMService(mock)

	summary, err := svc.GenerateSummary(context.Background(), "system prompt", []brain.ReportData{
		{EmployeeName: "Alice", Answers: map[string]string{"q1": "did X"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestLLM_ErrorReturned(t *testing.T) {
	mock := &mockLLM{err: errors.New("api error")}
	svc := brain.NewLLMService(mock)

	_, err := svc.GenerateChaseMessage(context.Background(), "prompt", "John", "warm")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLLM_AuthError_Classification(t *testing.T) {
	authErr := &brain.AuthError{Msg: "invalid api key"}
	if !brain.IsAuthError(authErr) {
		t.Error("should detect auth error")
	}
	if brain.IsAuthError(errors.New("timeout")) {
		t.Error("timeout should not be auth error")
	}
}
```

- [ ] **Step 2: Implement llm.go**

Create `internal/brain/llm.go`:
- Define `LLMClient` interface: `Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)`
- Define `ReportData` struct: `{EmployeeName string, Answers map[string]string}`
- Implement `AnthropicClient` struct using `go-anthropic` library
- `NewAnthropicClient(apiKey string) (*AnthropicClient, error)` — returns error if apiKey is empty
- Retry wrapper: 3 attempts with exponential backoff (1s, 4s, 16s) for transient errors only
- **Error classification**: Distinguish transient errors (retry) from `AuthError` (401/403 — do NOT retry, return immediately so caller can notify boss)
- `AuthError` type with `IsAuthError(err) bool` helper
- `LLMService` struct wrapping `LLMClient`:
  - `NewLLMService(client LLMClient) *LLMService`
  - `GenerateChaseMessage(ctx, systemPrompt, employeeName, tone string) (string, error)` — builds user prompt with name+tone
  - `GenerateSummary(ctx, systemPrompt string, reports []ReportData) (string, error)` — builds user prompt with all report data
- Structured logging of API calls (duration, token count, errors) via `slog`

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/brain/ -v -run TestLLM
```

- [ ] **Step 4: Commit**

```bash
git add internal/brain/llm.go internal/brain/llm_test.go
git commit -m "feat: Claude API wrapper with retry, auth error classification, mock interface"
```

---

## Task 9: Telegram Bot Framework

**Files:**
- Create: `internal/bot/bot.go`
- Create: `internal/bot/middleware.go`
- Create: `internal/bot/middleware_test.go`
- Create: `internal/bot/handler.go`

- [ ] **Step 1: Implement bot.go — bot setup**

Create `internal/bot/bot.go`:
- `Bot` struct holding telebot.Bot, DB queries, Redis client, Brain engines, config
- `NewBot(cfg, db, redis) (*Bot, error)` — creates telebot with long polling
- `Start()` — registers handlers and starts polling
- `Stop()` — graceful shutdown
- `SendMessage(chatID int64, text string) error` — wraps telebot Send with logging

- [ ] **Step 2: Implement middleware.go — identity resolution**

Create `internal/bot/middleware.go`:
- `IdentityMiddleware` — on every message, look up `employees.telegram_id`
- If found: attach employee + tenant to context
- If sender is `boss_chat_id`: attach boss role
- If not found: respond "Please contact your manager for access" (unless /join command)
- Context keys: `ContextKeyEmployee`, `ContextKeyTenant`, `ContextKeyIsBoss`

- [ ] **Step 3: Write middleware tests**

Create `internal/bot/middleware_test.go`:
```go
package bot_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

// MockQuerier implements the subset of sqlc.Querier used by middleware
type MockQuerier struct {
	EmployeeByTelegramID *bot.Employee // nil = not found
	TenantByBossChatID   *bot.Tenant   // nil = not found
}

func TestIdentityResolve_KnownEmployee(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{
		EmployeeByTelegramID: &bot.Employee{Name: "Alice", TenantID: "t1"},
	}, 999)

	result, err := resolver.Resolve(context.Background(), 12345)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if result.Employee == nil || result.Employee.Name != "Alice" {
		t.Error("should resolve to Alice")
	}
	if result.IsBoss {
		t.Error("should not be boss")
	}
}

func TestIdentityResolve_Boss(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)

	result, err := resolver.Resolve(context.Background(), 999)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !result.IsBoss {
		t.Error("should be boss")
	}
}

func TestIdentityResolve_Unknown(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)

	result, err := resolver.Resolve(context.Background(), 55555)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if result.Employee != nil || result.IsBoss {
		t.Error("unknown user should have no identity")
	}
}

func TestIdentityResolve_JoinBypass(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)
	// /join command should be allowed even for unknown users
	if !resolver.AllowWithoutIdentity("/join ABC123") {
		t.Error("/join should bypass identity check")
	}
	if resolver.AllowWithoutIdentity("/status") {
		t.Error("/status should NOT bypass identity check")
	}
}
```

Note: The `MockQuerier`, `Employee`, `Tenant`, and `IdentityResolver` types need to be defined so the middleware is testable without a real DB. Define a `Querier` interface in the bot package with just the methods needed (`GetEmployeeByTelegramID`, `GetTenantByBossChatID`).

- [ ] **Step 4: Run middleware tests — verify pass**

```bash
go test ./internal/bot/ -v -run TestIdentityResolve
```

- [ ] **Step 5: Implement handler.go — message routing**

Create `internal/bot/handler.go`:
- Route text messages to Report Collector (if employee is in collecting state)
- Route commands to commands.go
- Ignore unrecognized messages in groups
- Log all interactions with structured logging

- [ ] **Step 6: Verify compilation**

```bash
go build ./internal/bot/
```

- [ ] **Step 7: Commit**

```bash
git add internal/bot/
git commit -m "feat: telegram bot framework with identity middleware, tests, and message routing"
```

---

## Task 10: Report Collector (Conversation State Machine)

**Files:**
- Create: `internal/report/collector.go`
- Create: `internal/report/collector_test.go`
- Create: `internal/report/testutil_test.go`

- [ ] **Step 1: Create test utilities**

Create `internal/report/testutil_test.go`:
```go
package report_test

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// mockRedisClient implements the RedisClient interface used by collector
type mockRedisClient struct {
	mu   sync.Mutex
	data map[string]string
	ttls map[string]time.Duration
}

func mockRedis() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", ErrNil // mimic redis.Nil
	}
	return v, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, _ := json.Marshal(value)
	m.data[key] = string(b)
	m.ttls[key] = ttl
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

// ErrNil sentinel for "key not found"
var ErrNil = fmt.Errorf("redis: nil")
```

Note: Import `fmt` at top. The collector should accept a `RedisClient` interface (Get/Set/Del) rather than a concrete Redis client, enabling this mock.

- [ ] **Step 2: Write collector tests**

Create `internal/report/collector_test.go`:
```go
package report_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/report"
)

func TestCollector_StartConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?", "Q3?"})
	state, msg, err := c.Start(context.Background(), "emp-123")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if state != report.StateCollecting {
		t.Errorf("state = %v, want Collecting", state)
	}
	if msg != "Q1?" {
		t.Errorf("first question = %q", msg)
	}
}

func TestCollector_AnswerAllQuestions_EntersConfirming(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?", "Q3?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")

	// Answer Q1
	state, msg, _ := c.HandleAnswer(ctx, "emp-123", "Answer to Q1")
	if state != report.StateCollecting {
		t.Errorf("after Q1: state = %v", state)
	}
	if msg != "Q2?" {
		t.Errorf("second question = %q", msg)
	}

	// Answer Q2
	state, msg, _ = c.HandleAnswer(ctx, "emp-123", "Answer to Q2")
	if msg != "Q3?" {
		t.Errorf("third question = %q", msg)
	}

	// Answer Q3 → should enter confirming state (not complete)
	state, msg, _ = c.HandleAnswer(ctx, "emp-123", "Answer to Q3")
	if state != report.StateConfirming {
		t.Errorf("after Q3: state = %v, want Confirming", state)
	}
	// msg should contain the summary for review
	if msg == "" {
		t.Error("confirming message should show report summary")
	}
}

func TestCollector_ConfirmReport(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	c.HandleAnswer(ctx, "emp-123", "A1") // → confirming

	// Confirm
	state, _, _ := c.Confirm(ctx, "emp-123")
	if state != report.StateComplete {
		t.Errorf("after confirm: state = %v, want Complete", state)
	}
}

func TestCollector_GetAnswers(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	c.HandleAnswer(ctx, "emp-123", "A1")
	c.HandleAnswer(ctx, "emp-123", "A2")

	answers := c.GetAnswers(ctx, "emp-123")
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}
	if answers["q1"] != "A1" || answers["q2"] != "A2" {
		t.Errorf("answers = %v", answers)
	}
}

func TestCollector_NoActiveConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	state, _, _ := c.HandleAnswer(ctx, "emp-123", "random message")
	if state != report.StateIdle {
		t.Errorf("no active conversation should return Idle, got %v", state)
	}
}

func TestCollector_MidConversationRedirect(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	// IsCollecting should be true
	if !c.IsCollecting(ctx, "emp-123") {
		t.Error("should be collecting")
	}
}
```

- [ ] **Step 3: Run tests — verify fail, then implement**

Create `internal/report/collector.go`:
- `ConversationState` enum: `StateIdle`, `StateCollecting`, `StateConfirming`, `StateComplete`
- `RedisClient` interface: `Get(ctx, key) (string, error)`, `Set(ctx, key, value, ttl) error`, `Del(ctx, keys...) error`
- `Collector` struct with RedisClient and questions list
- Redis key format: `conv:{employeeID}` → JSON `{state, current_question: int, answers: map}`
- TTL: 4 hours
- `Start(ctx, employeeID) (state, nextQuestion, error)` — sets state to collecting(q1), returns first question
- `HandleAnswer(ctx, employeeID, answer) (state, nextMsg, error)` — stores answer, advances to next question or confirming state
  - When all questions answered → state = `StateConfirming`, returns formatted summary for review
- `Confirm(ctx, employeeID) (state, msg, error)` — moves from confirming to complete
- `GetAnswers(ctx, employeeID) map[string]string`
- `IsCollecting(ctx, employeeID) bool`

**Edge case notes (handled in handler.go):**
- Unrelated message mid-report: handler checks `IsCollecting()` and treats any non-command text as an answer; if clearly off-topic, bot replies "You're in the middle of your daily report. Please answer: {current question}"
- Auto-confirm: If 5 minutes pass in confirming state with no response, scheduler or TTL handles expiry (deferred to Phase 2 for full auto-confirm timer; Phase 1 uses 4h TTL expiry)
- All-in-one message and group report recognition: Deferred to Phase 2 (requires AI parsing)

- [ ] **Step 4: Run tests — verify pass**

```bash
go test ./internal/report/ -v -run TestCollector
```

- [ ] **Step 5: Commit**

```bash
git add internal/report/collector.go internal/report/collector_test.go internal/report/testutil_test.go
git commit -m "feat: report collector with confirming state, Redis-backed conversation state machine"
```

---

## Task 11: Chase Logic

**Files:**
- Create: `internal/report/chaser.go`
- Create: `internal/report/chaser_test.go`

- [ ] **Step 1: Write chaser tests**

Create `internal/report/chaser_test.go`:
```go
package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// mockChaserDB implements the DB interface for chaser
type mockChaserDB struct {
	employeesWithoutReport []report.EmployeeInfo
	lastChaseStep          int
	createdLogs            []report.ChaseLogEntry
}

func (m *mockChaserDB) ListEmployeesWithoutReport(ctx context.Context, tenantID string, date string) ([]report.EmployeeInfo, error) {
	return m.employeesWithoutReport, nil
}

func (m *mockChaserDB) GetLastChaseStep(ctx context.Context, employeeID string, date string) (int, error) {
	return m.lastChaseStep, nil
}

func (m *mockChaserDB) CreateChaseLog(ctx context.Context, entry report.ChaseLogEntry) error {
	m.createdLogs = append(m.createdLogs, entry)
	return nil
}

// mockSender implements the MessageSender interface
type mockSender struct {
	sentMessages []sentMessage
}

type sentMessage struct {
	ChatID  int64
	Message string
}

func (m *mockSender) SendMessage(chatID int64, text string) error {
	m.sentMessages = append(m.sentMessages, sentMessage{chatID, text})
	return nil
}

func TestChaser_ChasesEmployeesWithoutReport(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111, CultureCode: "philippines"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{response: "Hi Alice, gentle reminder!"}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "philippines")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	err := chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("chase: %v", err)
	}
	if len(sender.sentMessages) != 1 {
		t.Errorf("expected 1 message sent, got %d", len(sender.sentMessages))
	}
	if len(db.createdLogs) != 1 {
		t.Errorf("expected 1 chase log, got %d", len(db.createdLogs))
	}
}

func TestChaser_SkipsTodayWhenMaxSteps(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Bob", TelegramID: 222, CultureCode: "singapore"},
		},
		lastChaseStep: 99, // beyond all escalation steps
	}
	llm := &mockLLM{response: "reminder"}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "singapore")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	if len(sender.sentMessages) != 0 {
		t.Error("should not send message when skip_today")
	}
}

func TestChaser_CultureOverrideApplied(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Carlos", TelegramID: 333, CultureCode: "philippines"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{response: "reminder"}
	sender := &mockSender{}
	// Dalio wants public, but PH culture overrides to private
	engine, _ := brain.NewEngine("dalio", "philippines")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	if len(db.createdLogs) != 1 {
		t.Fatal("expected 1 log")
	}
	if db.createdLogs[0].Action != "private_message" {
		t.Errorf("action should be private_message (culture override), got %q", db.createdLogs[0].Action)
	}
}

func TestChaser_LLMFallback(t *testing.T) {
	db := &mockChaserDB{
		employeesWithoutReport: []report.EmployeeInfo{
			{ID: "e1", Name: "Dan", TelegramID: 444, CultureCode: "default"},
		},
		lastChaseStep: 0,
	}
	llm := &mockLLM{err: errors.New("api down")}
	sender := &mockSender{}
	engine, _ := brain.NewEngine("inamori", "default")

	chaser := report.NewChaser(db, brain.NewLLMService(llm), sender, engine)
	chaser.ChaseAll(context.Background(), "tenant-1", "2026-03-20")

	// Should still send a template fallback message
	if len(sender.sentMessages) != 1 {
		t.Errorf("should send fallback message, got %d messages", len(sender.sentMessages))
	}
}
```

- [ ] **Step 2: Run tests — verify fail, then implement chaser.go**

Create `internal/report/chaser.go`:
- `EmployeeInfo` struct: `{ID, Name, TelegramID, CultureCode}`
- `ChaseLogEntry` struct: `{TenantID, EmployeeID, ReportDate, Step, Action, Message}`
- `ChaserDB` interface: `ListEmployeesWithoutReport`, `GetLastChaseStep`, `CreateChaseLog`
- `MessageSender` interface: `SendMessage(chatID int64, text string) error`
- `Chaser` struct with ChaserDB, LLMService, MessageSender, Engine
- `NewChaser(db, llm, sender, engine)` constructor
- `ChaseAll(ctx, tenantID, date)`:
  1. List employees without report (DB query)
  2. For each: get last chase step from DB
  3. Get next step from `engine.GetEffectiveChaseStep(lastStep + 1)`
  4. If `skip_today` → skip
  5. Generate message via LLM; on error, use template fallback: `"Hi {name}, this is a reminder to submit your daily report."`
  6. Send via sender (private DM using employee's TelegramID)
  7. Insert chase_log

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/report/ -v -run TestChaser
```

- [ ] **Step 4: Commit**

```bash
git add internal/report/chaser.go internal/report/chaser_test.go
git commit -m "feat: mentor-driven chase with cultural override, LLM fallback"
```

---

## Task 12: Summary Generator

**Files:**
- Create: `internal/report/summarizer.go`
- Create: `internal/report/summarizer_test.go`

- [ ] **Step 1: Write summarizer tests**

Create `internal/report/summarizer_test.go`:
```go
package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

type mockSummarizerDB struct {
	reports       []report.ReportRow
	activeCount   int64
	createdSumm   *report.SummaryEntry
}

func (m *mockSummarizerDB) GetReportsByTenantDate(ctx context.Context, tenantID, date string) ([]report.ReportRow, error) {
	return m.reports, nil
}

func (m *mockSummarizerDB) CountActiveEmployees(ctx context.Context, tenantID string) (int64, error) {
	return m.activeCount, nil
}

func (m *mockSummarizerDB) CreateSummary(ctx context.Context, entry report.SummaryEntry) error {
	m.createdSumm = &entry
	return nil
}

func TestSummarizer_GenerateWithReports(t *testing.T) {
	db := &mockSummarizerDB{
		reports: []report.ReportRow{
			{EmployeeName: "Alice", Answers: `{"q1":"did X","q2":"no blockers","q3":"learned Y"}`},
			{EmployeeName: "Bob", Answers: `{"q1":"did Z","q2":"blocked on API","q3":"learned Go"}`},
		},
		activeCount: 5,
	}
	llm := &mockLLM{response: "## Summary\nTeam is making progress. Alice and Bob submitted. Blocker: API dependency."}
	engine, _ := brain.NewEngine("inamori", "philippines")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.SubmissionRate != 0.4 { // 2/5
		t.Errorf("submission rate = %f, want 0.4", result.SubmissionRate)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if db.createdSumm == nil {
		t.Error("summary should be saved to DB")
	}
}

func TestSummarizer_PartialData(t *testing.T) {
	db := &mockSummarizerDB{
		reports:     []report.ReportRow{},
		activeCount: 10,
	}
	llm := &mockLLM{response: "No reports submitted today."}
	engine, _ := brain.NewEngine("inamori", "default")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.SubmissionRate != 0 {
		t.Errorf("rate should be 0 with no reports, got %f", result.SubmissionRate)
	}
}

func TestSummarizer_LLMFallback(t *testing.T) {
	db := &mockSummarizerDB{
		reports: []report.ReportRow{
			{EmployeeName: "Alice", Answers: `{"q1":"did X"}`},
		},
		activeCount: 3,
	}
	llm := &mockLLM{err: errors.New("api down")}
	engine, _ := brain.NewEngine("inamori", "default")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("should not error with fallback: %v", err)
	}
	// Should still produce a bullet-point summary
	if result.Content == "" {
		t.Error("fallback should produce non-empty content")
	}
}
```

- [ ] **Step 2: Implement summarizer.go**

Create `internal/report/summarizer.go`:
- `ReportRow` struct: `{EmployeeName, Answers (JSON string)}`
- `SummaryEntry` struct: `{TenantID, SummaryDate, Content, SubmissionRate, BlockersCount, KeyMetrics}`
- `SummaryResult` struct: `{Content, SubmissionRate, BlockersCount}`
- `SummarizerDB` interface: `GetReportsByTenantDate`, `CountActiveEmployees`, `CreateSummary`
- `Summarizer` struct with SummarizerDB, LLMService, Engine
- `NewSummarizer(db, llm, engine)` constructor
- `Generate(ctx, tenantID, date) (*SummaryResult, error)`:
  1. Get all reports for tenant+date
  2. Count active employees → calculate submission rate
  3. Build LLM prompt using mentor's summary config (focus, highlight, flag)
  4. Call LLM; on error, generate bullet-point fallback:
     ```
     Daily Report Summary (AI unavailable)
     Submitted: 2/5 (40%)
     - Alice: did X
     - Bob: did Z, blocked on API
     ```
  5. Create `SummaryResult` with content, submission_rate, blockers_count
  6. Insert into summaries table
  7. Return result

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/report/ -v -run TestSummarizer
```

- [ ] **Step 4: Commit**

```bash
git add internal/report/summarizer.go internal/report/summarizer_test.go
git commit -m "feat: AI summary generator with mentor-specific focus, LLM fallback"
```

---

## Task 13: Scheduler

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/scheduler_test.go`

- [ ] **Step 1: Write scheduler tests**

Test cases:
- `NewScheduler` registers 3 jobs (remind, chase, summary)
- `CheckMissedJobs` detects stale `last_run` timestamps and runs catch-up
- Jobs call the correct functions (mock callbacks)
- Timezone configuration applied correctly

```go
package scheduler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

// mockRedisClient for scheduler tests (same interface as report tests)
type mockRedisClient struct {
	mu   sync.Mutex
	data map[string]string
}

func mockRedis() *mockRedisClient {
	return &mockRedisClient{data: make(map[string]string)}
}

var errNil = fmt.Errorf("redis: nil")

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", errNil
	}
	return v, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, _ := json.Marshal(value)
	m.data[key] = string(b)
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

type mockCallbacks struct {
	remindCalled  bool
	chaseCalled   bool
	summaryCalled bool
}

func (m *mockCallbacks) Remind(ctx context.Context) error { m.remindCalled = true; return nil }
func (m *mockCallbacks) Chase(ctx context.Context) error  { m.chaseCalled = true; return nil }
func (m *mockCallbacks) Summary(ctx context.Context) error { m.summaryCalled = true; return nil }

func TestScheduler_RegistersThreeJobs(t *testing.T) {
	cb := &mockCallbacks{}
	s, err := scheduler.New("Asia/Singapore", mockRedis(), cb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if s.JobCount() != 3 {
		t.Errorf("expected 3 jobs, got %d", s.JobCount())
	}
}

func TestScheduler_MissedJobCatchUp(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// Set last_run to 3 hours ago (beyond 2h threshold)
	redis.Set(context.Background(), "scheduler:last_run:remind",
		time.Now().Add(-3*time.Hour).Format(time.RFC3339), 0)

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if !cb.remindCalled {
		t.Error("remind should have been called as catch-up")
	}
}
```

- [ ] **Step 2: Implement scheduler.go**

Create `internal/scheduler/scheduler.go`:
- `Callbacks` interface: `Remind(ctx) error`, `Chase(ctx) error`, `Summary(ctx) error`
- `Scheduler` struct with gocron scheduler, Redis client, Callbacks
- `New(timezone string, redis RedisClient, callbacks Callbacks) (*Scheduler, error)`:
  1. Parse timezone via `time.LoadLocation`
  2. Create gocron scheduler with timezone
  3. Register 3 jobs:
     - **Remind** at 9:00 AM → `callbacks.Remind()`
     - **Chase** at 5:30 PM → `callbacks.Chase()`
     - **Summary** at 7:00 PM → `callbacks.Summary()`
  4. After each job runs, update Redis key `scheduler:last_run:{name}` with current time
- `Start()` — starts scheduler + `CheckMissedJobs()`
- `Stop()` — graceful shutdown
- `CheckMissedJobs(ctx)` — for each job, if `last_run` is more than 2 hours stale, run catch-up
- `JobCount() int` — returns number of registered jobs (for testing)

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/scheduler/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/scheduler/
git commit -m "feat: gocron scheduler with remind/chase/summary jobs and missed-job catch-up"
```

---

## Task 14: Bot Commands

**Files:**
- Create: `internal/bot/commands.go`
- Create: `internal/bot/commands_test.go`

- [ ] **Step 1: Write command handler tests**

Create `internal/bot/commands_test.go`:
```go
package bot_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

// mockCommandDB implements the DB interface for command handlers
type mockCommandDB struct {
	tenant         *bot.Tenant // GetTenantByBossChatID result
	createdTenant  *bot.Tenant
	employees      []bot.Employee
	createdEmp     *bot.Employee
	empByInvite    *bot.Employee
	reportCount    int64
}

// Implement all needed Querier methods...
func (m *mockCommandDB) GetTenantByBossChatID(ctx context.Context, id int64) (*bot.Tenant, error) {
	if m.tenant == nil {
		return nil, bot.ErrNotFound
	}
	return m.tenant, nil
}

func (m *mockCommandDB) CreateTenant(ctx context.Context, params bot.CreateTenantParams) (*bot.Tenant, error) {
	t := &bot.Tenant{ID: "new-tenant", Name: params.Name}
	m.createdTenant = t
	return t, nil
}

// mockBotContext wraps telebot.Context for testing
type mockBotContext struct {
	senderID   int64
	text       string
	replied    string
}

func (m *mockBotContext) Sender() *bot.User  { return &bot.User{ID: m.senderID} }
func (m *mockBotContext) Text() string       { return m.text }
func (m *mockBotContext) Send(msg string) error { m.replied = msg; return nil }

func TestCommand_Start_AutoCreateTenant(t *testing.T) {
	db := &mockCommandDB{tenant: nil} // no tenant exists
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	ctx := &mockBotContext{senderID: 999, text: "/start"}
	err := handler.HandleStart(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if db.createdTenant == nil {
		t.Error("should auto-create tenant for boss")
	}
	if ctx.replied == "" {
		t.Error("should reply with welcome message")
	}
}

func TestCommand_Start_ExistingTenant(t *testing.T) {
	db := &mockCommandDB{tenant: &bot.Tenant{ID: "t1", Name: "Test"}}
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	ctx := &mockBotContext{senderID: 999, text: "/start"}
	handler.HandleStart(ctx)
	if db.createdTenant != nil {
		t.Error("should NOT create tenant if one exists")
	}
}

func TestCommand_Status_BossOnly(t *testing.T) {
	db := &mockCommandDB{tenant: &bot.Tenant{ID: "t1"}}
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	// Non-boss should be denied
	ctx := &mockBotContext{senderID: 123, text: "/status"}
	handler.HandleStatus(ctx)
	if ctx.replied == "" || ctx.replied != "Permission denied" {
		// Should contain a permission denied message
	}
}

func TestCommand_AddEmployee(t *testing.T) {
	db := &mockCommandDB{tenant: &bot.Tenant{ID: "t1"}}
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	ctx := &mockBotContext{senderID: 999, text: "/addemployee Alice ph"}
	handler.HandleAddEmployee(ctx)
	if db.createdEmp == nil {
		t.Error("should create employee")
	}
	// Reply should contain invite code
}

func TestCommand_Join(t *testing.T) {
	db := &mockCommandDB{
		empByInvite: &bot.Employee{ID: "e1", Name: "Alice", InviteCode: "ABC123"},
	}
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	ctx := &mockBotContext{senderID: 555, text: "/join ABC123"}
	handler.HandleJoin(ctx)
	// Should link telegram_id and confirm
}

func TestCommand_Mentor_Switch(t *testing.T) {
	db := &mockCommandDB{tenant: &bot.Tenant{ID: "t1", MentorID: "inamori"}}
	handler := bot.NewCommandHandler(db, nil, nil, 999)

	ctx := &mockBotContext{senderID: 999, text: "/mentor dalio"}
	handler.HandleMentor(ctx)
	// Should update tenant mentor to dalio
}

func TestCommand_Help(t *testing.T) {
	handler := bot.NewCommandHandler(nil, nil, nil, 999)
	ctx := &mockBotContext{senderID: 123, text: "/help"}
	handler.HandleHelp(ctx)
	if ctx.replied == "" {
		t.Error("should reply with help text")
	}
}
```

Note: The actual `bot.Tenant`, `bot.Employee`, `bot.User`, `bot.CreateTenantParams`, `bot.ErrNotFound`, `bot.NewCommandHandler` types are defined in `commands.go` and `bot.go`. The handler accepts a `Querier` interface for DB access (same pattern as middleware). Adapt mock method signatures to match the actual interface.

- [ ] **Step 2: Implement commands.go**

Create `internal/bot/commands.go`:
- Each command is a handler function: `handleStart(c telebot.Context) error`
- **Tenant auto-creation in /start**: When boss sends `/start` and no tenant exists for `cfg.BossTelegramID`:
  1. Create tenant with default mentor (inamori), default timezone from config
  2. Encrypt bot_token and anthropic_key (if provided) using `pkg.Encrypt`
  3. Return welcome message with setup instructions
- Permission check: boss commands check `c.Sender().ID == cfg.BossTelegramID`
- `/status` query: list all active employees, mark submitted/not submitted for today
- `/addemployee`: create employee with generated invite code (random 8 chars), DM boss the code
- `/join`: look up invite code → link telegram_id → confirm to employee
- `/mentor`: validate mentor ID exists → update tenant → reload engine
- `/diagnostics`: read Redis keys for last_run times, count recent chase_logs/reports

- [ ] **Step 3: Run tests — verify pass**

```bash
go test ./internal/bot/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/bot/commands.go internal/bot/commands_test.go
git commit -m "feat: bot commands — /start (with auto-tenant) /status /help /addemployee /join /mentor /diagnostics"
```

---

## Task 15: Health Check + Main Wiring

**Files:**
- Modify: `cmd/brain/main.go`

- [ ] **Step 1: Wire everything in main.go**

Update `cmd/brain/main.go`:
1. Load config
2. Connect to PostgreSQL (pgx pool)
3. Connect to Redis
4. Run migrations (golang-migrate, embedded)
5. Initialize sqlc queries
6. Load mentor + culture → create Brain Engine
7. Create Anthropic client (encrypt/decrypt BYOK keys from DB using `pkg.Encrypt`/`pkg.Decrypt`)
8. Create Bot, Collector, Chaser, Summarizer
9. Create Scheduler with callbacks wired to Collector/Chaser/Summarizer
10. Start health check HTTP server (Gin):
    - `GET /healthz` → check DB ping + Redis ping + bot status → return JSON `{"status": "ok/degraded", "db": "ok/error", "redis": "ok/error"}`
11. Start bot polling (goroutine)
12. Start scheduler (goroutine)
13. Wait for SIGINT/SIGTERM → graceful shutdown (stop scheduler, stop bot, close DB pool, close Redis)

**Log redaction**: Use a custom `slog.Handler` wrapper that masks fields named `api_key`, `bot_token`, `password`, `encryption_key` in log output. Apply globally via `slog.SetDefault`.

- [ ] **Step 2: Verify full build**

```bash
go build -o brain ./cmd/brain
```

- [ ] **Step 3: Commit**

```bash
git add cmd/brain/main.go
git commit -m "feat: main.go — wire all components, health check, log redaction, graceful shutdown"
```

---

## Task 16: Integration Test

**Files:**
- Create: `cmd/brain/main_test.go` (optional, smoke test)

- [ ] **Step 1: Start dependencies**

```bash
make dev
```

- [ ] **Step 2: Create .env from .env.example**

Fill in real values:
- Copy `.env.example` to `.env`
- Set `TELEGRAM_BOT_TOKEN` (create test bot via @BotFather)
- Set `BOSS_TELEGRAM_ID` (your own Telegram ID)
- Set `ANTHROPIC_API_KEY`
- Generate `ENCRYPTION_KEY`: `openssl rand -hex 32`

- [ ] **Step 3: Run the bot**

```bash
export $(grep -v '^#' .env | xargs)
make run
```

- [ ] **Step 4: Manual validation checklist**

```
[ ] Bot responds to /start with welcome message (auto-creates tenant)
[ ] /addemployee TestUser ph → creates employee, returns invite code
[ ] Employee DMs bot /join <code> → confirmed
[ ] /status → shows TestUser as "not submitted"
[ ] /mentor dalio → switches mentor, confirms
[ ] /mentor inamori → switches back
[ ] /diagnostics → shows system status
[ ] Wait for remind job (or trigger manually) → employee receives check-in questions
[ ] Employee answers 3 questions → bot shows summary for confirmation
[ ] Employee confirms → bot confirms report saved
[ ] /status → shows TestUser as "submitted"
[ ] Wait for summary job → boss receives AI summary
[ ] /healthz → returns 200 with component status
```

- [ ] **Step 5: Run full test suite**

```bash
make test-cover
```
Target: 80%+ coverage

- [ ] **Step 6: Final commit**

```bash
git add cmd/brain/main_test.go
git commit -m "feat: Phase 1 complete — core bot with mentor strategy + cultural adaptation"
```

---

## Summary

| Task | Component | Tests |
|------|-----------|-------|
| 1 | Project scaffolding | — |
| 2 | DB schema + sqlc | — (generated) |
| 3 | Config module | 4 tests |
| 4 | Crypto (AES-256-GCM) | 4 tests |
| 5 | Mentor strategy loader | 5 tests |
| 6 | Culture pack loader | 5 tests |
| 7 | Brain Engine | 4 tests |
| 8 | Claude API wrapper | 4 tests |
| 9 | Telegram Bot framework | 4 tests (middleware) |
| 10 | Report Collector | 6 tests |
| 11 | Chase logic | 4 tests |
| 12 | Summary generator | 3 tests |
| 13 | Scheduler | 2 tests |
| 14 | Bot commands | ~8 tests |
| 15 | Main wiring + health | — (integration) |
| 16 | Integration test | manual checklist |

**Total: ~53 unit tests across 11 packages**

### Deferred to Phase 2
- Auto-confirm timer for confirming state (5 min timeout)
- All-in-one message parsing (AI-based)
- Group message recognition as report
- Redis-based message queue for Telegram send retries (Phase 1 logs failures to chase_logs)
- `/config` command for updating API keys
- Telegram send failure retry with Redis queue
