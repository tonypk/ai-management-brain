# Multi-Channel Admin Backend Design

## Goal

Extend AI Management Brain to support 4 messaging channels (Telegram, Signal, Slack, Lark) with a full-featured admin backend for channel management, report viewing, mentor configuration, scheduler control, and memory inspection.

## Architecture

Extend the existing Vue3+Go monolith (Approach 1). Add channel columns to the employees table, refactor the Telegram-centric `MessageSender` to a channel-agnostic `Sender`, create admin API endpoints, and build 5 admin frontend tabs. Reuse the existing `channel.Channel` interface and `channel.Router` for multi-channel dispatch.

## Tech Stack

- Go 1.25 (Gin + sqlc + pgx)
- Vue3 + TypeScript (existing SPA)
- PostgreSQL 16 (pgvector)
- Redis 7
- signal-cli-rest-api (Signal)
- Slack Web API + Events API
- Lark/Feishu Open API

---

## Section 1: Database Schema Changes

### Migration `000007_multi_channel.up.sql`

> **Note**: Migration 000006 is applied inline in `cmd/brain/main.go` (vector dimension change from 1024→384). This migration is 000007. As a cleanup step, create `000006_vector384.up.sql` with the inline migration content to normalize the migration chain.

#### Employee Channel Columns

```sql
ALTER TABLE employees ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN slack_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN lark_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN preferred_channel VARCHAR(20) NOT NULL DEFAULT 'telegram';

CREATE UNIQUE INDEX idx_employees_signal ON employees(signal_phone) WHERE signal_phone IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_slack ON employees(slack_id) WHERE slack_id IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_lark ON employees(lark_id) WHERE lark_id IS NOT NULL;
```

#### Tenant Channel Configuration

```sql
ALTER TABLE tenants ADD COLUMN slack_bot_token TEXT;
ALTER TABLE tenants ADD COLUMN slack_signing_secret TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_id TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_secret TEXT;
ALTER TABLE tenants ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN enabled_channels TEXT[] NOT NULL DEFAULT '{telegram}';
```

### Design Decisions

- **Direct columns vs junction table**: Use direct columns on `employees` (signal_phone, slack_id, lark_id) since channels are a fixed set of 4. No need for a junction table.
- **`preferred_channel`**: Determines which channel gets messages first. Defaults to `telegram` for backward compatibility.
- **`enabled_channels`**: Array on `tenants` controls which channels are available for the organization.
- **Channel credentials**: Stored in tenant table. Sensitive values (tokens, secrets) encrypted at rest using existing `ENCRYPTION_KEY`.
- **Backward compatibility**: Existing `telegram_id` column unchanged. Default `preferred_channel = 'telegram'` means no behavior change for existing users.

### Down Migration

```sql
ALTER TABLE employees DROP COLUMN signal_phone;
ALTER TABLE employees DROP COLUMN slack_id;
ALTER TABLE employees DROP COLUMN lark_id;
ALTER TABLE employees DROP COLUMN preferred_channel;

ALTER TABLE tenants DROP COLUMN slack_bot_token;
ALTER TABLE tenants DROP COLUMN slack_signing_secret;
ALTER TABLE tenants DROP COLUMN lark_app_id;
ALTER TABLE tenants DROP COLUMN lark_app_secret;
ALTER TABLE tenants DROP COLUMN signal_phone;
ALTER TABLE tenants DROP COLUMN enabled_channels;
```

### sqlc Queries

New queries needed in `sql/queries/`. After adding queries, run `~/go/bin/sqlc generate` to regenerate Go code. All existing `SELECT *` queries will automatically include new columns after migration.

> **Naming convention**: Existing queries use `Get` prefix (e.g., `GetEmployeeByTelegramID`). New queries follow the same convention.

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

-- name: UpdateTenantChannels :exec
UPDATE tenants
SET slack_bot_token = $2, slack_signing_secret = $3,
    lark_app_id = $4, lark_app_secret = $5,
    signal_phone = $6, enabled_channels = $7
