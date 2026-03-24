# AI C-Suite: Virtual Management Team Design

## Goal

Transform the system from "pick a mentor style" to "assemble your AI management team." Users build a virtual C-suite (CEO, CFO, CMO, etc.) where each seat is assigned a specific expert persona from a unified persona library. All seats share company-level memory. A board discussion mechanism allows multi-role deliberation on strategic topics.

## Architecture

The C-suite is a **strategic layer** that sits above the existing execution layer (AI roles from OrgEngine). The existing system remains untouched — seats are additive.

```
Strategic Layer: Board + C-suite (CEO, CFO, CMO...)    ← NEW
   ↓ guides decisions
Execution Layer: Existing AI roles (Chief of Staff...)  ← UNCHANGED
```

**Key decisions:**
- Seat-based architecture (Approach A) — seats as first-class entities
- Unified persona library — existing mentors + new domain experts, tagged by specialty
- Company-level shared memory — all seats share tenant memory pool, each retrieves by scope
- Telegram role switching — `/talk cmo` to switch, independent chat history per seat
- Sequential board discussion — each seat responds with accumulative context

## Tech Stack

- Go (Gin + sqlc + pgx) — new `internal/seats/` package
- PostgreSQL — new `seats` table
- Redis — seat chat history + active seat tracking
- Vue3 + TypeScript — new `/seats` management page
- Claude API — LLM calls for seat chat + board discussion

---

## Section 1: Unified Persona Library

### Current State

14 mentors defined in `configs/mentors/*.yaml`, each with MentorConfig (philosophy, strategy, system_prompt). Loaded by `internal/brain/mentor.go`.

### Changes

Add fields to MentorConfig (Go struct + YAML):

```yaml
# configs/mentors/trout.yaml
id: trout
name: 杰克·特劳特
name_en: Jack Trout
company: Trout & Partners
philosophy: "定位理论 — 占据用户心智中的独特位置"
version: 1

# New fields
domain: marketing
tags: [marketing, strategy, positioning, branding]
recommended_seats: [cmo]

strategy:
  system_prompt: |
    You are Jack Trout, the father of Positioning theory...
  checkin_questions: []
  summary:
    focus_areas: [市场定位, 品牌差异化, 竞争分析]
```

### New Personas (v1: 2 experts)

**Jack Trout** (`trout`):
- Domain: `marketing`
- Tags: `[marketing, strategy, positioning, branding]`
- Recommended seats: `[cmo]`
- Philosophy: Positioning theory — own a unique position in the customer's mind
- System prompt: Trout's analytical framework for market positioning, differentiation, category creation

**Erin Meyer** (`meyer`):
- Domain: `cross_cultural`
- Tags: `[cross_cultural, communication, leadership, international]`
- Recommended seats: `[chro, coo]`
- Philosophy: The Culture Map — navigate cultural differences in business
- System prompt: Meyer's 8-scale cultural framework (communicating, evaluating, persuading, leading, deciding, trusting, disagreeing, scheduling)

### Existing Mentors

All 14 existing mentors get `domain: general_management` and appropriate tags added. Their existing functionality is unchanged.

### Domain Types

`general_management`, `marketing`, `cross_cultural`, `finance`, `technology`, `hr` (extensible — just a string field).

### Go Changes

In `internal/brain/mentor.go`, add to MentorConfig:

```go
type MentorConfig struct {
    // ... existing fields ...
    Domain           string   `yaml:"domain"`
    Tags             []string `yaml:"tags"`
    RecommendedSeats []string `yaml:"recommended_seats"`
}
```

Update `ValidMentors` map in `internal/brain/engine.go` and `mentorDescriptions` map in `internal/bot/commands.go` to include `trout` and `meyer`.

New API endpoint to list mentors with domain info:
- `GET /api/v1/mentors` — extends existing `/admin/mentors` endpoint to include domain/tags/recommended_seats fields (or add a separate public endpoint)

---

## Section 2: Seat System

### Database

Migration `000010_seats.up.sql`:

```sql
CREATE TABLE seats (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    seat_type   VARCHAR(50) NOT NULL,
    title       VARCHAR(100) NOT NULL,
    persona_id  VARCHAR(50) NOT NULL,
    scope       TEXT NOT NULL,
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, seat_type)
);

CREATE INDEX idx_seats_tenant ON seats(tenant_id);
```

Down migration `000010_seats.down.sql`:
```sql
DROP TABLE IF EXISTS seats;
```

