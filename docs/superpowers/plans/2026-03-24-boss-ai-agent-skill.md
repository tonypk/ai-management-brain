# Boss AI Agent — SKILL.md Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the OpenClaw skill from `management-brain` to `boss-ai-agent` — a full-stack AI management middleware leveraging OpenClaw native tools across 7 scenarios with 14 mentors and 7 culture packs.

**Architecture:** Single SKILL.md file (~800-1200 lines) containing agent instructions organized by: identity, onboarding, tool reference, 7 scenarios, mentor system, cultural adaptation, cloud API, and Chinese docs. Separate README.md for user-facing install guide. Published to ClawHub as new `boss-ai-agent` slug.

**Tech Stack:** Markdown (YAML frontmatter), ClawHub CLI v0.9.0, OpenClaw tool API

**Spec:** `docs/superpowers/specs/2026-03-24-boss-ai-agent-skill-design.md`

---

### Task 1: Scaffold SKILL.md with frontmatter + Identity + First Run

**Files:**
- Create: `openclaw-skill/SKILL.md` (overwrite existing)

- [ ] **Step 1: Write YAML frontmatter**

```yaml
---
name: boss-ai-agent
version: "1.0.0"
description: "Boss AI Agent — your AI management middleware. Connects boss to all systems (Telegram/Slack/GitHub/Notion/Email), 14 mentor philosophies, 7 culture packs, 7 automated scenarios. OpenClaw native-first, zero external dependency."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    primaryEnv: "BOSS_AI_AGENT_API_KEY"
    requires:
      env:
        - "BOSS_AI_AGENT_API_KEY"
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---
```

- [ ] **Step 2: Write Identity section**

Content must include:
- Role definition: "You are Boss AI Agent — the boss's AI management middleware..."
- Core principle: proactive, not passive
- Mentor permeation principle: mentor philosophy affects ALL decisions (risk assessment, priority, communication, escalation), not just question style
- Language: respond in the boss's language (auto-detect from conversation)

- [ ] **Step 3: Write First Run onboarding flow**

Content must include:
- Greeting and self-introduction
- 3 onboarding questions (team size, communication tools, project management tools)
- Auto-detect connected channels via OpenClaw
- Config generation: `[write]` to `~/.openclaw/skills/boss-ai-agent/config.json`
- Cron registration: `[cron add]` for each scheduled task
- Channel verification: `[message send]` test message
- Mentor recommendation based on answers
- Env var fallback: check `MANAGEMENT_BRAIN_API_KEY` if `BOSS_AI_AGENT_API_KEY` not set
- Empty team guard: if 0 people, enter solo founder mode (skip check-in/chase/summary crons, keep briefing/patrol)

- [ ] **Step 4: Verify file renders correctly**

Run: `wc -l openclaw-skill/SKILL.md` — expect ~80-120 lines so far
Run: `head -20 openclaw-skill/SKILL.md` — verify frontmatter parses

- [ ] **Step 5: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: scaffold Boss AI Agent SKILL.md with identity and onboarding"
```

---

### Task 2: Tool Reference section

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Tool Reference cheat sheet**

For each tool, provide:
- **What it does** (1 line)
- **Invocation example** (exact tool call syntax)
- **When Boss AI Agent uses it** (1 line)

Tools to document (in order of importance):

1. **`message`** — send/read/search across all channels
   - `message send --channel telegram --to {channelId} --text "..."`
   - `message read --channel telegram --limit 50 --since "30m ago"`
   - `message search --channel slack --query "blocked" --limit 20`

2. **`cron`** — schedule recurring tasks
   - `cron add --label "daily-checkin" --schedule "0 9 * * 1-5" --task "Send check-in questions to all active employees"`
   - `cron list` / `cron remove --label "daily-checkin"`

3. **`memory_search` / `memory_get`** — persistent agent memory
   - `memory_search --query "John Santos performance"` — semantic search
   - `memory_get --key "emp:john-santos"` — exact key lookup
   - Key prefixes: `emp:{name}`, `decision:{date}`, `project:{name}`, `boss:pref`

4. **`sessions_spawn`** — dispatch sub-agents
   - `sessions_spawn --task "Scan GitHub repos for stale PRs" --tools "web_fetch" --label "github-scanner"`
   - Output format: JSON `{status, findings[], summary}`
   - Timeout: 60s, failure = skip source

5. **`web_fetch`** — fetch external data
   - `web_fetch --url "https://api.github.com/repos/{owner}/{repo}/pulls?state=open"`
   - Used for: GitHub, Linear, Jira, Notion, email APIs

6. **`web_search`** — search the web
   - `web_search --query "AI industry news March 2026"`

7. **`read` / `write`** — local file I/O
   - `read ~/.openclaw/skills/boss-ai-agent/config.json`
   - `write ~/.openclaw/skills/boss-ai-agent/config.json --content '{...}'`

8. **`browser`** — optional, screenshot dashboards
   - `browser screenshot --url "https://github.com/orgs/{org}/projects"`

9. **`exec`** — run shell commands
   - `exec --command "curl -s https://api.example.com/health"`

10. **`nodes`** — device notifications (optional fallback)
    - `nodes notify --message "🚨 URGENT: deploy failure detected"`

11. **`image`** — analyze images
    - `image --path "/tmp/screenshot.png" --prompt "What issues do you see in this UI?"`

- [ ] **Step 2: Verify section completeness**

Each tool must have: description, invocation syntax, and Boss AI Agent use case.

- [ ] **Step 3: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add Tool Reference section to Boss AI Agent"
```

