# Intelligent Mentor Chat Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable intelligent mentor-powered conversations for employees and boss via the existing Telegram/Slack/Lark/Signal channels, filling the empty `default` branch in text handlers.

**Architecture:** Extend existing `AnthropicClient` with multi-turn `ChatWithHistory()`, create a `ChatService` that orchestrates Redis-backed conversation history, role-based system prompts, rate limiting, and gap-based memory extraction. Wire into `main.go` text handlers (both Telegram and UnifiedHandler).

**Tech Stack:** Go 1.25, anthropic-sdk-go, Redis (go-redis/v9), sqlc/pgx, existing brain/memory/events packages.

**Spec:** `docs/superpowers/specs/2026-03-22-mentor-chat-design.md`

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `internal/brain/chat.go` | ChatService — orchestrates employee/boss chat: Redis history, rate limiting, prompt assembly, gap-based extraction |
| `internal/brain/chat_test.go` | Unit tests for ChatService |
| `internal/brain/llm_chat_test.go` | Unit tests for ChatWithHistory |

### Modified Files
| File | Changes |
|------|---------|
| `internal/brain/llm.go` | Add `ChatLLMClient` interface, `ChatMessage` type, `ChatWithHistory()` method on `AnthropicClient` |
| `internal/brain/engine.go` | Add `BuildBossPrompt()` method, `MentorName()` accessor |
| `internal/events/bus.go` | Add `ChatCompleted` event constant + `ChatCompletedPayload` struct |
| `cmd/brain/main.go` | Create `ChatService`, add boss check in Telegram handler, fill both `default` branches, wire `ChatService` to `UnifiedHandler` |

---

### Task 1: Add ChatLLMClient interface and ChatWithHistory to llm.go

**Files:**
- Modify: `internal/brain/llm.go:17-19` (after LLMClient interface)
- Test: `internal/brain/llm_chat_test.go`

- [ ] **Step 1: Write the failing test for ChatMessage and ChatWithHistory**

Create `internal/brain/llm_chat_test.go`:

```go
package brain

import (
	"context"
	"testing"
)

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{Role: "user", Content: "hello"}
	if msg.Role != "user" || msg.Content != "hello" {
		t.Fatalf("unexpected fields: %+v", msg)
	}
}

// TestAnthropicClient_ChatWithHistory_NoKey verifies that a nil/empty API key
// still results in a valid AnthropicClient (created elsewhere) — this test
// focuses on the interface satisfaction.
func TestChatLLMClient_InterfaceSatisfaction(t *testing.T) {
	// AnthropicClient must satisfy ChatLLMClient at compile time.
	var _ ChatLLMClient = (*AnthropicClient)(nil)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run TestChatMessage -v`
Expected: FAIL — `ChatMessage` and `ChatLLMClient` not defined

- [ ] **Step 3: Implement ChatMessage type, ChatLLMClient interface, and ChatWithHistory method**

In `internal/brain/llm.go`, add after line 19 (after `LLMClient` interface closing brace):

```go
// ChatMessage represents a single message in a multi-turn conversation.
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ChatLLMClient extends LLM capabilities with multi-turn conversation.
type ChatLLMClient interface {
	ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error)
}
```

Then add the `ChatWithHistory` method on `AnthropicClient` (after the `ChatLong` method, ~line 126):

```go
// ChatWithHistory sends a multi-turn conversation to Claude and returns the response.
func (a *AnthropicClient) ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	start := time.Now()
	var lastErr error

	// Build messages array from history + new user message
	messages := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	backoffs := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for attempt := 0; attempt <= 2; attempt++ {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 1024,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: messages,
		})
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") ||
				strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "unauthorized") {
				return "", &AuthError{Msg: errMsg}
			}
			lastErr = err
			slog.Warn("LLM ChatWithHistory API call failed",
				"attempt", attempt+1,
				"error", err,
				"duration", time.Since(start),
			)
			if attempt < 2 {
				time.Sleep(backoffs[attempt])
			}
			continue
		}

		var result string
		for _, block := range resp.Content {
			if block.Type == "text" {
				result += block.Text
			}
		}
		slog.Info("LLM ChatWithHistory succeeded",
			"duration", time.Since(start),
			"input_tokens", resp.Usage.InputTokens,
			"output_tokens", resp.Usage.OutputTokens,
			"history_len", len(history),
		)
		return result, nil
	}

	return "", fmt.Errorf("LLM ChatWithHistory failed after 3 attempts: %w", lastErr)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestChatMessage|TestChatLLMClient" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/llm.go internal/brain/llm_chat_test.go
git commit -m "feat: add ChatLLMClient interface and ChatWithHistory method"
```

---

### Task 2: Add ChatCompleted event to events/bus.go

**Files:**
- Modify: `internal/events/bus.go:17-26` (event constants)
- Test: `internal/events/bus_test.go` (if exists, add test; otherwise create)

- [ ] **Step 1: Write the failing test**

Create or add to `internal/events/bus_test.go`:

