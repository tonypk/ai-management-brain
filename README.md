# AI Management Brain

> Open-source AI management middleware: connect your Telegram, select a management mentor philosophy, and let AI handle daily reports, team communication, and executive summaries — culturally adapted per employee.

**Invisible but indispensable.** Customers don't replace any existing tools. The brain quietly takes over management logic.

---

## Design Specification (v1.0 · 2026-03-20)

### Product Definition

**Target:** Global remote team managers / founders / CEOs (5-100 people, multi-country)

**Business Model:** Open-source core (Apache 2.0) + paid cloud service

**Core Differentiators:**

| Dimension | Competitors (DailyBot, Geekbot, Xembly) | AI Management Brain |
|-----------|----------------------------------------|---------------------|
| Channel | Slack-first | Telegram-first (expand later) |
| AI Depth | Simple summaries | Claude deep analysis + anomaly detection |
| Mentor System | None | Mentor = Management OS (strategy, not just tone) |
| Cultural Intelligence | None | Auto-adapt per employee nationality/culture |
| Open Source | Closed SaaS | Core open source, self-hostable |

---

### Architecture

```
Data Input Layer (any source)
  Lark / DingTalk / Telegram / Slack / HR SaaS / Email / API
                          |
                          v
AI Management Intelligence Layer (the product)
  ┌─────────────────────────────────────────────┐
  │          AI Management Brain                │
  │                                             │
  │  ┌──────────────┐ ┌────────────┐ ┌────────┐│
  │  │ Mentor       │ │ Agent      │ │Culture ││
  │  │ Strategy     │ │ Orchestr.  │ │Adapt + ││
  │  │ Engine       │ │            │ │Skills  ││
  │  └──────────────┘ └────────────┘ └────────┘│
  │                                             │
  │  BYOK · Multi-tenant · Event-driven · Memory│
  └─────────────────────────────────────────────┘
                          |
                          v
Output Action Layer
  Auto Chase    Decision Push    Policy Publish    Anomaly Alert
  daily/weekly  boss summary     multi-channel     progress risk
                          |
                          v
Write Back to Original Channel
  Lark / DingTalk / Telegram / Email
```

---

### Core Concept: Mentor = Management Operating System

Selecting a mentor changes **what the system does, how it does it, and what it focuses on**:

1. **Check-in Questions** — what to ask employees daily
2. **Chase Strategy** — escalation path, forbidden actions
3. **Summary Focus** — what metrics/signals the boss sees
4. **Proactive Actions** — weekly recognition, pulse surveys, etc.
5. **Trigger Rules** — event-driven responses (3-day miss, sentiment drop, etc.)
6. **AI System Prompt** — personality and reasoning framework

#### Mentor Comparison

| Dimension | Inamori (稻盛和夫) | Dalio | Grove | Ren Zhengfei (任正非) |
|-----------|-------------------|-------|-------|---------------------|
| Philosophy | Altruism, Amoeba | Radical Transparency | High Output, OKR | Wolf Culture, Self-criticism |
| Questions | Contribution-focused | Decision-focused | Output-focused | Goal/battle-focused |
| Chase | Private, warm, never public | Public, direct, principled | Data-driven, deadline flag | Direct, competitive ranking |
| Summary | Morale, collaboration, support | Decision quality, mistakes, principles | OKR progress, efficiency, bottlenecks | Achievement rate, top fighters |
| Triggers | Sentiment drop → caring check-in | Same mistake twice → create principle | Output drop → suggest 1:1 | Low output → performance warning |

#### Mentor x Culture Matrix

Same action, different execution per culture:

| | Philippines (low direct, high face) | Singapore (high direct) |
|---|---|---|
| **Inamori** | Private: warm, thank effort first | Private: polite but clear |
| **Dalio** | Private (culture override) | Group: direct transparency |
| **Grove** | Private: output framing | Group: @name deadline |

Culture can **override** mentor when there's a conflict.

---

### Tech Stack