WHERE id = $1;
```

> **Tenant resolution for inbound messages**: Employee lookup queries do NOT filter by `tenant_id`. Since `signal_phone`, `slack_id`, and `lark_id` are unique per employee across all tenants (enforced by unique partial indexes), a single query suffices. The existing `GetEmployeeByTelegramID` also doesn't filter by tenant_id. The tenant is resolved from the employee's `tenant_id` column after lookup.

---

## Section 2: Message Routing Refactor

### Problem

The current `MessageSender` interface in `internal/report/chaser.go`, `internal/report/triggers.go`, `internal/report/actions.go`, and `internal/roles/sender.go` is Telegram-centric:

```go
type MessageSender interface {
    SendMessage(chatID int64, text string) error
}
```

This hardcodes `int64` chat IDs (Telegram-specific). Business logic cannot send to Signal, Slack, or Lark.

### Solution

Replace `MessageSender` with the existing `channel.Sender` interface:

```go
// Already exists in internal/channel/adapter.go
type Sender interface {
    Send(ctx context.Context, channelType Type, userID string, text string) error
}
```

The `channel.Router` already implements this interface by dispatching to the correct adapter.

### Channel Resolution Helper

New file `internal/channel/resolve.go`:

```go
package channel

import (
    "strconv"
    "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ResolveChannel returns the preferred channel type and user ID for an employee.
// Falls back through available channels if preferred is not configured.
func ResolveChannel(emp db.Employee) (Type, string) {
    // Try preferred channel first
    switch Type(emp.PreferredChannel) {
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
    case TypeTelegram:
        if emp.TelegramID.Valid && emp.TelegramID.Int64 != 0 {
            return TypeTelegram, strconv.FormatInt(emp.TelegramID.Int64, 10)
        }
    }

    // Fallback: try all channels in order
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
```

### Refactor Touchpoints

4 files need updating to use `channel.Sender` instead of `MessageSender`:

1. **`internal/report/chaser.go`**
   - Change `sender MessageSender` field to `sender channel.Sender`
   - In `ChaseAll()`: resolve employee channel, call `sender.Send(ctx, chType, chID, msg)`

2. **`internal/report/triggers.go`**
   - Same pattern: resolve channel per employee before sending

3. **`internal/report/actions.go`**
   - Same pattern for weekly/monthly action messages

4. **`internal/roles/sender.go`**
   - `BossSender`: resolve boss's preferred channel from tenant config

### Backward Compatibility

- `TelegramAdapter.SendMessage(chatID int64, text string)` method stays (used by bot command handlers)
- `TelegramAdapter` also implements `Channel.SendToUser(ctx, userID string, text)` by parsing userID to int64
- Existing Telegram bot commands continue to work unchanged

---

## Section 3: Admin API Endpoints

### Route Structure

All routes under `/api/v1/admin/`, protected by `RequireRole("boss")` middleware.

```
/api/v1/admin
├── /channels
│   ├── GET    /                    # List enabled channels + credential status
│   ├── PUT    /                    # Update enabled channels + credentials
│   └── POST   /test/:channel       # Send test message to boss
│
├── /employees
│   ├── GET    /                    # List employees with all channel bindings
│   ├── PUT    /:id/channels        # Assign channel IDs to employee
│   └── PUT    /:id/preferred       # Set preferred channel
│
├── /reports
│   ├── GET    /                    # List reports with filters
│   ├── GET    /stats               # Submission rates by channel/employee
│   └── GET    /:id                 # Single report detail + AI summary
│
├── /mentors
│   ├── GET    /                    # List all mentors with descriptions
│   ├── GET    /current             # Current tenant mentor config
│   ├── PUT    /                    # Set single mentor
│   └── PUT    /blend               # Set mentor blend
│
├── /scheduler
│   ├── GET    /                    # List jobs: name, cron, last_run, next_run
│   ├── PUT    /:job/schedule       # Update cron expression
│   ├── POST   /:job/trigger        # Manually trigger job
│   └── PUT    /timezone            # Update timezone
│
└── /memories
    ├── GET    /                    # List memories with filters
    ├── GET    /:id                 # Single memory detail
    ├── POST   /search              # Semantic similarity search
    ├── DELETE /:id                 # Delete memory
    └── GET    /stats               # Memory counts by type/tier
```

### Handler File Structure

New file `internal/api/admin_handlers.go` with all admin handlers. Follow the existing closure-based handler pattern used in `handlers.go` (e.g., `handleGetMentor(queries *sqlc.Queries) gin.HandlerFunc`). Dependencies are passed via `RouterConfig` (already exists in `router.go`).

Add new fields to `RouterConfig`:

```go
// In internal/api/router.go, add to RouterConfig:
type RouterConfig struct {
    // ... existing fields ...
    ChannelRouter  *channel.Router        // for admin channel management
    Scheduler      *scheduler.Scheduler   // nil = scheduler admin disabled
}
```

Admin handlers use the existing `mentorDescriptions` map (defined in `handlers.go`, currently 14 mentors) and `*sqlc.Queries` directly — no new interface needed.

> **Additional RouterConfig fields needed**: `SlackAdapter *channel.SlackAdapter` and `LarkAdapter *channel.LarkAdapter` for the test-channel endpoint and webhook registration.

> **Scheduler gap**: The existing `Scheduler` struct has no method to update a job's cron expression at runtime. Add `UpdateJobSchedule(name, cron string) error` (remove + re-add the job in gocron).

### Common Patterns

**Error responses** follow existing codebase pattern:
```json
{ "error": "descriptive error message" }
```

**Pagination** for list endpoints (reports, memories, employees):
- Query params: `?page=1&limit=20` (defaults: page=1, limit=20, max limit=100)
- Response includes metadata:
```json
{
  "data": [...],
  "meta": { "page": 1, "limit": 20, "total": 150, "total_pages": 8 }
}
```

### Endpoint Details

#### GET /admin/channels
Returns:
```json
{
  "enabled_channels": ["telegram", "signal"],
  "channels": {
    "telegram": { "configured": true, "status": "active" },
    "signal": { "configured": true, "phone": "+12133474323", "status": "active" },
    "slack": { "configured": false },
    "lark": { "configured": false }
  }
}
```

#### PUT /admin/channels
Request:
```json
{
  "enabled_channels": ["telegram", "signal", "slack"],
  "slack_bot_token": "xoxb-...",
  "slack_signing_secret": "..."
}
```
Validates credentials, updates tenant, restarts affected channel adapters.

#### POST /admin/channels/test/:channel
Sends a test message to the boss via the specified channel. Returns success/failure with error detail.

#### PUT /admin/employees/:id/channels
Request:
```json
{
  "signal_phone": "+639177918392",
  "slack_id": "U01ABC123",
  "lark_id": "ou_abc123"
}
```

#### PUT /admin/employees/:id/preferred
Request:
```json
{
  "preferred_channel": "signal"
}
```
Validates that the employee has a configured ID for the chosen channel.

#### GET /admin/reports?from=2026-03-01&to=2026-03-22&employee_id=xxx&channel=signal
Returns paginated reports with channel info.

#### GET /admin/reports/stats
Returns:
```json
{
  "total_reports": 150,
  "submission_rate": 0.85,
  "by_channel": { "telegram": 120, "signal": 30 },
  "by_employee": [{ "name": "Alice", "count": 20, "rate": 0.95 }]
}
```

#### GET /admin/scheduler
Returns:
```json
{
  "timezone": "Asia/Singapore",
  "jobs": [
    { "name": "remind", "schedule": "0 9 * * *", "last_run": "...", "next_run": "...", "status": "active" },
    { "name": "chase", "schedule": "30 17 * * *", "last_run": "...", "next_run": "...", "status": "active" }
  ]
}
```

#### POST /admin/scheduler/:job/trigger
Manually triggers a job. Returns `{ "triggered": true, "job": "remind" }`.

#### GET /admin/memories?type=behavioral&tier=long_term&employee_id=xxx
Paginated memory list with filters.

#### POST /admin/memories/search
Request: `{ "query": "communication style", "limit": 10 }`
Uses embedding similarity search.

---

## Section 4: Frontend Admin Pages

### Router Changes

> **Note**: The existing router uses `createWebHashHistory()`, so actual URLs will be `/#/admin/channels` etc. All paths below are the Vue Router path values (without `#`).

Add to `frontend/src/router/index.ts`:

```typescript
{ path: "/admin/channels", name: "AdminChannels", component: () => import("@/views/admin/ChannelsView.vue"), meta: { requiresAuth: true, role: "boss" } },
{ path: "/admin/team-channels", name: "AdminTeamChannels", component: () => import("@/views/admin/TeamChannelsView.vue"), meta: { requiresAuth: true, role: "boss" } },
{ path: "/admin/reports", name: "AdminReports", component: () => import("@/views/admin/ReportsView.vue"), meta: { requiresAuth: true, role: "boss" } },
{ path: "/admin/mentor-scheduler", name: "AdminMentorScheduler", component: () => import("@/views/admin/MentorSchedulerView.vue"), meta: { requiresAuth: true, role: "boss" } },
{ path: "/admin/memory", name: "AdminMemory", component: () => import("@/views/admin/MemoryView.vue"), meta: { requiresAuth: true, role: "boss" } },
```

### Sidebar Navigation

Add "Admin" group to `App.vue` sidebar, only visible for boss role:

```
--- Admin ---
- Channels (渠道管理)
- Team Channels (团队渠道)
- Reports (报表中心)
- Mentor & Scheduler (导师 & 排程)
- Memory (记忆查看器)
```

### Page Specifications

#### Tab 1: Channels (渠道管理) — `ChannelsView.vue`

- 4 channel cards in a grid layout
- Each card shows: channel name, icon, status indicator (green/red), toggle switch
- Expandable credential forms per channel:
  - Telegram: Already configured (bot token in env), read-only display
  - Signal: Phone number display (configured via env)
  - Slack: Bot token + signing secret input fields
  - Lark: App ID + app secret input fields
- "Test" button per channel — calls `POST /admin/channels/test/:channel`, shows toast result
- "Save" button — calls `PUT /admin/channels`

#### Tab 2: Team Channels (团队渠道) — `TeamChannelsView.vue`

- Data table with columns: Name, Role, Telegram ID, Signal Phone, Slack ID, Lark ID, Preferred Channel
- Inline editing: click a cell to edit channel ID value
- Preferred channel: dropdown selector per row
- Bulk action bar: "Set preferred channel for selected" dropdown
- Search/filter bar at top
- Calls `PUT /admin/employees/:id/channels` and `PUT /admin/employees/:id/preferred` on save

#### Tab 3: Reports (报表中心) — `ReportsView.vue`

- Filter bar: date range picker, employee dropdown, channel dropdown
- Stats cards row: Total Reports, Submission Rate (%), By Channel (mini bar chart)
- Data table: Employee, Date, Channel, Time, Sentiment (emoji), Blockers (count)
- Row click → expand panel with full report answers + AI summary text
- Pagination controls
- Calls `GET /admin/reports` and `GET /admin/reports/stats`

#### Tab 4: Mentor & Scheduler (导师 & 排程) — `MentorSchedulerView.vue`

Two sections with divider:

**Mentor Section:**
- Current mentor display: name, photo/icon, philosophy summary
- Mentor selector: all mentor cards in a grid (currently 14 mentors), click to select
- Blend mode toggle: enable → shows secondary mentor selector + weight slider (0-100%)
- Save button → calls `PUT /admin/mentors` or `PUT /admin/mentors/blend`

**Scheduler Section:**
- Table: Job Name, Cron Expression (editable input), Last Run (relative time), Next Run, Status
- "Run Now" button per row → calls `POST /admin/scheduler/:job/trigger`
- Timezone selector dropdown at top → calls `PUT /admin/scheduler/timezone`
- Save button for cron changes → calls `PUT /admin/scheduler/:job/schedule`

#### Tab 5: Memory (记忆查看器) — `MemoryView.vue`

- Stats row: Total Memories, Short-term count, Long-term count, By Type breakdown
- Filter bar: memory type dropdown, tier dropdown, employee dropdown
- Search bar: semantic search input → calls `POST /admin/memories/search`
- Data table: Content (truncated 100 chars), Type, Tier, Importance (bar), Employee, Created, Expires
- Row click → modal with full content, summary, metadata JSON viewer
- Delete button per row (with confirmation) → calls `DELETE /admin/memories/:id`
- Calls `GET /admin/memories` and `GET /admin/memories/stats`

### Shared Components

- `AdminLayout.vue` — wrapper with admin sub-navigation tabs
- Reuse existing `composables/api.ts` for API calls (Bearer JWT auth)

---

## Section 5: Multi-Channel Report Collection

### Problem

Currently only Telegram users can submit daily reports (via bot long-polling). Employees on Signal, Slack, or Lark cannot participate.

### Inbound Message Flow

```
Employee sends message via any channel
    ↓
Channel-specific handler receives message
    ↓
Parse into channel.Message struct (already defined)
    ↓
Unified CommandHandler processes intent
    ↓
Report collector creates/appends report
    ↓
Event bus publishes ReportSubmitted (with channel info)
```

### Per-Channel Ingress

| Channel | Mechanism | Endpoint | Setup |
|---------|-----------|----------|-------|
| Telegram | Long polling (existing) | N/A | Already working |
| Signal | signal-cli webhook | `POST /api/v1/signal/webhook` (exists) | Already wired |
| Slack | Events API | `POST /api/v1/slack/events` (new) | Needs Slack app config |
| Lark | Event subscription | `POST /api/v1/lark/events` (new) | Needs Lark app config |

### Unified Command Handler

New file `internal/channel/commands.go`:

> **Codebase note**: The existing `bot` package has its own `IdentityQuerier` and `CommandQuerier` interfaces in `internal/bot/middleware.go` and `internal/bot/commands.go` with a `DBAdapter` bridge in `internal/bot/adapter.go`. The `CommandHandler` here uses `*sqlc.Queries` directly (same pattern as API handlers), NOT the bot package interfaces.

```go
package channel

// Named MessageHandler to avoid collision with existing CommandHandler func type in adapter.go
type MessageHandler struct {
    queries   *sqlc.Queries
    collector *report.Collector
    brain     *brain.Engine
    sender    Sender
}

// HandleMessage processes an incoming message from any channel.
func (h *CommandHandler) HandleMessage(ctx context.Context, msg Message) error {
    // 1. Resolve employee from channel type + user ID
    emp, err := h.resolveEmployee(ctx, msg.ChannelType, msg.UserID)
    if err != nil {
        return err
    }

    // 2. Handle commands
    if msg.IsCommand {
        switch msg.Command {
        case "start", "help":
            return h.handleHelp(ctx, msg, emp)
        case "status":
            return h.handleStatus(ctx, msg, emp)
        case "checkin":
            return h.handleCheckin(ctx, msg, emp)
        default:
            return h.handleUnknownCommand(ctx, msg, emp)
        }
    }

    // 3. Non-command text during active check-in → report answer
    //    Note: Collector.HandleAnswer returns (State, string, error)
    //    The returned string is the next question/response to send back
    state, response, err := h.collector.HandleAnswer(ctx, emp.ID.String(), msg.Text)
    if err != nil {
        return err
    }
    if response != "" {
        chType, chID := ResolveChannel(emp)
        return h.sender.Send(ctx, chType, chID, response)
    }
    _ = state
    return nil
}

// resolveEmployee finds employee by channel-specific ID.
// Uses Get* prefix to match existing sqlc naming convention.
// Telegram uses pgtype.Int8 wrapper (sqlc generates typed params).
func (h *CommandHandler) resolveEmployee(ctx context.Context, ct Type, userID string) (sqlc.Employee, error) {
    switch ct {
    case TypeTelegram:
        id, _ := strconv.ParseInt(userID, 10, 64)
        return h.queries.GetEmployeeByTelegramID(ctx, pgtype.Int8{Int64: id, Valid: true})
    case TypeSignal:
        return h.queries.GetEmployeeBySignalPhone(ctx, userID)
    case TypeSlack:
        return h.queries.GetEmployeeBySlackID(ctx, userID)
    case TypeLark:
        return h.queries.GetEmployeeByLarkID(ctx, userID)
    }
    return sqlc.Employee{}, fmt.Errorf("unknown channel type: %s", ct)
}
```

### Telegram Bot Refactor

The existing Telegram bot handler delegates to `CommandHandler`:

```go
// In telegram.go OnText handler:
func (t *TelegramAdapter) handleText(c tele.Context) error {
    msg := Message{
        ChannelType: TypeTelegram,
        UserID:      strconv.FormatInt(c.Sender().ID, 10),
        Text:        c.Text(),
        IsCommand:   false,
    }
    return t.msgHandler.HandleMessage(context.Background(), msg)
}
```

Existing command registrations (`/start`, `/help`, etc.) also delegate to `CommandHandler`.

### SlackConfig Extension

Add `SigningSecret` to existing `SlackConfig` in `internal/channel/slack.go`:

```go
type SlackConfig struct {
    BotToken      string // xoxb-...
    AppToken      string // xapp-... (for Socket Mode, optional)
    WebhookURL    string // Incoming webhook URL (optional)
    SigningSecret string // For verifying webhook event signatures
}
```

### Slack Webhook Handler

New file `internal/channel/slack_webhook.go` (on the channel adapter, not API handler — consistent with `SignalAdapter.HandleWebhook` pattern):

```go
// HandleSlackEvent processes Slack Events API callbacks.
// Registered on public v1 group: POST /api/v1/slack/events (no JWT, signature-verified)
func (s *SlackAdapter) HandleSlackEvent(c *gin.Context) {
    // 1. Verify Slack request signature using signing secret
    // 2. Handle URL verification challenge (return challenge value)
    // 3. Parse event payload (event_callback type)
    // 4. For message events: create channel.Message, call msgHandler.HandleMessage()
}
```

### Lark Webhook Handler

New file `internal/channel/lark_webhook.go`:

```go
// HandleLarkEvent processes Lark/Feishu event subscription callbacks.
// Registered on public v1 group: POST /api/v1/lark/events (no JWT, encrypted body)
func (l *LarkAdapter) HandleLarkEvent(c *gin.Context) {
    // 1. Decrypt event body using lark_app_secret
    // 2. Handle URL verification challenge
    // 3. Parse event payload
    // 4. For message events: create channel.Message, call msgHandler.HandleMessage()
}
```

> **Webhook registration**: Both Slack and Lark webhook endpoints are registered on the public `v1` group (no JWT auth), same as the existing Signal webhook. They use platform-specific signature/encryption verification instead. Register in `router.go` alongside the existing signal webhook:
> ```go
> // In SetupRouter, after signal webhook:
> if cfg.SlackAdapter != nil {
>     v1.POST("/slack/events", cfg.SlackAdapter.HandleSlackEvent)
> }
> if cfg.LarkAdapter != nil {
>     v1.POST("/lark/events", cfg.LarkAdapter.HandleLarkEvent)
> }
> ```

### Event Bus Enhancement

Add `Channel` field to event payloads:

```go
type ReportSubmittedPayload struct {
    EmployeeID   string `json:"employee_id"`
    EmployeeName string `json:"employee_name"`
    ReportDate   string `json:"report_date"`
    Channel      string `json:"channel"` // NEW
}

type ChaseCompletedPayload struct {
    EmployeeID   string `json:"employee_id"`
    EmployeeName string `json:"employee_name"`
    ReportDate   string `json:"report_date"`
    ChaseLogID   string `json:"chase_log_id"`
    Step         int    `json:"step"`
    Action       string `json:"action"`
    Message      string `json:"message"`
    Channel      string `json:"channel"` // NEW
}
```

### Reports Table Enhancement

Add `channel` column to track which channel a report came from:

```sql
-- In migration 000007
ALTER TABLE reports ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
ALTER TABLE chase_logs ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
```

---

## Implementation Order

Recommended build sequence:

1. **Normalize migration 000006** — extract inline vector dimension migration from `main.go` into `sql/migrations/000006_vector384.up.sql`
2. **Database migration 000007** — employee channels, tenant channels, reports.channel, chase_logs.channel
3. **sqlc queries + regenerate** — add new queries to `sql/queries/`, run `~/go/bin/sqlc generate`
4. **SlackConfig extension** — add `SigningSecret` field
5. **Channel resolution** — `internal/channel/resolve.go` helper
6. **MessageSender refactor** — replace int64-based sender with `channel.Sender` in chaser, triggers, actions, roles
7. **Admin API endpoints** — add to `RouterConfig`, create `admin_handlers.go` with closure-based handlers
8. **Frontend admin pages** — 5 views under `views/admin/`, sidebar update, router update
9. **Unified CommandHandler** — `internal/channel/commands.go`, channel-agnostic message processing
10. **Slack/Lark webhook handlers** — on channel adapters, register in router
11. **Event payload enhancement** — add `Channel` field to event payloads
12. **Testing** — unit tests for each component, integration tests for message flow

## Non-Goals (Explicit Exclusions)

- No WhatsApp, Discord, or other channels in this iteration
- No per-employee channel credential management (channels are org-level)
- No message delivery tracking / read receipts
- No real-time channel status monitoring (health checks are manual via "Test" button)
- No message queuing / retry logic (rely on channel adapter's built-in error handling)