```go
package events_test

import (
	"encoding/json"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/events"
)

func TestChatCompletedPayload_Marshal(t *testing.T) {
	p := events.ChatCompletedPayload{
		EmployeeID: "emp-123",
		Messages:   `[{"role":"user","content":"hi"}]`,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded events.ChatCompletedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EmployeeID != "emp-123" {
		t.Fatalf("unexpected: %+v", decoded)
	}
}

func TestChatCompletedEventType(t *testing.T) {
	if events.ChatCompleted != "chat.completed" {
		t.Fatalf("unexpected event type: %s", events.ChatCompleted)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/events/ -run TestChatCompleted -v`
Expected: FAIL — `ChatCompleted` and `ChatCompletedPayload` not defined

- [ ] **Step 3: Implement ChatCompleted event**

In `internal/events/bus.go`, add to the const block (after line 25, `EmployeeJoined`):

```go
ChatCompleted    EventType = "chat.completed"
```

Add the payload struct (after `ChaseCompletedPayload`, ~line 71):

```go
// ChatCompletedPayload is sent when a chat conversation is closed for memory extraction.
type ChatCompletedPayload struct {
	EmployeeID string `json:"employee_id"`
	Messages   string `json:"messages"` // JSON array of ChatMessage
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/events/ -run TestChatCompleted -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/events/bus.go internal/events/bus_test.go
git commit -m "feat: add ChatCompleted event type for mentor chat"
```

---

### Task 3: Add BuildBossPrompt and MentorName to engine.go

**Files:**
- Modify: `internal/brain/engine.go:265-306` (after BuildSystemPromptWithMemory)
- Test: `internal/brain/engine_chat_test.go` (new file — `engine_test.go` uses `package brain_test`)

- [ ] **Step 1: Write the failing test**

Create `internal/brain/engine_chat_test.go` (separate file to avoid package conflict with existing `engine_test.go` which uses `package brain_test`):

```go
package brain

import (
	"context"
	"strings"
	"testing"
)

func TestEngine_MentorName(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	name := e.MentorName()
	if name == "" {
		t.Fatal("MentorName should not be empty")
	}
}

func TestEngine_BuildBossPrompt(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildBossPrompt(context.Background(), "tenant-1", BuildBossContext{
		LatestSummary:  "Team performed well today.",
		SubmissionRate: "80% (4/5)",
		EmployeeList:   "1. Alice (member, active)\n2. Bob (manager, active)",
		MemorySection:  "",
	})
	if !strings.Contains(prompt, "chairman") || !strings.Contains(prompt, "CEO") {
		t.Fatal("boss prompt should reference chairman/CEO roles")
	}
	if !strings.Contains(prompt, "Team performed well") {
		t.Fatal("boss prompt should contain the latest summary")
	}
	if !strings.Contains(prompt, "80%") {
		t.Fatal("boss prompt should contain submission rate")
	}
}

func TestEngine_BuildEmployeeChatPrompt(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", "Alice", "I have a problem")
	if !strings.Contains(prompt, "Alice") {
		t.Fatal("employee chat prompt should contain employee name")
	}
	if !strings.Contains(prompt, "coach") {
		t.Fatal("employee chat prompt should reference coaching role")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestEngine_MentorName|TestEngine_BuildBoss|TestEngine_BuildEmployee" -v`
Expected: FAIL — methods not defined

- [ ] **Step 3: Implement MentorName, BuildBossPrompt, and BuildEmployeeChatPrompt**

In `internal/brain/engine.go`, add after `MentorID()` method (~line 74):

```go
// MentorName returns the loaded mentor's display name.
func (e *Engine) MentorName() string {
	return e.mentor.Name
}
```

Add `BuildBossContext` struct and `BuildBossPrompt` method after `BuildSystemPromptWithMemory` (~line 306):

```go
// BuildBossContext holds the team data to inject into the boss system prompt.
type BuildBossContext struct {
	LatestSummary  string
	SubmissionRate string
	EmployeeList   string
	MemorySection  string // pre-formatted memory recall (empty if no employee mentioned)
}

// BuildBossPrompt assembles the system prompt for boss (chairman) conversations.
func (e *Engine) BuildBossPrompt(ctx context.Context, tenantID string, bctx BuildBossContext) string {
	prompt := e.BuildSystemPrompt()

	prompt += "\n\n<team_context>\n"
	prompt += "## Latest Team Summary\n"
	if bctx.LatestSummary != "" {
		prompt += bctx.LatestSummary
	} else {
		prompt += "(No summary available yet)"
	}
	prompt += "\n\n## Today's Status\n"
	prompt += "Submission rate: " + bctx.SubmissionRate + "\n"
	prompt += "\n## Team Roster\n"
	prompt += bctx.EmployeeList + "\n"
	prompt += "</team_context>\n"

	if bctx.MemorySection != "" {
		prompt += "\n" + bctx.MemorySection + "\n"
	}

	prompt += fmt.Sprintf("\nYou are %s, acting as CEO reporting to the chairman. "+
		"The chairman is consulting you about management decisions. "+
		"Provide data-driven insights based on team performance. Be candid and strategic.",
		e.MentorName())

	return prompt
}

// BuildEmployeeChatPrompt assembles the system prompt for employee mentor conversations.
func (e *Engine) BuildEmployeeChatPrompt(ctx context.Context, tenantID, employeeID, employeeName, queryText string) string {
	prompt := e.BuildSystemPromptWithMemory(ctx, tenantID, employeeID, queryText)

	prompt += fmt.Sprintf("\nYou are %s, acting as CEO and management coach. "+
		"The employee %q is asking you for guidance. "+
		"Respond based on your management philosophy. Keep responses concise and actionable.",
		e.MentorName(), employeeName)

	return prompt
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestEngine_MentorName|TestEngine_BuildBoss|TestEngine_BuildEmployee" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/engine.go internal/brain/engine_chat_test.go
git commit -m "feat: add BuildBossPrompt, BuildEmployeeChatPrompt, MentorName to engine"
```

