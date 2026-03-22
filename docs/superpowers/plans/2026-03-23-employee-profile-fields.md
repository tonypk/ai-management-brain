# Employee Profile Fields Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add job_title, responsibilities, country, and language fields to employees — injected into AI mentor prompts and exposed in bot/API/frontend.

**Architecture:** 4 new TEXT columns with empty string defaults. Brain layer gets `EmployeeChatRequest` struct (replacing parameter list) and `EmployeeContext` for prompt building. `<employee_context>` block injected into mentor prompts; `language` field forces AI reply language. All entry points (Telegram bot, REST API, Vue frontend) updated.

**Tech Stack:** Go 1.25 (Gin, sqlc, pgx), Vue3+TS, PostgreSQL 16, telebot/v3

**Spec:** `docs/superpowers/specs/2026-03-23-employee-profile-fields-design.md`

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `cmd/brain/main.go` | Inline migrations, Telegram handler wiring, fetchBossContext | Modify |
| `sql/queries/employees.sql` | sqlc query definitions | Modify |
| `internal/db/sqlc/` | Generated code (models.go, employees.sql.go) | Regenerate |
| `internal/brain/chat.go` | ChatService, EmployeeChatRequest, RosterEntry | Modify |
| `internal/brain/chat_test.go` | ChatService tests | Modify |
| `internal/brain/engine.go` | BuildEmployeeChatPrompt, EmployeeContext | Modify |
| `internal/brain/engine_chat_test.go` | Engine prompt tests | Modify |
| `internal/bot/middleware.go` | bot.Employee struct | Modify |
| `internal/bot/adapter.go` | CreateEmployeeParams, sqlcEmployeeToBot, CreateEmployee | Modify |
| `internal/bot/commands.go` | HandleAddEmployee, HandleProfile, HandleHelp, HandleStart | Modify |
| `internal/api/handlers.go` | createEmployeeRequest, list/get/create handlers, new updateProfile | Modify |
| `internal/api/router.go` | Employee routes | Modify |
| `internal/channel/message_handler.go` | OnText callback signature, resolveEmployee | Modify |
| `frontend/src/composables/api.ts` | Employee interface, createEmployee, updateEmployeeProfile | Modify |
| `frontend/src/views/EmployeesView.vue` | Create form, table, edit modal | Modify |

---

### Task 1: Database Migration & sqlc Queries

**Files:**
- Modify: `cmd/brain/main.go:370-387` (add migration 000008)
- Modify: `sql/queries/employees.sql` (update CreateEmployee, add UpdateEmployeeProfile, update ListEmployeesWithChannels)
- Regenerate: `internal/db/sqlc/` (models.go, employees.sql.go)

- [ ] **Step 1: Add migration 000008 to main.go**

In `cmd/brain/main.go`, after the `migration007` block (line 384), add a new migration constant and execute it. Find the line:

```go
	_, err := pool.Exec(ctx, migration007)
	return err
```

Replace with:

```go
	if _, err := pool.Exec(ctx, migration007); err != nil {
		return err
	}

	const migration008 = `
-- 000008: employee profile fields
ALTER TABLE employees ADD COLUMN IF NOT EXISTS job_title       TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS country         TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS language        TEXT NOT NULL DEFAULT '';
`
	_, err := pool.Exec(ctx, migration008)
	return err
```

- [ ] **Step 2: Update CreateEmployee query in employees.sql**

Replace the current CreateEmployee query (lines 10-13 of `sql/queries/employees.sql`):

```sql
-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
```

With:

```sql
-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code, job_title, responsibilities, country, language)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;
```

- [ ] **Step 3: Add UpdateEmployeeProfile query**

Append after the `UpdateEmployeePreferredChannel` query (after line 44):

```sql
-- name: UpdateEmployeeProfile :exec
UPDATE employees
SET job_title = $2, responsibilities = $3, country = $4, language = $5
WHERE id = $1;
```

- [ ] **Step 4: Update ListEmployeesWithChannels query**

Replace the current query (lines 46-50):

```sql
-- name: ListEmployeesWithChannels :many
SELECT id, tenant_id, name, telegram_id, signal_phone, slack_id, lark_id, preferred_channel, culture_code, role, is_active
FROM employees
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;
```

With:

```sql
-- name: ListEmployeesWithChannels :many
SELECT id, tenant_id, name, telegram_id, signal_phone, slack_id, lark_id, preferred_channel, culture_code, role, is_active, job_title, responsibilities, country, language
FROM employees
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;
```

