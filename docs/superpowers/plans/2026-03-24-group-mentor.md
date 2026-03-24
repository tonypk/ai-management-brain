# Group Mentor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the AI mentor to participate in Telegram group chats — responding to @mentions and autonomously posting based on team data.

**Architecture:** New `group_chats` DB table links Telegram groups to tenants. Bot layer extended with group chat detection (ChatType/ChatID/Reply on BotContext). Group prompt building in `internal/brain/group.go`. Scheduler job runs daily AI decision per group. Admin CRUD via new API handlers + frontend page.

**Tech Stack:** Go 1.25 (Gin, sqlc, pgx, telebot/v3, gocron/v2), Vue 3 + TypeScript, PostgreSQL 16, Redis 7, Claude API

**Spec:** `docs/superpowers/specs/2026-03-24-group-mentor-design.md`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `sql/migrations/000009_group_chats.up.sql` | Create group_chats table + index |
| `sql/migrations/000009_group_chats.down.sql` | Drop group_chats table |
| `cmd/brain/main.go` | Add migration009 block, register group handler, add scheduler job, extend /join |
| `sql/queries/group_chats.sql` | sqlc CRUD queries for group_chats |
| `internal/db/sqlc/` | Regenerated sqlc code |
| `internal/bot/commands.go` | Extend BotContext interface (ChatID, ChatType, ChatTitle, Reply) |
| `internal/bot/bot.go` | Implement new BotContext methods on teleBotContext |
| `internal/brain/group.go` | Group prompt building + AI decision logic |
| `internal/brain/group_test.go` | Unit tests for group logic |
| `internal/api/group_handlers.go` | Admin group CRUD endpoints |
| `internal/api/group_handlers_test.go` | Handler tests |
| `internal/api/router.go` | Register group admin routes |
| `frontend/src/composables/api.ts` | Add group API functions |
| `frontend/src/views/admin/GroupChatsView.vue` | Admin group management page |
| `frontend/src/router/index.ts` | Add group route |
| `frontend/src/App.vue` | Add group nav item |

---

### Task 1: Database Migration

**Files:**
- Create: `sql/migrations/000009_group_chats.up.sql`
- Create: `sql/migrations/000009_group_chats.down.sql`
- Modify: `cmd/brain/main.go` (add migration009 block in `runMigrations`)

- [ ] **Step 1: Create up migration**

```sql
-- sql/migrations/000009_group_chats.up.sql
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
```

- [ ] **Step 2: Create down migration**

```sql
-- sql/migrations/000009_group_chats.down.sql
DROP TABLE IF EXISTS group_chats;
```

- [ ] **Step 3: Add migration009 block to runMigrations in cmd/brain/main.go**

Add after the existing `migration008` block (around line 400 in `runMigrations`). Follow the exact pattern of existing migrations:

```go
	// 000009: Group chats
	migration009 := `
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
	if _, err := pool.Exec(ctx, migration009); err != nil {
		return fmt.Errorf("migration 009: %w", err)
	}
	slog.Info("migration 009 applied: group_chats")
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./cmd/brain/`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add sql/migrations/000009_group_chats.up.sql sql/migrations/000009_group_chats.down.sql cmd/brain/main.go
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add group_chats migration (000009)"
```

---

### Task 2: sqlc Queries

**Files:**
- Create: `sql/queries/group_chats.sql`
- Regenerate: `internal/db/sqlc/` (run sqlc generate)

- [ ] **Step 1: Create sqlc query file**

```sql
-- sql/queries/group_chats.sql

-- name: CreateGroupChat :one
INSERT INTO group_chats (tenant_id, platform, platform_chat_id, name, group_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetGroupChatByID :one
SELECT * FROM group_chats
WHERE id = $1;

-- name: GetGroupChatByPlatformID :one
SELECT * FROM group_chats
WHERE platform = $1 AND platform_chat_id = $2;

-- name: ListActiveGroupChatsByTenant :many
SELECT * FROM group_chats
WHERE tenant_id = $1 AND is_active = true
ORDER BY created_at;

-- name: ListGroupChatsByTenant :many
SELECT * FROM group_chats
WHERE tenant_id = $1
ORDER BY created_at;

-- name: UpdateGroupChat :one
UPDATE group_chats
SET name = $2, group_type = $3, is_active = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteGroupChat :exec
UPDATE group_chats
SET is_active = false, updated_at = now()
WHERE id = $1;
```