---

### Task 4: Create ChatService (internal/brain/chat.go)

This is the core task — the ChatService orchestrating all mentor chat logic.

**Files:**
- Create: `internal/brain/chat.go`
- Create: `internal/brain/chat_test.go`

**Dependencies:** Tasks 1-3 must be complete.

- [ ] **Step 1: Write the failing test for ChatService construction**

Create `internal/brain/chat_test.go`:

```go
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// --- Mocks ---

type mockChatLLM struct {
	response string
	err      error
	calls    []mockChatCall
	mu       sync.Mutex
}

type mockChatCall struct {
	SystemPrompt string
	History      []ChatMessage
	UserMessage  string
}

func (m *mockChatLLM) ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockChatCall{systemPrompt, history, userMessage})
	return m.response, m.err
}

type mockChatRedis struct {
	data map[string]string
	ttls map[string]time.Duration
	incr map[string]int64
	mu   sync.Mutex
}

func newMockChatRedis() *mockChatRedis {
	return &mockChatRedis{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
		incr: make(map[string]int64),
	}
}

func (m *mockChatRedis) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("redis: nil")
	}
	return v, nil
}

func (m *mockChatRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = fmt.Sprint(value)
	m.ttls[key] = ttl
	return nil
}

func (m *mockChatRedis) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *mockChatRedis) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incr[key]++
	return m.incr[key], nil
}

func (m *mockChatRedis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttls[key] = ttl
	return nil
}

// --- Tests ---

func TestNewChatService(t *testing.T) {
	llm := &mockChatLLM{response: "hello"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})
	if svc == nil {
		t.Fatal("ChatService should not be nil")
	}
}

func TestChatService_HandleEmployee_Basic(t *testing.T) {
	llm := &mockChatLLM{response: "I recommend focusing on priorities."}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "How do I manage my team?")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "I recommend focusing on priorities." {
		t.Fatalf("unexpected response: %s", resp)
	}

	// Verify history was stored
	histKey := "chat:emp-1"
	stored, err := rdb.Get(context.Background(), histKey)
	if err != nil {
		t.Fatal("history should be stored in Redis")
	}
	var msgs []chatHistoryMessage
	if err := json.Unmarshal([]byte(stored), &msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user+assistant), got %d", len(msgs))
	}
}

func TestChatService_HandleEmployee_RateLimit(t *testing.T) {
	llm := &mockChatLLM{response: "ok"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	// Send 5 messages (should all succeed)
	for i := 0; i < 5; i++ {
		_, err := svc.HandleEmployee(context.Background(), "emp-rate", "tenant-1", "Test", "inamori", "default", "msg")
		if err != nil {
			t.Fatalf("message %d should succeed: %v", i, err)
		}
	}

	// 6th message should be rate-limited
	resp, err := svc.HandleEmployee(context.Background(), "emp-rate", "tenant-1", "Test", "inamori", "default", "msg")
	if err != nil {
		t.Fatal(err)
	}
	if resp != rateLimitMessage {
		t.Fatalf("expected rate limit message, got: %s", resp)
	}
}

func TestChatService_HandleBoss_Basic(t *testing.T) {
	llm := &mockChatLLM{response: "Based on today's data, the team is performing well."}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	resp, err := svc.HandleBoss(context.Background(), "tenant-1", "inamori", "default", "How is the team doing?",
		BossContext{
			LatestSummary:  "Good progress today.",
			SubmittedCount: 4,
			TotalEmployees: 5,
			EmployeeRoster: []RosterEntry{{Name: "Alice", Role: "member", IsActive: true}},
		})
	if err != nil {
		t.Fatal(err)
	}
	if resp == "" {
		t.Fatal("response should not be empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestNewChatService|TestChatService" -v`
Expected: FAIL — `ChatService`, `NewChatService` not defined

- [ ] **Step 3: Implement ChatService**

Create `internal/brain/chat.go`:

```go
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	maxHistoryMessages = 10
	historyTTL         = 24 * time.Hour
	rateLimitWindow    = 60 * time.Second
	rateLimitMax       = 5
	gapThreshold       = 6 * time.Hour
	rateLimitMessage   = "请稍等一下再继续对话"
	aiDisabledMessage  = "AI功能未启用，请联系管理员"
	aiErrorMessage     = "系统繁忙，请稍后再试"
)

// ChatRedisClient defines the Redis operations needed by ChatService.
// Extends the basic Get/Set/Del with Incr/Expire for rate limiting.
type ChatRedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// chatHistoryMessage represents a single message stored in Redis history.
type chatHistoryMessage struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	TS      time.Time `json:"ts"`
}

// RosterEntry holds basic employee info for boss context.
type RosterEntry struct {
	Name     string
	Role     string
	IsActive bool
}

// BossContext holds team data fetched from DB for boss chat.
type BossContext struct {
	LatestSummary  string
	SubmittedCount int
	TotalEmployees int
	EmployeeRoster []RosterEntry
}

// ChatServiceConfig holds dependencies for creating a ChatService.
type ChatServiceConfig struct {
	LLM           ChatLLMClient
	Redis         ChatRedisClient
	EngineFactory *EngineFactory
	BossTgID      int64
}

// ChatService orchestrates mentor chat for employees and boss.
type ChatService struct {
	llm           ChatLLMClient
	redis         ChatRedisClient
	engineFactory *EngineFactory
	bossTgID      int64
}

// NewChatService creates a new ChatService.
func NewChatService(cfg ChatServiceConfig) *ChatService {
	return &ChatService{
		llm:           cfg.LLM,
		redis:         cfg.Redis,
		engineFactory: cfg.EngineFactory,
		bossTgID:      cfg.BossTgID,
	}
}

// HandleEmployee processes a chat message from an employee.
func (s *ChatService) HandleEmployee(ctx context.Context, employeeID, tenantID, employeeName, mentorID, cultureCode, text string) (string, error) {
	if s.llm == nil {
		return aiDisabledMessage, nil
	}

	// Rate limiting
	if limited, err := s.checkRateLimit(ctx, employeeID); err != nil {
		slog.Error("rate limit check failed", "employee_id", employeeID, "error", err)
	} else if limited {
		return rateLimitMessage, nil
	}

	// Load engine for this tenant's mentor+culture
	engine, err := s.engineFactory.ForTenant(mentorID, cultureCode)
	if err != nil {
		engine, _ = s.engineFactory.ForTenant("inamori", "default")
	}

	// Load history and check for gap-based extraction
	history, err := s.loadHistory(ctx, chatKey(employeeID))
	if err != nil {
		slog.Warn("load chat history failed", "employee_id", employeeID, "error", err)
	}
	history = s.checkGapAndTrim(ctx, chatKey(employeeID), history, employeeID, tenantID)

	// Build system prompt with memory recall
	systemPrompt := engine.BuildEmployeeChatPrompt(ctx, tenantID, employeeID, employeeName, text)

	// Convert history to ChatMessage format
	chatHistory := historyToChatMessages(history)

	// Call LLM
	response, err := s.llm.ChatWithHistory(ctx, systemPrompt, chatHistory, text)
	if err != nil {
		slog.Error("chat LLM call failed", "employee_id", employeeID, "error", err)
		if IsAuthError(err) {
			return aiDisabledMessage, nil
		}
		return aiErrorMessage, nil
	}

	// Append user message and assistant response to history
	now := time.Now()
	history = append(history,
		chatHistoryMessage{Role: "user", Content: text, TS: now},
		chatHistoryMessage{Role: "assistant", Content: response, TS: now},
	)

	// Trim to max and save
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}
	if err := s.saveHistory(ctx, chatKey(employeeID), history); err != nil {
		slog.Error("save chat history failed", "employee_id", employeeID, "error", err)
	}

	return response, nil
}

// HandleBoss processes a chat message from the boss (chairman).
// Boss has no rate limit (per spec).
func (s *ChatService) HandleBoss(ctx context.Context, tenantID, mentorID, cultureCode, text string, bctx BossContext) (string, error) {
	if s.llm == nil {
		return aiDisabledMessage, nil
	}

	engine, err := s.engineFactory.ForTenant(mentorID, cultureCode)
	if err != nil {
		engine, _ = s.engineFactory.ForTenant("inamori", "default")
	}

	// Load boss history
	bossKey := bossHistoryKey(tenantID)
	history, err := s.loadHistory(ctx, bossKey)
	if err != nil {
		slog.Warn("load boss chat history failed", "tenant_id", tenantID, "error", err)
	}
	history = s.checkGapAndTrim(ctx, bossKey, history, "", tenantID)

	// Build employee roster text
	var rosterSB strings.Builder
	for i, emp := range bctx.EmployeeRoster {
		status := "active"
		if !emp.IsActive {
			status = "inactive"
		}
		fmt.Fprintf(&rosterSB, "%d. %s (%s, %s)\n", i+1, emp.Name, emp.Role, status)
	}

	// Check if boss mentions an employee by name
	memorySection := ""
	if engine.MemoryEngine() != nil {
		for _, emp := range bctx.EmployeeRoster {
			if matchEmployeeName(text, emp.Name) {
				// TODO: need employeeID for recall — for V1, skip dynamic employee memory
				break
			}
		}
	}

	// Build rate string
	rate := "0% (0/0)"
	if bctx.TotalEmployees > 0 {
		pct := float64(bctx.SubmittedCount) / float64(bctx.TotalEmployees) * 100
		rate = fmt.Sprintf("%.0f%% (%d/%d)", pct, bctx.SubmittedCount, bctx.TotalEmployees)
	}

	systemPrompt := engine.BuildBossPrompt(ctx, tenantID, BuildBossContext{
		LatestSummary:  bctx.LatestSummary,
		SubmissionRate: rate,
		EmployeeList:   rosterSB.String(),
		MemorySection:  memorySection,
	})

	chatHistory := historyToChatMessages(history)

	response, err := s.llm.ChatWithHistory(ctx, systemPrompt, chatHistory, text)
	if err != nil {
		slog.Error("boss chat LLM call failed", "tenant_id", tenantID, "error", err)
		if IsAuthError(err) {
			return aiDisabledMessage, nil
		}
		return aiErrorMessage, nil
	}

	now := time.Now()
	history = append(history,
		chatHistoryMessage{Role: "user", Content: text, TS: now},
		chatHistoryMessage{Role: "assistant", Content: response, TS: now},
	)
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}
	if err := s.saveHistory(ctx, bossKey, history); err != nil {
		slog.Error("save boss chat history failed", "tenant_id", tenantID, "error", err)
	}

	return response, nil
}

// --- Rate Limiting ---

func (s *ChatService) checkRateLimit(ctx context.Context, employeeID string) (bool, error) {
	key := "chat_rate:" + employeeID
	count, err := s.redis.Incr(ctx, key)
	if err != nil {
		return false, err
	}
	if count == 1 {
		// First message in this window — set expiry
		if err := s.redis.Expire(ctx, key, rateLimitWindow); err != nil {
			slog.Warn("set rate limit expire failed", "error", err)
		}
	}
	return count > rateLimitMax, nil
}

// --- History Management ---

func chatKey(employeeID string) string {
	return "chat:" + employeeID
}

func bossHistoryKey(tenantID string) string {
	return "chat:boss:" + tenantID
}

func (s *ChatService) loadHistory(ctx context.Context, key string) ([]chatHistoryMessage, error) {
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Key not found = empty history
	}
	var history []chatHistoryMessage
	if err := json.Unmarshal([]byte(data), &history); err != nil {
		return nil, err
	}
	return history, nil
}

func (s *ChatService) saveHistory(ctx context.Context, key string, history []chatHistoryMessage) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, key, string(data), historyTTL)
}

// checkGapAndTrim checks if there's a >6h gap since last message.
// If so, publishes ChatCompleted event for memory extraction and clears history.
func (s *ChatService) checkGapAndTrim(ctx context.Context, key string, history []chatHistoryMessage, employeeID, tenantID string) []chatHistoryMessage {
	if len(history) == 0 {
		return history
	}

	lastMsg := history[len(history)-1]
	if time.Since(lastMsg.TS) > gapThreshold {
		slog.Info("chat gap detected, clearing history for extraction",
			"key", key,
			"last_message", lastMsg.TS,
			"gap", time.Since(lastMsg.TS),
		)
		// Clear history — memory extraction happens via event subscriber
		_ = s.redis.Del(ctx, key)
		return nil
	}
	return history
}

// --- Helpers ---

func historyToChatMessages(history []chatHistoryMessage) []ChatMessage {
	msgs := make([]ChatMessage, len(history))
	for i, h := range history {
		msgs[i] = ChatMessage{Role: h.Role, Content: h.Content}
	}
	return msgs
}

// matchEmployeeName checks if the text mentions an employee by name.
// Case-insensitive exact match on full name or first name.
func matchEmployeeName(text, employeeName string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, strings.ToLower(employeeName)) {
		return true
	}
	// Also match first name
	parts := strings.Fields(employeeName)
	if len(parts) > 1 {
		if strings.Contains(lower, strings.ToLower(parts[0])) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestNewChatService|TestChatService" -v`