---

### Task 3: Scenarios 1-3 (Daily Cycle, Project Patrol, Briefing)

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Scenario 1 — Daily Management Cycle**

Must include:
- **Trigger**: `[cron]` at configured `schedule.checkin` time (default 9 AM weekdays)
- **Check-in flow**: load config → get mentor → get culture per employee → `[memory_search]` employee history → personalize questions → `[message send]` to each employee
- **Chase flow**: `[cron]` at `schedule.chase` → `[message read]` check replies → identify non-responders → apply mentor chase strategy + culture override → `[message send]` reminders → `[memory]` record chase events
- **Summary flow**: `[cron]` at `schedule.summary` → collect all replies → `[memory_search]` historical trends → generate mentor-perspective summary (submission rate, highlights, concerns, recommended 1:1s) → `[message send]` to boss → `[memory]` store summary
- **Empty team guard**: skip all if `team` is empty

- [ ] **Step 2: Write Scenario 2 — Project Health Patrol**

Must include:
- **Trigger**: boss says "check project status" / "项目状态" OR `[cron]` weekly Monday
- **Sub-agent dispatch**: `[sessions_spawn]` 3 parallel agents (github-scanner, pm-scanner, chat-scanner) with prompt templates from spec
- **Aggregation**: collect results, deduplicate, apply mentor risk framework
- **Report format**: structured output with severity levels, findings, recommended actions
- **Failure handling**: skip unavailable sources with ⚠️ note
- **Conditional**: only spawn agents for enabled integrations (check config)

- [ ] **Step 3: Write Scenario 3 — Smart Daily Briefing**

Must include:
- **Trigger**: boss says "what's important today" / "今天有什么重要的" OR `[cron]` at `schedule.briefing`
- **Data gathering**: `[message read]` unread important messages → `[web_fetch]` calendar/email → optional `[web_search]` industry news → `[memory_search]` historical context
- **Priority framework**: sort by mentor priority (Musk: blockers first, Inamori: people first, Ma: customer first)
- **Output format**: concise briefing with numbered items, severity tags