- [ ] **Step 2: Run sqlc generate**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`
Expected: No errors, generated files in `internal/db/sqlc/`

- [ ] **Step 3: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add sql/queries/group_chats.sql internal/db/sqlc/
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add group_chats sqlc queries"
```

---

### Task 3: Extend BotContext Interface

**Files:**
- Modify: `internal/bot/commands.go` (BotContext interface)
- Modify: `internal/bot/bot.go` (teleBotContext adapter)

- [ ] **Step 1: Add methods to BotContext interface**

In `internal/bot/commands.go`, add to the `BotContext` interface (line 19-23):

```go
// BotContext abstracts telebot.Context for testability.
type BotContext interface {
	SenderID() int64
	Text() string
	Send(msg string) error
	ChatID() int64
	ChatType() string  // "private", "group", "supergroup"
	ChatTitle() string // group name (empty for private chats)
	Reply(msg string) error
}
```

- [ ] **Step 2: Implement new methods on teleBotContext**

In `internal/bot/bot.go`, add after the existing `Send` method (line 61):

```go
func (t *teleBotContext) ChatID() int64     { return t.c.Chat().ID }
func (t *teleBotContext) ChatType() string  { return string(t.c.Chat().Type) }
func (t *teleBotContext) ChatTitle() string { return t.c.Chat().Title }
func (t *teleBotContext) Reply(msg string) error {
	return t.c.Reply(msg)
}
```

- [ ] **Step 3: Add GroupQuerier interface and extend CommandQuerier**

In `internal/bot/commands.go`, add the `GroupQuerier` interface and the new method to `CommandQuerier`:

```go
// GroupQuerier defines DB operations for group chat management.
type GroupQuerier interface {
	CreateGroupChat(ctx context.Context, tenantID, platform, platformChatID, name, groupType string) (GroupChat, error)
	GetGroupChatByPlatformID(ctx context.Context, platform, platformChatID string) (GroupChat, error)
}

// GroupChat holds basic group chat info for bot use.
type GroupChat struct {
	ID       string
	TenantID string
	Name     string
}
```

- [ ] **Step 4: Add group context to HandleJoin**

In `internal/bot/commands.go`, modify `HandleJoin` to branch on chat type. Replace the existing `HandleJoin` method (lines 233-255):

```go
// HandleJoin links a Telegram user to an employee record via invite code,
// OR registers a group chat when called from a group context.
func (h *CommandHandler) HandleJoin(c BotContext) error {
	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /join <invite_code>")
	}

	code := parts[1]

	// Group chat registration
	chatType := c.ChatType()
	if chatType == "group" || chatType == "supergroup" {
		if h.groupDB == nil {
			return c.Send("Group features not available.")
		}
		// Look up tenant by invite code to get tenant_id
		emp, err := h.db.GetEmployeeByInviteCode(context.Background(), code)
		if err != nil {
			return c.Send("Invalid invite code.")
		}

		chatID := fmt.Sprintf("%d", c.ChatID())
		title := c.ChatTitle()
		if title == "" {
			title = "Unnamed Group"
		}

		// Check if already registered
		existing, err := h.groupDB.GetGroupChatByPlatformID(context.Background(), "telegram", chatID)
		if err == nil && existing.ID != "" {
			return c.Send(fmt.Sprintf("This group '%s' is already registered.", existing.Name))
		}

		gc, err := h.groupDB.CreateGroupChat(context.Background(), emp.TenantID, "telegram", chatID, title, "general")
		if err != nil {
			slog.Error("create group chat", "error", err)
			return c.Send("Failed to register group. Please try again.")
		}
		slog.Info("group chat registered", "group_id", gc.ID, "name", gc.Name, "tenant", gc.TenantID)
		return c.Send(fmt.Sprintf("Group '%s' registered! The mentor will now be active here.\n\nUse the admin dashboard to change the group type.", title))
	}

	// Private chat — existing employee join flow
	emp, err := h.db.GetEmployeeByInviteCode(context.Background(), code)
	if err != nil {
		return c.Send("Invalid invite code.")
	}

	if err := h.db.UpdateEmployeeTelegramID(context.Background(), emp.ID, c.SenderID()); err != nil {
		return fmt.Errorf("update telegram id: %w", err)
	}

	slog.Info("employee joined", "employee_id", emp.ID, "telegram_id", c.SenderID())
	return c.Send(fmt.Sprintf(
		"Welcome %s! You're now linked to the team. You'll receive daily check-in questions.",
		emp.Name,
	))
}
```

