# AI Management Brain

> Open-source AI management middleware: connect your Telegram, select a management mentor philosophy, and let AI handle daily reports, team communication, and executive summaries — culturally adapted per employee.

**Invisible but indispensable.** Customers don't replace any existing tools. The brain quietly takes over management logic.

## Quick Start

```bash
# 1. Clone
git clone https://github.com/tonypk/ai-management-brain.git
cd ai-management-brain

# 2. Configure
cp .env.example .env
# Edit .env: set ENCRYPTION_KEY, JWT_SECRET (openssl rand -hex 32), TELEGRAM_BOT_TOKEN, BOSS_TELEGRAM_ID

# 3. Run
docker compose up -d

# 4. Verify
curl http://localhost/healthz
# {"db":"ok","redis":"ok","status":"ok"}
```

**9 Management Mentors:** Inamori (稻盛和夫) · Dalio · Grove · Ren (任正非) · Son (孙正义) · Jobs · Bezos · Ma (马云) · Musk (马斯克)

**6 Culture Packs:** Philippines · Singapore · Indonesia · Sri Lanka · Malaysia · China

**Features:** Daily check-ins, AI-powered chase & summaries, mentor blending, anomaly alerts, multi-channel (Telegram + Slack + Lark), Vue3 dashboard, OAuth, billing.

---

## Design Specification (v1.1 · 2026-03-20)

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

**Competitive Moat (vs DIY Claude/GPT setups):**
- Persistent memory across days/weeks (trend detection, not single-chat)
- Multi-employee orchestration (manage 20+ people independently)
- Cultural adaptation per employee (impossible in single-prompt setup)
- Structured strategy execution (not ad-hoc prompting)

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
  │  │ Engine       │ │ (Phase 4)  │ │Skills  ││
  │  └──────────────┘ └────────────┘ └────────┘│
  │                                             │
  │  BYOK · Multi-tenant · Scheduler · Memory   │
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

Phase 1-3 are scheduler-driven (cron). Event-driven architecture (pub/sub) introduced in Phase 4.

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

#### Mentor YAML Schema

```yaml
id: inamori                          # unique identifier (required)
name: 稻盛和夫                        # display name (required)
name_en: Kazuo Inamori               # english name (required)
company: 京瓷 · KDDI                  # associated company (required)
philosophy: 敬天爱人，自利利他          # one-line summary (required)
version: 1                           # schema version for upgrades

strategy:
  checkin_questions:                  # (required) daily questions sent to employees
    - "今天你为团队做了什么贡献？"
    - "遇到什么困难需要大家帮助？"
    - "你从今天的工作中学到了什么？"

  chase:
    method: private_first             # (required) private_first | public_direct | data_driven
    escalation:                       # (required) ordered steps
      - action: private_message       # private_message | public_reminder | manager_notify | issue_log | skip_today
        delay: "0"                    # duration from chase trigger
        tone: warm_reminder           # free-form tone hint for AI
      - action: manager_notify
        delay: "2h"
        tone: caring_concern
      - action: skip_today
        delay: "4h"
    forbidden:                        # (optional) actions this mentor NEVER does
      - public_naming
      - shame_based
    encouraged:                       # (optional) actions this mentor prefers
      - private_conversation
      - effort_recognition

  summary:
    focus:                            # (required) what to emphasize in boss summary
      - morale
      - collaboration
      - support_needs
    highlight: team_achievements      # (required) primary highlight category
    flag: emotional_signals           # (required) primary warning signal
    metrics:                          # (optional) mentor-specific metrics
      - name: 团队协作度
        source: mutual_mentions
      - name: 需要关怀
        source: sentiment_negative

  actions:                            # (optional, Phase 2+)
    weekly:
      - type: recognition
        desc: "感谢本周贡献最大的成员"
      - type: team_pulse
        desc: "团队氛围快速调查"
    monthly:
      - type: report
        desc: "利他贡献月报"
    triggers:                         # (optional, Phase 2+)
      - event: consecutive_miss_3days
        action: manager_private_chat
        message: "{name} 连续3天未提交，建议私下关心一下"
      - event: sentiment_drop
        action: private_checkin
        message: "最近感觉你状态不太好，有什么我能帮到你的吗？"

  system_prompt: |                    # (required) injected into all Claude API calls
    你融合了稻盛和夫的管理哲学。核心原则：
    1. 以利他心出发，先考虑对方感受再提要求
    2. 强调集体荣誉感和团队归属感
    3. 认可努力本身，不只看结果数字
    4. 温和而坚定，批评前先充分肯定
    5. 「全员经营」— 让每个人感觉自己是经营者而非打工人
```