Note: `seat_type` is a free-form string (e.g., `"ceo"`, `"cmo"`, `"head_of_design"`). The predefined types below are defaults for the UI, not enforced constraints. Each tenant can have at most one seat per `seat_type` value.

### Predefined Seat Types

| seat_type | Default title | Default scope |
|-----------|--------------|---------------|
| `ceo` | Chief Executive Officer | Strategic direction, overall operations, team culture |
| `cfo` | Chief Financial Officer | Financial health, budgeting, cost control |
| `cmo` | Chief Marketing Officer | Market positioning, branding, customer acquisition |
| `cto` | Chief Technology Officer | Tech strategy, architecture, R&D efficiency |
| `chro` | Chief Human Resources Officer | Talent, culture, organizational development |
| `coo` | Chief Operations Officer | Process optimization, execution efficiency, daily ops |

Users can also create seats with custom `seat_type` strings (e.g., `"head_of_design"`, `"vp_sales"`). The `seat_type` is a free-form identifier, not limited to the predefined list. The UNIQUE constraint ensures one seat per type per tenant.

### Go Package: `internal/seats/`

```go
// internal/seats/service.go
type Seat struct {
    ID        string
    TenantID  string
    SeatType  string
    Title     string
    PersonaID string
    Scope     string
    IsActive  bool
}

type SeatService struct {
    db            *sqlc.Queries
    engineFactory *brain.EngineFactory
    memory        *memory.MemoryEngine
    llm           brain.ChatLLMClient  // ChatLLMClient (not LLMClient) — needed for ChatWithHistory in seat chat; also implements Chat() for board discussion
    chatRedis     *brain.ChatRedisClient
}

func (s *SeatService) Chat(ctx context.Context, tenantID, seatType, userMessage string) (string, error)
func (s *SeatService) BoardDiscuss(ctx context.Context, tenantID, topic string) ([]BoardResponse, string, error)
func (s *SeatService) ListSeats(ctx context.Context, tenantID string) ([]Seat, error)
func (s *SeatService) AssignSeat(ctx context.Context, tenantID, seatType, personaID string) (*Seat, error)
```

### Chat Flow