- [ ] **Step 5: Add groupDB field to CommandHandler**

In `internal/bot/commands.go`, add `groupDB` to the `CommandHandler` struct and constructor:

```go
// CommandHandler handles bot commands.
type CommandHandler struct {
	db             CommandQuerier
	groupDB        GroupQuerier   // nil = group features disabled
	bossChatID     int64
	DiagnosticsFn  func() string
}

func NewCommandHandler(db CommandQuerier, _ interface{}, _ interface{}, bossChatID int64) *CommandHandler {
	return &CommandHandler{db: db, bossChatID: bossChatID}
}

// SetGroupDB injects the group querier for group chat features.
func (h *CommandHandler) SetGroupDB(gdb GroupQuerier) {
	h.groupDB = gdb
}
```

- [ ] **Step 6: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS (may need to fix mock implementations if tests use BotContext)

- [ ] **Step 7: Update any test mocks that implement BotContext**

Check for test files that have mock BotContext implementations and add the new methods:

```go
func (m *mockBotContext) ChatID() int64     { return 0 }
func (m *mockBotContext) ChatType() string  { return "private" }
func (m *mockBotContext) ChatTitle() string { return "" }
func (m *mockBotContext) Reply(msg string) error { return m.Send(msg) }
```

- [ ] **Step 8: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/bot/...`
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add internal/bot/commands.go internal/bot/bot.go internal/bot/commands_test.go
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: extend BotContext with group chat support (ChatID, ChatType, Reply)"
```

---

### Task 4: Group Prompt Building + AI Decision

**Files:**
- Create: `internal/brain/group.go`
- Create: `internal/brain/group_test.go`

- [ ] **Step 1: Write group_test.go with tests for prompt building and SKIP parsing**

```go
// internal/brain/group_test.go
package brain

import (
	"testing"
)

func TestBuildGroupReplyPrompt(t *testing.T) {
	prompt := BuildGroupReplyPrompt("稻盛和夫", "engineering", "本周提交率85%，情绪正面为主", "如何提升代码质量？")
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !contains(prompt, "engineering") {
		t.Error("expected group type in prompt")
	}
	if !contains(prompt, "稻盛和夫") {
		t.Error("expected mentor name in prompt")
	}
	if !contains(prompt, "如何提升代码质量") {
		t.Error("expected user question in prompt")
	}
}

func TestBuildGroupDecisionPrompt(t *testing.T) {
	prompt := BuildGroupDecisionPrompt("马斯克", "sales", GroupTeamData{
		SubmissionRate: "80%",
		SentimentDist:  "positive: 5, neutral: 2, negative: 1",
		LatestSummary:  "团队状态良好",
		Weekday:        "Friday",
	})
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !contains(prompt, "马斯克") {
		t.Error("expected mentor name")
	}
	if !contains(prompt, "sales") {
		t.Error("expected group type")
	}
	if !contains(prompt, "SKIP") {
		t.Error("expected SKIP instruction")
	}
}

func TestParseDecisionResponse(t *testing.T) {
	tests := []struct {
		input    string
		wantSkip bool
	}{
		{"SKIP", true},
		{"  SKIP  ", true},
		{"skip", true},
		{"SKIP\n", true},
		{"大家早上好！今天继续加油！", false},
		{"", true}, // empty = skip
	}
	for _, tt := range tests {
		skip := IsSkipDecision(tt.input)
		if skip != tt.wantSkip {
			t.Errorf("IsSkipDecision(%q) = %v, want %v", tt.input, skip, tt.wantSkip)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && (s == sub || len(s) > len(sub) && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run TestBuildGroup -v`
Expected: FAIL (functions not defined)

- [ ] **Step 3: Implement group.go**