Expected: PASS

- [ ] **Step 5: Add more targeted tests for gap detection and name matching**

Add to `internal/brain/chat_test.go`:

```go
func TestMatchEmployeeName(t *testing.T) {
	tests := []struct {
		text string
		name string
		want bool
	}{
		{"How is Alice doing?", "Alice Smith", true},
		{"how is alice doing?", "Alice Smith", true},
		{"Tell me about Bob", "Alice Smith", false},
		{"What about Smith?", "Alice Smith", true},
		{"", "Alice", false},
	}
	for _, tt := range tests {
		got := matchEmployeeName(tt.text, tt.name)
		if got != tt.want {
			t.Errorf("matchEmployeeName(%q, %q) = %v, want %v", tt.text, tt.name, got, tt.want)
		}
	}
}

func TestChatService_GapDetection(t *testing.T) {
	rdb := newMockChatRedis()
	svc := &ChatService{redis: rdb}

	// Create history with old timestamp (>6h ago)
	old := []chatHistoryMessage{
		{Role: "user", Content: "hi", TS: time.Now().Add(-7 * time.Hour)},
		{Role: "assistant", Content: "hello", TS: time.Now().Add(-7 * time.Hour)},
	}
	data, _ := json.Marshal(old)
	rdb.Set(context.Background(), "chat:emp-gap", string(data), historyTTL)

	// checkGapAndTrim should detect gap and clear
	result := svc.checkGapAndTrim(context.Background(), "chat:emp-gap", old, "emp-gap", "tenant-1")
	if len(result) != 0 {
		t.Fatalf("expected empty history after gap, got %d messages", len(result))
	}
}

func TestChatService_HistoryTrimming(t *testing.T) {
	llm := &mockChatLLM{response: "ok"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	// Send 6 messages (each creates 2 entries: user + assistant = 12, trimmed to 10)
	for i := 0; i < 6; i++ {
		_, _ = svc.HandleEmployee(context.Background(), "emp-trim", "tenant-1", "Test", "inamori", "default", fmt.Sprintf("msg %d", i))
	}

	history, _ := svc.loadHistory(context.Background(), "chat:emp-trim")
	if len(history) > maxHistoryMessages {
		t.Fatalf("expected max %d messages, got %d", maxHistoryMessages, len(history))
	}
}
```

