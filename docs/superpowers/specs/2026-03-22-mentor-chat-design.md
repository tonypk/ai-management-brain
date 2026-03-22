# Intelligent Mentor Chat Design Spec

## Problem

When employees send free-form text messages to the Telegram bot (outside of report collection), the text handler's `default` branch returns `nil` — the user receives silence. The mentor infrastructure (14 mentors, 6 cultures, memory engine, Claude API) exists but is never invoked for interactive conversation.

## Solution

Fill the `default` branch in the text handler to route free-form messages to an AI mentor chat powered by Claude, with multi-turn conversation history stored in Redis and role-based context injection.

## Role Model

- **Boss = Chairman (董事长)** — has full visibility into team data and individual employees
- **Mentor = CEO** — reports to the chairman, coaches employees
- **Employee = Team member** — can only discuss their own work with the CEO-mentor

## Architecture: Approach A (Extend Existing)

Minimal changes to the existing codebase. No new database tables. Three key additions:

1. `ChatWithHistory()` in `brain/llm.go` — multi-turn Claude API call
2. `ChatService` in `brain/chat.go` — orchestrates history, prompts, rate limiting
3. Fill `default` branches in text handlers (`main.go` + `UnifiedHandler`)

### Component Overview

```
User Message (any channel)
    │
    ▼
Text Handler
    │
    ├── Boss (Telegram only) → ChatService.HandleBoss()
    │       detect via cfg.BossTelegramID, resolve tenant via GetTenantByBossChatID
    │
    ├── Employee (any channel) →
    │       ├── StateCollecting → Report Collector (unchanged)
    │       ├── StateConfirming → Report Collector (unchanged)
    │       └── StateIdle (default) → ChatService.HandleEmployee()
    │
    └── Unknown sender → silent ignore
```

### Boss Detection Flow

The boss is NOT in the `employees` table — they are identified by `cfg.BossTelegramID` (Telegram) or by the employee's `role = "boss"` (multi-channel). The text handler must check boss identity BEFORE the employee lookup:

**Telegram handler:**
```go
// Check if sender is the boss first
if senderID == cfg.BossTelegramID {
    tenant, err := botDB.GetTenantByBossChatID(ctx, senderID)
    response := chatService.HandleBoss(ctx, tenantID, text)
    sendReply(response)
    return nil
}
// Then try employee lookup
emp, err := botDB.GetEmployeeByTelegramID(ctx, senderID)
```

**UnifiedHandler (Slack/Lark/Signal):**
```go
// resolveEmployee already runs — check if emp.Role == "boss"
// Boss chat via non-Telegram channels uses employee record's role field
```

**V1 Limitation:** Boss chat with full team data injection works on all channels where the boss has an employee record with `role = "boss"`. The `cfg.BossTelegramID` path is Telegram-specific for backwards compatibility.

## Section 1: Conversation Roles & System Prompts

### Employee Mode (member/manager)

- System prompt = `BuildSystemPromptWithMemory(ctx, tenantID, employeeID, userMessage)`
- Mentor acts as CEO coaching a team member
- Context: employee's own memories, profile, report history
- Cannot access other employees' data
- Tone: supportive, coaching, aligned with mentor philosophy

### Boss Mode (boss)

- System prompt = `BuildBossPrompt(ctx, tenantID, userMessage)`
- Mentor acts as CEO reporting to the chairman
- Additional context injected:
  - Latest team summary (from `summaries` table)
  - Today's submission rate (submitted / total employees)
  - Employee roster (name, role, is_active)
- When boss mentions an employee by name, dynamically recall that employee's memories and inject into the next turn's system prompt
- Tone: peer-level, strategic, data-informed

### Employee Name Matching (Boss Mode)

When the boss mentions an employee, match against roster using case-insensitive exact match on full name or first name (split by space). Known limitations: no fuzzy matching, no nickname support. This is acceptable for V1 — teams are small (typically <50 people).

```go
func matchEmployeeName(text string, roster []Employee) *Employee {
    lower := strings.ToLower(text)
    for _, emp := range roster {
        if strings.Contains(lower, strings.ToLower(emp.Name)) {
            return &emp
        }
    }
    return nil
}
```