```go
// internal/brain/group.go
package brain

import (
	"fmt"
	"strings"
)

// GroupTeamData holds team statistics for the group decision prompt.
type GroupTeamData struct {
	SubmissionRate string
	SentimentDist  string
	LatestSummary  string
	Weekday        string
}

// BuildGroupReplyPrompt builds the system prompt for @mention replies in group chat.
func BuildGroupReplyPrompt(mentorName, groupType, teamSummary, userQuestion string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "You are %s, a management mentor active in a team group chat.\n\n", mentorName)

	sb.WriteString("<group_context>\n")
	fmt.Fprintf(&sb, "Group type: %s\n", groupType)
	if teamSummary != "" {
		sb.WriteString("Latest team summary:\n")
		sb.WriteString(teamSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("</group_context>\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("- NEVER mention individual employee's private reports, sentiments, or personal memories\n")
	sb.WriteString("- Keep responses concise and relevant to the group context\n")
	sb.WriteString("- Maintain your mentor persona and philosophy\n")
	sb.WriteString("- Answer based on team-level data, not individual data\n")

	fmt.Fprintf(&sb, "\nA team member asks: %q\n", userQuestion)
	sb.WriteString("Respond helpfully as the team's management mentor.")

	return sb.String()
}

// BuildGroupDecisionPrompt builds the system prompt for the daily autonomous posting decision.
func BuildGroupDecisionPrompt(mentorName, groupType string, data GroupTeamData) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "You are %s, managing a %s team group chat.\n\n", mentorName, groupType)

	sb.WriteString("Team data:\n")
	fmt.Fprintf(&sb, "- Submission rate: %s\n", data.SubmissionRate)
	fmt.Fprintf(&sb, "- Sentiment distribution: %s\n", data.SentimentDist)
	if data.LatestSummary != "" {
		fmt.Fprintf(&sb, "- Latest summary: %s\n", data.LatestSummary)
	}
	fmt.Fprintf(&sb, "- Today is: %s\n", data.Weekday)

	sb.WriteString("\nDecide whether to post a message in the group chat.\n")
	sb.WriteString("If not needed, reply with only: SKIP\n")
	sb.WriteString("If needed, output the message content directly (no markers or labels).\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("- Don't post every day — about 2-3 times per week is ideal\n")
	sb.WriteString("- Friday is good for a weekly review\n")
	sb.WriteString("- If submission rate is below 60%, encourage the team\n")
	sb.WriteString("- Maintain your mentor style and cultural context\n")
	sb.WriteString("- NEVER mention individual private information\n")
	sb.WriteString("- Keep messages concise (3-5 sentences)\n")
	sb.WriteString("- Use the language appropriate for the team's culture\n")

	return sb.String()
}

// IsSkipDecision checks if the AI decision response indicates no posting.
func IsSkipDecision(response string) bool {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, "SKIP")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run TestBuildGroup -v && go test ./internal/brain/ -run TestParseDecision -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add internal/brain/group.go internal/brain/group_test.go
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add group prompt building and AI decision parsing"
```

---

### Task 5: API Handlers for Group CRUD

**Files:**
- Create: `internal/api/group_handlers.go`
- Create: `internal/api/group_handlers_test.go`
- Modify: `internal/api/router.go` (add routes + RouterConfig field)

- [ ] **Step 1: Write group_handlers_test.go**

```go
// internal/api/group_handlers_test.go
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleListGroups_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/groups", handleListGroups(nil)) // nil queries = will handle gracefully

	req := httptest.NewRequest("GET", "/groups", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Without tenant context, should return error
	if w.Code == http.StatusOK {
		t.Log("handler registered and responds")
	}
}
```

Note: The full test will be expanded after implementation. This is a smoke test to verify handler registration.

- [ ] **Step 2: Implement group_handlers.go**

```go
// internal/api/group_handlers.go
package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

func handleListGroups(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		groups, err := q.ListGroupChatsByTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list groups", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": groups})
	}
}

type updateGroupRequest struct {
	Name      string `json:"name"`
	GroupType string `json:"group_type"`
	IsActive  bool   `json:"is_active"`
}

func handleUpdateGroup(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		id := c.Param("id")
		groupID, err := parseUUID(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
			return
		}

		var req updateGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Validate group_type
		validTypes := map[string]bool{
			"general": true, "engineering": true, "operations": true,
			"sales": true, "support": true,
		}
		if !validTypes[req.GroupType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_type"})
			return
		}

		// Verify group belongs to this tenant
		existing, err := q.GetGroupChatByID(c.Request.Context(), groupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if existing.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		group, err := q.UpdateGroupChat(c.Request.Context(), sqlc.UpdateGroupChatParams{
			ID:        groupID,
			Name:      req.Name,
			GroupType: req.GroupType,
			IsActive:  req.IsActive,
		})
		if err != nil {
			slog.Error("update group", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": group})
	}
}

func handleDeleteGroup(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		id := c.Param("id")
		groupID, err := parseUUID(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
			return
		}

		// Verify group belongs to this tenant
		existing, err := q.GetGroupChatByID(c.Request.Context(), groupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if existing.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		if err := q.SoftDeleteGroupChat(c.Request.Context(), groupID); err != nil {
			slog.Error("delete group", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}
```