- [ ] **Step 6: Run all ChatService tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/brain/ -run "TestChatService|TestMatchEmployee" -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/brain/chat.go internal/brain/chat_test.go
git commit -m "feat: add ChatService with employee/boss chat, rate limiting, history"
```

---

### Task 5: Wire ChatService into main.go

This connects everything — creates ChatService, adds boss detection, fills both `default` branches.

**Files:**
- Modify: `cmd/brain/main.go:86-101` (redisWrapper — add Incr/Expire)
- Modify: `cmd/brain/main.go:410-426` (after Anthropic client creation — create ChatService)
- Modify: `cmd/brain/main.go:596-667` (Telegram text handler — add boss check + fill default)
- Modify: `cmd/brain/main.go:669-722` (UnifiedHandler OnText — fill default)

- [ ] **Step 1: Extend redisWrapper with Incr and Expire**

In `cmd/brain/main.go`, after the existing `Del` method on `redisWrapper` (~line 101), add:

```go
func (r *redisWrapper) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *redisWrapper) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}
```

- [ ] **Step 2: Create ChatService after Anthropic client initialization**

In `cmd/brain/main.go`, after the LLM client block (~line 426), add:

```go
// Create ChatService for mentor chat (requires LLM)
var chatService *brain.ChatService
if cfg.AnthropicKey != "" {
	chatRedis := &redisWrapper{client: rdb}
	chatService = brain.NewChatService(brain.ChatServiceConfig{
		LLM:           llmClient,
		Redis:         chatRedis,
		EngineFactory: engineFactory,
		BossTgID:      cfg.BossTelegramID,
	})
	slog.Info("ChatService initialized for mentor chat")
}
```

Note: `llmClient` is the `*AnthropicClient` created at line 415. It satisfies both `LLMClient` and `ChatLLMClient`.

- [ ] **Step 3: Add boss check and fill default in Telegram text handler**

Replace the Telegram text handler (lines 597-667) with:

```go
tgBot.RegisterTextHandler(func(senderID int64, text string, sendReply func(string) error) error {
	ctx := context.Background()

	// Check if sender is the boss FIRST (boss is not in employees table)
	if senderID == cfg.BossTelegramID {
		if chatService == nil {
			return sendReply(brain.AIDisabledMessage())
		}
		// Send typing indicator
		tgAdapter.Bot().ChatAction(tele.ChatID(senderID), tele.Typing)

		tenant, err := botDB.GetTenantByBossChatID(ctx, senderID)
		if err != nil {
			slog.Error("boss chat: get tenant", "error", err)
			return sendReply(brain.AIErrorMessage())
		}

		// Fetch team context
		bossCtx := fetchBossContext(ctx, queries, tenant.ID, loc)

		resp, err := chatService.HandleBoss(ctx, tenant.ID, tenant.MentorID, "default", text, bossCtx)
		if err != nil {
			slog.Error("boss chat failed", "error", err)
			return sendReply(brain.AIErrorMessage())
		}
		return sendReply(resp)
	}

	// Look up employee by telegram_id
	emp, err := botDB.GetEmployeeByTelegramID(ctx, senderID)
	if err != nil {
		return nil
	}

	empID := emp.ID
	state := collector.GetState(ctx, empID)
	lower := strings.ToLower(strings.TrimSpace(text))

	switch state {
	case report.StateConfirming:
		if lower == "confirm" {
			answers := collector.GetAnswers(ctx, empID)
			cState, msg, err := collector.Confirm(ctx, empID)
			if err != nil {
				slog.Error("confirm report", "employee_id", empID, "error", err)
				return sendReply("Error confirming report. Please try again.")
			}
			if cState == report.StateComplete && answers != nil {
				today := time.Now().In(loc).Format("2006-01-02")
				if err := reportDB.CreateReport(ctx, emp.TenantID, empID, today, answers); err != nil {
					slog.Error("save report", "employee_id", empID, "error", err)
					return sendReply("Report confirmed but failed to save. Please contact your manager.")
				}
				slog.Info("report saved", "employee_id", empID, "date", today)
				_ = eventBus.PublishPayload(ctx, events.ReportSubmitted, emp.TenantID, events.ReportSubmittedPayload{
					EmployeeID:   empID,
					EmployeeName: emp.Name,
					ReportDate:   today,
					Channel:      "telegram",
				})
				go func(eid, tid, date string) {
					if err := analyzer.Analyze(context.Background(), eid, date); err != nil {
						slog.Error("report analysis failed", "employee_id", eid, "error", err)
					}
				}(empID, emp.TenantID, today)
			}
			return sendReply(msg)
		}
		if lower == "edit" {
			_, firstQ, err := collector.Start(ctx, empID)
			if err != nil {
				return sendReply("Error restarting. Please try again.")
			}
			return sendReply("Let's start over.\n\n" + firstQ)
		}
		return sendReply("Please reply 'confirm' to submit or 'edit' to start over.")

	case report.StateCollecting:
		cState, nextMsg, err := collector.HandleAnswer(ctx, empID, text)
		if err != nil {
			slog.Error("handle answer", "employee_id", empID, "error", err)
			return sendReply("Error processing your answer. Please try again.")
		}
		_ = cState
		if nextMsg != "" {
			return sendReply(nextMsg)
		}

	default:
		// Mentor chat — idle state
		if chatService == nil {
			return nil // AI not enabled, stay silent
		}
		// Send typing indicator for better UX
		tgAdapter.Bot().ChatAction(tele.ChatID(senderID), tele.Typing)

		tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
		if err != nil {
			slog.Warn("mentor chat: failed to get tenant", "error", err)
			return nil
		}
		resp, err := chatService.HandleEmployee(ctx, empID, emp.TenantID, emp.Name, tenant.MentorID, emp.CultureCode, text)
		if err != nil {
			slog.Error("mentor chat failed", "employee_id", empID, "error", err)
			return nil
		}
		if resp != "" {
			return sendReply(resp)
		}
	}

	return nil
})
```

- [ ] **Step 4: Fill default branch in UnifiedHandler OnText**

Replace the UnifiedHandler OnText function (lines 673-721) to add boss detection and fill the default branch:

```go
OnText: func(ctx context.Context, employeeID, tenantID, text, channelType string) (string, error) {
	// Check if this employee is the boss (multi-channel: detected via role in resolveEmployee)
	// Boss check via non-Telegram is handled by emp.Role in the handler;
	// for V1, UnifiedHandler doesn't have access to emp.Role directly in OnText.
	// Boss chat via Telegram is handled above. Non-TG boss chat deferred to future.

	state := collector.GetState(ctx, employeeID)
	lower := strings.ToLower(strings.TrimSpace(text))

	switch state {
	case report.StateConfirming:
		if lower == "confirm" {
			answers := collector.GetAnswers(ctx, employeeID)
			cState, msg, err := collector.Confirm(ctx, employeeID)
			if err != nil {
				return "Error confirming report. Please try again.", nil
			}
			if cState == report.StateComplete && answers != nil {
				today := time.Now().In(loc).Format("2006-01-02")
				if err := reportDB.CreateReport(ctx, tenantID, employeeID, today, answers); err != nil {
					return "Report confirmed but failed to save.", nil
				}
				_ = eventBus.PublishPayload(ctx, events.ReportSubmitted, tenantID, events.ReportSubmittedPayload{
					EmployeeID:   employeeID,
					EmployeeName: "",
					ReportDate:   today,
					Channel:      channelType,
				})
				go func() {
					if err := analyzer.Analyze(context.Background(), employeeID, today); err != nil {
						slog.Error("report analysis failed", "employee_id", employeeID, "error", err)
					}
				}()
			}
			return msg, nil
		}
		if lower == "edit" {
			_, firstQ, err := collector.Start(ctx, employeeID)
			if err != nil {
				return "Error restarting. Please try again.", nil
			}
			return "Let's start over.\n\n" + firstQ, nil
		}
		return "Please reply 'confirm' to submit or 'edit' to start over.", nil

	case report.StateCollecting:
		_, nextMsg, err := collector.HandleAnswer(ctx, employeeID, text)
		if err != nil {
			return "Error processing your answer. Please try again.", nil
		}
		return nextMsg, nil

	default:
		// Mentor chat — idle state
		if chatService == nil {
			return "", nil
		}
		tenant, err := botDB.GetTenantByBossChatID(ctx, cfg.BossTelegramID)
		if err != nil {
			return "", nil
		}
		resp, err := chatService.HandleEmployee(ctx, employeeID, tenantID, "", tenant.MentorID, "default", text)
		if err != nil {
			slog.Error("unified mentor chat failed", "employee_id", employeeID, "error", err)
			return "", nil
		}
		return resp, nil
	}
},
```

- [ ] **Step 5: Add fetchBossContext helper function**

Add before `main()` in `cmd/brain/main.go`:

```go
// fetchBossContext gathers team data for boss chat from the database.
func fetchBossContext(ctx context.Context, queries *sqlc.Queries, tenantID string, loc *time.Location) brain.BossContext {
	uid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return brain.BossContext{}
	}

	// Latest summary
	summary, err := queries.GetLatestSummary(ctx, uid)
	latestSummary := ""
	if err == nil {
		latestSummary = summary.Content
	}

	// Today's submission count
	today := time.Now().In(loc).Format("2006-01-02")
	todayDate, _ := time.Parse("2006-01-02", today)
	pgDate := pgtype.Date{Time: todayDate, Valid: true}
	submitted, _ := queries.CountReportsByTenantDate(ctx, sqlc.CountReportsByTenantDateParams{
		TenantID:   uid,
		ReportDate: pgDate,
	})

	// Active employees
	emps, _ := queries.ListActiveEmployees(ctx, uid)
	roster := make([]brain.RosterEntry, 0, len(emps))
	for _, e := range emps {
		roster = append(roster, brain.RosterEntry{
			Name:     e.Name,
			Role:     e.Role,
			IsActive: e.IsActive,
		})
	}

	return brain.BossContext{
		LatestSummary:  latestSummary,
		SubmittedCount: int(submitted),
		TotalEmployees: len(emps),
		EmployeeRoster: roster,
	}
}