| Layer | Choice | Reason |
|-------|--------|--------|
| Language | Go 1.22+ | Single binary, user's primary stack |
| Web Framework | Gin | Consistent with existing projects |
| DB Access | sqlc | Type-safe SQL |
| Database | PostgreSQL 16 | Multi-tenant RLS, JSONB |
| Cache/Queue | Redis 7 | Scheduler state, message dedup |
| AI | Claude API (anthropic-go) | BYOK per tenant |
| Bot | telebot/v3 | Mature Go Telegram Bot library |
| Scheduler | go-co-op/gocron | Lightweight cron |
| Deploy | Docker Compose | Single machine start |

### Project Structure

```
ai-management-brain/
├── cmd/
│   └── brain/main.go              # Single binary: API + Bot + Scheduler
├── internal/
│   ├── bot/                        # Telegram Bot
│   │   ├── handler.go
│   │   ├── commands.go
│   │   └── middleware.go
│   ├── brain/                      # Core AI Engine (Strategy Executor)
│   │   ├── engine.go               # Assemble prompt (constitution+mentor+culture)
│   │   ├── mentor.go               # Load/blend mentor strategies
│   │   ├── culture.go              # Cultural adaptation layer
│   │   └── llm.go                  # Claude API wrapper
│   ├── report/                     # Report business logic
│   │   ├── collector.go
│   │   ├── chaser.go
│   │   └── summarizer.go
│   ├── scheduler/
│   │   └── jobs.go
│   ├── channel/                    # Channel abstraction (Phase 3+)
│   │   ├── adapter.go
│   │   └── telegram.go
│   ├── db/
│   │   ├── sqlc/
│   │   ├── queries/
│   │   └── migrations/
│   ├── config/
│   │   └── config.go
│   └── pkg/
│       ├── crypto.go
│       └── response.go
├── api/                            # REST API (Phase 3+)
├── configs/
│   ├── mentors/                    # Mentor YAML files
│   └── cultures/                   # Culture pack YAML files
├── frontend/                       # Vue3 Dashboard (Phase 3+)
├── sql/
│   ├── schema.sql
│   ├── migrations/
│   └── queries/
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── .env.example
```

### Database Schema

```sql
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,
    bot_token     TEXT,
    boss_chat_id  BIGINT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    telegram_id   BIGINT UNIQUE,
    culture_code  TEXT NOT NULL DEFAULT 'default',
    role          TEXT NOT NULL DEFAULT 'member',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,
    blockers      TEXT,
    sentiment     TEXT,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);

CREATE TABLE chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,
    message       TEXT NOT NULL,
    chased_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary_date    DATE NOT NULL,
    content         TEXT NOT NULL,
    submission_rate FLOAT NOT NULL DEFAULT 0,
    blockers_count  INT NOT NULL DEFAULT 0,
    key_metrics     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);
```

### Brain Engine

Strategy executor, not just a prompt builder:

```
MentorStrategy (loaded from YAML)
    ├── GetCheckinQuestions()       -> []string
    ├── GetChaseStep(stepNum)      -> ChaseAction{action, delay, tone}
    ├── GetSummaryConfig()         -> SummaryConfig{focus, highlight, flag, metrics}
    ├── GetProactiveActions(freq)  -> []Action (Phase 2+)
    ├── GetTriggerRules()          -> []TriggerRule (Phase 2+)
    ├── BuildSystemPrompt()        -> string
    ├── BlendWith(other, weight)   -> MentorStrategy (Phase 2+)
    └── AdaptForCulture(code)      -> applies cultural overrides

CulturePack (loaded from YAML)
    ├── GetOverrides()             -> communication rules
    ├── GetForbiddenPatterns()     -> []string
    ├── GetPreferredPatterns()     -> []string
    └── ShouldOverride(action)     -> bool
```

---

### Phased Roadmap

#### Phase 1: Core Bot (Week 1-2)

**Goal:** Complete daily loop with mentor-driven strategy.

