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
    ├── StateCollecting → Report Collector (unchanged)
    ├── StateConfirming → Report Collector (unchanged)
    └── StateIdle (default) → ChatService
                                  │
                                  ├── Load history (Redis)
                                  ├── Build system prompt
                                  │   ├── Employee: mentor + culture + employee memories
                                  │   └── Boss: mentor + culture + team summary + employee list + on-demand employee memories
                                  ├── Call ChatWithHistory()
                                  ├── Save history (Redis)
                                  └── Return response
```

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

- Key: `chat:{employeeID}`
- Value: JSON array of messages
- Structure: `[{"role": "user"|"assistant", "content": "...", "ts": "2026-03-22T10:00:00Z"}]`
- Max messages: 20 (oldest trimmed when exceeded)
- TTL: 24 hours

### State Coordination with Report Collection

- Report state: `conv:{employeeID}` (existing, unchanged)
- Chat history: `chat:{employeeID}` (new, independent key)
- Priority in text handler:
  1. `StateCollecting` / `StateConfirming` → report flow (unchanged)
  2. `StateIdle` (default) → mentor chat
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

### LLM Method Addition

```go
// In brain/llm.go
type ChatMessage struct {
    Role    string // "user" or "assistant"
    Content string
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
- Existing `Chat()` unchanged — other features continue using it

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
- Match name against roster
- If matched, recall that employee's memories via `MemoryEngine.Recall()`
- Inject into system prompt's `<memory>` section for the next turn

### Conversation → Memory Extraction

- When chat history expires (24h TTL) or user sends `/endchat`:
  - Publish `ChatCompleted` event on event bus
  - Memory extractor processes the conversation
  - Extracts key insights as `SourceConversation` memories
  - Stored as `TierShortTerm`, subject to normal consolidation

### Token Budget

| Component | Tokens |
|-----------|--------|
| System prompt (mentor + culture) | ~500 |
| Memory recall | ~800 |
| Boss team data | ~500 |
| Conversation history (20 msgs) | ~2000 |
| Response | ≤1024 |
| **Total per turn** | **≤5000** |

## Section 4: Multi-Channel Support & Error Handling

### Multi-Channel

- Telegram: fill `default` branch in `tgBot.RegisterTextHandler()`
- Slack/Lark/Signal: fill `default` branch in `UnifiedHandler.OnText`
- Reply on the **originating channel** (use `msg.ChannelType`, not `ResolveChannel`)
- All channels share the same chat history (keyed by `employeeID`)

### Error Handling

| Scenario | Response |
|----------|----------|
| `ANTHROPIC_API_KEY` not configured | "AI功能未启用，请联系管理员" |
| API call fails (after 3 retries) | "系统繁忙，请稍后再试" |
| Unregistered user | Silent ignore (existing behavior) |
| Employee has no tenant mentor | Use default mentor (inamori) |

### Rate Limiting

- Redis key: `chat_rate:{employeeID}`, TTL: 60 seconds
- Employee limit: 5 messages/minute
- Exceeded: reply "请稍等一下再继续对话"
- Boss: no rate limit

## Files Changed

### New Files
- `internal/brain/chat.go` — ChatService: Redis history management, role detection, prompt assembly, rate limiting

### Modified Files
- `internal/brain/llm.go` — add `ChatWithHistory()` method and `ChatMessage` type
- `internal/brain/engine.go` — add `BuildBossPrompt()` method
- `internal/events/bus.go` — add `ChatCompleted` event constant
- `cmd/brain/main.go` — fill text handler `default` branch, create ChatService, wire to UnifiedHandler

### Unchanged Files
- `internal/report/collector.go` — report flow untouched
- Frontend — no UI changes (users interact via messaging channels)
- Database schema — no new tables (Redis only)
- All existing handlers, routes, middleware

## Out of Scope

- Streaming responses (nice-to-have for future)
- Chat history persistence in PostgreSQL (Redis-only for now)
- Frontend chat interface (users chat via Telegram/Slack/Lark/Signal)
- `/endchat` command (can be added later; TTL handles cleanup)
- Image/file message handling (text only)
