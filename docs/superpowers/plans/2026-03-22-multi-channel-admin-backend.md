# Multi-Channel Admin Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend AI Management Brain with 4-channel support (Telegram, Signal, Slack, Lark) and a full admin backend (channels, reports, mentors, scheduler, memory viewer).

**Architecture:** Extend the existing Vue3+Go monolith. Add channel columns to employees/tenants tables, refactor Telegram-centric MessageSender to channel-agnostic Sender, add admin API endpoints with closure-based handlers, build 5 admin Vue3 pages, and unify report collection across all channels.

**Tech Stack:** Go 1.25 (Gin+sqlc+pgx), Vue3+TS, PostgreSQL 16 (pgvector), Redis 7, signal-cli-rest-api, Slack Web API, Lark Open API

**Spec:** `docs/superpowers/specs/2026-03-22-multi-channel-admin-backend-design.md`

---

## File Structure

### New Files
- `sql/migrations/000006_vector384.up.sql` — normalize inline migration
- `sql/migrations/000006_vector384.down.sql`
- `sql/migrations/000007_multi_channel.up.sql` — employee/tenant channel columns
- `sql/migrations/000007_multi_channel.down.sql`
- `internal/channel/resolve.go` — channel resolution helper
- `internal/channel/resolve_test.go`
- `internal/channel/message_handler.go` — unified message handler
- `internal/channel/message_handler_test.go`
- `internal/channel/slack_webhook.go` — Slack Events API handler
- `internal/channel/lark_webhook.go` — Lark event handler
- `internal/api/admin_handlers.go` — admin API handlers
- `internal/api/admin_handlers_test.go`
- `frontend/src/views/admin/ChannelsView.vue`
- `frontend/src/views/admin/TeamChannelsView.vue`
- `frontend/src/views/admin/ReportsView.vue`
- `frontend/src/views/admin/MentorSchedulerView.vue`
- `frontend/src/views/admin/MemoryView.vue`

### Modified Files
- `sql/queries/employees.sql` — add channel lookup queries
- `sql/queries/tenants.sql` — add channel config query
- `cmd/brain/main.go` — extract migration006, wire new components
- `internal/config/config.go` — add Slack/Lark config fields
- `internal/channel/adapter.go` — no changes needed (interfaces already correct)
- `internal/channel/slack.go` — add SigningSecret to SlackConfig
- `internal/channel/telegram.go` — delegate to MessageHandler
- `internal/channel/signal.go` — delegate to MessageHandler
- `internal/report/chaser.go` — replace MessageSender with channel.Sender
- `internal/report/triggers.go` — replace MessageSender with channel.Sender
- `internal/report/actions.go` — replace MessageSender with channel.Sender
- `internal/roles/sender.go` — replace MessageSender with channel.Sender
- `internal/events/bus.go` — add Channel field to payloads
- `internal/api/router.go` — add admin routes, RouterConfig fields
- `internal/scheduler/scheduler.go` — add UpdateJobSchedule method
- `frontend/src/router/index.ts` — add admin routes
- `frontend/src/App.vue` — add admin sidebar group
- `frontend/src/composables/api.ts` — add admin API functions

---

## Task 1: Normalize Migration 000006

Extract the inline vector dimension migration from `main.go` into a proper SQL migration file.

**Files:**
- Create: `sql/migrations/000006_vector384.up.sql`
- Create: `sql/migrations/000006_vector384.down.sql`
- Modify: `cmd/brain/main.go:304-307`

- [ ] **Step 1: Create migration 000006 up file**

```sql
-- 000006_vector384.up.sql
ALTER TABLE memories ALTER COLUMN embedding TYPE vector(384);
```

- [ ] **Step 2: Create migration 000006 down file**

```sql
-- 000006_vector384.down.sql
ALTER TABLE memories ALTER COLUMN embedding TYPE vector(1024);
```

- [ ] **Step 3: Update runMigrations in main.go**

The current `runMigrations` function in `cmd/brain/main.go` executes inline SQL strings (not file-based). Keep this pattern for now but replace the inline migration006 with a call that reads from the SQL file:

```go
// Replace inline migration006 block (lines ~304-307) with:
migration006, err := os.ReadFile("sql/migrations/000006_vector384.up.sql")
if err != nil {
    // Fallback for when running from different directory
    migration006 = []byte("ALTER TABLE memories ALTER COLUMN embedding TYPE vector(384);")
}
if _, err := pool.Exec(ctx, string(migration006)); err != nil {
    return err
}
```

Or simpler: just keep the inline string but add a comment referencing the SQL file for documentation purposes. The SQL file exists for the migration chain record; the inline execution is the actual runtime mechanism.

- [ ] **Step 4: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add sql/migrations/000006_vector384.up.sql sql/migrations/000006_vector384.down.sql cmd/brain/main.go
git commit -m "refactor: extract inline migration 000006 to SQL file"
```

---

## Task 2: Database Migration 000007 — Multi-Channel

Add channel columns to employees and tenants tables, plus channel tracking on reports and chase_logs.

**Files:**
- Create: `sql/migrations/000007_multi_channel.up.sql`
- Create: `sql/migrations/000007_multi_channel.down.sql`

- [ ] **Step 1: Create migration 000007 up file**

```sql
-- 000007_multi_channel.up.sql

-- Employee channel columns
ALTER TABLE employees ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN slack_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN lark_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN preferred_channel VARCHAR(20) NOT NULL DEFAULT 'telegram';

CREATE UNIQUE INDEX idx_employees_signal ON employees(signal_phone) WHERE signal_phone IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_slack ON employees(slack_id) WHERE slack_id IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_lark ON employees(lark_id) WHERE lark_id IS NOT NULL;

-- Tenant channel configuration
ALTER TABLE tenants ADD COLUMN slack_bot_token TEXT;
ALTER TABLE tenants ADD COLUMN slack_signing_secret TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_id TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_secret TEXT;
ALTER TABLE tenants ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN enabled_channels TEXT[] NOT NULL DEFAULT '{telegram}';

-- Track which channel reports/chases came from
ALTER TABLE reports ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
ALTER TABLE chase_logs ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
```

- [ ] **Step 2: Create migration 000007 down file**

```sql
-- 000007_multi_channel.down.sql
ALTER TABLE chase_logs DROP COLUMN IF EXISTS channel;
ALTER TABLE reports DROP COLUMN IF EXISTS channel;

ALTER TABLE tenants DROP COLUMN IF EXISTS enabled_channels;
ALTER TABLE tenants DROP COLUMN IF EXISTS signal_phone;
ALTER TABLE tenants DROP COLUMN IF EXISTS lark_app_secret;
ALTER TABLE tenants DROP COLUMN IF EXISTS lark_app_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS slack_signing_secret;
ALTER TABLE tenants DROP COLUMN IF EXISTS slack_bot_token;

DROP INDEX IF EXISTS idx_employees_lark;
DROP INDEX IF EXISTS idx_employees_slack;
DROP INDEX IF EXISTS idx_employees_signal;