### System Prompt Structure

**Employee:**
```
{mentor_philosophy}
{cultural_context}

<memory>
{recalled_employee_memories}
</memory>

You are {mentor_name}, acting as CEO and management coach. The employee "{employee_name}" is asking you for guidance. Respond based on your management philosophy. Keep responses concise and actionable.
```

**Boss:**
```
{mentor_philosophy}
{cultural_context}

<team_context>
## Latest Team Summary
{latest_summary}

## Today's Status
Submission rate: {rate}% ({submitted}/{total})

## Team Roster
{employee_list}
</team_context>

<memory>
{recalled_memories_if_employee_mentioned}
</memory>

You are {mentor_name}, acting as CEO reporting to the chairman. The chairman is consulting you about management decisions. Provide data-driven insights based on team performance. Be candid and strategic.
```

## Section 2: Conversation History & State Management

### Redis Storage

- Key: `chat:{employeeID}` (for employees), `chat:boss:{tenantID}` (for boss via Telegram)
- Value: JSON array of messages
- Structure: `[{"role": "user"|"assistant", "content": "...", "ts": "2026-03-22T10:00:00Z"}]`
- Max messages: 10 (oldest trimmed when exceeded; keeps token budget realistic)
- TTL: 24 hours

### State Coordination with Report Collection

- Report state: `conv:{employeeID}` (existing, unchanged)
- Chat history: `chat:{employeeID}` (new, independent key)
- Priority in text handler:
  1. Boss check (Telegram: `cfg.BossTelegramID`, other: `emp.Role == "boss"`)
  2. `StateCollecting` / `StateConfirming` → report flow (unchanged)
  3. `StateIdle` (default) → mentor chat
- Chat history persists across report interruptions

### Report Interruption Flow

```
Employee chatting (StateIdle) → chat:{id} has history
    │
    ▼
9:00 AM remind job fires → sends check-in questions
    │
    ▼
collector.Start() → state = StateCollecting
    │
    ▼
Next message → routed to report collector (not chat)
    │
    ▼
Report complete → state = StateIdle
    │
    ▼
Next message → back to mentor chat (history preserved)
```

### LLM Interface Addition

Introduce a new `ChatLLMClient` interface to keep the existing `LLMClient` unchanged and maintain testability:

```go
// In brain/llm.go

type ChatMessage struct {
    Role    string // "user" or "assistant"
    Content string
}

// ChatLLMClient extends LLM capabilities with multi-turn conversation.
type ChatLLMClient interface {
    ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error)
}

func (a *AnthropicClient) ChatWithHistory(
    ctx context.Context,
    systemPrompt string,
    history []ChatMessage,
    userMessage string,
) (string, error)
```

- Converts history + userMessage into Claude API `messages` array
- Uses existing retry logic (3 attempts with exponential backoff)
- Max tokens: 1024 (same as `Chat()`)
- Existing `LLMClient` interface and `Chat()` unchanged
- `AnthropicClient` satisfies both `LLMClient` and `ChatLLMClient`

### ChatService Dependencies

```go
type ChatService struct {
    llm          ChatLLMClient
    redis        *redis.Client    // Direct redis client (needs INCR for rate limiting)
    queries      *sqlc.Queries
    engineFactory *brain.EngineFactory
    memoryEngine *memory.MemoryEngine  // nil = no memory
    bossTgID     int64                 // cfg.BossTelegramID
}
```

Using `*redis.Client` directly (not `RedisClient` interface) because rate limiting requires `INCR` + `EXPIRE` which the existing interface doesn't support. This is acceptable since `ChatService` is a new file with no legacy constraints.

## Section 3: Memory & Data Injection

### Employee Chat — Memory Recall

- Each conversation turn calls `BuildSystemPromptWithMemory()` with the user's message as query
- Automatically recalls relevant `EmployeeInsight`, `StrategyResult`, `OrgKnowledge` memories
- Enables continuity: "You mentioned a blocker last week — how did that resolve?"

### Boss Chat — Team Data Injection

