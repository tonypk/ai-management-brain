# Chat-Native Deep Onboarding Design

## Overview

Transform Boss AI Agent's onboarding from a minimal `/start` + `/addemployee` flow into an LLM-driven consultative dialogue. The AI acts as a management consultant, collecting deep context about the company, industry, and projects through natural conversation, then generates a complete management plan (mentor, board, policies, schedules) for the boss to confirm step by step.

After onboarding, the mentor joins team groups as a semi-autonomous management assistant — running daily cycles, tracking projects, sharing insights, and responding to @mentions.

## Goals

1. **Deep onboarding**: AI-guided dialogue collects industry, company stage, business model, team size, current projects, management pain points, culture preferences, and communication tools
2. **AI-generated management plan**: mentor selection, board configuration, management policies, and daily schedules — all tailored to the collected context
3. **Step-by-step confirmation**: boss reviews and adjusts each part before activation
4. **Enhanced in-group mentor**: semi-autonomous + responsive behavior after joining team groups
5. **Dual-track**: chat experience and OpenClaw skill coexist, sharing the same backend

## Core Management Framework

Boss AI Agent's capabilities are built on the four pillars of management (Plan → Organize → Lead → Control), extended with people management, decision support, communication, and strategy:

| Pillar | Sub-capabilities | How Boss AI Agent Implements It |
|--------|-----------------|-------------------------------|
| **Planning** | Goal setting (OKR/KPI/Scrum/MBO/BSC), strategy, resource allocation | AI recommends framework during onboarding; mentor tracks goals in daily cycles |
| **Organizing** | Org structure design, roles & responsibilities, SOPs, process design | AI designs org tree during onboarding; `org_units` tracks structure; mentor suggests reorgs |
| **Leading** | Team execution, communication, motivation, culture management | Mentor persona drives all interactions; culture packs adapt per-employee; check-ins motivate |
| **Controlling** | Data monitoring, deviation correction, retrospectives | Signal scanning, risk alerts, weekly/monthly reports, sentiment tracking |
| **People Mgmt** | Hiring, training, performance, incentives | Employee profiles, 1:1 prep, performance reports, Maslow/dual-factor awareness in mentor prompts |
| **Decision Making** | Data-driven decisions, risk assessment, rapid decisions | C-Suite board discussions, data-backed recommendations, structured decision frameworks |
| **Communication** | Upward (boss), downward (employees), cross-team | Multi-channel messaging, group management, mentor mediates communication |
| **Strategy** | Positioning, business model, competitive advantage | Onboarding collects strategic context; C-Suite board analyzes strategy; industry insights |

The onboarding dialogue collects enough context to activate ALL pillars. The AI mentor then operationalizes them through daily cycles, proactive suggestions, and on-demand analysis.

## Priority

Phase 1 (this spec): Onboarding flow + org structure. Group mentor enhancements are designed here but implemented later.

---

## 1. Onboarding Dialogue Architecture

### Trigger

Boss sends first message (or `/start`) to the bot via any channel (Telegram/Lark/Slack). If the tenant has no completed onboarding (`onboarding_sessions.status != 'active'` or no session exists), the bot enters guided dialogue mode.

### `/start` Interception

The current `HandleStart` in `internal/bot/commands.go` auto-creates a tenant and returns a fixed welcome message. This must be changed:

1. `CommandHandler` receives `OnboardingService` as an injected dependency
2. `HandleStart` still auto-creates the tenant (needed for FK references), but instead of returning a welcome message, it calls `OnboardingService.HandleMessage(ctx, tenantID, channelType, userID, "/start")` to begin the onboarding dialogue
3. The auto-created tenant gets `onboarding_completed_at = NULL` to indicate onboarding is pending
4. All subsequent boss messages are routed to `OnboardingService` until onboarding completes

### State Machine

```
(no session) → onboarding → configuring → confirming → active
```

| State | Description |
|-------|-------------|
| (no session) | No `onboarding_sessions` row — first `/start` creates one |
| `onboarding` | AI-guided dialogue collecting information (multi-turn) |
| `configuring` | Collection complete, AI generating management plan |
| `confirming` | Presenting plan to boss step by step (3 steps) |
| `active` | All confirmed, mentor begins working |