| Task | Output |
|------|--------|
| Project init + Docker Compose | Go module + PG + Redis |
| DB schema + migrations (5 tables) | sql/migrations/ |
| Mentor YAML loader (Inamori + Dalio) | internal/brain/mentor.go |
| Culture pack loader (PH + SG) | internal/brain/culture.go |
| Brain Engine v1 (strategy executor) | internal/brain/engine.go |
| Telegram Bot framework | internal/bot/ |
| Report Collector | internal/report/collector.go |
| Chase logic (mentor-driven + cultural) | internal/report/chaser.go |
| Summary generation (mentor-driven) | internal/report/summarizer.go |
| Scheduler (remind/chase/summary) | internal/scheduler/jobs.go |
| Bot commands: /start /status /help | internal/bot/commands.go |
| Claude API wrapper | internal/brain/llm.go |

**Done when:**
- Own Telegram group connected
- Mentor-specific check-in questions sent
- Chase uses mentor strategy + cultural adaptation
- Boss receives mentor-focused AI summary
- Switching Inamori ↔ Dalio visibly changes everything

#### Phase 2: Full Mentor + Culture (Week 3-4)

**Goal:** Complete strategy system with blending, custom mentors, proactive actions.

| Task | Output |
|------|--------|
| 4 mentor YAMLs (Inamori/Dalio/Grove/Ren) | configs/mentors/ |
| 4 culture packs (PH/SG/ID/LK) | configs/cultures/ |
| Mentor blending (weighted mix) | brain/mentor.go |
| Custom mentor (name → Claude → YAML) | brain/mentor.go |
| Proactive actions engine | scheduler/jobs.go |
| Trigger rules engine | brain/engine.go |
| Blocker analysis + sentiment detection | report/summarizer.go |
| Bot: /mentor /culture /blend | bot/commands.go |

**Done when:**
- Switching mentors changes questions + chase + summary
- Culture adaptation works per employee
- Custom mentor creation works
- Mentor blending works
- Proactive actions fire on schedule

#### Phase 3: Open Source + Dashboard (Week 5-7)

**Goal:** Distributable open-source. docker compose up.

| Task | Output |
|------|--------|
| Multi-tenant (tenant_id + RLS) | DB + middleware |
| REST API (Gin) | api/ |
| Vue3 Web Dashboard | frontend/ |
| BYOK (encrypted key per tenant) | pkg/crypto.go |
| Onboarding flow | API + frontend |
| Docker one-click deploy | docker-compose.yml |
| Open source prep (README, LICENSE) | root |
| GitHub Actions CI | .github/workflows/ |

**Done when:**
- Anyone: fork → clone → docker compose up → connect Telegram
- Web UI handles all configuration
- Multi-tenant isolation works

#### Phase 4: Multi-Channel + Agents (Week 8-10)

**Goal:** Slack/Lark support. Boss NL commands.

| Task | Output |
|------|--------|
| Channel abstraction (Adapter interface) | channel/adapter.go |
| Slack + Lark adapters | channel/ |
| Agent Orchestrator | brain/orchestrator.go |
| Chief of Staff Agent | brain/chief.go |
| Alert Agent | report/alert.go |
| More Skills (policy, perf review) | internal/skills/ |

**Done when:**
- Slack + Telegram simultaneously
- Boss NL commands work
- Proactive anomaly alerts

#### Phase 5: Cloud SaaS (Week 11-13)

**Goal:** Revenue-generating cloud service.

Landing page, Stripe billing, analytics dashboard, 8 mentors, 6 culture packs, SSO/RBAC.

---

### Commercial Tiers

**Open Source (Apache 2.0, self-hosted)**
- Complete Bot + API + Dashboard
- 4 mentors + 4 culture packs
- Single tenant, Telegram + Slack
- Docker one-click deploy

**Cloud Pro ($29/mo, ≤20 people)**
- Cloud hosted, no maintenance
- 8 mentors + custom mentors
- 6 cultures + custom cultures
- Multi-channel (+Lark/DingTalk)
- Agent orchestration + Analytics

**Cloud Enterprise ($99/mo, unlimited)**
- Multi-tenant, BYOK, SSO + RBAC
- Custom Skills plugins
- SLA + priority support

---

### Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Claude API cost | BYOK — tenants pay own AI costs |
| Mentor YAML too rigid | YAML as hints, Claude reasons dynamically |
| Cultural errors | Forbidden patterns + human review |
| Telegram rate limits | Queue + batch messages |
| Open source forks | Move fast, community, cloud network effects |
| Scope creep | Clear DoD per phase |