- [ ] **Step 5: Regenerate sqlc**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`

Verify `internal/db/sqlc/models.go` now has `JobTitle`, `Responsibilities`, `Country`, `Language` fields on the `Employee` struct. Verify `CreateEmployeeParams` now has the 4 new fields. Verify `ListEmployeesWithChannelsRow` also has them.

- [ ] **Step 6: Verify build compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: Compilation errors in files that call `CreateEmployee` with the old 6-param struct (adapter.go, handlers.go). This is expected — they will be fixed in subsequent tasks.

- [ ] **Step 7: Commit**

```bash
git add cmd/brain/main.go sql/queries/employees.sql internal/db/sqlc/
git commit -m "feat: add employee profile fields migration and sqlc queries"
```

---

### Task 2: Brain Layer — EmployeeChatRequest & EmployeeContext

**Files:**
- Modify: `internal/brain/chat.go:40-44,87-113` (RosterEntry, HandleEmployee, HandleBoss)
- Modify: `internal/brain/chat_test.go` (update all HandleEmployee call sites)
- Modify: `internal/brain/engine.go:350-360` (BuildEmployeeChatPrompt)
- Modify: `internal/brain/engine_chat_test.go:57-69` (update BuildEmployeeChatPrompt test)

- [ ] **Step 1: Add EmployeeChatRequest struct and refactor HandleEmployee signature in chat.go**

Add the struct before the `HandleEmployee` function. Replace the `RosterEntry` struct (lines 40-44) and `HandleEmployee` function signature (line 87):

**RosterEntry** — add `JobTitle` field:

```go
type RosterEntry struct {
	Name     string
	JobTitle string
	Role     string
	IsActive bool
}
```

**Add EmployeeChatRequest** after `BossContext` (after line 52):

```go
// EmployeeChatRequest holds all data needed for an employee chat message.
type EmployeeChatRequest struct {
	EmployeeID       string
	TenantID         string
	Name             string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
	MentorID         string
	CultureCode      string
	Text             string
}
```

**Refactor HandleEmployee** — change signature from:

```go
func (s *ChatService) HandleEmployee(ctx context.Context, employeeID, tenantID, employeeName, mentorID, cultureCode, text string) (string, error) {
```

To:

```go
func (s *ChatService) HandleEmployee(ctx context.Context, req EmployeeChatRequest) (string, error) {
```

Update all internal references:
- `employeeID` → `req.EmployeeID`
- `tenantID` → `req.TenantID`
- `mentorID` → `req.MentorID`
- `cultureCode` → `req.CultureCode`
- `text` → `req.Text`
- `employeeName` → `req.Name`

The `BuildEmployeeChatPrompt` call (line 113) changes from:

```go
systemPrompt := engine.BuildEmployeeChatPrompt(ctx, tenantID, employeeID, employeeName, text)
```

To:

```go
systemPrompt := engine.BuildEmployeeChatPrompt(ctx, req.TenantID, req.EmployeeID, EmployeeContext{
	Name:             req.Name,
	JobTitle:         req.JobTitle,
	Responsibilities: req.Responsibilities,
	Country:          req.Country,
	Language:         req.Language,
}, req.Text)
```

- [ ] **Step 2: Update HandleBoss roster formatting in chat.go**

In `HandleBoss` (around line 168-173), the roster building loop currently formats:

```go
fmt.Fprintf(&rosterSB, "%d. %s (%s, %s)\n", i+1, emp.Name, emp.Role, status)
```

Change to include job title:

```go
if emp.JobTitle != "" {
	fmt.Fprintf(&rosterSB, "%d. %s - %s (%s, %s)\n", i+1, emp.Name, emp.JobTitle, emp.Role, status)
} else {
	fmt.Fprintf(&rosterSB, "%d. %s (%s, %s)\n", i+1, emp.Name, emp.Role, status)
}
```

- [ ] **Step 3: Add EmployeeContext struct and refactor BuildEmployeeChatPrompt in engine.go**

Add the struct before `BuildEmployeeChatPrompt` (before line 350):

```go
// EmployeeContext holds employee profile data for prompt building.
type EmployeeContext struct {
	Name             string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
}
```

Replace the current `BuildEmployeeChatPrompt` (lines 350-360):

```go
func (e *Engine) BuildEmployeeChatPrompt(ctx context.Context, tenantID, employeeID string, profile EmployeeContext, queryText string) string {
	prompt := e.BuildSystemPromptWithMemory(ctx, tenantID, employeeID, queryText)

	// Inject employee context if any profile fields are set
	var ctxParts []string
	if profile.JobTitle != "" {
		ctxParts = append(ctxParts, "Job Title: "+profile.JobTitle)
	}
	if profile.Responsibilities != "" {
		ctxParts = append(ctxParts, "Responsibilities: "+profile.Responsibilities)
	}
	if profile.Country != "" {
		ctxParts = append(ctxParts, "Country: "+profile.Country)
	}
	if len(ctxParts) > 0 {
		prompt += "\n\n<employee_context>\n" + strings.Join(ctxParts, "\n") + "\n</employee_context>"
	}

	prompt += fmt.Sprintf("\nYou are %s, acting as CEO and management coach. "+
		"The employee %q is asking you for guidance. "+
		"Respond based on your management philosophy. Keep responses concise and actionable.",
		e.MentorName(), profile.Name)

	if profile.Language != "" {
		prompt += fmt.Sprintf("\nReply in %s.", profile.Language)
	}

	return prompt
}
```

**Note:** Ensure `"strings"` is in the import list of `engine.go`. Check if it's already imported; if not, add it.

- [ ] **Step 4: Update chat_test.go — all HandleEmployee call sites**

Every call to `svc.HandleEmployee(...)` must change from positional params to `EmployeeChatRequest` struct. For example, `TestChatService_HandleEmployee_Basic` (line 118):

From:

```go
resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "How do I manage my team?")
```

To:

```go
resp, err := svc.HandleEmployee(context.Background(), brain.EmployeeChatRequest{
	EmployeeID:  "emp-1",
	TenantID:    "tenant-1",
	Name:        "Alice",
	MentorID:    "inamori",
	CultureCode: "default",
	Text:        "How do I manage my team?",
})
```

Do the same for ALL HandleEmployee calls in chat_test.go:
- `TestChatService_HandleEmployee_Basic` (line 118)
- `TestChatService_HandleEmployee_RateLimit` (lines 153, 161) — 6 calls inside loop + 1 after
- `TestChatService_HandleEmployee_AIDisabled` (line 179)
- `TestChatService_HandleEmployee_LLMError` (line 198)
- `TestChatService_HandleEmployee_AuthError` (line 217)
- `TestChatService_HandleEmployee_HistoryTrimming` (line 326) — inside loop

- [ ] **Step 5: Update engine_chat_test.go — BuildEmployeeChatPrompt test**

In `TestEngine_BuildEmployeeChatPrompt` (line 62), change from:

```go
prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", "Alice", "I have a problem")
```

To:

```go
prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", brain.EmployeeContext{
	Name: "Alice",
}, "I have a problem")
```

Add a new test for the `<employee_context>` block:

```go
func TestEngine_BuildEmployeeChatPrompt_WithProfile(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", EmployeeContext{
		Name:             "Alice",
		JobTitle:         "Frontend Developer",
		Responsibilities: "Handles UI/UX",
		Country:          "Philippines",
		Language:         "Chinese",
	}, "I have a problem")
	if !strings.Contains(prompt, "<employee_context>") {
		t.Fatal("prompt should contain employee_context block")
	}
	if !strings.Contains(prompt, "Frontend Developer") {
		t.Fatal("prompt should contain job title")
	}
	if !strings.Contains(prompt, "Reply in Chinese") {
		t.Fatal("prompt should contain language instruction")
	}
}