- [ ] **Step 4: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add scenarios 1-3 (daily cycle, project patrol, briefing)"
```

---

### Task 4: Scenarios 4-7 (1:1, Signal Scan, KB, Emergency)

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Scenario 4 — 1:1 Meeting Assistant**

Must include:
- **Trigger**: boss says "1:1 with {name}" / "和{name}做1:1"
- **Data collection**: `[memory_search]` employee last 30 days → `[web_fetch]` GitHub/Linear contributions (if integration enabled) → `[message search]` employee channel sentiment
- **Output**: prep document with performance trends, mood shifts, blockers history, suggested topics, conversation strategy per mentor framework

- [ ] **Step 2: Write Scenario 5 — Periodic Signal Scanning**

Must include:
- **Trigger**: `[cron]` at `schedule.signalScan` (default every 30min during work hours)
- **Process**: `[message read]` recent messages from team channels → keyword + sentiment detection → classify signals (🔴🟡🟢) → `[memory]` record significant signals
- **Alert threshold**: 2+ red signals in 1 hour → `[message send]` alert boss
- **Manual trigger**: boss can say "scan channels" anytime
- **Keywords**: from `config.alerts.urgentKeywords` + built-in patterns

- [ ] **Step 3: Write Scenario 6 — Knowledge Base Management**

Must include:
- **Trigger**: boss says "record this decision" / "update Notion" / "记下来"
- **Process**: `[web_fetch]` connect to configured knowledge base (Notion/Sheets) → `[write]` generate structured content → update KB → `[memory]` index for future reference
- **Supported backends**: Notion (via MCP/API), Google Sheets (via MCP/API), local markdown via `[write]`

- [ ] **Step 4: Write Scenario 7 — Emergency Response**

Must include:
- **Trigger**: detected via signal scanning OR employee direct message with urgent keywords
- **Immediate alert**: `[message send]` to boss on preferred channel, fallback chain: preferred → all channels → `[nodes notify]`
- **Intel gathering**: `[sessions_spawn]` rapid investigation agents
- **Response plan**: generate mentor-recommended actions (Musk: act fast, Inamori: stabilize people, Ma: turn crisis into opportunity)
- **Execution**: after boss approves → `[message send]` to relevant people

- [ ] **Step 5: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add scenarios 4-7 (1:1, signal scan, knowledge base, emergency)"
```

---

### Task 5: Mentor System

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Mentor System header and architecture**

Explain the 3-tier system:
- 3 fully-embedded with complete decision matrices
- 6 standard with check-in questions + tags
- 5 light-touch with tags only
- Cloud extension via `POST /api/v1/openclaw/command`

- [ ] **Step 2: Write 3 complete Mentor Decision Matrices**

Copy from spec — Musk, Inamori, Ma (马云). Each matrix covers 7 decision points:
check-in questions, chase intensity, risk assessment, project patrol focus, info priority, 1:1 advice, emergency style.

Additionally, write full check-in question sets (3 questions each) for all 3:
- Musk: "What did you push forward today? Any breakthroughs?" / "What process or blocker can we eliminate?" / "If you had half the time, what would you do?"
- Inamori: "What did you contribute to the team today?" / "Any difficulties you need help with?" / "What did you learn from today's work?"
- Ma: "How did you help a teammate or customer today?" / "What change did you embrace?" / "What's your biggest learning?"

- [ ] **Step 3: Write 6 Standard Mentor entries**

Each with: ID, name, 3 check-in questions, core tags. Copy from spec:
dalio, grove, ren, son, jobs, bezos

- [ ] **Step 4: Write 5 Light-touch Mentor entries**

Each with: ID, name, core tags. Agent infers behavior from tags:
buffett, zhangyiming, leijun, caodewang, chushijian

- [ ] **Step 5: Write Mentor Blending rules**

- How to blend two mentors (weight 50-90%)
- Questions: merge from both
- Decision framework: primary leads, secondary supplements
- Cloud override when available

- [ ] **Step 6: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add mentor system (3 full + 6 standard + 5 light-touch + blending)"
```

---

### Task 6: Cultural Adaptation + Cloud API

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Cultural Adaptation section**

7 culture packs (including default) with table from spec:
- Each culture: directness, hierarchy, key rules
- Override rule: culture > mentor when conflict
- Examples: Filipino employee with Dalio mentor → private feedback (culture overrides radical transparency)
- Chase adaptations per culture

- [ ] **Step 2: Write Cloud API section**

Optional section — only active when `BOSS_AI_AGENT_API_KEY` (or fallback `MANAGEMENT_BRAIN_API_KEY`) is set.

Document:
- Base URL: `BOSS_AI_AGENT_URL` env var or `https://api.manageaibrain.com`
- Auth: `Authorization: Bearer {apiKey}`
- Available endpoints:
  - `GET /api/v1/openclaw/status` — team status
  - `GET /api/v1/openclaw/report?period=weekly|monthly` — rankings
  - `POST /api/v1/openclaw/command` — execute commands (switch mentor, list mentors, list employees)
  - `GET /api/v1/openclaw/alerts` — anomaly alerts
- Mentor fetch: `POST /api/v1/openclaw/command {"command": "list mentors"}` returns full mentor configs
- When to use: prefer cloud API for mentor configs, team data, and analytics when available

