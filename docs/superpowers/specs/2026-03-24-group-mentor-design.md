# Group Mentor Design

## Goal

Enable the AI mentor to participate in team group chats — responding to @mentions with team-aware context, and proactively posting messages when the AI determines it's appropriate based on team data and group type.

## Background

The AI Management Brain currently handles 1-on-1 conversations with employees (check-ins, mentor chat) and the boss (management advice). Group chats are a natural extension: the mentor can reinforce team culture, share relevant insights, and respond to team-level questions — all while maintaining its current mentor persona (inamori, dalio, etc.).

## Requirements

1. **Mentor identity**: Appears with full mentor persona in group chats (same personality as 1-on-1)
2. **Multi-group support**: One tenant can have multiple groups (engineering, operations, sales, etc.)
3. **@mention replies**: Responds when @mentioned, using group context + team-level memory (no individual private data)
4. **Autonomous posting**: AI decides daily whether to post, based on team data and group type
5. **Privacy**: Never reveals individual employee reports, sentiments, or personal memories in group chat

## Architecture

### Data Model

New `group_chats` table:

```sql
CREATE TABLE group_chats (
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

CREATE INDEX idx_group_chats_tenant ON group_chats(tenant_id) WHERE is_active = true;
```

**Fields:**
- `platform` / `platform_chat_id`: Channel type and platform-specific chat ID (Telegram group IDs are negative integers)
- `group_type`: One of `general`, `engineering`, `operations`, `sales`, `support` — tells the AI what kind of group this is
- `is_active`: Soft-delete flag

### Group Registration

Bot registers groups via the existing `/join` command in group chat context:

1. Boss adds bot to a Telegram group
2. Someone sends `/join <invite_code>` in the group
3. Bot detects group chat context (chat ID is negative)
4. Creates `group_chats` record with `platform_chat_id` = group chat ID
5. Auto-fills `name` from Telegram API (`c.Chat().Title`)
6. Default `group_type` = `general` (boss changes via frontend)

### @Mention Reply Flow

```
Telegram group message
  → Bot receives message (group chat type)
  → Check: is bot @mentioned? → No → ignore
  → Yes → lookup group_chats by (platform, platform_chat_id)
  → Not found → ignore (unregistered group)
  → Found → load tenant's mentor engine
  → Build group system prompt:
      - Mentor base persona + culture
      - Group type context ("You are in an engineering team group chat")
      - Latest team summary (reuse existing summary data)
      - Privacy rule ("Never mention individual private information")
  → LLM.Chat() (single-turn, no history)
  → Reply to the original message in group
```

**Key decisions:**
- **No chat history** for groups — group messages are high volume and noisy. Each @reply is independent.
- **Single-turn `Chat()`** — not `ChatWithHistory()`. No need for conversational continuity in groups.
- **Reply-to-message** — uses Telegram's reply feature so everyone knows which question the mentor is answering.
- **Team memory, not individual** — uses latest summary and aggregate team data, never individual employee memories.

### Autonomous Posting (Daily AI Decision)

**Trigger:** Scheduler job at 9:00 AM daily. Gets the current tenant via `GetTenantByBossChatID`, then iterates all active group_chats for that tenant via `ListActiveGroupChats(ctx, tenantID)`.

**Per-group decision flow:**

```
1. Load tenant's mentor engine
2. Collect team data:
   - Yesterday's submission rate + sentiment distribution
   - Latest summary excerpt
   - Day of week (Friday = good for weekly review)
3. Check anti-spam: Redis key group:last_post:{groupID}
   - If posted in last 24h → skip (hard limit: max 1 proactive message per group per day)
4. Build decision prompt:

   你是{mentor_name}，管理一个{group_type}类型的团队群聊。

   团队数据：
   - 昨日提交率: {rate}%
   - 情绪分布: {sentiment}
   - 最新总结: {summary}
   - 今天是: {weekday}

   决定你是否需要在群里发一条消息。
   如果不需要，只回复 SKIP（不要加任何其他内容）。
   如果需要，直接输出消息内容。

   规则：
   - 不要每天都发，大约每周2-3次即可
   - 周五适合发本周回顾
   - 提交率低于60%时可以鼓励大家
   - 保持你的导师风格和文化语境
   - 不要提及任何个人的私人信息
   - 消息简洁，控制在3-5句话
   - 使用团队的文化语境对应的语言

5. Call LLM.Chat()
6. If response != "SKIP":
   - Send to group
   - Set Redis key group:last_post:{groupID} with 24h TTL
```

**Anti-spam safeguards:**
- AI prompt instructs "~2-3 times per week"
- Redis hard limit: 1 proactive message per group per 24 hours
- `is_active` flag lets boss disable per group

**Token cost estimate:**
- Decision call per group per day: ~500-800 tokens (most return SKIP)
- 10 groups x 30 days ≈ ~200K tokens/month ≈ $0.60 (Claude Sonnet)

### API Endpoints

```
GET    /api/v1/admin/groups        — List all groups for tenant
PUT    /api/v1/admin/groups/:id    — Update group type / is_active
DELETE /api/v1/admin/groups/:id    — Soft delete (set is_active=false)
```