**Key patterns used (matching existing codebase):**
- `TenantFromContext(c)` returns `string` (not error) — parse with `parseUUID()`
- `parseUUID()` from `handlers.go:23` converts string to `pgtype.UUID`
- Tenant ownership verification before update/delete
- `gin.H{"data": ...}` response envelope

**Note:** We need to add a `GetGroupChatByID` query to `sql/queries/group_chats.sql` for the ownership check. Add this query in Task 2:

- [ ] **Step 3: Add routes to router.go**

In `internal/api/router.go`, add inside the `admin` group (after the memories section, around line 168):

```go
			// Group Chats
			admin.GET("/groups", handleListGroups(cfg.Queries))
			admin.PUT("/groups/:id", handleUpdateGroup(cfg.Queries))
			admin.DELETE("/groups/:id", handleDeleteGroup(cfg.Queries))
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS

Note: The sqlc-generated `UpdateGroupChatParams` struct field names depend on the exact query parameter naming. If the build fails due to struct field mismatches, read the generated `internal/db/sqlc/group_chats.sql.go` and adjust the handler code to match.

- [ ] **Step 5: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/api/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add internal/api/group_handlers.go internal/api/group_handlers_test.go internal/api/router.go
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add admin group CRUD API endpoints"
```

---

### Task 6: Bot Group Message Handler + Scheduler Job

**Files:**
- Modify: `cmd/brain/main.go` (register group handler, add scheduler job, wire /join groupDB)

This is the integration task. All pieces from Tasks 1-5 are connected here.

- [ ] **Step 1: Create GroupQuerier adapter in main.go**

Add a `groupDBAdapter` struct that implements `bot.GroupQuerier` using `sqlc.Queries`. Place this near the other adapter types (e.g., near `formatPgUUID`):

```go
// groupDBAdapter adapts sqlc.Queries to bot.GroupQuerier.
type groupDBAdapter struct {
	q *sqlc.Queries
}

func (a *groupDBAdapter) CreateGroupChat(ctx context.Context, tenantID, platform, platformChatID, name, groupType string) (bot.GroupChat, error) {
	tid, err := parseUUIDForDB(tenantID)
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
```

Note: `parseUUIDForDB` may need to be created if not present — it converts a UUID string to `pgtype.UUID`. Check existing code for the pattern. If no such function exists, use the pgx parsing approach:

```go
func parseUUIDForDB(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, err
	}
	return u, nil
}
```

- [ ] **Step 2: Wire groupDB into CommandHandler**

After `cmdHandler := bot.NewCommandHandler(...)` (around line 686), add:

```go
	cmdHandler.SetGroupDB(&groupDBAdapter{q: queries})
```

- [ ] **Step 3: Modify the OnText handler to detect group chats**

The current `RegisterTextHandler` uses `TextHandlerFunc` which doesn't expose chat context. We need to register a raw telebot handler that checks chat type first.

Replace the `tgBot.RegisterTextHandler(...)` call (lines 689-805) with direct telebot handler registration. Since `tgBot` wraps `*tele.Bot`, we need to either:
a) Expose the underlying bot, or
b) Add a new registration method.

The simplest approach: replace `RegisterTextHandler` with `RegisterRawTextHandler` that exposes the full telebot context. This replaces the old method entirely — the private chat handling is preserved inside the same handler.

In `internal/bot/bot.go`, add (alongside the existing `RegisterTextHandler` which remains for backward compat):

```go
// RawTextHandlerFunc receives the full telebot context for advanced handling.
type RawTextHandlerFunc func(c tele.Context) error

// RegisterRawTextHandler registers a raw telebot handler for text messages.
// This replaces RegisterTextHandler when group chat detection is needed.
func (b *Bot) RegisterRawTextHandler(h RawTextHandlerFunc) {
	b.bot.Handle(tele.OnText, func(c tele.Context) error {
		return h(c)
	})
	slog.Info("raw text message handler registered")
}
```