func TestEngine_BuildEmployeeChatPrompt_EmptyProfile(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", EmployeeContext{
		Name: "Alice",
	}, "I have a problem")
	if strings.Contains(prompt, "<employee_context>") {
		t.Fatal("prompt should NOT contain employee_context block when all fields empty")
	}
	if strings.Contains(prompt, "Reply in") {
		t.Fatal("prompt should NOT contain language instruction when language empty")
	}
}
```

- [ ] **Step 6: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/...`

Expected: All tests pass (may still fail to compile if other packages reference the old HandleEmployee signature — that's OK, we fix those in later tasks).

- [ ] **Step 7: Commit**

```bash
git add internal/brain/chat.go internal/brain/chat_test.go internal/brain/engine.go internal/brain/engine_chat_test.go
git commit -m "feat: add EmployeeChatRequest struct and employee_context prompt injection"
```

---

### Task 3: Bot Layer — Employee Struct & Commands

**Files:**
- Modify: `internal/bot/middleware.go:10-17` (Employee struct)
- Modify: `internal/bot/adapter.go:34-39,67-83,228-242` (CreateEmployeeParams, CreateEmployee, sqlcEmployeeToBot)
- Modify: `internal/bot/commands.go:78-105,140-154,156-190,324-380` (HandleStart, HandleHelp, HandleAddEmployee, HandleProfile)

- [ ] **Step 1: Add 4 fields to bot.Employee struct in middleware.go**

In `internal/bot/middleware.go`, change the `Employee` struct (lines 10-17) from:

```go
type Employee struct {
	ID          string
	Name        string
	TenantID    string
	TelegramID  int64
	CultureCode string
	InviteCode  string
}
```

To:

```go
type Employee struct {
	ID               string
	Name             string
	TenantID         string
	TelegramID       int64
	CultureCode      string
	InviteCode       string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
}
```

- [ ] **Step 2: Update CreateEmployeeParams and sqlcEmployeeToBot in adapter.go**

**CreateEmployeeParams** (lines 34-39) — add 4 fields:

```go
type CreateEmployeeParams struct {
	TenantID         string
	Name             string
	CultureCode      string
	InviteCode       string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
}
```

**CreateEmployee method** (lines 67-83) — pass new fields to sqlc:

```go
func (a *DBAdapter) CreateEmployee(ctx context.Context, params CreateEmployeeParams) (*Employee, error) {
	uid, err := parseUUID(params.TenantID)
	if err != nil {
		return nil, err
	}
	e, err := a.q.CreateEmployee(ctx, sqlc.CreateEmployeeParams{
		TenantID:         uid,
		Name:             params.Name,
		CultureCode:      params.CultureCode,
		Role:             "member",
		InviteCode:       pgtype.Text{String: params.InviteCode, Valid: true},
		JobTitle:         params.JobTitle,
		Responsibilities: params.Responsibilities,
		Country:          params.Country,
		Language:         params.Language,
	})
	if err != nil {
		return nil, err
	}
	return sqlcEmployeeToBot(e), nil
}
```

**sqlcEmployeeToBot** (lines 228-242) — add new field mappings:

```go
func sqlcEmployeeToBot(e sqlc.Employee) *Employee {
	emp := &Employee{
		ID:               formatUUID(e.ID),
		Name:             e.Name,
		TenantID:         formatUUID(e.TenantID),
		CultureCode:      e.CultureCode,
		JobTitle:         e.JobTitle,
		Responsibilities: e.Responsibilities,
		Country:          e.Country,
		Language:         e.Language,
	}
	if e.TelegramID.Valid {
		emp.TelegramID = e.TelegramID.Int64
	}
	if e.InviteCode.Valid {
		emp.InviteCode = e.InviteCode.String
	}
	return emp
}
```

- [ ] **Step 3: Refactor HandleAddEmployee to pipe-separated format in commands.go**

Replace the current `HandleAddEmployee` function (lines 156-190):

```go
func (h *CommandHandler) HandleAddEmployee(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	raw := strings.TrimSpace(c.Text())
	// Remove the command prefix
	idx := strings.Index(raw, " ")
	if idx < 0 {
		return c.Send("Usage: /addemployee name | job_title | responsibilities | country | language\nOnly name is required.\nExample: /addemployee Alice | Frontend Developer | Handles UI/UX | Philippines | Chinese")
	}
	body := strings.TrimSpace(raw[idx+1:])

	parts := strings.SplitN(body, "|", 5)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	name := parts[0]
	if name == "" {
		return c.Send("Name is required.\nUsage: /addemployee name | job_title | responsibilities | country | language")
	}

	var jobTitle, responsibilities, country, language string
	if len(parts) > 1 {
		jobTitle = parts[1]
	}
	if len(parts) > 2 {
		responsibilities = parts[2]
	}
	if len(parts) > 3 {
		country = parts[3]
	}
	if len(parts) > 4 {
		language = parts[4]
	}

	inviteCode := generateInviteCode()

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	emp, err := h.db.CreateEmployee(context.Background(), CreateEmployeeParams{
		TenantID:         tenant.ID,
		Name:             name,
		CultureCode:      "default",
		InviteCode:       inviteCode,
		JobTitle:         jobTitle,
		Responsibilities: responsibilities,
		Country:          country,
		Language:         language,
	})
	if err != nil {
		return fmt.Errorf("create employee: %w", err)
	}

	msg := fmt.Sprintf("Employee added!\nName: %s", emp.Name)
	if jobTitle != "" {
		msg += fmt.Sprintf("\nJob: %s", jobTitle)
	}
	if country != "" {
		msg += fmt.Sprintf("\nCountry: %s", country)
	}
	if language != "" {
		msg += fmt.Sprintf("\nLanguage: %s", language)
	}
	msg += fmt.Sprintf("\nInvite code: %s\n\nShare this code with %s to join via /join %s", inviteCode, name, inviteCode)

	return c.Send(msg)
}
```

- [ ] **Step 4: Update HandleProfile to show new fields in commands.go**

Replace the profile output in `HandleProfile` (lines 368-379). Change from:

```go
	return c.Send(fmt.Sprintf(
		"Employee Profile: %s\n\n"+
			"Status: %s\n"+
			"Culture: %s\n\n"+
			"Last 7 days: %d/7 submitted\n"+
			"Last 30 days: %d submitted\n"+
			"Current streak: %d days\n"+
			"Sentiment trend: %s",
		found.Name, status, found.CultureCode,
		profile.SubmittedLast7, profile.SubmittedLast30,
		profile.CurrentStreak, profile.SentimentTrend,
	))
```

To:

```go
	msg := fmt.Sprintf("Employee Profile: %s\n\nStatus: %s", found.Name, status)
	if found.JobTitle != "" {
		msg += fmt.Sprintf("\nJob Title: %s", found.JobTitle)
	}
	if found.Responsibilities != "" {
		msg += fmt.Sprintf("\nResponsibilities: %s", found.Responsibilities)
	}
	if found.Country != "" {
		msg += fmt.Sprintf("\nCountry: %s", found.Country)
	}
	if found.Language != "" {
		msg += fmt.Sprintf("\nLanguage: %s", found.Language)
	}
	msg += fmt.Sprintf("\nCulture: %s", found.CultureCode)
	msg += fmt.Sprintf("\n\nLast 7 days: %d/7 submitted\nLast 30 days: %d submitted\nCurrent streak: %d days\nSentiment trend: %s",
		profile.SubmittedLast7, profile.SubmittedLast30,
		profile.CurrentStreak, profile.SentimentTrend)
	return c.Send(msg)
```

- [ ] **Step 5: Update HandleHelp and HandleStart text in commands.go**

**HandleHelp** (line 145): Change `/addemployee <name> <culture>` to `/addemployee <name> | job | resp | country | lang`

**HandleStart** (line 102): Change `/addemployee <name> <culture>` to `/addemployee <name> | job | resp | country | lang`

- [ ] **Step 6: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/bot/...`

Expected: Tests pass (bot tests use mock interfaces, not real sqlc).

- [ ] **Step 7: Commit**

```bash
git add internal/bot/middleware.go internal/bot/adapter.go internal/bot/commands.go
git commit -m "feat: add profile fields to bot Employee struct and commands"
```

---

### Task 4: API Layer — Handlers & Route

**Files:**
- Modify: `internal/api/handlers.go:134-221,224-265` (list/get/create employee handlers)
- Modify: `internal/api/handlers.go` (add handleUpdateProfile)
- Modify: `internal/api/router.go:74` (add profile route)

- [ ] **Step 1: Update createEmployeeRequest and handleCreateEmployee**

In `internal/api/handlers.go`, update the request struct (lines 167-170):

```go
type createEmployeeRequest struct {
	Name             string `json:"name" binding:"required,min=1"`
	CultureCode      string `json:"culture_code"`
	JobTitle         string `json:"job_title"`
	Responsibilities string `json:"responsibilities"`
	Country          string `json:"country"`
	Language         string `json:"language"`
}
```

In `handleCreateEmployee`, update the `CreateEmployeeParams` (lines 200-206):

```go
		emp, err := queries.CreateEmployee(c.Request.Context(), sqlc.CreateEmployeeParams{
			TenantID:         tenantID,
			Name:             req.Name,
			CultureCode:      cultureCode,
			Role:             "member",
			InviteCode:       pgtype.Text{String: inviteCode, Valid: true},
			JobTitle:         req.JobTitle,
			Responsibilities: req.Responsibilities,
			Country:          req.Country,
			Language:         req.Language,
		})
```

Update the response (lines 213-220) to include new fields:

```go
		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{
				"id":               formatUUID(emp.ID),
				"name":             emp.Name,
				"culture_code":     emp.CultureCode,
				"invite_code":      inviteCode,
				"job_title":        emp.JobTitle,
				"responsibilities": emp.Responsibilities,
				"country":          emp.Country,
				"language":         emp.Language,
			},
		})