#### Culture Pack YAML Schema

```yaml
market: Philippines                   # (required)
language: Filipino / English          # (required)
timezone: Asia/Manila                 # (required)
version: 1                           # schema version

communication_style:
  directness: low                     # low | medium | high
  hierarchy_respect: high             # low | medium | high
  relationship_first: true
  group_face: high                    # low | medium | high

chase_rules:
  never_name_in_group: true           # culture overrides mentor if true
  private_before_escalate: true
  warmth_required: true
  acknowledge_effort: true

forbidden_patterns:                   # natural language rules for Claude (not regex)
  - "Why haven't you..."
  - "You are the only one"
  - "As I mentioned"

preferred_patterns:
  - "Hope you're doing well"
  - "The team really values your input"
  - "Whenever you have a moment"
```

#### Mentor x Culture Matrix

Same action, different execution per culture. Culture can **override** mentor when there's a conflict (e.g., Dalio wants public chase but PH culture requires private-first → private wins):

| | Philippines (low direct, high face) | Singapore (high direct) |
|---|---|---|
| **Inamori** | Private: warm, thank effort first | Private: polite but clear |
| **Dalio** | Private (culture override) | Group: direct transparency |
| **Grove** | Private: output framing | Group: @name deadline |

#### Mentor Blending (Phase 2)

Users can blend mentors with weights (e.g., 70% Inamori + 30% Dalio):

- **Questions**: Use primary mentor's questions, optionally append 1 from secondary
- **Chase strategy**: Primary mentor's method; secondary influences tone
- **Summary**: Weighted merge of focus areas
- **System prompt**: Primary prompt with secondary's key principles appended

---

### Authentication & Authorization

#### Phase 1: Bot-based Auth

```
Boss Setup Flow:
1. Boss creates Telegram bot via @BotFather
2. Boss runs the Brain binary with BOT_TOKEN + BOSS_TELEGRAM_ID in .env
3. Boss adds bot to team group
4. Bot recognizes boss by telegram_id → grants admin commands

Employee Registration:
1. Boss uses /addemployee <name> <culture_code> in bot DM
2. Bot generates an invite link or code
3. Employee DMs the bot with /join <code>
4. Bot links employee.telegram_id → confirmed
5. Unrecognized users get: "Please contact your manager for access"
```

#### Phase 3+: API Auth

- JWT tokens for dashboard login (email + password)
- API keys for programmatic access (per tenant)
- Tenant isolation: every API request passes through `TenantFromContext()` middleware
- RBAC: boss (full access), manager (team-scoped), member (own data only)

#### Command Permissions

| Command | Boss | Manager | Employee |
|---------|------|---------|----------|
| /start, /help | yes | yes | yes |
| /status | yes | yes (own team) | no |
| /mentor, /culture | yes | no | no |
| /addemployee | yes | no | no |
| /config | yes | no | no |
| /join | no | no | yes |

---

### Secret Encryption (BYOK)

All sensitive fields (`anthropic_key`, `bot_token`) are encrypted at rest using envelope encryption:

1. **Master Key**: AES-256 key loaded from `ENCRYPTION_KEY` environment variable (32 bytes)
2. **Per-field encryption**: AES-256-GCM with random nonce per value
3. **Storage format**: `nonce:ciphertext` (base64 encoded) in TEXT column
4. **Key rotation**: Re-encrypt all values with new master key via admin command
5. **Lost key**: Encrypted values are unrecoverable — tenants must re-enter their API keys

Implementation: `internal/pkg/crypto.go` provides `Encrypt(plaintext, masterKey) → ciphertext` and `Decrypt(ciphertext, masterKey) → plaintext`.

---

### Bot Interaction Model

#### Group Chat vs DM

- **Group chat**: Bot listens for commands (/start, /status) and recognizes free-form reports
- **DM (private chat)**: Primary channel for report collection (privacy), chase messages, and admin commands
- **Report collection preference**: Bot sends check-in questions via DM; employees reply in DM

#### Conversation State Machine

Report collection is a multi-turn conversation tracked in Redis:

```
States: idle → collecting(q1) → collecting(q2) → collecting(q3) → confirming → idle

Redis key: conv:{employee_id}
Value: {state, current_question, answers_so_far, started_at}
TTL: 4 hours (auto-expire abandoned conversations)

Flow:
1. Bot sends Q1 → state = collecting(q1)
2. Employee replies → store answer, send Q2 → state = collecting(q2)
3. Employee replies → store answer, send Q3 → state = collecting(q3)
4. Employee replies → show summary "Here's your report: ..." → state = confirming
5. Employee confirms (or bot auto-confirms after 5min) → save to DB → state = idle

Edge cases:
- Employee goes silent: TTL expires after 4h, partial answers discarded
- Employee sends unrelated message mid-report: bot gently redirects
- Employee sends all answers in one message: AI parses and maps to questions
- Group message recognized as report: bot DMs confirmation, saves to DB
```

---

### Error Handling & Failure Modes

| Failure | Handling |
|---------|----------|
| Claude API down/timeout | Retry 3x with exponential backoff (1s, 4s, 16s). If all fail: skip AI features, send raw data to boss with "AI unavailable" note |
| Claude API key invalid | Mark tenant's key as invalid, notify boss via bot: "Your API key is no longer valid. Please update via /config" |
| Telegram send failure | Message queue in Redis with retry (3 attempts). Failed messages logged to `chase_logs` with `action=send_failed` |
| Scheduler missed job | On startup, check for missed jobs (last_run timestamp in Redis). Run catch-up if within 2-hour window |
| Partial report data | Generate summary with available data. Note in summary: "3/10 employees submitted today" |
| DB connection lost | Health check retries every 5s. Bot responds with "Service temporarily unavailable" |

---

### Observability

- **Logging**: Structured JSON via `slog` (Go stdlib). Log levels: DEBUG, INFO, WARN, ERROR
- **Log redaction**: Mask API keys, bot tokens, employee personal data in logs
- **Health endpoint**: `GET /healthz` returns DB, Redis, Telegram Bot status
- **Bot diagnostic**: `/diagnostics` command (boss only) shows: last remind/chase/summary time, message success rate, API key status
- **Metrics** (Phase 3+): Prometheus endpoint with: messages_sent_total, reports_collected_total, chases_triggered_total, summaries_generated_total, llm_calls_total, llm_errors_total

---

### Tech Stack

| Layer | Choice | Reason |
|-------|--------|--------|
| Language | Go 1.25+ | Single binary, user's primary stack |
| Web Framework | Gin | Consistent with existing projects |
| DB Access | sqlc | Type-safe SQL |
| Database | PostgreSQL 16 | Multi-tenant RLS, JSONB |
| Cache/Queue | Redis 7 | Conversation state, scheduler state, message queue |
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
│   │   ├── handler.go              # Message routing + conversation state
│   │   ├── commands.go             # /start /status /help /mentor /addemployee
│   │   └── middleware.go           # Identity resolution, tenant routing, permissions
│   ├── brain/                      # Core AI Engine (Strategy Executor)
│   │   ├── engine.go               # Assemble prompt (mentor+culture)
│   │   ├── mentor.go               # Load/blend mentor strategies from YAML
│   │   ├── culture.go              # Cultural adaptation layer
│   │   └── llm.go                  # Claude API wrapper with retry
│   ├── report/                     # Report business logic
│   │   ├── collector.go            # Multi-turn conversation collector
│   │   ├── chaser.go               # Mentor-driven escalation + cultural override
│   │   └── summarizer.go           # AI summary with mentor-specific focus
│   ├── scheduler/
│   │   └── jobs.go                 # remind / chase / summarize + missed job catch-up
│   ├── channel/                    # Channel abstraction (Phase 3+)
│   │   ├── adapter.go              # interface: Send(), Receive(), Reply()
│   │   └── telegram.go
│   ├── db/                         # sqlc generated code
│   │   └── sqlc/
│   ├── config/
│   │   └── config.go               # Env-based config with validation
│   └── pkg/
│       ├── crypto.go               # AES-256-GCM envelope encryption
│       └── response.go             # Unified API response format
├── api/                            # REST API (Phase 3+)
├── configs/
│   ├── mentors/                    # Mentor YAML files
│   └── cultures/                   # Culture pack YAML files
├── frontend/                       # Vue3 Dashboard (Phase 3+)
├── sql/
│   ├── migrations/                 # golang-migrate files
│   └── queries/                    # sqlc query files
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── .env.example
```

Single binary in MVP. Split into cmd/bot + cmd/api + cmd/worker when scaling needed.

### Database Schema

```sql
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,              -- AES-256-GCM encrypted
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,             -- optional: {"inamori": 0.7, "dalio": 0.3}
    bot_token     TEXT,              -- AES-256-GCM encrypted
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
    role          TEXT NOT NULL DEFAULT 'member',  -- boss | manager | member
    invite_code   TEXT,              -- for /join registration
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,    -- {"q1": "...", "q2": "...", "q3": "..."}
    blockers      TEXT,              -- AI-extracted blockers
    sentiment     TEXT,              -- positive | neutral | negative
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);