// parseUUIDForChat parses a UUID string into pgtype.UUID.
func parseUUIDForChat(s string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(s); err != nil {
		return uid, err
	}
	return uid, nil
}
```

- [ ] **Step 6: Add exported message constants to brain package**

Add to `internal/brain/chat.go`:

```go
// AIDisabledMessage returns the user-facing message when AI is not configured.
func AIDisabledMessage() string { return aiDisabledMessage }

// AIErrorMessage returns the user-facing message when AI encounters an error.
func AIErrorMessage() string { return aiErrorMessage }
```

- [ ] **Step 7: Add necessary imports to main.go**

Ensure these imports are present in `cmd/brain/main.go`:

```go
tele "gopkg.in/telebot.v3"               // for typing indicator (tele.ChatID, tele.Typing)
"github.com/jackc/pgx/v5/pgtype"         // for fetchBossContext UUID/Date parsing
```

The codebase uses `tele` as the import alias for telebot (see `internal/channel/telegram.go`). `main.go` does NOT currently import telebot, so add it. The `pgtype` import is also not currently in `main.go` — add it for `fetchBossContext`.

- [ ] **Step 8: Build and verify compilation**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./cmd/brain/`
Expected: Successful compilation, no errors.

- [ ] **Step 9: Run all tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./... 2>&1 | tail -30`
Expected: All existing tests pass, new tests pass.

- [ ] **Step 10: Commit**

```bash
git add cmd/brain/main.go internal/brain/chat.go
git commit -m "feat: wire ChatService into text handlers for mentor chat"
```

---

### Task 6: Integration test and deploy

**Files:**
- No new files — testing and deployment

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./... -count=1`
Expected: All tests pass.