### Dialogue Storage

| Store | Key/Table | TTL | Purpose |
|-------|-----------|-----|---------|
| Redis | `onboarding:chat:{tenantID}` | 7 days | Conversation history (LLM context) |
| PostgreSQL | `onboarding_sessions.collected_data` | Permanent | Extracted structured info (source of truth) |
| PostgreSQL | `onboarding_sessions` | Permanent | Session state, collected data, proposed plan |

7-day TTL allows the boss to complete onboarding across multiple sessions.

**Redis cache miss handling**: If `onboarding:chat:{tenantID}` expires (boss returns after 7+ days), the system rebuilds a summary context from `onboarding_sessions.collected_data` and continues. The boss does not need to start over — AI acknowledges the gap: "Welcome back! Based on our previous conversation, here's what I know so far: [summary]. Let me continue from where we left off."

### Turn Limits

- AI proactively wraps up around turn 15
- Hard cap at 20 turns — forces transition to summary
- `onboarding_sessions.message_count` tracks progress
- **Incomplete required fields at turn 20**: AI summarizes what it has, explicitly lists missing fields, and asks the boss to fill them in directly (e.g., "I still need to know your industry. Could you tell me?"). The system stays in `onboarding` state with a 5-turn grace period.

### Concurrency Control

A Redis processing lock prevents concurrent LLM calls from racing:
- Key: `onboarding:lock:{tenantID}`, TTL: 60s
- Before processing, acquire lock with `SET NX EX 60`
- If lock exists, reply: "I'm still thinking about your last message, one moment..."
- Release lock in a deferred call (`defer redis.Del(ctx, lockKey)`) so it executes regardless of panics or early returns

### LLM System Prompt Design

AI plays a "management consultant" role. The prompt includes:

- Role definition: experienced management consultant conducting an initial assessment
- Collection targets with required/optional flags (see table below)
- Already-collected information (injected dynamically each turn from `onboarding_sessions.collected_data`)
- Missing required items (so AI knows what to steer toward)
- Instructions to be conversational, not interrogative — follow up on interesting points, ask one thing at a time

### Information to Collect

| Category | Details | Required |
|----------|---------|----------|
| Industry | Specific industry / sub-sector | Yes |
| Company stage | Startup / growth / mature / transformation | Yes |
| Business model | B2B / B2C / SaaS / platform / services / etc. | Yes |
| Team size | Headcount + rough structure | Yes |
| Organizational structure | Departments, teams, reporting lines, key roles — AI decides depth based on company size/stage | Yes |
| Current projects | What they're building, goals, timelines | Yes |
| Management pain points | Top 1-3 management challenges | Yes |
| Communication tools | Which tools, how many groups | Yes |
| Culture preferences | Team country/region distribution, communication style | No |
| Goal management | Existing framework or need recommendation | No |

### Information Extraction

After each dialogue turn, a lightweight LLM call (Haiku-class) extracts newly revealed structured information and updates `onboarding_sessions.collected_data` in PostgreSQL (source of truth). Redis `onboarding:extracted:{tenantID}` is removed — the database is the single source.

When all required fields are covered, the main dialogue LLM is prompted to wrap up: "I think I have a good picture now. Let me put together a management plan for you."

---

## 2. Management Plan Generation & Step-by-Step Confirmation

### Plan Generation

When collection completes, AI transitions to `configuring` state. A single LLM call with all collected context generates a complete management plan as structured JSON, stored in `onboarding_sessions.proposed_plan`.

**Plan validation**: The LLM output is unmarshaled into a typed `ProposedPlan` Go struct and validated before storage. If the JSON is malformed, the system retries up to 2 times with clarifying instructions. If all retries fail, the system tells the boss: "I'm having trouble putting together the plan. Let me try again." and resets to `configuring` state.

