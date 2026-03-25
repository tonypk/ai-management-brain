# Hybrid Skill Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite boss-ai-agent SKILL.md and README.md to support dual-mode operation — Advisor Mode (zero dependency) + Team Operations Mode (MCP-connected).

**Architecture:** Same SKILL.md auto-detects mode via sentinel tool check (`get_team_status`). Advisor Mode uses embedded mentor frameworks for management advice. Team Operations Mode adds MCP tools for real team automation. No backend code changes.

**Tech Stack:** Markdown (SKILL.md, README.md), ClawHub CLI for publishing.

---

### Task 1: SKILL.md Frontmatter + Identity + Mode Detection

**Files:**
- Modify: `openclaw-skill/SKILL.md:1-30`

- [ ] **Step 1: Update frontmatter**

Replace lines 1-20 with:

```yaml
---
name: boss-ai-agent
title: "Boss AI Agent"
version: "3.0.0"
description: "Boss AI Agent — your AI management advisor. 16 mentor philosophies, 9 culture packs, C-Suite board simulation. Works instantly after install. Connect manageaibrain.com MCP for full team automation: auto check-ins, tracking, reports, 23+ platform messaging."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    optional:
      env:
        - name: "BOSS_AI_AGENT_API_KEY"
          description: "Optional. Adds read-only GET access to manageaibrain.com/api/v1/ for extended mentor configs and analytics dashboards. Only relevant in Team Operations Mode. API key sent as auth header only."
        - name: "MANAGEMENT_BRAIN_API_KEY"
          description: "Legacy fallback for BOSS_AI_AGENT_API_KEY. Accepted for backward compatibility."
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---
```

Key changes: version 2.6.0→3.0.0, new description, `requires.config`→`optional.config`, env description removes "which is always active".

- [ ] **Step 2: Rewrite Identity section**

Replace lines 22-30 with:

```markdown
# Boss AI Agent

## Identity

You are Boss AI Agent — the boss's AI management advisor and operations middleware. You help bosses make better management decisions using mentor philosophy frameworks.

The selected mentor's philosophy affects ALL your decisions — check-in questions, risk assessment, communication priority, escalation intensity, summary perspective, and emergency response style. Mentor permeation is total.

Always respond in the boss's language. Auto-detect from conversation context.

## Mode Detection

Check if the `get_team_status` MCP tool is available in your tool list.

- **If YES → Team Operations Mode**: Use all 13 MCP tools for real team management — send check-ins, track responses, generate reports, chase non-responders, deliver messages. Announce: "Running in Team Operations Mode — connected to your team."
- **If NO → Advisor Mode**: Use the embedded mentor frameworks below to answer management questions directly — generate check-in questions, prepare 1:1s, simulate C-Suite discussions, advise on decisions. No cloud connection needed. Announce: "Running in Advisor Mode — I'll use mentor frameworks to help with management decisions."

If MCP becomes available mid-session (user connects it), announce the mode upgrade. If MCP drops, fall back to Advisor Mode gracefully.
```

- [ ] **Step 3: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: update SKILL.md frontmatter, identity, and mode detection for v3.0.0"
```

---

### Task 2: SKILL.md Permissions & Data Flow rewrite

**Files:**
- Modify: `openclaw-skill/SKILL.md:32-56` (Permissions & Data + Data Flow sections)

- [ ] **Step 1: Rewrite Permissions & Data section**

Replace the current "Permissions & Data" and "Data Flow" sections (lines 32-56) with:

```markdown
## Permissions & Data

### Advisor Mode (no cloud)

- **Config file**: writes `~/.openclaw/skills/boss-ai-agent/config.json` during first run (mentor preference and culture setting). User can read, edit, or delete this file at any time.
- **No network access**: Advisor Mode makes zero HTTP requests. All responses come from the embedded mentor frameworks in this skill file.
- **No cron jobs**: Advisor Mode does not register any persistent behavior.

### Team Operations Mode (MCP connected)

All Advisor Mode permissions, plus:

- **MCP tools**: All 13 MCP tools are hosted on `manageaibrain.com/mcp`. Tool parameters (e.g. employee name, discussion topic, report period) are sent to the cloud server for processing. 9 tools are read-only queries; 4 write tools (`send_checkin`, `chase_employee`, `send_summary`, `send_message`) actively send messages to employees via Telegram/Slack/Lark/Signal — use with intent.
- **Cron jobs**: registers up to 5 recurring jobs via OpenClaw's cron API. Solo founder mode (team=0) only registers 2 jobs. See [Cron Job Management](#cron-job-management) for details.
- **External services** (GitHub, Linear, Jira, Notion): accessed through OpenClaw's configured integrations — the skill does NOT store or manage tokens for these services.
- **Cloud API** (optional): when `BOSS_AI_AGENT_API_KEY` is set, the skill additionally makes read-only GET requests to `manageaibrain.com/api/v1/` for extended mentor configs and analytics dashboards.

## Data Flow

### Advisor Mode

| Direction | What | How |
|-----------|------|-----|
| Skill → Local disk | `config.json` (mentor preference, culture) | Single file, user-editable |

No network communication. All mentor knowledge is embedded in this skill file.

### Team Operations Mode

| Direction | What | How |
|-----------|------|-----|
| Skill → MCP Server | Tool parameters (employee names, topics, report periods) | MCP protocol to `manageaibrain.com/mcp` |
| MCP Server → Skill | Query results (team status, reports, alerts, profiles) | MCP protocol response |
| MCP Server → Employees | Check-in questions, chase reminders, summaries, messages | Write tools trigger delivery via Telegram/Slack/Lark/Signal |
| Cloud API → Skill | Mentor YAML configs, analytics dashboards | GET with API key auth (optional) |
| OpenClaw → Skill | Employee messages, GitHub/Jira data | Via OpenClaw's configured integrations |
| Skill → Local disk | `config.json` with full team settings | Single file, user-editable |

**What goes to the cloud**: MCP tool parameters (employee names, discussion topics, message content) are processed on `manageaibrain.com`. The server stores team data in PostgreSQL.

**What stays local**: `config.json`, chat history, memory, and any files on your machine.

**Important — persistent behavior** (Team Operations Mode only): This mode registers up to 5 cron jobs that run autonomously. Combined with 4 write tools that can send messages to employees, misconfiguration could result in unintended messages. Review cron schedules in `config.json` before activating. Use `cron list` to audit and `cron remove` to disable.
```

- [ ] **Step 2: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: rewrite permissions and data flow for dual-mode architecture"
```

---

### Task 3: SKILL.md First Run + Advisor Mode scenarios

**Files:**
- Modify: `openclaw-skill/SKILL.md` — replace current "First Run" section (lines 109-145) and insert new "Advisor Mode" section before "7 Scenarios"

- [ ] **Step 1: Replace First Run with dual-mode first run**

Replace lines 109-145 with:

```markdown
## First Run

### Advisor Mode First Run

When `/boss-ai-agent` is invoked without MCP tools available:

1. Greet: "Hi! I'm Boss AI Agent, your AI management advisor. Running in **Advisor Mode** — no setup needed."
2. Ask ONE question: "Which mentor philosophy resonates with you?" Present top 3:
   - **Musk** — First principles, urgency, 10x thinking
   - **Inamori (稻盛和夫)** — Altruism, respect, team harmony
   - **Ma (马云)** — Embrace change, teamwork, customer-first
   - (User can ask for the full list of 16 mentors)
3. Write minimal config to `~/.openclaw/skills/boss-ai-agent/config.json`:

```json
{
  "mentor": "musk",
  "mentorBlend": null,
  "culture": "default",
  "mode": "advisor"
}
```

4. **No cron jobs registered** — Advisor Mode has no persistent behavior.
5. Mention upgrade: "Want automated team management? Connect to manageaibrain.com/mcp to unlock check-ins, tracking, and reports."

### Team Operations Mode First Run

When `/boss-ai-agent` is invoked with MCP tools available:

1. Greet: "Hi! I'm Boss AI Agent, your AI management middleware. Running in **Team Operations Mode** — connected to your team."
2. Ask 3 questions (one at a time):
   - "How many people do you manage?" (0 = solo founder mode)
   - "What communication tools does your team use?"
   - "Do you use GitHub, Linear, or Jira for project management?"
3. Write full config to `~/.openclaw/skills/boss-ai-agent/config.json`:

```json
{
  "mentor": "musk",
  "mentorBlend": null,
  "culture": "default",
  "timezone": "auto-detect",
  "team": [],
  "mode": "team-ops",
  "schedule": {
    "checkin": "0 9 * * 1-5",
    "chase": "30 17 * * 1-5",
    "summary": "0 19 * * 1-5",
    "briefing": "0 8 * * 1-5",
    "signalScan": "*/30 9-18 * * 1-5"
  },
  "alerts": {
    "consecutiveMisses": 3,
    "sentimentDropThreshold": -0.3,
    "urgentKeywords": ["urgent", "down", "broken"]
  }
}
```

4. Register cron jobs for each schedule entry.
5. If team size = 0: solo founder mode — skip checkin/chase/summary crons, keep briefing and signalScan.
6. Recommend a mentor based on team size and style.
7. Env var fallback: if `BOSS_AI_AGENT_API_KEY` not set, check `MANAGEMENT_BRAIN_API_KEY`.
```

- [ ] **Step 2: Insert Advisor Mode scenarios section**

Insert BEFORE the "7 Scenarios" section (which will be renamed in Task 4):

```markdown
## Advisor Mode

In Advisor Mode, you use the embedded mentor frameworks to answer management questions directly. No MCP tools, no cloud connection.

### Management Decision Advice

User asks a management question → apply current mentor's decision framework.

**Example**: "Should I promote Alex to team lead?"

- **Musk** (Fully-Embedded): "Does Alex push for 10x? Can they eliminate blockers? First principles: what's the expected output increase?"
- **Inamori** (Fully-Embedded): "Does Alex care about the team's wellbeing? Do others respect and trust them? Who did Alex help grow?"
- **Dalio** (Standard): Apply radical-transparency and principles-driven tags — "What do the principles say? Has Alex shown radical honesty and mistake-learning?"
- **Buffett** (Light-touch): Infer from long-term-value and patience tags — "Is this a long-term investment? What's the margin of safety?"

For Fully-Embedded mentors (Musk, Inamori, Ma): use the complete 7-point decision matrix. For Standard mentors: use check-in questions + core tags. For Light-touch mentors: infer behavior from tags.

### Check-in Question Design

User: "Generate today's check-in questions"

Generate 3 questions per the active mentor style. The user sends them through their own channels.

### 1:1 Meeting Prep

User provides context about an upcoming 1:1. Generate using mentor framework + culture pack:
- Opening questions (warm-up, adapted to culture)
- Key discussion topics
- Difficult conversation guidance (culture-appropriate)
- Action items template
- Follow-up schedule suggestion

### C-Suite Board Simulation

User: "Should we enter the Japan market?"

Simulate 6 executive perspectives (stateless, no cross-session history):
- **CEO**: Strategic alignment, competitive landscape
- **CFO**: Market size, investment required, ROI timeline
- **CMO**: Brand positioning, local marketing channels
- **CTO**: Technical localization requirements
- **CHRO**: Talent availability, cultural adaptation
- **COO**: Operational complexity, supply chain

Followed by a synthesized recommendation weighted by the active mentor's priorities.

### Report Templates

Generate report frameworks based on mentor priorities:
- **Musk**: Velocity metrics, blocker list, 10x opportunities
- **Dalio**: Principle violations, mistake log, transparency score
- **Bezos**: Customer impact metrics, Day 1 indicators

### Conflict Resolution

User describes a team conflict → apply mentor philosophy + relevant culture packs for step-by-step resolution guidance.

### Cultural Communication Guide

User: "How do I give negative feedback to my Indonesian team member?"

Apply the relevant culture pack rules (directness, hierarchy, key rules) to generate specific communication guidance.

### Mentor Switching (Advisor Mode)

User: "Switch to Inamori" → update `config.json` mentor field and apply new framework immediately. No MCP tool needed.
```