- [ ] **Step 2: Build Linux binary**

Run: `cd /Users/anna/Documents/ai-management-brain && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain/`
Expected: `brain` binary created.

- [ ] **Step 3: Push to GitHub**

```bash
cd /Users/anna/Documents/ai-management-brain && git push origin main
```

- [ ] **Step 4: Deploy to server**

```bash
ssh ai-brain "cd ~/ai-management-brain && git pull && docker compose -f docker-compose.prod.yml up -d --build"
```

- [ ] **Step 5: Verify health check**

```bash
ssh ai-brain "curl -s localhost/healthz | python3 -m json.tool"
```
Expected: `{"status": "ok", "db": "ok", "redis": "ok"}`

- [ ] **Step 6: Manual test — send message to bot as employee and as boss**

Send a free-form message to `@aimanagerbrainbot` on Telegram:
- As employee: "How do I improve my productivity?"
- As boss: "How is the team performing today?"

Expected: Bot responds with mentor-style advice (not silence).

- [ ] **Step 7: Commit any fixes discovered during manual testing**

---

## Notes

- **V1 Limitation:** Boss chat via non-Telegram channels (Slack/Lark) not supported in this iteration because the `OnText` callback doesn't receive `emp.Role`. For V2, pass employee role through `UnifiedHandlerConfig.OnText` or add a separate `OnBossText` callback. The fix is straightforward: modify `HandleMessage` in `message_handler.go` to pass `emp.Role` through to `OnText`.
- **Memory extraction on gap:** The `checkGapAndTrim` method clears history when >6h gap is detected. The `ChatCompleted` event type is defined (Task 2) and ready for a subscriber that calls `memEngine.ExtractFromChat`. The subscriber itself is not implemented here — add as a follow-up by subscribing to `ChatCompleted` in main.go, similar to the existing `ReportSubmitted` subscriber.
- **Boss rate limiting:** Intentionally omitted per spec. A comment `// Boss has no rate limit (per spec)` should be in `HandleBoss`.
- **Typing indicator:** Uses `tgAdapter.Bot().ChatAction()` with `tele` import alias (matching `internal/channel/telegram.go` convention).
- **UnifiedHandler employee name:** The `OnText` callback doesn't receive `employeeName`. Passing `""` for now — the mentor still works but without personalization. For V2, add `employeeName` to `OnText` signature.