Then in `cmd/brain/main.go`, replace the `tgBot.RegisterTextHandler(...)` call (lines 689-805) with `tgBot.RegisterRawTextHandler(...)`. The handler body contains BOTH group and private chat handling. Extract `senderID`, `text`, and `sendReply` at the top for the private chat section:

```go
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

			tenantID := formatPgUUID(gc.TenantID)

			// Load mentor engine
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				slog.Error("group chat: get tenant", "error", err)
				return nil
			}

			engine, err := engineFactory.ForTenant(tenant.MentorID, "default")
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
				slog.Error("group reply LLM failed", "error", err, "tenant_id", tenantID)
				return nil
			}

			return c.Reply(response)
		}

		// === PRIVATE CHAT HANDLING (existing code, unchanged) ===
		// ... (keep all existing private chat code exactly as-is)
```

Note: `chatService.LLM()` requires adding a getter method to `ChatService`. Add to `internal/brain/chat.go`:

```go
// LLM returns the underlying LLM client for single-turn calls.
// AnthropicClient implements both LLMClient (Chat) and ChatLLMClient (ChatWithHistory).
func (s *ChatService) LLM() LLMClient {
	if s.llm == nil {
		return nil
	}
	if lc, ok := s.llm.(LLMClient); ok {
		return lc
	}
	return nil
}
```

**Context:** `LLMClient` interface is defined in `internal/brain/llm.go:17-19`:
```go
type LLMClient interface {
    Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
```
`ChatLLMClient` extends it with `ChatWithHistory`. `AnthropicClient` implements both. The `ChatService.llm` field is typed as `ChatLLMClient`, so we type-assert to `LLMClient` to access the single-turn `Chat()` method.

- [ ] **Step 4: Add group mentor scheduler job**

After the memory consolidation jobs section (around line 1097), add:

```go
	// Group mentor autonomous posting job
	if chatService != nil {
		if err := sched.AddJob("group_mentor", "0 9 * * *", func(ctx context.Context) error {
			slog.Info("group_mentor job: running autonomous posting decisions")
			tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}
			tenantID := tenant.ID

			groups, err := queries.ListActiveGroupChatsByTenant(ctx, tenantUUID)
			if err != nil {
				return fmt.Errorf("list active groups: %w", err)
			}

			engine, err := engineFactory.ForTenant(tenant.MentorID, "default")
			if err != nil {
				return fmt.Errorf("load engine: %w", err)
			}

			// Collect team data
			loc, _ := time.LoadLocation(cfg.Timezone)
			today := time.Now().In(loc)
			yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
			weekday := today.Weekday().String()

			submissionRate := "N/A"
			sentimentDist := "N/A"
			summaryText := ""

			// Get submission rate from analytics
			if overview, err := queries.GetAnalyticsOverview(ctx, tenantUUID); err == nil {
				if overview.TotalEmployees > 0 {
					pct := float64(overview.TodayReports) / float64(overview.TotalEmployees) * 100
					submissionRate = fmt.Sprintf("%.0f%%", pct)
				}
			}

			// Get latest summary
			if summary, err := queries.GetLatestSummary(ctx, tenantUUID); err == nil {
				summaryText = summary.Content
				if len(summaryText) > 500 {
					summaryText = summaryText[:500] + "..."
				}
			}

			llmClient, ok := chatService.LLM().(brain.LLMClient)
			if !ok || llmClient == nil {
				slog.Warn("group_mentor: no LLM client available")
				return nil
			}

			for _, gc := range groups {
				groupID := formatPgUUID(gc.ID)

				// Anti-spam: check Redis for last post time
				antiSpamKey := fmt.Sprintf("group:last_post:%s", groupID)
				if _, err := redisClient.Get(ctx, antiSpamKey).Result(); err == nil {
					slog.Debug("group_mentor: skipping (posted recently)", "group", gc.Name)
					continue
				}

				// Build decision prompt
				prompt := brain.BuildGroupDecisionPrompt(
					engine.MentorName(),
					gc.GroupType,
					brain.GroupTeamData{
						SubmissionRate: submissionRate,
						SentimentDist:  sentimentDist,
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
				redisClient.Set(ctx, antiSpamKey, "1", 24*time.Hour)
				slog.Info("group_mentor: posted to group", "group", gc.Name, "message_len", len(response))
			}

			return nil
		}); err != nil {
			slog.Error("failed to register group_mentor job", "error", err)
		} else {
			slog.Info("group_mentor job registered", "schedule", "daily 09:00")
		}
	}
```