```go
type ProposedPlan struct {
    Mentor      MentorPlan       `json:"mentor"`
    Board       []SeatPlan       `json:"board"`
    OrgDesign   OrgDesignPlan    `json:"org_design"`
    Policies    PolicyPlan       `json:"policies"`
    Schedule    SchedulePlan     `json:"schedule"`
    Reasoning   string           `json:"reasoning"`
}

type MentorPlan struct {
    PrimaryID   string  `json:"primary_id"`
    SecondaryID string  `json:"secondary_id,omitempty"`
    BlendWeight float64 `json:"blend_weight,omitempty"`
    Reasoning   string  `json:"reasoning"`
}

type SeatPlan struct {
    SeatType  string `json:"seat_type"`
    PersonaID string `json:"persona_id"`
    Reasoning string `json:"reasoning"`
}

type PolicyPlan struct {
    Framework        string   `json:"framework"`        // okr/kpi/scrum/mbo/bsc
    CheckinQuestions []string `json:"checkin_questions"`
    TrackingFocus    []string `json:"tracking_focus"`
    RiskRules        RiskRules `json:"risk_rules"`
    Cadence          Cadence   `json:"cadence"`
    Reasoning        string   `json:"reasoning"`
}

type RiskRules struct {
    ConsecutiveMisses      int     `json:"consecutive_misses"`       // alert after N missed check-ins
    SentimentDropThreshold float64 `json:"sentiment_drop_threshold"` // alert on sentiment drop > threshold
    UrgentKeywords         []string `json:"urgent_keywords"`         // keywords triggering immediate alert
}

type Cadence struct {
    DailyActions   []string `json:"daily_actions"`    // e.g., ["checkin", "chase", "summary"]
    WeeklyActions  []string `json:"weekly_actions"`   // e.g., ["review", "1on1_prep"]
    WeeklyDay      string   `json:"weekly_day"`       // e.g., "friday"
    MonthlyActions []string `json:"monthly_actions"`  // e.g., ["performance_review", "okr_check"]
    MonthlyDay     int      `json:"monthly_day"`      // e.g., 1 (1st of month)
}

type SchedulePlan struct {
    Checkin    string `json:"checkin"`     // cron expression, e.g., "0 9 * * 1-5"
    Chase      string `json:"chase"`       // cron expression
    Summary    string `json:"summary"`     // cron expression
    Briefing   string `json:"briefing"`    // cron expression
    SignalScan string `json:"signal_scan"` // cron expression
    Timezone   string `json:"timezone"`    // e.g., "Asia/Manila"
}

// OrgDesignPlan — AI-designed organizational structure
// The AI decides depth and shape based on company size, stage, and industry.
type OrgDesignPlan struct {
    Units     []OrgUnitPlan `json:"units"`      // flat list with parent references forming a tree
    Reasoning string        `json:"reasoning"`  // why this structure
}

type OrgUnitPlan struct {
    RefID        string `json:"ref_id"`         // temporary ID for parent references, e.g., "eng", "eng-frontend"
    ParentRefID  string `json:"parent_ref_id"`  // empty = top-level (reports to boss)
    Name         string `json:"name"`           // e.g., "Engineering", "Frontend Team"
    UnitType     string `json:"unit_type"`      // "department" / "team" / "squad" / "division" — AI decides
    HeadRole     string `json:"head_role"`      // e.g., "VP Engineering", "Team Lead"
    Responsibilities string `json:"responsibilities"`
}
```

### Four Confirmation Steps

**Step 1: Mentor + Board Configuration**

AI recommends based on industry, stage, and pain points:

- Primary mentor + reasoning (e.g., "Your SaaS startup suits Musk's first-principles + rapid iteration style")
- Optional: secondary mentor blend (e.g., Musk 70% + Inamori 30%)
- 6 C-Suite seat persona assignments (e.g., CTO seat uses Grove's OKR-driven style)

Boss can: accept / change mentor / adjust blend ratio / modify seat assignments

**Step 2: Organizational Structure**

AI designs the org structure based on company size, stage, industry, and team info collected during onboarding. The AI decides the depth (flat for 5-person startup, multi-level for 100-person company) and unit types (departments, teams, squads, etc.):

- Visual tree of organizational units with reporting lines
- Each unit: name, type, head role, responsibilities
- Reasoning for the structure design (e.g., "At your stage, a flat structure with 3 functional teams is optimal for speed")

Boss can: accept / add/remove/rename units / change reporting lines / adjust head roles

Example output for a 20-person SaaS startup:
```
CEO (You)
├── Engineering (VP Eng)
│   ├── Frontend Team (Team Lead)
│   └── Backend Team (Team Lead)
├── Product (Product Manager)
├── Marketing (Marketing Lead)
└── Operations (Ops Manager)
```

After confirmation, org units are written to `org_units` table. Employees are assigned to units later (via `/addemployee` or group setup).

**Step 3: Management Policies**

Generated from industry + stage + pain points + org structure:

- Goal management framework recommendation (OKR / KPI / Scrum / MBO / BSC) + reasoning
- Customized check-in questions (based on mentor + industry, not generic templates)
- Per-unit tracking focus (e.g., Engineering: delivery velocity + code quality; Marketing: campaign ROI + lead gen)
- Risk alert rules (consecutive missed check-ins threshold, sentiment drop threshold)
- Management cadence (what happens daily / weekly / monthly)

Boss can: confirm or modify each item

**Step 4: Daily Plan + Group Setup**

- Scheduled task timetable (check-in, chase, summary, briefing, signal scan times)
- Group setup instructions: boss adds bot to team group → bot auto-detects → mentor sends self-introduction in mentor persona
- Group-to-unit mapping: which chat group corresponds to which org unit
- Employee onboarding: bot generates invite link in group, or boss uses `/addemployee`

### Data Writes Per Step

All writes within a step use a single database transaction.

| Step | Database writes (single transaction) |
|------|--------------------------------------|
| Step 1 confirmed | `tenants.mentor_id`, `tenants.mentor_blend`, `seats` table, `onboarding_sessions.confirm_step = 1` |
| Step 2 confirmed | `org_units` table (bulk insert all units), `onboarding_sessions.confirm_step = 2` |
| Step 3 confirmed | `organizations.management_plan` (JSONB — full policy plan), `onboarding_sessions.confirm_step = 3` |
| Step 4 confirmed | `tenants.onboarding_completed_at = now()`, `onboarding_sessions.status = 'active'`, `onboarding_sessions.confirm_step = 4` |

**Scheduler registration (Step 3)**: The existing scheduler jobs (remind, chase, summary, etc.) already run globally at startup via gocron. They read per-tenant config from the database at execution time. No dynamic per-tenant job registration is needed. Step 3 writes the tenant's schedule config to `organizations.management_plan.schedule`, and the existing jobs pick it up on their next run. This is consistent with the current architecture.

**Scheduler guard**: All scheduler callbacks must skip tenants with `onboarding_completed_at = NULL` to avoid sending check-ins to tenants still in onboarding (no employees yet, mentor not confirmed). Add an early return check: `if tenant.OnboardingCompletedAt == nil { return }`.

### Post-Onboarding Modifications

Boss can modify settings anytime via chat:

- "I want to switch mentors" → single-item modification flow
- "Change check-in time to 10am" → update schedule
- "Re-do onboarding" → reset: clear `onboarding_sessions` (status, collected_data, proposed_plan, message_count, confirm_step all reset), delete Redis chat history, keep tenant and organizations rows intact

No need to re-run full onboarding. AI recognizes intent and walks through the specific change.

### Relationship with OrgWizard

The existing `OrgWizard` (API-only, used via frontend dashboard) is a separate entry point that also writes to `organizations.management_plan`. After this change:

- **OrgWizard**: remains available via API/dashboard for users who prefer the web interface
- **OnboardingService**: the chat-native path that achieves the same result via dialogue
- Both write to the same `organizations.management_plan` field — last write wins
- `wizard_sessions` table is deprecated and dropped in migration 000011 (see Migration Plan below). OrgWizard will use `onboarding_sessions` going forward, or be refactored separately if needed.

---

## 3. Data Model Changes

### Existing Table Extensions

**`organizations` table** (add fields, all nullable to support incremental collection):

| New Field | Type | Description |
|-----------|------|-------------|
| `management_pain_points` | TEXT[] | List of management pain points |
| `current_projects` | JSONB | Current project descriptions (name, goal, timeline) |
| `target_framework` | VARCHAR(50) | Goal management framework: okr/kpi/scrum/mbo/bsc |
| `team_structure` | JSONB | Team structure (departments, role distribution) |
| `communication_tools` | TEXT[] | Communication tools in use |
| `culture_preferences` | JSONB | Culture preferences (country distribution, comm style) |

Note: `onboarding_status` is NOT added to `organizations`. The `onboarding_sessions` table is the sole source of truth for onboarding state.

Note: Existing `organizations` fields (`industry`, `size`, `stage`) have `NOT NULL` constraints. Migration 000011 relaxes them to nullable with `ALTER COLUMN ... DROP NOT NULL` so the `organizations` row can be created at tenant creation time with empty values, then populated during onboarding.

**`tenants` table** (add fields):

| New Field | Type | Description |
|-----------|------|-------------|
| `onboarding_completed_at` | TIMESTAMPTZ | When onboarding completed, NULL = not done |
| `boss_slack_id` | TEXT | Boss's Slack user ID |
| `boss_lark_id` | TEXT | Boss's Lark user ID |

Note: Using `TEXT` instead of `VARCHAR(50)` for Slack/Lark IDs — Lark open IDs (`ou_xxx`) can exceed 50 chars in some regions. No performance difference in PostgreSQL.

### New Table: `org_units`

Flexible N-level organizational tree. AI decides the structure during onboarding.

```sql
CREATE TABLE org_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    parent_id UUID REFERENCES org_units(id),  -- NULL = top-level (reports to boss)
    name VARCHAR(200) NOT NULL,
    unit_type VARCHAR(50) NOT NULL,            -- department / team / squad / division / etc.
    head_role VARCHAR(200),                    -- e.g., "VP Engineering", "Team Lead"
    head_employee_id UUID REFERENCES employees(id),  -- NULL until employee assigned as head
    responsibilities TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_units_tenant ON org_units(tenant_id);
CREATE INDEX idx_org_units_parent ON org_units(parent_id) WHERE parent_id IS NOT NULL;
```

- `parent_id`: self-referencing FK forming the tree. NULL = reports directly to boss.
- `unit_type`: AI-chosen (department, team, squad, division, etc.). No fixed enum — the AI adapts to the company's vocabulary.
- `head_employee_id`: filled when an employee is assigned as head. Can be NULL (position exists but not filled).
- `sort_order`: for display ordering among siblings.

**Employee-to-unit assignment**: `employees` table gets a new `org_unit_id UUID REFERENCES org_units(id)` column. An employee belongs to one unit. Unit heads are tracked both via `org_units.head_employee_id` (for the unit) and `employees.org_unit_id` (for the employee).

### Org-Aware Mentor Intelligence

With organizational structure, the mentor gains these capabilities:

| Capability | Example |
|-----------|---------|
| **Targeted check-ins** | Different questions for Engineering vs Marketing |
| **Scoped tracking** | "Frontend team has 3 overdue PRs" instead of generic "team has overdue PRs" |
| **Delegation routing** | "Let the Backend Lead follow up on this API issue" |
| **Unit-level reports** | Weekly report broken down by department |
| **Structure suggestions** | "Your Engineering team has 12 people — consider splitting into Frontend and Backend squads" |
| **Head accountability** | Chase unit heads first, not individual members |
| **Span of control alerts** | "Marketing Lead manages 8 people — consider adding a sub-team" |
| **Reorg recommendations** | "Based on the new project, consider moving Alice from Backend to the new Data team" |

The mentor uses `org_units` context in all system prompts — for check-ins, chases, summaries, group messages, and 1:1 prep.

### New Table: `onboarding_sessions`

Replaces the deprecated `wizard_sessions` table.

```sql
CREATE TABLE onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    status VARCHAR(20) NOT NULL DEFAULT 'onboarding',
    confirm_step INT NOT NULL DEFAULT 0,
    collected_data JSONB NOT NULL DEFAULT '{}',
    proposed_plan JSONB NOT NULL DEFAULT '{}',
    message_count INT NOT NULL DEFAULT 0,
    channel_type VARCHAR(20) NOT NULL DEFAULT 'telegram',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_onboarding_sessions_tenant
    ON onboarding_sessions(tenant_id);
```

- `status`: onboarding / configuring / confirming / active
- `confirm_step`: 0 = not started, 1 = mentor+board confirmed, 2 = org structure confirmed, 3 = policies confirmed, 4 = plan confirmed
- `collected_data`: structured info extracted from dialogue (updated in real-time, source of truth)
- `proposed_plan`: AI-generated management plan (pending confirmation), typed as `ProposedPlan`
- `message_count`: dialogue turn counter
- `channel_type`: which channel the onboarding was initiated on (updated on each message if boss switches channels)
- `UNIQUE(tenant_id)`: one active onboarding per tenant (explicit index, following codebase convention)

### Redis Keys

| Key | Purpose | TTL |
|-----|---------|-----|
| `onboarding:chat:{tenantID}` | Dialogue history (LLM context) | 7 days |
| `onboarding:lock:{tenantID}` | Per-turn processing lock | 60s |

---

## 4. Dialogue Routing & Multi-Channel Unification

### Boss Identity Resolution

Current: only `tenants.boss_chat_id` (Telegram ID), checked in `main.go` with `if senderID == cfg.BossTelegramID`.

After: a new `ResolveBoss` function handles all channels:

```go
// internal/channel/boss_resolve.go (new file — separate from outbound resolve.go)
func ResolveBoss(ctx context.Context, db *sqlc.Queries, channelType channel.Type, userID string) (*sqlc.Tenant, error)
```

| Channel | Query |
|---------|-------|
| Telegram | `GetTenantByBossChatID(telegramID int64)` (existing) |
| Slack | `GetTenantByBossSlackID(slackID string)` (new) |
| Lark | `GetTenantByBossLarkID(larkID string)` (new) |

### Unified Routing Logic

**Telegram** (`main.go` raw text handler — modified):

```
Message arrives
  → if senderID == bossChatID:
      → tenant = GetTenantByBossChatID(senderID)
      → session = GetOnboardingSession(tenant.ID)
      → if session == nil OR session.Status != 'active':
          → response = OnboardingService.HandleMessage(ctx, tenant.ID, "telegram", senderID, text)
          → c.Send(response)
          → return
      → (existing routing: seat chat / mentor chat)
  → (existing employee routing)
```

**Non-Telegram** (`internal/channel/message_handler.go` — modified):

```go
func (h *UnifiedHandler) HandleMessage(ctx context.Context, msg Message) error {
    // 1. Try boss resolution FIRST
    tenant, err := channel.ResolveBoss(ctx, h.db, msg.ChannelType, msg.UserID)
    if err == nil && tenant != nil {
        session, _ := h.db.GetOnboardingSession(ctx, tenant.ID)
        if session == nil || session.Status != "active" {
            response, _ := h.onboarding.HandleMessage(ctx, tenant.ID, msg.ChannelType, msg.UserID, msg.Text)
            return h.router.Send(ctx, msg.ChannelType, msg.UserID, response)
        }
        // boss with completed onboarding — route to existing boss handlers
        return h.handleBossMessage(ctx, tenant, msg)
    }

    // 2. Fall through to employee resolution (existing logic)
    emp, err := h.resolveEmployee(ctx, msg.ChannelType, msg.UserID)
    ...
}
```

Key change: `UnifiedHandler` gets `OnboardingService` injected, and boss resolution is attempted BEFORE employee resolution. This ensures Slack/Lark boss messages are never silently dropped.

### OnboardingService (New Module)

```
internal/onboarding/
  service.go      -- Main logic: receive message → check state → dispatch
  prompt.go       -- System prompt construction (consultant role + targets + collected info)
  extractor.go    -- Extract structured info from dialogue (lightweight LLM call)
  planner.go      -- Generate management plan from collected info, validate ProposedPlan struct
  confirmer.go    -- Manage 3-step confirmation flow
```

**Key interface:**

```go
type OnboardingService struct {
    db        *sqlc.Queries
    redis     *redis.Client
    llm       LLMClient
    engine    *brain.EngineFactory
    mentors   *brain.MentorLoader
}

// HandleMessage processes a boss message during onboarding.
// channelType and userID are passed for context but not used for sending.
// Returns response text — caller sends via the appropriate channel.
func (s *OnboardingService) HandleMessage(
    ctx context.Context,
    tenantID uuid.UUID,
    channelType string,
    userID string,
    text string,
) (string, error)
```

OnboardingService returns text only. The caller (channel adapter or main.go) is responsible for sending via the correct channel. This keeps all onboarding logic channel-agnostic.

---

## 5. Enhanced In-Group Mentor Behavior

> **Note**: Designed here, implemented in a later phase.

### Responsive (Triggered)

| Trigger | Behavior |
|---------|----------|
| @mention question | Reply with mentor persona + team context + project context, multi-turn (Redis group chat history, 5 messages) |
| @mention "review" / "recap" | Pull weekly check-in data + project progress, output structured review |
| @mention "advice" / "suggest" | Combine management policies and current data for targeted advice |
| Boss private chat "send notice to group" | Mentor rewrites in their style and sends to specified group |

### Semi-Autonomous (Scheduled + Event-Driven)

| Behavior | Trigger | Content |
|----------|---------|---------|
| Morning briefing | Daily morning (configurable) | Today's priorities, reminders, yesterday's incomplete items — generated from management policies |
| Project progress tracking | Daily afternoon (configurable) | Check GitHub PRs / task boards, flag lagging items, remind owners |
| Weekly review | Friday afternoon | Weekly check-in rate, project progress, highlights, risk items |
| Industry insight | Monday morning | Industry/management insight based on industry + project context |
| Risk alert | Real-time (signal scan) | Consecutive missed check-ins / sentiment anomaly / urgent keywords → @boss in group |

### Constraints

- **Privacy**: never mention individual check-in content or sentiment scores in groups — team aggregate data only
- **Frequency cap**: max 3 autonomous messages per group per day (briefing + tracking + 1 extra) to prevent spam
- **Boss override**: boss can say "be quiet" or "skip today" and mentor complies
- **Persona consistency**: all autonomous messages carry mentor persona and management policy context

### Group Type Expansion

The existing `group_chats.group_type VARCHAR(50)` already supports free-text values. No schema change needed — this is application-level semantics:

| Type | Description | Mentor behavior difference |
|------|-------------|---------------------------|
| `general` | All-hands group | Morning briefings, weekly reviews, team-level info |
| `project` | Project group | Track that project's progress, PRs, deadlines |
| `management` | Leadership group | Deep analysis, personnel suggestions, sensitive topics |

Boss specifies group type during onboarding or later setup.

---

## 6. Migration Plan

### Migration: `000011_onboarding.up.sql`

```sql
-- 1. Drop deprecated wizard_sessions table
DROP TABLE IF EXISTS wizard_sessions;

-- 2. Create onboarding_sessions table
CREATE TABLE onboarding_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'onboarding',
    confirm_step    INT NOT NULL DEFAULT 0,
    collected_data  JSONB NOT NULL DEFAULT '{}',
    proposed_plan   JSONB NOT NULL DEFAULT '{}',
    message_count   INT NOT NULL DEFAULT 0,
    channel_type    VARCHAR(20) NOT NULL DEFAULT 'telegram',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_onboarding_sessions_tenant ON onboarding_sessions(tenant_id);

-- 3. Create org_units table (flexible N-level tree)
CREATE TABLE org_units (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    parent_id       UUID REFERENCES org_units(id),
    name            VARCHAR(200) NOT NULL,
    unit_type       VARCHAR(50) NOT NULL,
    head_role       VARCHAR(200),
    head_employee_id UUID REFERENCES employees(id),
    responsibilities TEXT,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_org_units_tenant ON org_units(tenant_id);
CREATE INDEX idx_org_units_parent ON org_units(parent_id) WHERE parent_id IS NOT NULL;

-- 4. Add org_unit_id to employees
ALTER TABLE employees ADD COLUMN IF NOT EXISTS org_unit_id UUID REFERENCES org_units(id);
CREATE INDEX IF NOT EXISTS idx_employees_org_unit ON employees(org_unit_id) WHERE org_unit_id IS NOT NULL;

-- 5. Extend organizations table (all new fields nullable)
ALTER TABLE organizations ALTER COLUMN industry DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN size DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN stage DROP NOT NULL;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS management_pain_points TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS current_projects JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS target_framework VARCHAR(50);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS team_structure JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS communication_tools TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS culture_preferences JSONB;

-- 6. Extend tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_slack_id TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_lark_id TEXT;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_slack ON tenants(boss_slack_id) WHERE boss_slack_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_lark ON tenants(boss_lark_id) WHERE boss_lark_id IS NOT NULL;
```

### Migration: `000011_onboarding.down.sql`

```sql
-- Note: wizard_sessions data is not recoverable after drop.
-- Recreate empty table for rollback compatibility.
CREATE TABLE IF NOT EXISTS wizard_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    mentor_id       TEXT NOT NULL,
    current_step    TEXT NOT NULL DEFAULT 'start',
    conversation    JSONB NOT NULL DEFAULT '[]',
    company_profile JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TABLE IF EXISTS onboarding_sessions;

ALTER TABLE employees DROP COLUMN IF EXISTS org_unit_id;
DROP TABLE IF EXISTS org_units;

ALTER TABLE organizations DROP COLUMN IF EXISTS management_pain_points;
ALTER TABLE organizations DROP COLUMN IF EXISTS current_projects;
ALTER TABLE organizations DROP COLUMN IF EXISTS target_framework;
ALTER TABLE organizations DROP COLUMN IF EXISTS team_structure;
ALTER TABLE organizations DROP COLUMN IF EXISTS communication_tools;
ALTER TABLE organizations DROP COLUMN IF EXISTS culture_preferences;
-- Note: NOT NULL constraints on industry/size/stage are not restored
-- to avoid breaking existing rows that may have NULL values.

ALTER TABLE tenants DROP COLUMN IF EXISTS onboarding_completed_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS boss_slack_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS boss_lark_id;
```

### Inline Migration Update

Migration 000011 must also be added to the `runMigrations()` function in `cmd/brain/main.go` (the codebase runs migrations from both file-based and inline sources).

### New sqlc Queries

- `CreateOnboardingSession`
- `GetOnboardingSession` (by tenant_id)
- `UpdateOnboardingSession` (status, confirm_step, collected_data, proposed_plan, message_count)
- `GetTenantByBossSlackID`
- `GetTenantByBossLarkID`
- `UpdateOrganizationFromOnboarding` (bulk update all new fields)
- `CreateOrgUnit`
- `ListOrgUnits` (by tenant_id, returns all units for tree construction)
- `GetOrgUnit` (by id)
- `UpdateOrgUnit` (name, unit_type, parent_id, head_role, head_employee_id)
- `DeleteOrgUnit` (soft delete: set is_active = false)
- `AssignEmployeeToUnit` (update employees.org_unit_id)
- `ListEmployeesByUnit` (by org_unit_id)

---

## File Changes Summary

| Path | Action |
|------|--------|
| `internal/onboarding/service.go` | New — OnboardingService main logic |
| `internal/onboarding/prompt.go` | New — system prompt construction |
| `internal/onboarding/extractor.go` | New — structured info extraction |
| `internal/onboarding/planner.go` | New — management plan generation + ProposedPlan validation |
| `internal/onboarding/confirmer.go` | New — 3-step confirmation flow |
| `sql/migrations/000011_onboarding.up.sql` | New — schema changes (drop wizard_sessions, create onboarding_sessions + org_units, extend employees/organizations/tenants) |
| `sql/migrations/000011_onboarding.down.sql` | New — rollback |
| `sql/queries/onboarding.sql` | New — sqlc queries for onboarding_sessions |
| `sql/queries/org_units.sql` | New — sqlc queries for org_units + employee assignment |
| `internal/channel/boss_resolve.go` | New — `ResolveBoss(ctx, db, channelType, userID)` for inbound boss identity resolution |
| `internal/channel/message_handler.go` | Modify — add boss resolution + onboarding routing before employee resolution |
| `internal/bot/commands.go` | Modify — `HandleStart` delegates to OnboardingService |
| `cmd/brain/main.go` | Modify — inject OnboardingService, add onboarding routing in raw text handler, add migration 011 to runMigrations() |
