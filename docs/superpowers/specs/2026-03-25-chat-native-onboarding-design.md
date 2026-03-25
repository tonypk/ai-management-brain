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

## Priority

Phase 1 (this spec): Onboarding flow only. Group mentor enhancements are designed here but implemented later.

---

## 1. Onboarding Dialogue Architecture

### Trigger

Boss sends first message (or `/start`) to the bot via any channel (Telegram/Lark/Slack). If the tenant has no completed onboarding (`onboarding_sessions.status != 'active'`), the bot enters guided dialogue mode.

### State Machine

```
idle → onboarding → configuring → confirming → active
```

| State | Description |
|-------|-------------|
| `idle` | Not started |
| `onboarding` | AI-guided dialogue collecting information (multi-turn) |
| `configuring` | Collection complete, AI generating management plan |
| `confirming` | Presenting plan to boss step by step (3 steps) |
| `active` | All confirmed, mentor begins working |

### Dialogue Storage

| Store | Key | TTL | Purpose |
|-------|-----|-----|---------|
| Redis | `onboarding:chat:{tenantID}` | 7 days | Conversation history (LLM context) |
| Redis | `onboarding:extracted:{tenantID}` | 7 days | Extracted structured info cache |
| PostgreSQL | `onboarding_sessions` | Permanent | Session state, collected data, proposed plan |

7-day TTL allows the boss to complete onboarding across multiple sessions.

### Turn Limits

- AI proactively wraps up around turn 15
- Hard cap at 20 turns — forces transition to summary
- `onboarding_sessions.message_count` tracks progress

### LLM System Prompt Design

AI plays a "management consultant" role. The prompt includes:

- Role definition: experienced management consultant conducting an initial assessment
- Collection targets with required/optional flags (see table below)
- Already-collected information (injected dynamically each turn)
- Missing required items (so AI knows what to steer toward)
- Instructions to be conversational, not interrogative — follow up on interesting points, ask one thing at a time

### Information to Collect

| Category | Details | Required |
|----------|---------|----------|
| Industry | Specific industry / sub-sector | Yes |
| Company stage | Startup / growth / mature / transformation | Yes |
| Business model | B2B / B2C / SaaS / platform / services / etc. | Yes |
| Team size | Headcount + rough structure | Yes |
| Current projects | What they're building, goals, timelines | Yes |
| Management pain points | Top 1-3 management challenges | Yes |
| Communication tools | Which tools, how many groups | Yes |
| Culture preferences | Team country/region distribution, communication style | No |
| Goal management | Existing framework or need recommendation | No |

### Information Extraction

After each dialogue turn, a lightweight LLM call (Haiku-class) extracts newly revealed structured information and updates `onboarding_sessions.collected_data`. This is separate from the main dialogue LLM call to keep extraction reliable and cost-efficient.

When all required fields are covered, the main dialogue LLM is prompted to wrap up: "I think I have a good picture now. Let me put together a management plan for you."

---

## 2. Management Plan Generation & Step-by-Step Confirmation

### Plan Generation

When collection completes, AI transitions to `configuring` state. A single LLM call with all collected context generates a complete management plan as structured JSON, stored in `onboarding_sessions.proposed_plan`.

### Three Confirmation Steps

**Step 1: Mentor + Board Configuration**

AI recommends based on industry, stage, and pain points:

- Primary mentor + reasoning (e.g., "Your SaaS startup suits Musk's first-principles + rapid iteration style")
- Optional: secondary mentor blend (e.g., Musk 70% + Inamori 30%)
- 6 C-Suite seat persona assignments (e.g., CTO seat uses Grove's OKR-driven style)

Boss can: accept / change mentor / adjust blend ratio / modify seat assignments

**Step 2: Management Policies**

Generated from industry + stage + pain points:

- Goal management framework recommendation (OKR / KPI / Scrum / MBO / BSC) + reasoning
- Customized check-in questions (based on mentor + industry, not generic templates)
- Tracking focus areas (e.g., startup: burn rate + delivery speed; mature: retention + efficiency)
- Risk alert rules (consecutive missed check-ins threshold, sentiment drop threshold)
- Management cadence (what happens daily / weekly / monthly)

Boss can: confirm or modify each item

**Step 3: Daily Plan + Group Setup**

- Scheduled task timetable (check-in, chase, summary, briefing, signal scan times)
- Group setup instructions: boss adds bot to team group → bot auto-detects → mentor sends self-introduction in mentor persona
- Employee onboarding: bot generates invite link in group, or boss uses `/addemployee`

### Data Writes Per Step

| Step | Database writes |
|------|----------------|
| Step 1 confirmed | `tenants.mentor_id`, `tenants.mentor_blend`, `seats` table |
| Step 2 confirmed | `organizations.management_plan` (JSONB) |
| Step 3 confirmed | Register scheduler jobs, set `onboarding_sessions.status = 'active'`, set `tenants.onboarding_completed_at` |

### Post-Onboarding Modifications

Boss can modify settings anytime via chat:

- "I want to switch mentors" → single-item modification flow
- "Change check-in time to 10am" → update schedule
- "Re-do onboarding" → reset `onboarding_sessions.status` to `idle`

No need to re-run full onboarding. AI recognizes intent and walks through the specific change.

---

## 3. Data Model Changes

### Existing Table Extensions

**`organizations` table** (add fields):

| New Field | Type | Description |
|-----------|------|-------------|
| `management_pain_points` | TEXT[] | List of management pain points |
| `current_projects` | JSONB | Current project descriptions (name, goal, timeline) |
| `target_framework` | VARCHAR(50) | Goal management framework: okr/kpi/scrum/mbo/bsc |
| `team_structure` | JSONB | Team structure (departments, role distribution) |
| `communication_tools` | TEXT[] | Communication tools in use |
| `culture_preferences` | JSONB | Culture preferences (country distribution, comm style) |
| `onboarding_status` | VARCHAR(20) | idle/onboarding/configuring/confirming/active |

**`tenants` table** (add fields):

| New Field | Type | Description |
|-----------|------|-------------|
| `onboarding_completed_at` | TIMESTAMPTZ | When onboarding completed, NULL = not done |
| `boss_slack_id` | VARCHAR(50) | Boss's Slack user ID |
| `boss_lark_id` | VARCHAR(50) | Boss's Lark user ID |

### New Table: `onboarding_sessions`

```sql
CREATE TABLE onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    status VARCHAR(20) NOT NULL DEFAULT 'onboarding',
    confirm_step INT NOT NULL DEFAULT 0,
    collected_data JSONB NOT NULL DEFAULT '{}',
    proposed_plan JSONB NOT NULL DEFAULT '{}',
    message_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- `status`: onboarding / configuring / confirming / active
- `confirm_step`: 0 = not started, 1 = mentor confirming, 2 = policies confirming, 3 = plan confirming
- `collected_data`: structured info extracted from dialogue (updated in real-time)
- `proposed_plan`: AI-generated management plan (pending confirmation)
- `message_count`: dialogue turn counter
- `UNIQUE(tenant_id)`: one active onboarding per tenant

### Redis Keys

| Key | Purpose | TTL |
|-----|---------|-----|
| `onboarding:chat:{tenantID}` | Dialogue history (LLM context) | 7 days |
| `onboarding:extracted:{tenantID}` | Extracted structured info cache | 7 days |

---

## 4. Dialogue Routing & Multi-Channel Unification

### Unified Routing Logic

```
Message arrives (any channel)
  → IdentityResolver: identify sender (boss / employee / unknown)
  → If boss:
      → Check onboarding_sessions.status
      → idle / onboarding / configuring / confirming → route to OnboardingService
      → active → existing routing (seat chat / mentor chat)
  → If employee:
      → existing routing (check-in / mentor chat)
```

### OnboardingService (New Module)

```
internal/onboarding/
  service.go      -- Main logic: receive message → check state → dispatch
  prompt.go       -- System prompt construction (consultant role + targets + collected info)
  extractor.go    -- Extract structured info from dialogue (lightweight LLM call)
  planner.go      -- Generate management plan from collected info
  confirmer.go    -- Manage 3-step confirmation flow
```

**Key interface:**

```go
type OnboardingService struct {
    db        *sqlc.Queries
    redis     *redis.Client
    llm       LLMClient
    engine    *brain.EngineFactory
    scheduler *scheduler.Scheduler
}

// HandleMessage processes a boss message during onboarding.
// Returns response text — caller sends via the appropriate channel.
func (s *OnboardingService) HandleMessage(ctx context.Context, tenantID uuid.UUID, text string) (string, error)
```

OnboardingService returns text only. The caller (channel adapter) is responsible for sending via the correct channel. This keeps all onboarding logic channel-agnostic.

### Boss Identity Expansion

Current: only `tenants.boss_chat_id` (Telegram ID).

After: IdentityResolver checks by channel type:

| Channel | Field |
|---------|-------|
| Telegram | `boss_chat_id` |
| Slack | `boss_slack_id` |
| Lark | `boss_lark_id` |

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

Extend `group_chats.group_type`:

| Type | Description | Mentor behavior difference |
|------|-------------|---------------------------|
| `general` | All-hands group | Morning briefings, weekly reviews, team-level info |
| `project` | Project group | Track that project's progress, PRs, deadlines |
| `management` | Leadership group | Deep analysis, personnel suggestions, sensitive topics |

Boss specifies group type during onboarding or later setup.

---

## Migration Plan

### New Migration: `000011_onboarding.up.sql`

1. Create `onboarding_sessions` table
2. Add fields to `organizations`: `management_pain_points`, `current_projects`, `target_framework`, `team_structure`, `communication_tools`, `culture_preferences`, `onboarding_status`
3. Add fields to `tenants`: `onboarding_completed_at`, `boss_slack_id`, `boss_lark_id`

### New sqlc Queries

- `CreateOnboardingSession`
- `GetOnboardingSession` (by tenant_id)
- `UpdateOnboardingSession` (status, confirm_step, collected_data, proposed_plan, message_count)
- `GetTenantByBossSlackID`
- `GetTenantByBossLarkID`
- `UpdateOrganizationFromOnboarding` (bulk update all new fields)

---

## File Changes Summary

| Path | Action |
|------|--------|
| `internal/onboarding/service.go` | New — OnboardingService main logic |
| `internal/onboarding/prompt.go` | New — system prompt construction |
| `internal/onboarding/extractor.go` | New — structured info extraction |
| `internal/onboarding/planner.go` | New — management plan generation |
| `internal/onboarding/confirmer.go` | New — 3-step confirmation flow |
| `sql/migrations/000011_onboarding.up.sql` | New — schema changes |
| `sql/migrations/000011_onboarding.down.sql` | New — rollback |
| `sql/queries/onboarding.sql` | New — sqlc queries |
| `internal/bot/middleware.go` | Modify — expand IdentityResolver for multi-channel boss |
| `cmd/brain/main.go` | Modify — add onboarding routing before existing routes |
| `internal/channel/message_handler.go` | Modify — add onboarding routing for non-Telegram channels |
| `internal/config/config.go` | Modify — no new env vars needed (uses existing DB/Redis/LLM) |