```

- [ ] **Step 2: Update handleListEmployees response**

In `handleListEmployees` (lines 149-159), add 4 fields to each employee in the result:

```go
		result := make([]gin.H, 0, len(employees))
		for _, e := range employees {
			result = append(result, gin.H{
				"id":               formatUUID(e.ID),
				"name":             e.Name,
				"culture_code":     e.CultureCode,
				"role":             e.Role,
				"is_active":        e.IsActive,
				"has_telegram":     e.TelegramID.Valid,
				"invite_code":      e.InviteCode.String,
				"job_title":        e.JobTitle,
				"responsibilities": e.Responsibilities,
				"country":          e.Country,
				"language":         e.Language,
			})
		}
```

- [ ] **Step 3: Update handleGetEmployee response**

In `handleGetEmployee` (lines 253-263), add 4 fields:

```go
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":               formatUUID(emp.ID),
				"name":             emp.Name,
				"culture_code":     emp.CultureCode,
				"role":             emp.Role,
				"is_active":        emp.IsActive,
				"has_telegram":     emp.TelegramID.Valid,
				"invite_code":      emp.InviteCode.String,
				"job_title":        emp.JobTitle,
				"responsibilities": emp.Responsibilities,
				"country":          emp.Country,
				"language":         emp.Language,
			},
		})