On each boss message, fetch from DB:
- Latest summary: `SELECT * FROM summaries WHERE tenant_id = $1 ORDER BY summary_date DESC LIMIT 1`
- Today's submission rate: count reports for today vs active employees
- Employee roster: `SELECT name, role, is_active FROM employees WHERE tenant_id = $1`

When boss mentions an employee name:
- Match name against roster (case-insensitive exact match, see Section 1)
- If matched, recall that employee's memories via `MemoryEngine.Recall()`
- Inject into system prompt's `<memory>` section for the next turn

### Conversation → Memory Extraction

Triggered on a **gap-based** approach: when a user sends a new message and the last message in their chat history is older than 6 hours, extract insights from the old conversation before starting a fresh one.

- `ChatService` checks `last_message_ts` in history
- If gap > 6 hours: extract old conversation → publish `ChatCompleted` event → clear history → start fresh
- Memory extractor processes the conversation content
- Extracts key insights as `SourceConversation` memories
- Stored as `TierShortTerm`, subject to normal consolidation

This avoids dependency on Redis keyspace notifications or `/endchat` commands.

### Token Budget

| Component | Tokens |
|-----------|--------|
| System prompt (mentor + culture) | ~500 |
| Memory recall | ~800 |
| Boss team data | ~500 |
| Conversation history (10 msgs) | ~2000 |
| Response | ≤1024 |
| **Total per turn** | **≤5000** |

Note: With 10 messages (5 user + 5 assistant), history averages ~2000 tokens. If responses are long, older messages are trimmed first to stay within budget.

## Section 4: Multi-Channel Support & Error Handling

### Multi-Channel

- Telegram: fill `default` branch in `tgBot.RegisterTextHandler()`, add boss check before employee lookup
- Slack/Lark/Signal: fill `default` branch in `UnifiedHandler.OnText`, boss detected via `emp.Role == "boss"`
- Reply on the **originating channel** (use `msg.ChannelType`, not `ResolveChannel`)
- All channels share the same chat history (keyed by `employeeID`)
- Send Telegram "typing" indicator (`sendChatAction`) before Claude API call for better UX

### Error Handling

| Scenario | Response |
|----------|----------|
| `ANTHROPIC_API_KEY` not configured | "AI功能未启用，请联系管理员" |
| API call fails (after 3 retries) | "系统繁忙，请稍后再试" |
| Unregistered user | Silent ignore (existing behavior) |
| Employee has no tenant mentor | Use default mentor (inamori) |

### Response Language

The mentor responds in the same language the user writes in. The system prompt includes culture context but does not force a specific language. Claude naturally mirrors the user's language.

### Rate Limiting

- Redis: `INCR` on `chat_rate:{employeeID}` with `EXPIRE 60`
- Employee limit: 5 messages/minute
- Exceeded: reply "请稍等一下再继续对话"
- Boss: no rate limit

## Files Changed

### New Files
- `internal/brain/chat.go` — ChatService with HandleEmployee(), HandleBoss(), Redis history, rate limiting, prompt assembly

### Modified Files
- `internal/brain/llm.go` — add `ChatLLMClient` interface, `ChatMessage` type, `ChatWithHistory()` method
- `internal/brain/engine.go` — add `BuildBossPrompt()` method
- `internal/events/bus.go` — add `ChatCompleted` event constant
- `cmd/brain/main.go` — add boss check in Telegram text handler, fill `default` branch, create ChatService, wire to UnifiedHandler

### Unchanged Files
- `internal/report/collector.go` — report flow untouched
- `internal/brain/llm.go` `LLMClient` interface — unchanged, existing callers unaffected
- Frontend — no UI changes (users interact via messaging channels)
- Database schema — no new tables (Redis only)
- All existing handlers, routes, middleware

## Out of Scope

- Streaming responses (nice-to-have for future)
- Chat history persistence in PostgreSQL (Redis-only for now)
- Frontend chat interface (users chat via Telegram/Slack/Lark/Signal)
- `/endchat` command (gap-based extraction handles cleanup)
- Image/file message handling (text only)