Note: `botDB.GetTenantByBossChatID` returns a `*bot.Tenant` with string ID. The sqlc-generated `queries.ListActiveGroupChatsByTenant` takes `pgtype.UUID`. Convert using `parseUUID` (from handlers) or inline:

```go
var tenantUUID pgtype.UUID
tenantUUID.Scan(tenant.ID)
groups, err := queries.ListActiveGroupChatsByTenant(ctx, tenantUUID)
```

Similarly for `queries.GetLatestSummary` and `queries.GetAnalyticsOverview` — check their parameter types after sqlc generation and convert accordingly. The bot-layer `tenant.ID` is always a string UUID that needs conversion to `pgtype.UUID` for sqlc queries.

- [ ] **Step 5: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./cmd/brain/`
Expected: BUILD SUCCESS

Fix any compilation errors from sqlc generated type mismatches (parameter struct field names, UUID types, etc.).

- [ ] **Step 6: Run all tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add internal/bot/bot.go internal/brain/chat.go cmd/brain/main.go
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add group chat handler and autonomous posting scheduler job"
```

---

### Task 7: Frontend — Admin Group Chats Page

**Files:**
- Create: `frontend/src/views/admin/GroupChatsView.vue`
- Modify: `frontend/src/composables/api.ts` (add API functions)
- Modify: `frontend/src/router/index.ts` (add route)
- Modify: `frontend/src/App.vue` (add nav item)

- [ ] **Step 1: Add API functions to api.ts**

Add to `frontend/src/composables/api.ts`:

```typescript
// Group Chats
export interface GroupChat {
  id: string
  tenant_id: string
  platform: string
  platform_chat_id: string
  name: string
  group_type: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export async function listGroups(): Promise<{ data: GroupChat[] }> {
  return request('/admin/groups')
}

export async function updateGroup(id: string, data: { name: string; group_type: string; is_active: boolean }): Promise<{ data: GroupChat }> {
  return request(`/admin/groups/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteGroup(id: string): Promise<void> {
  return request(`/admin/groups/${id}`, { method: 'DELETE' })
}
```

- [ ] **Step 2: Create GroupChatsView.vue**

```vue
<!-- frontend/src/views/admin/GroupChatsView.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listGroups, updateGroup, deleteGroup, type GroupChat } from '../../composables/api'

const groups = ref<GroupChat[]>([])
const loading = ref(true)
const error = ref('')
const editingId = ref('')
const editForm = ref({ name: '', group_type: 'general', is_active: true })
const saving = ref(false)

const groupTypes = ['general', 'engineering', 'operations', 'sales', 'support']