```

- [ ] **Step 4: Add handleUpdateProfile handler**

Add after `handleUpdateEmployeeCulture` (after line 323):

```go
// updateProfileRequest holds the request body for updating employee profile fields.
type updateProfileRequest struct {
	JobTitle         *string `json:"job_title"`
	Responsibilities *string `json:"responsibilities"`
	Country          *string `json:"country"`
	Language         *string `json:"language"`
}

// handleUpdateProfile updates an employee's profile fields (boss only).
func handleUpdateProfile(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		empID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee ID"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Verify employee belongs to tenant
		emp, err := queries.GetEmployee(c.Request.Context(), sqlc.GetEmployeeParams{
			ID:       empID,
			TenantID: tenantID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
				return
			}
			slog.Error("get employee for profile update", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Patch semantics: only update provided fields
		jobTitle := emp.JobTitle
		if req.JobTitle != nil {
			jobTitle = *req.JobTitle
		}
		responsibilities := emp.Responsibilities
		if req.Responsibilities != nil {
			responsibilities = *req.Responsibilities
		}
		country := emp.Country
		if req.Country != nil {
			country = *req.Country
		}
		language := emp.Language
		if req.Language != nil {
			language = *req.Language
		}

		if err := queries.UpdateEmployeeProfile(c.Request.Context(), sqlc.UpdateEmployeeProfileParams{
			ID:               empID,
			JobTitle:         jobTitle,
			Responsibilities: responsibilities,
			Country:          country,
			Language:         language,
		}); err != nil {
			slog.Error("update employee profile", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"job_title":        jobTitle,
				"responsibilities": responsibilities,
				"country":          country,
				"language":         language,
			},
		})
	}
}
```

- [ ] **Step 5: Add route in router.go**

In `internal/api/router.go`, after line 74 (`PUT /employees/:id/culture`), add:

```go
		protected.PUT("/employees/:id/profile", RequireRole("boss"), handleUpdateProfile(cfg.Queries))