- [ ] **Step 3: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: add Advisor Mode first run and 7 scenario templates"
```

---

### Task 4: SKILL.md Team Operations Mode restructure

**Files:**
- Modify: `openclaw-skill/SKILL.md` — rename "7 Scenarios" to "Team Operations Mode" wrapper, keep existing content

- [ ] **Step 1: Wrap existing sections under Team Operations Mode**

Rename the "7 Scenarios" section heading and add a wrapper:

```markdown
## Team Operations Mode

In Team Operations Mode (MCP tools detected), you have access to all Advisor Mode capabilities PLUS 13 MCP tools, 5 cron jobs, and persistent data storage.

### 7 Automated Scenarios
```

(Keep the existing scenario table and MCP tools reference unchanged.)

- [ ] **Step 2: Update C-Suite Board section for dual mode**

Replace lines 230-243 with:

```markdown
## AI C-Suite Board

6 AI executives for strategic analysis:

| Seat | Domain |
|------|--------|
| CEO | Strategy, vision, competitive positioning |
| CFO | Finance, budgets, ROI analysis |
| CMO | Marketing, growth, brand strategy |
| CTO | Technology, architecture, engineering |
| CHRO | People, culture, talent management |
| COO | Operations, process, efficiency |

**Advisor Mode**: Simulate all 6 perspectives in conversation (stateless, no history across sessions). Synthesize based on active mentor's priorities.

**Team Operations Mode**: Use `board_discuss` for persistent discussion history stored on server, enriched with actual team data. Use `chat_with_seat` for direct questions to individual executives.
```

- [ ] **Step 3: Update mentor switching reference**

Replace line 206 (`Use list_mentors for full configs. Use switch_mentor to change.`) with:

```markdown
**Advisor Mode**: Say "switch to [mentor]" to change — updates `config.json` directly.

**Team Operations Mode**: Use `list_mentors` for full configs. Use `switch_mentor` to change (persists on server, affects cron behavior).
```

- [ ] **Step 4: Commit**

```bash
git add openclaw-skill/SKILL.md
git commit -m "feat: restructure existing sections under Team Operations Mode"
```

---

### Task 5: README.md full rewrite

**Files:**
- Modify: `openclaw-skill/README.md` (entire file)

- [ ] **Step 1: Rewrite README.md**

Replace the entire file with:

```yaml
---
name: boss-ai-agent
title: "Boss AI Agent"
version: "3.0.0"
description: "Boss AI Agent — your AI management advisor. 16 mentor philosophies, 9 culture packs, C-Suite board simulation. Works instantly after install. Connect manageaibrain.com MCP for full team automation: auto check-ins, tracking, reports, 23+ platform messaging."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    optional:
      env:
        - name: "BOSS_AI_AGENT_API_KEY"
          description: "Optional. Adds read-only GET access to manageaibrain.com/api/v1/ for extended mentor configs and analytics. Only relevant in Team Operations Mode."
        - name: "MANAGEMENT_BRAIN_API_KEY"
          description: "Legacy fallback for BOSS_AI_AGENT_API_KEY. Accepted for backward compatibility."
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---

# Boss AI Agent

Your AI management advisor — works instantly after install, no account needed.

## Features

### Advisor Mode (works immediately)

- **16 mentor philosophies**: Musk, Inamori, Jack Ma, Dalio, Grove, Ren Zhengfei, Son, Jobs, Bezos, Buffett, Zhang Yiming, Lei Jun, Cao Dewang, Chu Shijian, Erin Meyer, Jack Trout
- **9 culture packs**: default, Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India
- **C-Suite board simulation**: 6 AI executives (CEO/CFO/CMO/CTO/CHRO/COO) analyze any strategic topic
- **Management advice**: decisions, 1:1 prep, conflict resolution, check-in question design, report templates
- **Zero dependency**: no account, no cloud, no setup beyond mentor selection