ALTER TABLE employees DROP COLUMN IF EXISTS preferred_channel;
ALTER TABLE employees DROP COLUMN IF EXISTS lark_id;
ALTER TABLE employees DROP COLUMN IF EXISTS slack_id;
ALTER TABLE employees DROP COLUMN IF EXISTS signal_phone;
```

- [ ] **Step 3: Commit**

```bash
git add sql/migrations/000007_multi_channel.*
git commit -m "feat: add migration 000007 for multi-channel support"
```

---

## Task 3: sqlc Queries + Regenerate

Add new queries for channel-based employee lookup and tenant channel management, then regenerate Go code.

**Files:**
- Modify: `sql/queries/employees.sql`
- Modify: `sql/queries/tenants.sql`
- Auto-generated: `internal/db/sqlc/*.go`

- [ ] **Step 1: Add employee channel queries**

Append to `sql/queries/employees.sql`:

```sql
-- name: GetEmployeeBySignalPhone :one
SELECT * FROM employees WHERE signal_phone = $1 AND is_active = true;

-- name: GetEmployeeBySlackID :one
SELECT * FROM employees WHERE slack_id = $1 AND is_active = true;

-- name: GetEmployeeByLarkID :one
SELECT * FROM employees WHERE lark_id = $1 AND is_active = true;

-- name: UpdateEmployeeChannels :exec
UPDATE employees
SET signal_phone = $2, slack_id = $3, lark_id = $4, preferred_channel = $5
WHERE id = $1;

-- name: UpdateEmployeePreferredChannel :exec
UPDATE employees SET preferred_channel = $2 WHERE id = $1;

-- name: ListEmployeesWithChannels :many
SELECT id, tenant_id, name, telegram_id, signal_phone, slack_id, lark_id, preferred_channel, culture_code, role, is_active
FROM employees
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;
```

- [ ] **Step 2: Add tenant channel queries**

Append to `sql/queries/tenants.sql`:

```sql
-- name: UpdateTenantChannels :exec
UPDATE tenants
SET slack_bot_token = $2, slack_signing_secret = $3,
    lark_app_id = $4, lark_app_secret = $5,
    signal_phone = $6, enabled_channels = $7
WHERE id = $1;

-- name: GetTenantChannelConfig :one
SELECT id, slack_bot_token, slack_signing_secret, lark_app_id, lark_app_secret, signal_phone, enabled_channels
FROM tenants WHERE id = $1;
```

- [ ] **Step 3: Add report channel queries**

Append to `sql/queries/reports.sql` (or create if needed):

```sql
-- name: ListReportsFiltered :many
SELECT r.*, e.name as employee_name
FROM reports r
JOIN employees e ON r.employee_id = e.id
WHERE r.tenant_id = $1
  AND ($2::date IS NULL OR r.report_date >= $2)
  AND ($3::date IS NULL OR r.report_date <= $3)
  AND ($4::uuid IS NULL OR r.employee_id = $4)
  AND ($5::text = '' OR r.channel = $5)
ORDER BY r.submitted_at DESC
LIMIT $6 OFFSET $7;

-- name: CountReportsFiltered :one
SELECT COUNT(*) FROM reports
WHERE tenant_id = $1
  AND ($2::date IS NULL OR report_date >= $2)
  AND ($3::date IS NULL OR report_date <= $3)
  AND ($4::uuid IS NULL OR employee_id = $4)
  AND ($5::text = '' OR channel = $5);

-- name: GetReportStatsByChannel :many
SELECT channel, COUNT(*) as count
FROM reports WHERE tenant_id = $1
  AND ($2::date IS NULL OR report_date >= $2)
  AND ($3::date IS NULL OR report_date <= $3)
GROUP BY channel;
```

- [ ] **Step 4: Regenerate sqlc**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`
Expected: no errors, updated files in `internal/db/sqlc/`

- [ ] **Step 5: Verify build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS (new model fields auto-included via `SELECT *`)

- [ ] **Step 6: Commit**

```bash
git add sql/queries/ internal/db/sqlc/
git commit -m "feat: add sqlc queries for multi-channel employee/tenant/report lookup"
```

---

## Task 4: Channel Resolution Helper

Create a helper to resolve an employee's preferred channel type and user ID, with fallback logic.

**Files:**
- Create: `internal/channel/resolve.go`
- Create: `internal/channel/resolve_test.go`

- [ ] **Step 1: Write tests for ResolveChannel**

Create `internal/channel/resolve_test.go`:

```go
package channel

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// mockEmployee creates a test employee with specified channels
func mockEmployee(telegramID int64, signalPhone, slackID, larkID, preferred string) resolveEmployee {
	e := resolveEmployee{PreferredChannel: preferred}
	if telegramID != 0 {
		e.TelegramID = pgtype.Int8{Int64: telegramID, Valid: true}
	}
	if signalPhone != "" {
		e.SignalPhone = pgtype.Text{String: signalPhone, Valid: true}
	}
	if slackID != "" {
		e.SlackID = pgtype.Text{String: slackID, Valid: true}
	}
	if larkID != "" {
		e.LarkID = pgtype.Text{String: larkID, Valid: true}
	}
	return e
}

func TestResolveChannel_PreferredTelegram(t *testing.T) {
	emp := mockEmployee(12345, "+639177918392", "", "", "telegram")
	chType, chID := ResolveChannel(emp)
	if chType != TypeTelegram || chID != "12345" {
		t.Errorf("expected telegram/12345, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_PreferredSignal(t *testing.T) {
	emp := mockEmployee(12345, "+639177918392", "", "", "signal")
	chType, chID := ResolveChannel(emp)
	if chType != TypeSignal || chID != "+639177918392" {
		t.Errorf("expected signal/+639177918392, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_PreferredSlack(t *testing.T) {
	emp := mockEmployee(0, "", "U01ABC", "", "slack")
	chType, chID := ResolveChannel(emp)
	if chType != TypeSlack || chID != "U01ABC" {
		t.Errorf("expected slack/U01ABC, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_FallbackToTelegram(t *testing.T) {
	emp := mockEmployee(12345, "", "", "", "signal") // preferred signal but no phone
	chType, chID := ResolveChannel(emp)
	if chType != TypeTelegram || chID != "12345" {
		t.Errorf("expected fallback to telegram/12345, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_NoChannels(t *testing.T) {
	emp := mockEmployee(0, "", "", "", "telegram")
	chType, chID := ResolveChannel(emp)
	if chType != "" || chID != "" {
		t.Errorf("expected empty, got %s/%s", chType, chID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/channel/ -run TestResolveChannel -v`
Expected: FAIL (ResolveChannel not defined)

- [ ] **Step 3: Implement ResolveChannel**

Create `internal/channel/resolve.go`:

```go
package channel

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// resolveEmployee is a subset of sqlc.Employee fields needed for channel resolution.
// Using a separate type avoids importing the sqlc package from the channel package.
type resolveEmployee struct {
	TelegramID       pgtype.Int8
	SignalPhone      pgtype.Text
	SlackID          pgtype.Text
	LarkID           pgtype.Text
	PreferredChannel string
}

// ResolveChannel returns the preferred channel type and user ID for an employee.
// Falls back through available channels if the preferred channel is not configured.
func ResolveChannel(emp resolveEmployee) (Type, string) {
	// Try preferred channel first
	switch Type(emp.PreferredChannel) {
	case TypeTelegram:
		if emp.TelegramID.Valid && emp.TelegramID.Int64 != 0 {
			return TypeTelegram, strconv.FormatInt(emp.TelegramID.Int64, 10)
		}
	case TypeSignal:
		if emp.SignalPhone.Valid && emp.SignalPhone.String != "" {
			return TypeSignal, emp.SignalPhone.String
		}
	case TypeSlack:
		if emp.SlackID.Valid && emp.SlackID.String != "" {
			return TypeSlack, emp.SlackID.String
		}
	case TypeLark:
		if emp.LarkID.Valid && emp.LarkID.String != "" {
			return TypeLark, emp.LarkID.String
		}
	}

	// Fallback: try all channels in priority order
	if emp.TelegramID.Valid && emp.TelegramID.Int64 != 0 {
		return TypeTelegram, strconv.FormatInt(emp.TelegramID.Int64, 10)
	}
	if emp.SignalPhone.Valid && emp.SignalPhone.String != "" {
		return TypeSignal, emp.SignalPhone.String
	}
	if emp.SlackID.Valid && emp.SlackID.String != "" {
		return TypeSlack, emp.SlackID.String
	}
	if emp.LarkID.Valid && emp.LarkID.String != "" {
		return TypeLark, emp.LarkID.String
	}

	return "", ""
}

// EmployeeToResolve converts sqlc Employee fields into resolveEmployee.
// Call this from packages that import both channel and sqlc.
func EmployeeToResolve(telegramID pgtype.Int8, signalPhone, slackID, larkID pgtype.Text, preferred string) resolveEmployee {
	return resolveEmployee{
		TelegramID:       telegramID,
		SignalPhone:      signalPhone,
		SlackID:          slackID,
		LarkID:           larkID,
		PreferredChannel: preferred,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/channel/ -run TestResolveChannel -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/channel/resolve.go internal/channel/resolve_test.go
git commit -m "feat: add channel resolution helper with fallback logic"
```

---

## Task 5: MessageSender Refactor

Replace the Telegram-centric `MessageSender` interface (int64 chatID) with `channel.Sender` in all report/roles business logic.

**Files:**
- Modify: `internal/report/chaser.go`
- Modify: `internal/report/triggers.go`
- Modify: `internal/report/actions.go`
- Modify: `internal/report/dbadapter.go`
- Modify: `internal/report/alert.go`
- Modify: `internal/roles/sender.go`
- Create: `internal/channel/router_sender.go`

- [ ] **Step 1: Extend EmployeeInfo with channel fields**

In `internal/report/chaser.go`, add channel fields to `EmployeeInfo` (lines 12-18):

```go
type EmployeeInfo struct {
    ID               string
    Name             string
    TelegramID       int64
    SignalPhone      string // NEW
    SlackID          string // NEW
    LarkID           string // NEW
    PreferredChannel string // NEW — default "telegram"
    CultureCode      string
}
```

- [ ] **Step 2: Update DBAdapter to populate channel fields**

In `internal/report/dbadapter.go`, update `ListEmployeesWithoutReport` (line ~44) and `ListActiveEmployeesWithTelegram` (line ~208) to populate the new fields from sqlc Employee model:

```go
result = append(result, EmployeeInfo{
    ID:               e.ID.String(),
    Name:             e.Name,
    TelegramID:       e.TelegramID.Int64,
    SignalPhone:      e.SignalPhone.String,      // NEW
    SlackID:          e.SlackID.String,           // NEW
    LarkID:           e.LarkID.String,            // NEW
    PreferredChannel: e.PreferredChannel,         // NEW
    CultureCode:      e.CultureCode,
})
```

Rename `ListActiveEmployeesWithTelegram` to `ListActiveEmployees` (since it's no longer Telegram-specific). Update the corresponding interface in `actions.go`, `triggers.go`, `alert.go`.

- [ ] **Step 3: Replace MessageSender with channel.Sender**

Use the existing `channel.Sender` interface from `internal/channel/adapter.go`:
```go
type Sender interface {
    Send(ctx context.Context, channelType Type, userID string, text string) error
}
```

In `internal/report/chaser.go`:
- Replace `MessageSender` interface with an import of `channel.Sender`
- Update `Chaser` struct: `sender channel.Sender`
- Update `NewChaser`: parameter type to `channel.Sender`
- In `ChaseAll`, change send call:

```go
// OLD: c.sender.SendMessage(emp.TelegramID, msg)
// NEW:
re := channel.EmployeeToResolve(
    pgtype.Int8{Int64: emp.TelegramID, Valid: emp.TelegramID != 0},
    pgtype.Text{String: emp.SignalPhone, Valid: emp.SignalPhone != ""},
    pgtype.Text{String: emp.SlackID, Valid: emp.SlackID != ""},
    pgtype.Text{String: emp.LarkID, Valid: emp.LarkID != ""},
    emp.PreferredChannel,
)
chType, chID := channel.ResolveChannel(re)
if chType == "" {
    slog.Warn("employee has no channel configured", "employee", emp.Name)
    continue
}
c.sender.Send(ctx, chType, chID, msg)
```

- [ ] **Step 4: Refactor triggers.go**

Same pattern: Replace `MessageSender` with `channel.Sender` in `TriggerChecker`. Update `executeAction` to resolve channel per employee before sending. Note: `bossChatID int64` parameter in `CheckAll` becomes `bossEmployeeInfo EmployeeInfo` so the boss channel can be resolved too.

- [ ] **Step 5: Refactor actions.go**

Replace `MessageSender` with `channel.Sender` in `ActionExecutor`. Change `RunWeekly`/`RunMonthly` boss message sending to resolve boss channel via `EmployeeInfo` instead of hardcoded `bossChatID int64`.

- [ ] **Step 6: Refactor roles/sender.go**

Replace `BossSender` to use `channel.Sender`. Change `bossChatID int64` to `bossChannelType channel.Type` + `bossChannelID string`.

- [ ] **Step 7: Create RouterSender adapter**

Create `internal/channel/router_sender.go`:

```go
package channel

import "context"

// RouterSender adapts channel.Router to the channel.Sender interface.
type RouterSender struct {
    router *Router
}

func NewRouterSender(r *Router) *RouterSender {
    return &RouterSender{router: r}
}

func (rs *RouterSender) Send(ctx context.Context, channelType Type, userID string, text string) error {
    return rs.router.Send(ctx, channelType, userID, text)
}
```

- [ ] **Step 8: Update main.go wiring**

In `cmd/brain/main.go`, create Router with Telegram adapter, create `RouterSender`, pass to Chaser/TriggerChecker/ActionExecutor/BossSender instead of `tgAdapter`.

- [ ] **Step 9: Update tests**

Update mock `MessageSender` in test files (`chaser_test.go`, `triggers_test.go`, `actions_test.go`, `alert_test.go`) to implement `channel.Sender` interface instead. Update `EmployeeInfo` in test data to include new channel fields (can be empty strings for backward compat).

- [ ] **Step 10: Run tests**

Run: `go test ./internal/report/ ./internal/roles/ -v`
Expected: PASS

- [ ] **Step 11: Verify build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 12: Commit**

```bash
git add internal/report/ internal/roles/ internal/channel/router_sender.go cmd/brain/main.go
git commit -m "refactor: replace Telegram-centric MessageSender with channel-agnostic Sender"
```

---

## Task 6: Event Payload Enhancement

Add `Channel` field to event payloads so downstream consumers know which channel generated the event.

**Files:**
- Modify: `internal/events/bus.go`

- [ ] **Step 1: Update payload structs**

In `internal/events/bus.go`, add `Channel string` field to:

```go
type ReportSubmittedPayload struct {
    EmployeeID   string `json:"employee_id"`
    EmployeeName string `json:"employee_name"`
    ReportDate   string `json:"report_date"`
    Channel      string `json:"channel"`
}

type ChaseCompletedPayload struct {
    EmployeeID   string `json:"employee_id"`
    EmployeeName string `json:"employee_name"`
    ReportDate   string `json:"report_date"`
    ChaseLogID   string `json:"chase_log_id"`
    Step         int    `json:"step"`
    Action       string `json:"action"`
    Message      string `json:"message"`
    Channel      string `json:"channel"`
}
```

- [ ] **Step 2: Update event publishers in main.go**

Wherever `ReportSubmittedPayload` or `ChaseCompletedPayload` is published, add `Channel: "telegram"` (default for now, will be dynamic when CommandHandler is wired).

- [ ] **Step 3: Run tests**

Run: `go test ./internal/events/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/events/bus.go cmd/brain/main.go
git commit -m "feat: add Channel field to event payloads"
```

---

## Task 7: Config + SlackConfig Updates

Add Slack/Lark configuration fields and extend SlackConfig with SigningSecret.

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/channel/slack.go`

- [ ] **Step 1: Add config fields**

In `internal/config/config.go`, add to Config struct:

```go
// Slack (optional)
SlackBotToken      string
SlackSigningSecret string

// Lark (optional)
LarkAppID     string
LarkAppSecret string
```

In `Load()`:
```go
cfg.SlackBotToken = os.Getenv("SLACK_BOT_TOKEN")
cfg.SlackSigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
cfg.LarkAppID = os.Getenv("LARK_APP_ID")
cfg.LarkAppSecret = os.Getenv("LARK_APP_SECRET")
```

- [ ] **Step 2: Update SlackConfig**

In `internal/channel/slack.go`, add `SigningSecret` to SlackConfig:

```go
type SlackConfig struct {
    BotToken      string
    AppToken      string
    WebhookURL    string
    SigningSecret string // For verifying Slack Events API signatures
}
```

- [ ] **Step 3: Update .env.example**

Add:
```
# Slack (optional)
SLACK_BOT_TOKEN=
SLACK_SIGNING_SECRET=

# Lark/Feishu (optional)
LARK_APP_ID=
LARK_APP_SECRET=
```

- [ ] **Step 4: Verify build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/channel/slack.go .env.example
git commit -m "feat: add Slack/Lark config fields and SlackConfig.SigningSecret"
```

---

## Task 8: Scheduler UpdateJobSchedule

Add the ability to update a job's cron expression at runtime.

**Files:**
- Modify: `internal/scheduler/scheduler.go`

- [ ] **Step 1: Write test**

Add to `internal/scheduler/scheduler_test.go` (create if needed):

```go
func TestScheduler_UpdateJobSchedule(t *testing.T) {
    // Test that updating a job's schedule works
    // This is an integration-style test that verifies the method exists
    // and returns appropriate errors for unknown jobs
    s := &Scheduler{jobs: []jobInfo{{name: "test", cron: "0 9 * * *"}}}
    err := s.UpdateJobSchedule("nonexistent", "0 10 * * *")
    if err == nil {
        t.Error("expected error for unknown job")
    }
}
```

- [ ] **Step 2: Update jobInfo struct**

First, update the internal `jobInfo` struct to store the callback function and gocron job ID:

```go
type jobInfo struct {
    name   string
    cron   string
    fn     func(ctx context.Context) error  // NEW: store callback for re-registration
    jobID  uuid.UUID                        // NEW: gocron job ID for removal
}
```

Update `AddJob` to populate these new fields when registering jobs.

- [ ] **Step 3: Implement UpdateJobSchedule and ListJobs**

In `internal/scheduler/scheduler.go`:

```go
// UpdateJobSchedule updates the cron expression for an existing job.
func (s *Scheduler) UpdateJobSchedule(name, cron string) error {
    var found *jobInfo
    for i := range s.jobs {
        if s.jobs[i].name == name {
            found = &s.jobs[i]
            break
        }
    }
    if found == nil {
        return fmt.Errorf("job %q not found", name)
    }

    // Remove old job from gocron
    if found.jobID != uuid.Nil {
        if err := s.scheduler.RemoveByID(found.jobID); err != nil {
            return fmt.Errorf("remove job %q: %w", name, err)
        }
    }

    // Re-add with new schedule
    j, err := s.scheduler.NewJob(
        gocron.CronJob(cron, false),
        gocron.NewTask(found.fn),
        gocron.WithName(name),
    )
    if err != nil {
        return fmt.Errorf("re-add job %q with cron %q: %w", name, cron, err)
    }

    found.cron = cron
    found.jobID = j.ID()
    return nil
}

// ListJobs returns info about all registered jobs.
func (s *Scheduler) ListJobs() []JobInfo {
    result := make([]JobInfo, len(s.jobs))
    for i, j := range s.jobs {
        result[i] = JobInfo{
            Name:    j.name,
            Cron:    j.cron,
            LastRun: s.getLastRun(j.name),
            NextRun: s.getNextRun(j.name),
        }
    }
    return result
}

type JobInfo struct {
    Name    string    `json:"name"`
    Cron    string    `json:"cron"`
    LastRun time.Time `json:"last_run"`
    NextRun time.Time `json:"next_run"`
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/scheduler/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/scheduler/
git commit -m "feat: add UpdateJobSchedule and ListJobs to scheduler"
```

---

## Task 9: Admin API — RouterConfig + Route Registration

Add admin routes to the API router with proper middleware.

**Files:**
- Modify: `internal/api/router.go`

- [ ] **Step 1: Extend RouterConfig**

Add new fields to `RouterConfig` in `internal/api/router.go`:

```go
type RouterConfig struct {
    // ... existing fields ...
    ChannelRouter  *channel.Router
    SlackAdapter   *channel.SlackAdapter
    LarkAdapter    *channel.LarkAdapter
    Scheduler      *scheduler.Scheduler
}
```

- [ ] **Step 2: Register admin routes**

In the `SetupRouter` function, add after existing protected routes:

```go
// Admin routes (boss only)
admin := protected.Group("/admin")
admin.Use(RequireRole("boss"))
{
    // Channels
    admin.GET("/channels", handleGetChannels(cfg.Queries, cfg.ChannelRouter))
    admin.PUT("/channels", handleUpdateChannels(cfg.Queries, cfg.ChannelRouter))
    admin.POST("/channels/test/:channel", handleTestChannel(cfg.ChannelRouter))

    // Employees with channels
    admin.GET("/employees", handleAdminListEmployees(cfg.Queries))
    admin.PUT("/employees/:id/channels", handleUpdateEmployeeChannels(cfg.Queries))
    admin.PUT("/employees/:id/preferred", handleUpdateEmployeePreferred(cfg.Queries))

    // Reports
    admin.GET("/reports", handleAdminListReports(cfg.Queries))
    admin.GET("/reports/stats", handleReportStats(cfg.Queries))
    admin.GET("/reports/:id", handleGetReport(cfg.Queries))

    // Mentors
    admin.GET("/mentors", handleListMentors())
    admin.GET("/mentors/current", handleGetMentor(cfg.Queries))  // reuse existing
    admin.PUT("/mentors", handleUpdateMentor(cfg.Queries))       // reuse existing
    admin.PUT("/mentors/blend", handleUpdateBlend(cfg.Queries))  // reuse existing

    // Scheduler
    admin.GET("/scheduler", handleListSchedulerJobs(cfg.Scheduler))
    admin.PUT("/scheduler/:job/schedule", handleUpdateJobSchedule(cfg.Scheduler))
    admin.POST("/scheduler/:job/trigger", handleTriggerJob(cfg.Scheduler))
    admin.PUT("/scheduler/timezone", handleUpdateTimezone(cfg.Queries))

    // Memories
    if cfg.MemoryEngine != nil {
        admin.GET("/memories", handleAdminListMemories(cfg.MemoryStore))
        admin.GET("/memories/:id", handleGetMemory(cfg.MemoryStore))
        admin.POST("/memories/search", handleAdminSearchMemories(cfg.MemoryStore, cfg.MemoryEngine))
        admin.DELETE("/memories/:id", handleDeleteMemory(cfg.MemoryStore))
        admin.GET("/memories/stats", handleMemoryStats(cfg.MemoryStore))
    }
}

// Webhook routes (public, signature-verified)
if cfg.SlackAdapter != nil {
    v1.POST("/slack/events", cfg.SlackAdapter.HandleSlackEvent)
}
if cfg.LarkAdapter != nil {
    v1.POST("/lark/events", cfg.LarkAdapter.HandleLarkEvent)
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/brain/`
Expected: FAIL (handlers not yet implemented — this is expected, will be resolved in Task 10)

- [ ] **Step 4: Commit**

```bash
git add internal/api/router.go
git commit -m "feat: register admin API routes and webhook endpoints"
```

---

## Task 10: Admin API Handlers

Implement all admin handler functions.

**Files:**
- Create: `internal/api/admin_handlers.go`
- Create: `internal/api/admin_handlers_test.go`

- [ ] **Step 1: Implement channel management handlers**

Create `internal/api/admin_handlers.go`:

```go
package api

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/tonypk/ai-management-brain/internal/channel"
    "github.com/tonypk/ai-management-brain/internal/db/sqlc"
    "github.com/tonypk/ai-management-brain/internal/memory"
    "github.com/tonypk/ai-management-brain/internal/scheduler"
)

func handleGetChannels(queries *sqlc.Queries, router *channel.Router) gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID, err := parseUUID(TenantFromContext(c))
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
            return
        }
        cfg, err := queries.GetTenantChannelConfig(c.Request.Context(), tenantID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get channel config"})
            return
        }
        channels := map[string]gin.H{
            "telegram": {"configured": true, "status": "active"},
            "signal":   {"configured": cfg.SignalPhone.Valid && cfg.SignalPhone.String != "", "phone": cfg.SignalPhone.String},
            "slack":    {"configured": cfg.SlackBotToken.Valid && cfg.SlackBotToken.String != ""},
            "lark":     {"configured": cfg.LarkAppID.Valid && cfg.LarkAppID.String != ""},
        }
        c.JSON(http.StatusOK, gin.H{
            "enabled_channels": cfg.EnabledChannels,
            "channels":         channels,
        })
    }
}
```

Continue implementing all handlers following this pattern. Key handlers:
- `handleUpdateChannels` — validates + updates tenant channel config
- `handleTestChannel` — sends test message via specified channel adapter
- `handleAdminListEmployees` — calls `ListEmployeesWithChannels`
- `handleUpdateEmployeeChannels` — calls `UpdateEmployeeChannels`
- `handleUpdateEmployeePreferred` — validates channel exists, calls `UpdateEmployeePreferredChannel`
- `handleAdminListReports` — calls `ListReportsFiltered` with pagination
- `handleReportStats` — calls `GetReportStatsByChannel`
- `handleListMentors` — returns `mentorDescriptions` map
- `handleListSchedulerJobs` — calls `scheduler.ListJobs()`
- `handleUpdateJobSchedule` — calls `scheduler.UpdateJobSchedule()`
- `handleTriggerJob` — calls job callback directly
- `handleAdminListMemories` — calls `memoryStore.List()` with filters
- `handleAdminSearchMemories` — embeds query, calls `memoryStore.SearchSimilar()`
- `handleMemoryStats` — calls `memoryStore.Count()` and type/tier breakdowns

- [ ] **Step 2: Write tests for key handlers**

Create `internal/api/admin_handlers_test.go` with httptest-based tests for at least:
- `handleListMentors` — verify returns all 14 mentors
- `handleGetChannels` — mock queries, verify JSON shape
- `handleUpdateEmployeePreferred` — verify validation (reject if channel not configured)

- [ ] **Step 3: Run tests**

Run: `go test ./internal/api/ -v -run TestAdmin`
Expected: PASS

- [ ] **Step 4: Verify full build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/api/admin_handlers.go internal/api/admin_handlers_test.go
git commit -m "feat: implement admin API handlers for channels, reports, mentors, scheduler, memory"
```

---

## Task 11: Frontend — API Functions + Types

Add admin API functions and TypeScript types to the composables layer.

**Files:**
- Modify: `frontend/src/composables/api.ts`

- [ ] **Step 1: Add admin types**

Append to types section in `api.ts`:

```typescript
// Admin types
export interface ChannelStatus {
  configured: boolean;
  status?: string;
  phone?: string;
}

export interface ChannelConfig {
  enabled_channels: string[];
  channels: Record<string, ChannelStatus>;
}

export interface EmployeeWithChannels {
  id: string;
  tenant_id: string;
  name: string;
  telegram_id: number | null;
  signal_phone: string | null;
  slack_id: string | null;
  lark_id: string | null;
  preferred_channel: string;
  culture_code: string;
  role: string;
  is_active: boolean;
}

export interface AdminReport {
  id: string;
  employee_id: string;
  employee_name: string;
  report_date: string;
  answers: Record<string, string>;
  blockers: string;
  sentiment: string;
  channel: string;
  submitted_at: string;
}

export interface ReportStats {
  total_reports: number;
  submission_rate: number;
  by_channel: Record<string, number>;
  by_employee: { name: string; count: number; rate: number }[];
}

export interface SchedulerJob {
  name: string;
  cron: string;
  last_run: string;
  next_run: string;
  status: string;
}

export interface SchedulerConfig {
  timezone: string;
  jobs: SchedulerJob[];
}

export interface MemoryItem {
  id: string;
  tenant_id: string;
  memory_type: string;
  memory_tier: string;
  employee_id: string | null;
  content: string;
  summary: string | null;
  importance: number;
  access_count: number;
  metadata: Record<string, unknown>;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface MemoryStats {
  total: number;
  by_type: Record<string, number>;
  by_tier: Record<string, number>;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
}
```

- [ ] **Step 2: Add admin API functions**

```typescript
// Admin - Channels
export const getChannelConfig = () => request<ChannelConfig>("/admin/channels");
export const updateChannelConfig = (data: {
  enabled_channels: string[];
  slack_bot_token?: string;
  slack_signing_secret?: string;
  lark_app_id?: string;
  lark_app_secret?: string;
}) => request<void>("/admin/channels", { method: "PUT", body: JSON.stringify(data) });
export const testChannel = (channel: string) =>
  request<{ success: boolean; error?: string }>(`/admin/channels/test/${channel}`, { method: "POST" });

// Admin - Employees
export const listEmployeesWithChannels = () =>
  request<EmployeeWithChannels[]>("/admin/employees");
export const updateEmployeeChannels = (id: string, data: {
  signal_phone?: string; slack_id?: string; lark_id?: string;
}) => request<void>(`/admin/employees/${id}/channels`, { method: "PUT", body: JSON.stringify(data) });
export const updateEmployeePreferred = (id: string, preferred_channel: string) =>
  request<void>(`/admin/employees/${id}/preferred`, { method: "PUT", body: JSON.stringify({ preferred_channel }) });

// Admin - Reports
export const listAdminReports = (params: {
  page?: number; limit?: number; from?: string; to?: string; employee_id?: string; channel?: string;
}) => {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => { if (v) qs.set(k, String(v)); });
  return request<PaginatedResponse<AdminReport>>(`/admin/reports?${qs}`);
};
export const getReportStats = (from?: string, to?: string) => {
  const qs = new URLSearchParams();
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  return request<ReportStats>(`/admin/reports/stats?${qs}`);
};

// Admin - Mentors
export const listAllMentors = () => request<MentorInfo[]>("/admin/mentors");

// Admin - Scheduler
export const getSchedulerConfig = () => request<SchedulerConfig>("/admin/scheduler");
export const updateJobSchedule = (job: string, cron: string) =>
  request<void>(`/admin/scheduler/${job}/schedule`, { method: "PUT", body: JSON.stringify({ cron }) });
export const triggerJob = (job: string) =>
  request<{ triggered: boolean }>(`/admin/scheduler/${job}/trigger`, { method: "POST" });
export const updateTimezone = (timezone: string) =>
  request<void>("/admin/scheduler/timezone", { method: "PUT", body: JSON.stringify({ timezone }) });

// Admin - Memories
export const listMemories = (params: {
  page?: number; limit?: number; type?: string; tier?: string; employee_id?: string;
}) => {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => { if (v) qs.set(k, String(v)); });
  return request<PaginatedResponse<MemoryItem>>(`/admin/memories?${qs}`);
};
export const searchMemories = (query: string, limit?: number) =>
  request<MemoryItem[]>("/admin/memories/search", {
    method: "POST", body: JSON.stringify({ query, limit: limit || 10 }),
  });
export const deleteMemory = (id: string) =>
  request<void>(`/admin/memories/${id}`, { method: "DELETE" });
export const getMemoryStats = () => request<MemoryStats>("/admin/memories/stats");
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/composables/api.ts
git commit -m "feat: add admin API types and functions to frontend composables"
```

---

## Task 12: Frontend — Router + Sidebar

Add admin routes to Vue router and admin section to sidebar navigation.

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add admin routes**

In `frontend/src/router/index.ts`, add before the closing `]` of routes array:

```typescript
  {
    path: "/admin/channels",
    name: "AdminChannels",
    component: () => import("../views/admin/ChannelsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/team-channels",
    name: "AdminTeamChannels",
    component: () => import("../views/admin/TeamChannelsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/reports",
    name: "AdminReports",
    component: () => import("../views/admin/ReportsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/mentor-scheduler",
    name: "AdminMentorScheduler",
    component: () => import("../views/admin/MentorSchedulerView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/memory",
    name: "AdminMemory",
    component: () => import("../views/admin/MemoryView.vue"),
    meta: { requiresAuth: true },
  },
```

- [ ] **Step 2: Add admin sidebar section**

In `frontend/src/App.vue`, add after the existing navigation items (Analytics):

```html
<!-- Admin Section -->
<div class="nav-section">Admin</div>
<router-link to="/admin/channels" class="nav-item">Channels</router-link>
<router-link to="/admin/team-channels" class="nav-item">Team Channels</router-link>
<router-link to="/admin/reports" class="nav-item">Reports</router-link>
<router-link to="/admin/mentor-scheduler" class="nav-item">Mentor & Scheduler</router-link>
<router-link to="/admin/memory" class="nav-item">Memory</router-link>
```

Add CSS for `.nav-section`:
```css
.nav-section {
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  color: #888;
  margin-top: 16px;
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/App.vue
git commit -m "feat: add admin routes and sidebar section"
```

---

## Task 13: Frontend — Channels View

Build the channel management admin page.

**Files:**
- Create: `frontend/src/views/admin/ChannelsView.vue`

- [ ] **Step 1: Create ChannelsView.vue**

Channel cards grid with toggle, credential forms, test buttons. Uses `getChannelConfig`, `updateChannelConfig`, `testChannel` from API composable.

Key elements:
- 4 cards: Telegram (read-only), Signal (phone display), Slack (token + secret inputs), Lark (app ID + secret inputs)
- Status indicator (green/red dot)
- Enable/disable toggle per channel
- "Test" button per channel → shows success/error toast
- "Save" button → PUT to `/admin/channels`

- [ ] **Step 2: Verify dev build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/ChannelsView.vue
git commit -m "feat: add Channels admin view"
```

---

## Task 14: Frontend — Team Channels View

Build the employee channel assignment page.

**Files:**
- Create: `frontend/src/views/admin/TeamChannelsView.vue`

- [ ] **Step 1: Create TeamChannelsView.vue**

Data table with inline editing for channel IDs and preferred channel. Uses `listEmployeesWithChannels`, `updateEmployeeChannels`, `updateEmployeePreferred`.

Key elements:
- Table columns: Name, Role, Telegram, Signal, Slack, Lark, Preferred
- Click cell to edit channel ID inline
- Preferred channel dropdown per row
- Save on blur/enter
- Search filter at top

- [ ] **Step 2: Verify dev build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/TeamChannelsView.vue
git commit -m "feat: add Team Channels admin view"
```

---

## Task 15: Frontend — Reports View

Build the admin reports view with filters and stats.

**Files:**
- Create: `frontend/src/views/admin/ReportsView.vue`

- [ ] **Step 1: Create ReportsView.vue**

Filter bar + stats cards + data table. Uses `listAdminReports`, `getReportStats`.

Key elements:
- Filter bar: date range, employee dropdown, channel dropdown
- Stats cards: Total, Submission Rate, By Channel
- Table: Employee, Date, Channel, Time, Sentiment, Blockers
- Row expand → full answers + AI summary
- Pagination

- [ ] **Step 2: Verify dev build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/ReportsView.vue
git commit -m "feat: add Reports admin view with filters and stats"
```

---

## Task 16: Frontend — Mentor & Scheduler View

Build the combined mentor configuration and scheduler management page.

**Files:**
- Create: `frontend/src/views/admin/MentorSchedulerView.vue`

- [ ] **Step 1: Create MentorSchedulerView.vue**

Two sections: mentor selector + scheduler table. Uses `listAllMentors`, `updateMentor`, `updateBlend`, `getSchedulerConfig`, `updateJobSchedule`, `triggerJob`, `updateTimezone`.

Key elements:
- **Mentor section**: Current mentor display, grid of all mentor cards, blend toggle + slider
- **Scheduler section**: Table of jobs with editable cron, "Run Now" buttons, timezone selector

- [ ] **Step 2: Verify dev build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/MentorSchedulerView.vue
git commit -m "feat: add Mentor & Scheduler admin view"
```

---

## Task 17: Frontend — Memory View

Build the memory viewer with search and filters.

**Files:**
- Create: `frontend/src/views/admin/MemoryView.vue`

- [ ] **Step 1: Create MemoryView.vue**

Stats + filters + search + data table + detail modal. Uses `listMemories`, `searchMemories`, `deleteMemory`, `getMemoryStats`.

Key elements:
- Stats row: Total, Short-term, Long-term, By Type
- Filter bar: type, tier, employee dropdowns
- Semantic search bar
- Table: Content (truncated), Type, Tier, Importance, Employee, Created, Expires
- Row click → modal with full content + metadata JSON
- Delete button with confirmation

- [ ] **Step 2: Verify dev build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/MemoryView.vue
git commit -m "feat: add Memory viewer admin page"
```

---

## Task 18: Unified MessageHandler

Create the channel-agnostic message handler that processes incoming messages from all channels.

**Files:**
- Create: `internal/channel/message_handler.go`
- Create: `internal/channel/message_handler_test.go`

- [ ] **Step 1: Write tests**

Test cases: command parsing, employee resolution by channel type, report answer forwarding.

- [ ] **Step 2: Implement MessageHandler**

Create `internal/channel/message_handler.go`:

```go
package channel

import (
    "context"
    "fmt"
    "log/slog"
    "strconv"

    "github.com/jackc/pgx/v5/pgtype"
    "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// MessageHandler processes incoming messages from any channel.
// Named to avoid collision with CommandHandler func type in adapter.go.
type MessageHandler struct {
    queries   *sqlc.Queries
    sender    Sender
    onText    func(ctx context.Context, employeeID, text, channelType string) (response string, err error)
    onCommand func(ctx context.Context, employeeID, command, args, channelType string) (response string, err error)
}

type MessageHandlerConfig struct {
    Queries   *sqlc.Queries
    Sender    Sender
    OnText    func(ctx context.Context, employeeID, text, channelType string) (response string, err error)
    OnCommand func(ctx context.Context, employeeID, command, args, channelType string) (response string, err error)
}

func NewMessageHandler(cfg MessageHandlerConfig) *MessageHandler {
    return &MessageHandler{
        queries:   cfg.Queries,
        sender:    cfg.Sender,
        onText:    cfg.OnText,
        onCommand: cfg.OnCommand,
    }
}

func (h *MessageHandler) HandleMessage(ctx context.Context, msg Message) error {
    emp, err := h.resolveEmployee(ctx, msg.ChannelType, msg.UserID)
    if err != nil {
        slog.Warn("unknown sender", "channel", msg.ChannelType, "userID", msg.UserID, "error", err)
        return nil // don't error on unknown senders
    }

    var response string
    if msg.IsCommand && h.onCommand != nil {
        response, err = h.onCommand(ctx, emp.ID.String(), msg.Command, msg.Args, string(msg.ChannelType))
    } else if h.onText != nil {
        response, err = h.onText(ctx, emp.ID.String(), msg.Text, string(msg.ChannelType))
    }

    if err != nil {
        return fmt.Errorf("handle message: %w", err)
    }
    if response != "" {
        re := EmployeeToResolve(emp.TelegramID, emp.SignalPhone, emp.SlackID, emp.LarkID, emp.PreferredChannel)
        chType, chID := ResolveChannel(re)
        if chType != "" {
            return h.sender.Send(ctx, chType, chID, response)
        }
    }
    return nil
}

func (h *MessageHandler) resolveEmployee(ctx context.Context, ct Type, userID string) (sqlc.Employee, error) {
    switch ct {
    case TypeTelegram:
        id, _ := strconv.ParseInt(userID, 10, 64)
        return h.queries.GetEmployeeByTelegramID(ctx, pgtype.Int8{Int64: id, Valid: true})
    case TypeSignal:
        return h.queries.GetEmployeeBySignalPhone(ctx, pgtype.Text{String: userID, Valid: true})
    case TypeSlack:
        return h.queries.GetEmployeeBySlackID(ctx, pgtype.Text{String: userID, Valid: true})
    case TypeLark:
        return h.queries.GetEmployeeByLarkID(ctx, pgtype.Text{String: userID, Valid: true})
    }
    return sqlc.Employee{}, fmt.Errorf("unknown channel type: %s", ct)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/channel/ -run TestMessageHandler -v`
Expected: PASS

- [ ] **Step 4: Wire into main.go**

In `cmd/brain/main.go`, create `MessageHandler` and wire the existing text handler logic (report collection state machine) as the `OnText` callback. Wire existing command handler as `OnCommand`.

Set `MessageHandler` as the message handler for Telegram, Signal adapters.

- [ ] **Step 5: Verify build and existing tests**

Run: `go build ./cmd/brain/ && go test ./...`
Expected: BUILD SUCCESS, tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/channel/message_handler.go internal/channel/message_handler_test.go cmd/brain/main.go
git commit -m "feat: add unified MessageHandler for multi-channel message processing"
```

---

## Task 19: Slack Webhook Handler

Implement Slack Events API handler for receiving messages from Slack users.

**Files:**
- Create: `internal/channel/slack_webhook.go`

- [ ] **Step 1: Implement HandleSlackEvent**

```go
package channel

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
)

// HandleSlackEvent processes Slack Events API callbacks.
func (s *SlackAdapter) HandleSlackEvent(c *gin.Context) {
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
        return
    }

    // Verify Slack signature
    if s.signingSecret != "" {
        if !s.verifySlackSignature(c.Request.Header, body) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
            return
        }
    }

    var payload struct {
        Type      string `json:"type"`
        Challenge string `json:"challenge"`
        Event     struct {
            Type    string `json:"type"`
            User    string `json:"user"`
            Text    string `json:"text"`
            Channel string `json:"channel"`
        } `json:"event"`
    }
    if err := json.Unmarshal(body, &payload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
        return
    }

    // URL verification challenge
    if payload.Type == "url_verification" {
        c.JSON(http.StatusOK, gin.H{"challenge": payload.Challenge})
        return
    }

    // Process message events
    if payload.Type == "event_callback" && payload.Event.Type == "message" {
        if s.msgHandler != nil {
            text := payload.Event.Text
            isCmd := strings.HasPrefix(text, "/")
            var cmd, args string
            if isCmd {
                parts := strings.SplitN(text, " ", 2)
                cmd = strings.TrimPrefix(parts[0], "/")
                if len(parts) > 1 {
                    args = parts[1]
                }
            }
            msg := Message{
                ChannelType: TypeSlack,
                ChannelID:   payload.Event.Channel,
                UserID:      payload.Event.User,
                Text:        text,
                IsCommand:   isCmd,
                Command:     cmd,
                Args:        args,
            }
            _ = s.msgHandler(c.Request.Context(), msg)
        }
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *SlackAdapter) verifySlackSignature(headers http.Header, body []byte) bool {
    timestamp := headers.Get("X-Slack-Request-Timestamp")
    sig := headers.Get("X-Slack-Signature")
    if timestamp == "" || sig == "" {
        return false
    }
    // Check timestamp is within 5 minutes
    ts, err := strconv.ParseInt(timestamp, 10, 64)
    if err != nil || time.Now().Unix()-ts > 300 {
        return false
    }
    baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
    mac := hmac.New(sha256.New, []byte(s.signingSecret))
    mac.Write([]byte(baseString))
    expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sig))
}
```

Also update `SlackAdapter` struct to store `signingSecret` and expose `msgHandler`:
- Add `signingSecret string` field
- Add `msgHandler MessageHandler` field (the func type from adapter.go)
- Add `SetMessageHandler(h MessageHandler)` method

- [ ] **Step 2: Run build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/channel/slack_webhook.go internal/channel/slack.go
git commit -m "feat: add Slack Events API webhook handler with signature verification"
```

---

## Task 20: Lark Webhook Handler

Implement Lark/Feishu event subscription handler.

**Files:**
- Create: `internal/channel/lark_webhook.go`

- [ ] **Step 1: Implement HandleLarkEvent**

Similar pattern to Slack: verify encryption, handle challenge, parse message events, delegate to msgHandler.

Lark specifics:
- Decrypt event body using AES-256-CBC with app secret hash
- Challenge response: `{"challenge": "..."}`
- Message events: extract `open_id` and text content

- [ ] **Step 2: Run build**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/channel/lark_webhook.go internal/channel/lark.go
git commit -m "feat: add Lark event subscription webhook handler"
```

---

## Task 21: Wire Everything in main.go

Final integration: wire all new components together in the main startup sequence.

**Files:**
- Modify: `cmd/brain/main.go`

- [ ] **Step 1: Create channel Router**

After adapter creation, create a Router and register all available adapters:

```go
channelRouter := channel.NewRouter()
channelRouter.Register(tgAdapter)
if signalAdapter != nil {
    channelRouter.Register(signalAdapter)
}
// Slack and Lark adapters created from tenant config if credentials exist
```

- [ ] **Step 2: Create RouterSender for business logic**

```go
routerSender := channel.NewRouterSender(channelRouter)
```

Pass `routerSender` to Chaser, TriggerChecker, ActionExecutor, BossSender.

- [ ] **Step 3: Create MessageHandler**

Wire the existing text handler (report collection) and command handler as callbacks:

```go
msgHandler := channel.NewMessageHandler(channel.MessageHandlerConfig{
    Queries: botDB,
    Sender:  channelRouter,
    OnText:  func(ctx context.Context, empID, text, chType string) (string, error) {
        // ... existing report collection logic ...
    },
    OnCommand: func(ctx context.Context, empID, cmd, args, chType string) (string, error) {
        // ... existing command handling ...
    },
})
```

- [ ] **Step 4: Wire MessageHandler to adapters**

```go
tgAdapter.SetMessageHandler(msgHandler.HandleMessage)
if signalAdapter != nil {
    signalAdapter.SetMessageHandler(msgHandler.HandleMessage)
}
```

- [ ] **Step 5: Update RouterConfig**

Pass new fields to `api.NewRouter`:
```go
ChannelRouter: channelRouter,
SlackAdapter:  slackAdapter,
LarkAdapter:   larkAdapter,
Scheduler:     sched,
```

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 7: Build final binary**

Run: `go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 8: Commit**

```bash
git add cmd/brain/main.go
git commit -m "feat: wire multi-channel components in main.go startup"
```

---

## Task 22: Deploy & Verify

Build, deploy to production, and verify everything works.

**Files:** None (deployment only)

- [ ] **Step 1: Run full test suite locally**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./...`
Expected: ALL PASS

- [ ] **Step 2: Build Linux binary**

Run: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain/`

- [ ] **Step 3: Build frontend**

Run: `cd frontend && npm run build`

- [ ] **Step 4: Push to GitHub**

Run: `git push origin main`
CI/CD will auto-deploy via `.github/workflows/deploy.yml`

- [ ] **Step 5: Verify deployment**

```bash
ssh ai-brain
curl localhost/healthz
docker compose -f docker-compose.prod.yml logs brain --tail=50
```

- [ ] **Step 6: Test admin backend**

Open `https://manageaibrain.com/#/admin/channels` in browser, verify:
- Channel config page loads
- Team channels page shows employees
- Reports page shows data
- Mentor & Scheduler page works
- Memory viewer works

- [ ] **Step 7: Test multi-channel sending**

Via the admin backend, assign a Signal phone to an employee and set preferred_channel to "signal". Trigger a chase or reminder and verify the message arrives via Signal instead of Telegram.