```

- [ ] **Step 6: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/api/...`

Expected: Tests pass (some pre-existing API test failures may persist — they are unrelated).

- [ ] **Step 7: Commit**

```bash
git add internal/api/handlers.go internal/api/router.go
git commit -m "feat: add employee profile fields to API handlers and update profile endpoint"
```

---

### Task 5: Wiring — main.go & message_handler.go

**Files:**
- Modify: `cmd/brain/main.go:114-149` (fetchBossContext)
- Modify: `cmd/brain/main.go:750-770` (Telegram text handler)
- Modify: `cmd/brain/main.go:826-841` (unified handler OnText)
- Modify: `internal/channel/message_handler.go:17,25,40-54` (OnText signature)

- [ ] **Step 1: Update fetchBossContext to include JobTitle**

In `cmd/brain/main.go`, update the roster building loop (lines 135-141):

From:

```go
	for _, e := range emps {
		roster = append(roster, brain.RosterEntry{
			Name:     e.Name,
			Role:     e.Role,
			IsActive: e.IsActive,
		})
	}
```

To:

```go
	for _, e := range emps {
		roster = append(roster, brain.RosterEntry{
			Name:     e.Name,
			JobTitle: e.JobTitle,
			Role:     e.Role,
			IsActive: e.IsActive,
		})
	}
```

- [ ] **Step 2: Update Telegram text handler to pass EmployeeChatRequest**

In `cmd/brain/main.go`, the employee mentor chat call (line 762):

From:

```go
			resp, err := chatService.HandleEmployee(ctx, empID, emp.TenantID, emp.Name, tenant.MentorID, emp.CultureCode, text)
```

To:

```go
			resp, err := chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
				EmployeeID:       empID,
				TenantID:         emp.TenantID,
				Name:             emp.Name,
				JobTitle:         emp.JobTitle,
				Responsibilities: emp.Responsibilities,
				Country:          emp.Country,
				Language:         emp.Language,
				MentorID:         tenant.MentorID,
				CultureCode:      emp.CultureCode,
				Text:             text,
			})
```

- [ ] **Step 3: Update OnText callback signature in message_handler.go**

In `internal/channel/message_handler.go`, the `onText` field and `OnText` config field need to pass the resolved employee data. Change the signature to include an employee struct.

Update the `onText` field type (line 17) and `OnText` config field (line 25):

From:

```go
	onText    func(ctx context.Context, employeeID, tenantID, text, channelType string) (response string, err error)
```

To:

```go
	onText    func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (response string, err error)
```

Do the same for `UnifiedHandlerConfig.OnText` (line 25).

Update the call site in `HandleMessage` (line 54):

From:

```go
		response, err = h.onText(ctx, empID, tenantID, msg.Text, string(msg.ChannelType))
```

To:

```go
		response, err = h.onText(ctx, empID, tenantID, msg.Text, string(msg.ChannelType), emp.Name, emp.JobTitle, emp.Responsibilities, emp.Country, emp.Language, emp.CultureCode)
```

- [ ] **Step 4: Update unified handler OnText in main.go**

In `cmd/brain/main.go`, the OnText callback (line 779):

From:

```go
	OnText: func(ctx context.Context, employeeID, tenantID, text, channelType string) (string, error) {
```

To:

```go
	OnText: func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (string, error) {
```

Update the `HandleEmployee` call in the default branch (line 836):

From:

```go
				resp, err := chatService.HandleEmployee(ctx, employeeID, tenantID, "", tenant.MentorID, "default", text)
```