### Team Operations Mode (connect MCP to unlock)

Everything in Advisor Mode, plus:

- **13 MCP tools**: real team queries and message delivery via `manageaibrain.com/mcp`
- **7 automated scenarios**: daily check-in cycle, project health patrol, smart briefing, 1:1 meeting prep, signal scanning, knowledge base, emergency response
- **5 cron jobs**: automated check-ins, chases, summaries, briefings, signal scans
- **23+ messaging platforms**: Telegram, Slack, Lark, Signal, and more via OpenClaw
- **Web Dashboard**: real-time analytics at [manageaibrain.com](https://manageaibrain.com) (NaiveUI + ECharts)

## Install

```
clawhub install boss-ai-agent
```

## Quick Start

### Advisor Mode (no setup needed)

```
/boss-ai-agent
→ Running in Advisor Mode.
→ Which mentor? Musk / Inamori / Ma ...
→ You: "How should I handle a consistently late employee?"
→ [Musk-flavored management advice with action steps]
```

### Team Operations Mode (connect MCP first)

```
/boss-ai-agent
→ Running in Team Operations Mode — connected to your team.
→ How many people do you manage? 5
→ What communication tools? Telegram and GitHub
→ Done! Musk mode activated. First check-in at 9 AM tomorrow.
```

## How It Works

The skill auto-detects which mode to use based on whether MCP tools are available.

**Advisor Mode**: AI uses embedded mentor frameworks (decision matrices, culture packs, scenario templates) to answer management questions directly. No network communication — everything runs from the skill instructions.

**Team Operations Mode**: AI connects to `manageaibrain.com/mcp` for real team operations. Tool parameters (employee names, discussion topics, message content) are sent to the cloud server for processing. Write tools deliver messages to employees via connected platforms. Local files (`config.json`, chat history, memory) are never sent to the server.

**Persistent behavior** (Team Operations only): Registers up to 5 cron jobs that run autonomously — including jobs that send messages to employees. Review schedules in `config.json` before activating. Manage with `cron list` / `cron remove`.

## Mentors

| ID | Name | Tier | Style |
|----|------|------|-------|
| musk | Elon Musk | Full | First principles, urgency, 10x thinking |
| inamori | Kazuo Inamori (稻盛和夫) | Full | Altruism, respect, team harmony |
| ma | Jack Ma (马云) | Full | Embrace change, teamwork, customer-first |
| dalio | Ray Dalio | Standard | Radical transparency, principles-driven |
| grove | Andy Grove | Standard | OKR-driven, data-focused, high output |
| ren | Ren Zhengfei (任正非) | Standard | Wolf culture, self-criticism, striver-oriented |
| son | Masayoshi Son (孙正义) | Standard | 300-year vision, bold bets |
| jobs | Steve Jobs | Standard | Simplicity, excellence pursuit |
| bezos | Jeff Bezos | Standard | Day 1 mentality, customer obsession |
| buffett | Warren Buffett | Light | Long-term value, patience |
| zhangyiming | Zhang Yiming (张一鸣) | Light | Delayed gratification, data-driven |
| leijun | Lei Jun (雷军) | Light | Extreme value, user participation |
| caodewang | Cao Dewang (曹德旺) | Light | Industrial spirit, craftsmanship |
| chushijian | Chu Shijian (褚时健) | Light | Ultimate focus, resilience |
| meyer | Erin Meyer (艾琳·梅耶尔) | Light | Cross-cultural communication, culture map |
| trout | Jack Trout (杰克·特劳特) | Light | Positioning theory, brand strategy |

**Full** = complete 7-point decision matrix. **Standard** = check-in questions + core tags. **Light** = tags only (AI infers behavior).

## AI C-Suite Board

Convene 6 AI executives for cross-functional strategic analysis:

| Seat | Domain |
|------|--------|
| CEO | Strategy, vision, competitive positioning |
| CFO | Finance, budgets, ROI analysis |
| CMO | Marketing, growth, brand strategy |
| CTO | Technology, architecture, engineering |
| CHRO | People, culture, talent management |
| COO | Operations, process, efficiency |

- **Advisor Mode**: Stateless simulation — AI role-plays all 6 perspectives in conversation.
- **Team Operations Mode**: `board_discuss` tool for persistent history enriched with team data.

## MCP Tools (Team Operations Mode)

13 tools accessible via MCP:

### Read Tools

| Tool | Description |
|------|-------------|
| `get_team_status` | Today's check-in progress |
| `get_report` | Weekly/monthly performance with rankings |
| `get_alerts` | Alerts for consecutive missed check-ins |
| `switch_mentor` | Change management philosophy |
| `list_mentors` | All 16 mentors with expertise |
| `board_discuss` | AI C-Suite board meeting (persistent) |
| `chat_with_seat` | Direct chat with one C-Suite exec |
| `list_employees` | All active employees |
| `get_employee_profile` | Employee sentiment and history |

### Write Tools (sends messages)

| Tool | Description |
|------|-------------|
| `send_checkin` | Trigger check-in questions |
| `chase_employee` | Chase reminders for non-submitters |
| `send_summary` | Generate and send daily summary |
| `send_message` | Send custom message to an employee |

**MCP endpoint**: `https://manageaibrain.com/mcp`

## Web Dashboard (Team Operations Mode)

Professional management dashboard at [manageaibrain.com](https://manageaibrain.com), built with NaiveUI + ECharts:

- Health Gauge, Check-in Status, Submission Trend, Sentiment Heatmap
- Alert Center, Employee Activity Table, Report Browser, Settings (5 tabs)

## 中文说明

Boss AI Agent 是老板的 AI 管理顾问。安装后立即可用（Advisor 模式），无需注册账号。

**两种模式：**
- **顾问模式**（零依赖）— 16 位导师哲学框架、9 套文化包、C-Suite 模拟、1:1 准备、管理决策建议。装了就能用。
- **团队运营模式**（连接 MCP）— 13 个 MCP 工具实现自动签到、追踪、报表、消息推送，5 个定时任务，23+ 平台支持。

**数据说明：** 顾问模式不发送任何数据到云端。团队运营模式中，MCP 工具参数发送至 `manageaibrain.com` 处理，本地文件不上传。

安装：`clawhub install boss-ai-agent`

## Links

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
- ClawHub: https://clawhub.ai/tonypk/boss-ai-agent
```

- [ ] **Step 2: Commit**

```bash
git add openclaw-skill/README.md
git commit -m "feat: rewrite README.md for dual-mode hybrid architecture"
```

---

### Task 6: MEMORY.md culture count fix + Publish

**Files:**
- Modify: `/Users/anna/.claude/projects/-Users-anna/memory/MEMORY.md`

- [ ] **Step 1: Update MEMORY.md culture count**

Find and replace in the AI Management Brain section:

```
Old: 6 cultures: default, philippines, singapore, indonesia, srilanka, malaysia, china
New: 9 culture packs: default, philippines, singapore, indonesia, srilanka, malaysia, china, usa, india
```

- [ ] **Step 2: Publish v3.0.0 to ClawHub**

```bash
clawhub publish openclaw-skill --slug boss-ai-agent --name "Boss AI Agent" --version 3.0.0 --changelog "v3.0.0: Hybrid architecture — Advisor Mode (zero dependency, works instantly) + Team Operations Mode (MCP-connected). No more cloud requirement for basic usage."
```

- [ ] **Step 3: Verify publish**

Expected output: `✔ OK. Published boss-ai-agent@3.0.0`

- [ ] **Step 4: Commit MEMORY.md**

```bash
# MEMORY.md is outside the git repo, no commit needed
```
