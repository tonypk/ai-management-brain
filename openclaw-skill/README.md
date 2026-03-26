---
name: boss-ai-agent
title: "Boss AI Agent"
version: "5.1.0"
description: "Boss AI Agent — your AI management advisor. 16 mentor philosophies, 9 culture packs, C-Suite board simulation, execution intelligence engine, AI recommendation engine. Works instantly after install. Connect manageaibrain.com MCP for full team automation: auto check-ins, tracking, KPI metrics, task management, risk signals, incentive scoring, AI recommendations, 23+ platform messaging. Integrates with OpenClaw MCP connectors (Notion, Jira, GitHub, Slack, etc.) to build a company context layer — the foundation for all management intelligence."
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

- **24 MCP tools**: real team queries, execution intelligence, AI recommendations, and message delivery via `manageaibrain.com/mcp`
- **11 automated scenarios**: daily check-in cycle, project health patrol, smart briefing, 1:1 meeting prep, signal scanning, knowledge base, emergency response, execution risk review, KPI health check, incentive review, AI recommendations
- **6 cron jobs**: automated check-ins, chases, summaries, briefings, signal scans, daily recommendation scan
- **23+ messaging platforms**: Telegram, Slack, Lark, Signal, and more via OpenClaw
- **OpenClaw connector integration**: storage tools (Notion/Jira/Sheets), dev tools (GitHub/Linear/Calendar), communication tools (Slack/Discord/Lark) feed data into the company context layer
- **AI Recommendation Engine**: daily scans + real-time triggers generate prioritized management suggestions with one-click actions
- **Web Dashboard**: real-time analytics at [manageaibrain.com](https://manageaibrain.com) — 20+ pages including Dashboard, Company State, KPI Metrics, Projects, Tasks, Incentives, Recommendations, Goals, Reviews, Skills, Training, Career Paths

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

## OpenClaw Integration Architecture

Boss AI Agent is the **brain layer** — it works with OpenClaw's MCP connector ecosystem to build a complete management intelligence system.

```
OpenClaw Runtime (user environment)
  ├── MCP Connectors (user self-installs)
  │    ├── Storage: Notion / Jira / Google Sheets
  │    ├── Development: GitHub / Linear / Calendar
  │    └── Communication: Telegram / Slack / Discord / Lark
  │
  └── Boss AI Agent Skill (brain layer)
       └── manageaibrain.com API
            ├── Company Context Layer  ← foundation
            ├── Execution Intelligence ← signals + risks
            ├── Communication Parser   ← messages → events
            ├── Incentive Engine       ← context-aware scoring
            └── AI Recommendations     ← proactive suggestions
```

**Company Context Layer** is the foundation — all intelligence engines depend on it:
- Storage connectors (Notion/Jira/Sheets) → project updates, task status, documentation flow into context
- Dev connectors (GitHub/Linear) → PR activity, commit patterns, CI status feed into execution signals
- Communication connectors (Telegram/Slack/Discord/Lark) → employee messages are parsed into structured management events

**Key principle**: the skill does NOT manage tokens for external tools — OpenClaw handles all tool connections. The skill consumes the data and builds management intelligence on top.

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

24 tools accessible via MCP:

### Read Tools — Daily Operations (9)

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

### Read Tools — Execution Intelligence (9)

| Tool | Description |
|------|-------------|
| `get_company_state` | Full operational snapshot: risks, tasks, events, blocked projects |
| `get_execution_signals` | AI risk signals: overload, delivery, engagement, anomalies |
| `get_communication_events` | Structured events from check-ins: blockers, completions, delays |
| `get_top_risks` | Highest-severity execution risks sorted by urgency |
| `get_working_memory` | AI situational awareness: focus areas, momentum, action items |
| `get_kpi_dashboard` | All KPI metrics with latest values vs targets |
| `get_overdue_tasks` | Tasks past due date with priority and assignee |
| `get_task_stats` | Task status breakdown: todo, in_progress, done, blocked |
| `get_incentive_scores` | Per-employee incentive scores with breakdowns |

### Write Tools (4 — sends messages)

| Tool | Description |
|------|-------------|
| `send_checkin` | Trigger check-in questions |
| `chase_employee` | Chase reminders for non-submitters |
| `send_summary` | Generate and send daily summary |
| `send_message` | Send custom message to an employee |

### AI Recommendations (2)

| Tool | Description |
|------|-------------|
| `get_recommendations` | Get pending AI management suggestions with priority, evidence, actions |
| `execute_recommendation` | Execute a suggested action (send message, schedule meeting, etc.) |

**MCP endpoint**: `https://manageaibrain.com/mcp`

## Web Dashboard (Team Operations Mode)

Professional management dashboard at [manageaibrain.com](https://manageaibrain.com), built with Vue3 + NaiveUI:

- **Observe**: Dashboard (health gauge, check-in status, submission trend, sentiment heatmap, AI recommendations summary), Company State, KPI Metrics, Alerts, Reports
- **Organize**: Team Members, Organization, Projects, Tasks, Skill Inventory, Mentor, C-Suite Board
- **Lead**: Sentiment Map, 1:1 Coaching, 1:1 Meetings, Reviews, Incentives, Training, Career Paths
- **Plan**: Board Records, Goals & KPIs
- **Analyze**: AI Insights, Weekly Digest, AI Recommendations (with one-click execute/dismiss)

## 中文说明

Boss AI Agent 是老板的 AI 管理中间件。安装后立即可用（Advisor 模式），无需注册账号。

**两种模式：**
- **顾问模式**（零依赖）— 16 位导师哲学框架（稻盛和夫、马云、马斯克等）、9 套文化包（中国、菲律宾、新加坡等）、C-Suite 董事会模拟、1:1 准备、管理决策建议。装了就能用，不联网。
- **团队运营模式**（连接 MCP）— 24 个 MCP 工具实现自动签到、追踪、报表、消息推送、执行力分析、KPI 仪表盘、任务管理、激励评分、AI 推荐引擎，6 个定时任务，23+ 平台支持。

**OpenClaw 集成架构（v5.1 新增）：** Boss AI Agent 作为"大脑层"，与 OpenClaw MCP 连接器配合使用：
- **储存工具**（Notion / Jira / Sheets）→ 项目、任务、文档自动汇入公司上下文
- **开发工具**（GitHub / Linear / Calendar）→ PR、提交、CI 状态转化为执行力信号
- **沟通工具**（Telegram / Slack / Discord / Lark）→ 员工消息解析为结构化管理事件

**公司上下文层**是所有智能引擎的地基，执行力分析、AI 推荐、激励评分都依赖它。

**AI 推荐引擎（v5.0 新增）：** 每日 10:30 自动扫描团队数据，结合导师视角生成管理建议（如：连续缺勤提醒、情绪下降预警、任务逾期跟进）。支持一键执行建议动作，也可通过实时触发器即时生成。

**数据说明：** 顾问模式不发送任何数据到云端。团队运营模式中，MCP 工具参数发送至 `manageaibrain.com` 处理。外部工具通过 OpenClaw 连接器访问，Skill 不管理令牌。

安装：`clawhub install boss-ai-agent`

## Links

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
- ClawHub: https://clawhub.ai/tonypk/boss-ai-agent