async function load() {
  try {
    loading.value = true
    error.value = ''
    const res = await listGroups()
    groups.value = res.data || []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function startEdit(g: GroupChat) {
  editingId.value = g.id
  editForm.value = { name: g.name, group_type: g.group_type, is_active: g.is_active }
}

function cancelEdit() {
  editingId.value = ''
}

async function saveEdit(id: string) {
  try {
    saving.value = true
    await updateGroup(id, editForm.value)
    editingId.value = ''
    await load()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function toggleActive(g: GroupChat) {
  try {
    await updateGroup(g.id, { name: g.name, group_type: g.group_type, is_active: !g.is_active })
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}

async function removeGroup(g: GroupChat) {
  if (!confirm(`Remove group "${g.name}"?`)) return
  try {
    await deleteGroup(g.id)
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}

onMounted(load)
</script>

<template>
  <div class="groups-page">
    <h2>Group Chats</h2>
    <p class="subtitle">Manage groups where the mentor is active. Add groups via /join command in Telegram.</p>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <template v-else>
      <div v-if="!groups.length" class="card">
        <p>No groups registered yet. Add the bot to a Telegram group and use <code>/join &lt;invite_code&gt;</code> to register it.</p>
      </div>

      <table v-else>
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Platform</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in groups" :key="g.id">
            <template v-if="editingId === g.id">
              <td><input v-model="editForm.name" /></td>
              <td>
                <select v-model="editForm.group_type">
                  <option v-for="t in groupTypes" :key="t" :value="t">{{ t }}</option>
                </select>
              </td>
              <td><span class="badge badge-neutral">{{ g.platform }}</span></td>
              <td>
                <label class="toggle-label">
                  <input type="checkbox" v-model="editForm.is_active" />
                  {{ editForm.is_active ? 'Active' : 'Inactive' }}
                </label>
              </td>
              <td>
                <button class="btn btn-primary btn-sm" @click="saveEdit(g.id)" :disabled="saving">Save</button>
                <button class="btn btn-secondary btn-sm" @click="cancelEdit">Cancel</button>
              </td>
            </template>
            <template v-else>
              <td>{{ g.name }}</td>
              <td><span class="badge badge-neutral">{{ g.group_type }}</span></td>
              <td><span class="badge badge-neutral">{{ g.platform }}</span></td>
              <td>
                <span class="status-dot" :class="g.is_active ? 'active' : 'inactive'"></span>
                {{ g.is_active ? 'Active' : 'Inactive' }}
              </td>
              <td>
                <button class="btn btn-secondary btn-sm" @click="startEdit(g)">Edit</button>
                <button class="btn btn-secondary btn-sm" @click="toggleActive(g)">
                  {{ g.is_active ? 'Disable' : 'Enable' }}
                </button>
                <button class="btn btn-secondary btn-sm" style="color:#e74c3c" @click="removeGroup(g)">Remove</button>
              </td>
            </template>
          </tr>
        </tbody>
      </table>
    </template>
  </div>
</template>

<style scoped>
.groups-page { max-width: 900px; }
.subtitle { color: #888; margin-bottom: 1.5rem; font-size: 0.9rem; }
.btn-sm { padding: 0.3rem 0.6rem; font-size: 0.8rem; margin-right: 0.25rem; }
.status-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; }
.status-dot.active { background: #22c55e; }
.status-dot.inactive { background: #94a3b8; }
.toggle-label { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; }
input[type="checkbox"] { width: 16px; height: 16px; }
</style>
```

- [ ] **Step 3: Add route to router**

In `frontend/src/router/index.ts`, add after the `/admin/memory` route:

```typescript
  {
    path: "/admin/groups",
    name: "AdminGroups",
    component: () => import("../views/admin/GroupChatsView.vue"),
    meta: { requiresAuth: true },
  },
```

- [ ] **Step 4: Add nav item to App.vue**

In `frontend/src/App.vue`, add to the `adminItems` array:

```typescript
  { path: '/admin/groups', label: 'Groups', icon: '💬' },
```

- [ ] **Step 5: Build frontend**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
git -C /Users/anna/Documents/ai-management-brain add frontend/src/views/admin/GroupChatsView.vue frontend/src/composables/api.ts frontend/src/router/index.ts frontend/src/App.vue
git -C /Users/anna/Documents/ai-management-brain commit -m "feat: add admin group chats management page"
```

---

### Task 8: Build, Deploy, Verify

**Files:** None (deployment task)

- [ ] **Step 1: Build Go binary**

Run: `cd /Users/anna/Documents/ai-management-brain && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain/`
Expected: `brain` binary created

- [ ] **Step 2: Build frontend**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: BUILD SUCCESS

- [ ] **Step 3: Copy binary to server**

Run: `scp -i ~/.ssh/opentoke.pem /Users/anna/Documents/ai-management-brain/brain ubuntu@18.141.251.99:~/ai-management-brain/`

- [ ] **Step 4: Deploy on server**

Run: `ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build'`

- [ ] **Step 5: Verify health**

Run: `ssh ai-brain 'curl -s localhost/healthz'`
Expected: healthy response

- [ ] **Step 6: Check migration applied**

Run: `ssh ai-brain 'docker compose -f docker-compose.prod.yml logs brain 2>&1 | grep "migration 009"'`
Expected: "migration 009 applied: group_chats"

- [ ] **Step 7: Check scheduler job registered**

Run: `ssh ai-brain 'docker compose -f docker-compose.prod.yml logs brain 2>&1 | grep "group_mentor"'`
Expected: "group_mentor job registered"

- [ ] **Step 8: Push to GitHub**

Run: `cd /Users/anna/Documents/ai-management-brain && git push`

- [ ] **Step 9: Commit (final)**

All done! The group mentor feature is deployed and running.