CREATE TABLE chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,      -- private_message | public_reminder | manager_notify | send_failed
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
    key_metrics     JSONB,           -- mentor-specific metrics
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);

-- Indexes
CREATE INDEX idx_employees_tenant ON employees(tenant_id);
CREATE INDEX idx_employees_telegram ON employees(telegram_id);
CREATE INDEX idx_reports_tenant_date ON reports(tenant_id, report_date);
CREATE INDEX idx_reports_employee_date ON reports(employee_id, report_date);
CREATE INDEX idx_chase_logs_tenant_date ON chase_logs(tenant_id, report_date);
CREATE INDEX idx_chase_logs_employee ON chase_logs(employee_id, report_date);
CREATE INDEX idx_summaries_tenant_date ON summaries(tenant_id, summary_date);
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
    ├── GetForbiddenPatterns()     -> []string (natural language, not regex)
    ├── GetPreferredPatterns()     -> []string
    └── ShouldOverride(action)     -> bool (culture trumps mentor on conflict)
```

---

### Completed Roadmap

All 5 phases are complete. The project evolved significantly beyond initial plans — delivering 16 mentors (vs planned 8), 9 culture packs (vs planned 6), MCP multi-client support, and a hybrid ClawHub skill.

| Phase | Goal | Key Deliverables | Status |
|-------|------|-----------------|--------|
| **1. Core Bot** | Daily management loop | Telegram Bot, scheduler, 2 mentors, tenant isolation, AES-256 secrets | Done |
| **2. Full Mentor + Culture** | Complete strategy system | 9 mentors (expanded to 16 with light-touch), 9 culture packs, mentor blending, proactive actions, sentiment detection | Done |
| **3. Open Source + Dashboard** | Distributable project | REST API (Gin), Vue3 Dashboard (NaiveUI + ECharts), Docker deploy, CI/CD, rate limiting, Prometheus metrics | Done |
| **4. Multi-Channel + MCP** | Multi-client support | MCP server (13 tools), OpenClaw Skill, Claude Code + ChatGPT + Gemini support, AI C-Suite Board (6 seats), Organization Architecture Engine | Done |
| **5. Cloud SaaS** | Production cloud service | `manageaibrain.com`, Web Dashboard v2.0, Organization Setup Wizard, boss-ai-agent@3.0.0 hybrid skill (Advisor + Team Ops modes), 23+ messaging platforms | Done |

### Current State (v3.0.0)

- **16 mentor philosophies**: 3 fully-embedded (Musk/Inamori/Ma) + 6 standard + 7 light-touch
- **9 culture packs**: default, Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India
- **6 AI C-Suite seats**: CEO, CFO, CMO, CTO, CHRO, COO
- **13 MCP tools**: 9 read + 4 write (message delivery)
- **ClawHub Skill**: `boss-ai-agent@3.0.0` — Advisor Mode (zero dependency) + Team Operations Mode (MCP-connected)
- **Web Dashboard**: Health gauge, submission trends, sentiment heatmap, alert center, organization setup
- **Production**: `manageaibrain.com` on AWS t3a.small, Docker Compose, PostgreSQL 16, Redis 7
- All channels (+ Lark/DingTalk)
- SLA + priority support
- Custom pricing for 100+ people

---

### Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Claude API cost | BYOK — tenants pay own AI costs |
| Claude API down | Retry 3x, fallback to raw data with "AI unavailable" note |
| Mentor YAML too rigid | YAML as structured hints, Claude reasons dynamically within framework |
| Cultural adaptation errors | Forbidden patterns + human review for new cultures |
| Telegram rate limits | Redis message queue with rate-aware batching |
| Open source forks | Move fast, build community, cloud has network effects (shared mentors, benchmarks) |
| Scope creep | Each phase has clear DoD; don't start next until current passes |
| Security: stored secrets | AES-256-GCM envelope encryption, master key from env var |
| Data privacy (GDPR) | Configurable retention, tenant data export, deletion cascade |
| Runaway AI costs | Per-tenant daily API call budget with configurable limit |