To:

```go
				resp, err := chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
					EmployeeID:       employeeID,
					TenantID:         tenantID,
					Name:             empName,
					JobTitle:         empJobTitle,
					Responsibilities: empResponsibilities,
					Country:          empCountry,
					Language:         empLanguage,
					MentorID:         tenant.MentorID,
					CultureCode:      empCultureCode,
					Text:             text,
				})
```

- [ ] **Step 5: Build and run all tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./... && go test ./internal/brain/... ./internal/bot/... ./internal/channel/...`

Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add cmd/brain/main.go internal/channel/message_handler.go
git commit -m "feat: wire employee profile fields through Telegram and unified handlers"
```

---

### Task 6: Frontend — Form, Table & Edit Modal

**Files:**
- Modify: `frontend/src/composables/api.ts:95-100,357-365` (createEmployee, Employee interface)
- Modify: `frontend/src/views/EmployeesView.vue` (form, table, edit modal)

- [ ] **Step 1: Update Employee interface in api.ts**

In `frontend/src/composables/api.ts`, update the `Employee` interface (lines 357-365):

```typescript
export interface Employee {
  id: string;
  name: string;
  culture_code: string;
  role: string;
  is_active: boolean;
  has_telegram: boolean;
  invite_code: string;
  job_title: string;
  responsibilities: string;
  country: string;
  language: string;
}
```

- [ ] **Step 2: Update createEmployee function**

Replace the current `createEmployee` function (lines 95-100):

```typescript
export async function createEmployee(data: {
  name: string;
  culture_code: string;
  job_title?: string;
  responsibilities?: string;
  country?: string;
  language?: string;
}) {
  return request<{ data: Employee }>("/employees", {
    method: "POST",
    body: JSON.stringify(data),
  });
}
```

- [ ] **Step 3: Add updateEmployeeProfile function**

After `createEmployee`, add:

```typescript
export async function updateEmployeeProfile(
  id: string,
  data: { job_title?: string; responsibilities?: string; country?: string; language?: string },
) {
  return request<{ data: { job_title: string; responsibilities: string; country: string; language: string } }>(
    `/employees/${id}/profile`,
    { method: "PUT", body: JSON.stringify(data) },
  );
}
```

- [ ] **Step 4: Rewrite EmployeesView.vue**

Replace the entire file with the updated version that includes:
- 4 new form fields in the create form
- Job Title column in the table
- Edit button per row
- Edit profile modal

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listEmployees, createEmployee, updateEmployeeProfile, type Employee } from '../composables/api'

const employees = ref<Employee[]>([])
const loading = ref(true)
const error = ref('')

const showAdd = ref(false)
const newName = ref('')
const newCulture = ref('default')
const newJobTitle = ref('')
const newResponsibilities = ref('')
const newCountry = ref('')
const newLanguage = ref('')
const addError = ref('')
const adding = ref(false)

// Edit modal state
const showEdit = ref(false)
const editEmployee = ref<Employee | null>(null)
const editJobTitle = ref('')
const editResponsibilities = ref('')
const editCountry = ref('')
const editLanguage = ref('')
const editError = ref('')
const saving = ref(false)

const cultures = [
  { value: 'default', label: 'Default' },
  { value: 'philippines', label: 'Philippines' },
  { value: 'chinese', label: 'Chinese' },
  { value: 'japanese', label: 'Japanese' },
  { value: 'western', label: 'Western' },
]