1. Look up seat by (tenant_id, seat_type) → get persona_id + scope
2. `EngineFactory.ForTenant(persona_id, culture)` → get Engine (culture resolved from tenant's culture_code, defaulting to `"default"`)
3. Retrieve memories: `memory.RecallForMentor(ctx, tenantID, "", scope + " " + userMessage)` — empty employeeID for company-level recall
4. Build system prompt with seat scope + recalled memories
5. `ChatWithHistory` with Redis key `seat:{tenantID}:{seatType}` — independent history per seat
6. Return reply

**Persona validation**: `AssignSeat` must validate `persona_id` against `ValidMentors` map before DB insertion. Return error if invalid.

### API Endpoints

- `GET /api/v1/seats` — list tenant's seats
- `POST /api/v1/seats` — create/assign seat `{ seat_type, persona_id, title?, scope? }`
- `PUT /api/v1/seats/:id` — update persona, scope, title
- `DELETE /api/v1/seats/:id` — remove seat

### sqlc Queries

File: `sql/queries/seats.sql`

```sql
-- name: ListSeatsByTenant :many
SELECT * FROM seats WHERE tenant_id = $1 ORDER BY seat_type;

-- name: ListActiveSeatsByTenant :many
SELECT * FROM seats WHERE tenant_id = $1 AND is_active = true ORDER BY seat_type;

-- name: GetSeatByType :one
SELECT * FROM seats WHERE tenant_id = $1 AND seat_type = $2;

-- name: GetSeatByID :one
SELECT * FROM seats WHERE id = $1;

-- name: CreateSeat :one
INSERT INTO seats (tenant_id, seat_type, title, persona_id, scope)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateSeat :one
UPDATE seats SET title = $2, persona_id = $3, scope = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteSeat :exec
DELETE FROM seats WHERE id = $1;
```

---

## Section 3: Board Discussion Mechanism

### Trigger

Telegram command: `/board <topic>`

### Flow

```
Boss: /board Should we enter the Southeast Asian market?

Bot: 📋 Board discussion started. Topic: "Should we enter the Southeast Asian market?"

[CEO - Inamori style]
From a strategic perspective, the Southeast Asian market aligns with...

[CFO - Financial perspective]
Considering current cash flow, I recommend a phased entry...

[CMO - Jack Trout positioning perspective]
In the Southeast Asian market, we need to find differentiated positioning...

[CHRO - Erin Meyer cross-cultural perspective]
Southeast Asian countries have significant cultural differences...

📊 Synthesis:
Based on all perspectives, we recommend starting with Philippines as pilot...
```

### Implementation

```go
type BoardResponse struct {
    SeatType  string
    Title     string
    PersonaID string
    Content   string
}

func (s *SeatService) BoardDiscuss(ctx context.Context, tenantID, topic string) ([]BoardResponse, string, error) {
    // 1. Get all active seats
    tenantUUID := parseUUID(tenantID)
    seats, err := s.db.ListActiveSeatsByTenant(ctx, tenantUUID)
    if err != nil {
        return nil, "", fmt.Errorf("list seats: %w", err)
    }

    // 2. Retrieve company-level memories (empty employeeID = company-level)
    memories, _ := s.memory.RecallForMentor(ctx, tenantID, "", topic)
    // Note: memory recall failure is non-fatal — proceed without memories

    // 3. Resolve tenant culture (default to "default")
    culture := resolveTenantCulture(ctx, s.db, tenantID) // helper that queries tenant, returns culture_code or "default"

    // 4. Sequential calls — each seat sees prior responses
    var responses []BoardResponse
    var priorContext strings.Builder

    for _, seat := range seats {
        engine, err := s.engineFactory.ForTenant(seat.PersonaID, culture)
        if err != nil {
            // Skip seat if persona/engine fails, add "[unavailable]" placeholder
            responses = append(responses, BoardResponse{SeatType: seat.SeatType, Title: seat.Title, PersonaID: seat.PersonaID, Content: "[unavailable]"})
            continue
        }
        prompt := buildBoardPrompt(engine, seat, topic, memories, priorContext.String())
        reply, err := s.llm.Chat(ctx, prompt, topic)
        if err != nil {
            responses = append(responses, BoardResponse{SeatType: seat.SeatType, Title: seat.Title, PersonaID: seat.PersonaID, Content: "[unavailable]"})
            continue
        }

        responses = append(responses, BoardResponse{
            SeatType:  seat.SeatType,
            Title:     seat.Title,
            PersonaID: seat.PersonaID,
            Content:   reply,
        })
        priorContext.WriteString(fmt.Sprintf("[%s - %s]: %s\n\n", seat.Title, seat.PersonaID, reply))
    }

    // 5. Synthesis (one additional LLM call)
    synthesis, err := s.llm.Chat(ctx, buildSynthesisPrompt(topic, responses), "")
    if err != nil {
        synthesis = "Synthesis unavailable."
    }

    return responses, synthesis, nil
}
```

### Prompt Strategy

Each seat's system prompt includes:
- Persona's philosophy + system prompt (from YAML)
- Seat scope (responsibilities)
- Company memories (filtered by relevance to scope + topic)
- Prior members' responses (accumulative context)
- Instruction: "Analyze from your professional perspective. You may agree or disagree with previous speakers."

Synthesis prompt: "You are the board secretary. Synthesize all executives' opinions into a balanced decision recommendation with clear action items."

### Constraints

- N seats = N+1 LLM calls per discussion (including synthesis)
- Recommend max 6 active seats
- Per-seat response: max_tokens = 500; synthesis: max_tokens = 800
- Discussion records stored in Redis with 24h TTL
- Rate limit: 1 board discussion per 5 minutes per tenant (Redis key: `board_rate:{tenantID}`, 5-min TTL)

### API

- `POST /api/v1/board/discuss` — body: `{ "topic": "..." }` — returns array of BoardResponse + synthesis

---

## Section 4: Telegram Integration

### New Commands

| Command | Function | Example |
|---------|----------|---------|
| `/talk <seat>` | Switch conversation target | `/talk cmo` |
| `/talk off` | Return to default mode | `/talk off` |
| `/board <topic>` | Start board discussion | `/board Enter SEA market?` |
| `/team` | View current C-suite roster | Shows all seats + personas |
| `/assign <seat> <persona>` | Assign persona to seat | `/assign cmo trout` |

### Active Seat Tracking

Redis key: `active_seat:{tenantID}:{telegramUserID}` → value: `"cmo"` (or empty)

### Text Handler Logic Change

In `cmd/brain/main.go`'s `RegisterRawTextHandler`, insert the active seat check **after** group chat handling and **after** boss identification, but **before** the existing boss chat / employee report collection logic:

```
Incoming private message:
  1. [existing] Group chat handling (unchanged)
  2. [existing] Boss identification (unchanged)
  3. [NEW] If boss: check active_seat Redis key
     → If active seat exists: route to SeatService.Chat(ctx, tenantID, seatType, message), return
  4. [existing] Boss chat / employee report collection (unchanged)
```

This ensures the report collection state machine is not broken — active seat mode takes priority over the default boss chat, but only when explicitly activated via `/talk`.

### Compatibility

- Existing boss chat (default mode, no `/talk`) — unchanged
- Employee report collection — unchanged
- Group chat @mention — unchanged
- The `/talk` command is boss-only (checked via existing boss identification)

---

## Section 5: Frontend Management Page

### New Page: `/seats` — C-Suite Team

**Navigation**: Add to navItems in App.vue:
```js
{ path: '/seats', label: 'C-Suite', icon: '👔' }
```

### Features

- **Card layout**: Each seat displayed as a card showing seat_type, title, persona name, domain badge, status
- **Assign persona**: Select from unified persona library with domain tag filtering
- **Add/remove seats**: Modal for creating new seats, confirm for removal
- **Board discussion**: Input topic → display discussion results (read-only, formatted)

### API Calls

- `GET /api/v1/seats` → card list
- `POST /api/v1/seats` → create seat modal
- `PUT /api/v1/seats/:id` → edit persona/scope
- `DELETE /api/v1/seats/:id` → remove
- `GET /api/v1/mentors` → persona library (with domain/tags for filtering)
- `POST /api/v1/board/discuss` → board discussion

### Router

New route in `frontend/src/router/index.ts`:
```js
{ path: '/seats', name: 'Seats', component: () => import('../views/SeatsView.vue'), meta: { requiresAuth: true } }
```

---

## Scope Summary

### In Scope (v1)

1. Unified persona library — add domain/tags to MentorConfig, 2 new expert personas (Trout, Meyer)
2. Seats table + CRUD API + SeatService
3. Seat chat via Telegram (`/talk <seat>`) with independent history + shared memory
4. Board discussion mechanism (`/board <topic>`) with sequential multi-role deliberation
5. Frontend seats management page
6. Telegram commands: `/talk`, `/board`, `/team`, `/assign`

### Out of Scope (future iterations)

- Autonomous seat behavior (proactive suggestions, scheduled analysis)
- Agent network architecture (EventBus-driven seats)
- Multi-round board debates (v1 is single-round)
- Web-based chat interface for seats
- Seat-to-seat communication (e.g., CMO asks CFO about budget)
- More expert personas beyond Trout + Meyer

---

## Data Flow

```
User: /talk cmo
  → Redis: set active_seat:{tenant}:{user} = "cmo"
  → Bot: "Switched to CMO (Jack Trout)"

User: "How should we position our product?"
  → Check Redis: active_seat = "cmo"
  → SeatService.Chat(tenant, "cmo", message)
    → DB: GetSeatByType(tenant, "cmo") → persona_id="trout", scope="市场定位..."
    → EngineFactory.ForTenant("trout", culture) → Engine
    → Memory.RecallForMentor(tenant, "", scope + message) → relevant memories
    → BuildSeatPrompt(engine, seat, memories, message)
    → ChatWithHistory(Redis key "seat:{tenant}:cmo") → reply
  → Bot sends reply

User: /board "Enter Southeast Asian market?"
  → SeatService.BoardDiscuss(tenant, topic)
    → ListActiveSeatsByTenant → [CEO, CFO, CMO, CHRO]
    → Memory.RecallForMentor(tenant, "", topic) → company memories
    → For each seat (sequential):
      → Build prompt (persona + scope + memories + prior responses)
      → LLM.Chat → seat response
    → Build synthesis prompt → LLM.Chat → synthesis
  → Bot sends formatted discussion (each seat's response + synthesis)
```

## Error Handling

- Seat not found: "No CMO assigned yet. Use /assign cmo <persona> to set one up."
- Persona not found: "Unknown persona 'xyz'. Use /team to see available options."
- LLM failure during board discussion: Skip that seat with "[CMO unavailable]", continue with remaining seats
- Rate limit: "Board discussions limited to once per 5 minutes."

## Testing Strategy

- Unit tests: SeatService.Chat, SeatService.BoardDiscuss with mocked LLM + DB
- Integration: Telegram command routing (/talk, /board, /team, /assign)
- E2E: Full board discussion flow with real LLM (manual verification)