No POST endpoint — groups are created automatically via Bot `/join` command in group chat.

### Frontend: Admin Group Chats Page

New route: `/admin/groups` (GroupChatsView.vue)

Simple table view:
| Column | Type |
|--------|------|
| Name | Text (from Telegram) |
| Type | Dropdown: general/engineering/operations/sales/support |
| Platform | Badge (telegram/slack/lark) |
| Status | Toggle (active/inactive) |
| Actions | Edit type, Deactivate |

Added to admin nav in App.vue.

### Bot Layer Changes

**Interface extensions required:**

The current `BotContext` interface and `TextHandlerFunc` type lack group chat capabilities. These must be extended:

1. **`BotContext` interface** (internal/bot/commands.go) — add:
   - `ChatID() int64` — returns the chat ID (negative for groups)
   - `ChatType() string` — returns "private", "group", or "supergroup"
   - `ChatTitle() string` — returns group name (empty for private chats)
   - `Reply(msg string) error` — replies to the specific message (vs `Send` which sends a new message)

2. **`teleBotContext` adapter** (internal/bot/commands.go) — implement new methods using telebot's `c.Chat().ID`, `c.Chat().Type`, `c.Chat().Title`, and `c.Reply(msg)`.

3. **`TextHandlerFunc` type** (internal/bot/bot.go) — extend or add a new type:
   - Current: `func(senderID int64, text string, sendReply func(string) error) error`
   - The group handler in `cmd/brain/main.go` will use the raw telebot `c` context directly (register a separate `tele.OnText` handler that checks chat type before routing to existing private chat handler)

**Modified files:**
- `internal/bot/commands.go` — Extend BotContext interface + teleBotContext adapter
- `internal/bot/bot.go` — Add RegisterGroupTextHandler or modify OnText registration
- `cmd/brain/main.go` — Register group text handler, add scheduler job, extend /join for group context

**Handler routing logic (in main.go's telebot OnText handler):**
```
if c.Chat().Type == "group" || c.Chat().Type == "supergroup":
    if bot is @mentioned in c.Text():
        lookup group_chats by (platform, platform_chat_id)
        handle group @reply using c.Reply()
    else:
        ignore (don't process every group message)
    return
// ... existing private chat handling (unchanged)
```

**`/join` command in group context:**
The existing `HandleJoin` in commands.go handles individual employee linking. With the new `ChatType()` method on `BotContext`, it can branch:
```
if c.ChatType() == "group" || c.ChatType() == "supergroup":
    // Group registration: create group_chats record
    // Use c.ChatID() for platform_chat_id, c.ChatTitle() for name
    // Lookup tenant by invite_code to get tenant_id
else:
    // Existing employee join flow (unchanged)
```

The `CommandQuerier` interface needs new methods: `CreateGroupChat`, `GetGroupChatByPlatformID`.

**Group detection:**
- Telegram group chat IDs are negative numbers
- `c.Chat().Type` is "group" or "supergroup"

**Sending autonomous messages to groups:**
For autonomous posting, use `tgBot.Send(&tele.Chat{ID: platformChatID}, message)` to send messages to a group by its chat ID. This works because Telegram group IDs are just negative integers that can be used as chat destinations.

### Scheduler Addition

New gocron job: `groupMentorFn` at 9:00 AM daily
- Iterates all active group_chats
- Executes the AI decision flow per group
- Uses existing `schedulerCallbacks` pattern

## Files Changed

| File | Change |
|------|--------|
| `sql/migrations/000009_group_chats.up.sql` | New table + index |
| `sql/migrations/000009_group_chats.down.sql` | Drop table |
| `cmd/brain/main.go` | Add migration009 block in runMigrations, register group handler, add scheduler job, extend /join |
| `sql/queries/group_chats.sql` | CRUD queries |
| `internal/db/sqlc/` | Generated code (sqlc generate) |
| `internal/brain/group.go` | New: group prompt building + AI decision logic |
| `internal/bot/commands.go` | Extend BotContext interface + teleBotContext (ChatID, ChatType, ChatTitle, Reply) |
| `internal/bot/bot.go` | Group text handler registration |
| `internal/api/group_handlers.go` | New: admin group CRUD endpoints |
| `internal/api/routes.go` | Register group routes |
| `frontend/src/views/admin/GroupChatsView.vue` | New admin page |
| `frontend/src/router/index.ts` | Add route |
| `frontend/src/App.vue` | Add nav item |
| `frontend/src/composables/api.ts` | Add API functions |

## Out of Scope

- Group chat history storage (no need for v1)
- Individual memory recall in group context (privacy risk)
- Event-driven posting (future enhancement, upgrade to approach C)
- Slack/Lark group support (Telegram first, same architecture extends later)
- Group member management (Telegram API handles this)

## Testing

- Unit tests for group prompt building
- Unit tests for AI decision parsing (SKIP vs content)
- Integration test for group registration via /join
- API handler tests for group CRUD
- Manual E2E: add bot to test group, @mention, verify reply