async function loadEmployees() {
  try {
    const res = await listEmployees()
    employees.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  addError.value = ''
  adding.value = true
  try {
    await createEmployee({
      name: newName.value,
      culture_code: newCulture.value,
      job_title: newJobTitle.value || undefined,
      responsibilities: newResponsibilities.value || undefined,
      country: newCountry.value || undefined,
      language: newLanguage.value || undefined,
    })
    newName.value = ''
    newCulture.value = 'default'
    newJobTitle.value = ''
    newResponsibilities.value = ''
    newCountry.value = ''
    newLanguage.value = ''
    showAdd.value = false
    loading.value = true
    await loadEmployees()
  } catch (e: any) {
    addError.value = e.message
  } finally {
    adding.value = false
  }
}

function openEdit(emp: Employee) {
  editEmployee.value = emp
  editJobTitle.value = emp.job_title || ''
  editResponsibilities.value = emp.responsibilities || ''
  editCountry.value = emp.country || ''
  editLanguage.value = emp.language || ''
  editError.value = ''
  showEdit.value = true
}

async function handleSaveEdit() {
  if (!editEmployee.value) return
  editError.value = ''
  saving.value = true
  try {
    await updateEmployeeProfile(editEmployee.value.id, {
      job_title: editJobTitle.value,
      responsibilities: editResponsibilities.value,
      country: editCountry.value,
      language: editLanguage.value,
    })
    showEdit.value = false
    loading.value = true
    await loadEmployees()
  } catch (e: any) {
    editError.value = e.message
  } finally {
    saving.value = false
  }
}

onMounted(loadEmployees)
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem">
      <h2>Team Members</h2>
      <button class="btn btn-primary" @click="showAdd = !showAdd">
        {{ showAdd ? 'Cancel' : '+ Add Employee' }}
      </button>
    </div>

    <div v-if="showAdd" class="card">
      <h3>Add New Employee</h3>
      <form @submit.prevent="handleAdd" style="display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap">
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Name *</label>
          <input v-model="newName" placeholder="Employee name" required />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Culture</label>
          <select v-model="newCulture">
            <option v-for="c in cultures" :key="c.value" :value="c.value">{{ c.label }}</option>
          </select>
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Job Title</label>
          <input v-model="newJobTitle" placeholder="e.g. Frontend Developer" />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Country</label>
          <input v-model="newCountry" placeholder="e.g. Philippines" />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Language</label>
          <input v-model="newLanguage" placeholder="e.g. Chinese" />
        </div>
        <div style="width: 100%">
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Responsibilities</label>
          <textarea v-model="newResponsibilities" placeholder="Brief description of role" rows="2" style="width: 100%"></textarea>
        </div>
        <button type="submit" class="btn btn-primary" :disabled="adding">
          {{ adding ? 'Adding...' : 'Add' }}
        </button>
      </form>
      <p v-if="addError" class="error-msg">{{ addError }}</p>
    </div>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <div v-else class="card">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Job Title</th>
            <th>Culture</th>
            <th>Role</th>
            <th>Telegram</th>
            <th>Invite Code</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="employees.length === 0">
            <td colspan="7" style="text-align: center; color: #888; padding: 2rem">
              No employees yet. Add your first team member above.
            </td>
          </tr>
          <tr v-for="emp in employees" :key="emp.id">
            <td><strong>{{ emp.name }}</strong></td>
            <td>{{ emp.job_title || '-' }}</td>
            <td>{{ emp.culture_code }}</td>
            <td>{{ emp.role }}</td>
            <td>
              <span :class="emp.has_telegram ? 'badge badge-positive' : 'badge badge-neutral'">
                {{ emp.has_telegram ? 'Connected' : 'Pending' }}
              </span>
            </td>
            <td><code>{{ emp.invite_code || '-' }}</code></td>
            <td>
              <button class="btn" style="font-size: 0.8rem; padding: 0.25rem 0.5rem" @click="openEdit(emp)">Edit</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Profile Modal -->
    <div v-if="showEdit" style="position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100" @click.self="showEdit = false">
      <div class="card" style="width: 100%; max-width: 500px; margin: 1rem">
        <h3>Edit Profile: {{ editEmployee?.name }}</h3>
        <form @submit.prevent="handleSaveEdit" style="display: flex; flex-direction: column; gap: 0.75rem">
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Job Title</label>
            <input v-model="editJobTitle" placeholder="e.g. Frontend Developer" style="width: 100%" />
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Responsibilities</label>
            <textarea v-model="editResponsibilities" placeholder="Brief description" rows="3" style="width: 100%"></textarea>
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Country</label>
            <input v-model="editCountry" placeholder="e.g. Philippines" style="width: 100%" />
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Language</label>
            <input v-model="editLanguage" placeholder="e.g. Chinese" style="width: 100%" />
          </div>
          <div style="display: flex; gap: 0.5rem; justify-content: flex-end">
            <button type="button" class="btn" @click="showEdit = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </div>
          <p v-if="editError" class="error-msg">{{ editError }}</p>
        </form>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 5: Build frontend**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/composables/api.ts frontend/src/views/EmployeesView.vue
git commit -m "feat: add employee profile fields to frontend form, table, and edit modal"
```

---

### Task 7: Deploy to Production

**Files:** None (operational task)

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./...`

Expected: All tests pass (pre-existing `internal/api` test failures are acceptable).

- [ ] **Step 2: Build Linux binary**

Run: `cd /Users/anna/Documents/ai-management-brain && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain`

- [ ] **Step 3: Push to GitHub**

Run: `cd /Users/anna/Documents/ai-management-brain && git push`

- [ ] **Step 4: Copy binary to server and deploy**

```bash
scp /Users/anna/Documents/ai-management-brain/brain ai-brain:/home/ubuntu/ai-management-brain/brain
ssh ai-brain "cd /home/ubuntu/ai-management-brain && git pull && docker compose -f docker-compose.prod.yml up -d --build"
```

- [ ] **Step 5: Verify health**

```bash
ssh ai-brain "curl -s localhost/healthz"
ssh ai-brain "docker logs ai-management-brain-brain-1 --tail 20 2>&1 | grep -i 'chat\|profile\|migration'"
```

Expected: Health check returns OK. Logs show migration 000008 applied (if first deploy with these changes).