- [ ] **Step 3: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add cultural adaptation and cloud API sections"
```

---

### Task 7: Chinese section + Links + Response Formatting

**Files:**
- Modify: `openclaw-skill/SKILL.md`

- [ ] **Step 1: Write Response Formatting section**

Rules for how Boss AI Agent formats output:
- Team status: concise summary with submission rate, pending list, alerts
- Rankings: table format with medals (🥇🥈🥉)
- Alerts: severity tags (🔴 critical, 🟡 warning, 🟢 info)
- Briefings: numbered list, most important first
- 1:1 prep: structured document with sections
- When mentor is switched: explain what changes

- [ ] **Step 2: Write 中文说明 section**

Brief Chinese summary of Boss AI Agent:
- 定位：老板的 AI 管理中间件
- 7 大场景概述
- 14 位导师 + 7 套文化包
- 安装和使用方式
- 云平台可选
- Keep concise (~30-50 lines) — link to full docs for details

- [ ] **Step 3: Write Links section**

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
- ClawHub: https://clawhub.ai/tonypk/boss-ai-agent

- [ ] **Step 4: Final line count check**

Run: `wc -l openclaw-skill/SKILL.md` — expect 800-1200 lines

- [ ] **Step 5: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add response formatting, Chinese docs, and links"
```

---

### Task 8: Write README.md

**Files:**
- Create: `openclaw-skill/README.md` (overwrite existing)

- [ ] **Step 1: Write user-facing README**

Structure:
```markdown
# Boss AI Agent 🤖

Your AI management middleware — connects you to all systems through mentor wisdom.

## Features
- 7 automated management scenarios
- 14 mentor philosophies
- 7 culture packs
- Works with 23+ messaging platforms
- Zero external dependency (OpenClaw native)

## Install
\`\`\`
clawhub install boss-ai-agent
\`\`\`

## Quick Start
\`\`\`
/boss-ai-agent
> How many people do you manage? 5
> What tools? Telegram and GitHub
> Done! Musk mode activated. First check-in at 9 AM tomorrow.
\`\`\`

## Mentors
[table of 14 mentors with ID, name, style — 1 line each]

## Scenarios
[brief 1-line description of each scenario]

## Cloud Platform (Optional)
Connect to manageaibrain.com for web dashboard and full mentor configs.

## 中文说明
[brief Chinese summary]

## Links
```

README should be ~100-150 lines. NO agent instructions — those stay in SKILL.md only.

- [ ] **Step 2: Commit**

```bash
git add openclaw-skill/README.md
git commit -m "feat: add Boss AI Agent user-facing README"
```

---

### Task 9: Publish to ClawHub

**Files:**
- No file changes, CLI operation only

- [ ] **Step 1: Verify ClawHub login**

Run: `clawhub whoami`
Expected: `tonypk`

- [ ] **Step 2: Publish as new skill**

Run:
```bash
clawhub publish openclaw-skill \
  --slug boss-ai-agent \
  --name "Boss AI Agent" \
  --version 1.0.0 \
  --changelog "Initial release: 7 scenarios, 14 mentors, 7 culture packs, OpenClaw native-first"
```

Expected: `OK. Published boss-ai-agent@1.0.0`

- [ ] **Step 3: Verify publication**

Run: `clawhub inspect boss-ai-agent`
Expected: shows version 1.0.0, owner tonypk

- [ ] **Step 4: Final commit with version tag**

```bash
git add -A
git commit -m "chore: publish boss-ai-agent@1.0.0 to ClawHub"
git tag boss-ai-agent-v1.0.0
```

---

## Summary

| Task | Description | Est. Lines | Commit |
|------|-------------|-----------|--------|
| 1 | Frontmatter + Identity + First Run | ~100 | `feat: scaffold Boss AI Agent SKILL.md` |
| 2 | Tool Reference | ~120 | `feat: add Tool Reference section` |
| 3 | Scenarios 1-3 | ~180 | `feat: add scenarios 1-3` |
| 4 | Scenarios 4-7 | ~150 | `feat: add scenarios 4-7` |
| 5 | Mentor System | ~200 | `feat: add mentor system` |
| 6 | Cultural Adaptation + Cloud API | ~80 | `feat: add cultural adaptation and cloud API` |
| 7 | Response Formatting + Chinese + Links | ~80 | `feat: add response formatting, Chinese docs` |
| 8 | README.md | ~120 | `feat: add Boss AI Agent README` |
| 9 | Publish to ClawHub | 0 | `chore: publish boss-ai-agent@1.0.0` |
| **Total** | | **~1030** | **9 commits** |
