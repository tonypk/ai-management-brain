---
name: boss-ai-agent
title: "Boss AI Agent"
version: "2.5.0"
description: "Boss AI Agent — your AI management middleware. 16 mentor philosophies, 6 AI C-Suite seats, 9 culture packs, 7 automated scenarios, real-time dashboard with ECharts analytics. Works with Claude Code, ChatGPT, and Gemini via MCP."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    optional:
      env:
        - name: "BOSS_AI_AGENT_API_KEY"
          description: "Optional. Makes read-only GET requests to manageaibrain.com/api/v1/ for mentor configs and analytics dashboards. API key sent as auth header only — no local files, memory, or chat history are sent. Without it, all 7 scenarios work locally with no degradation."
        - name: "MANAGEMENT_BRAIN_API_KEY"
          description: "Legacy fallback for BOSS_AI_AGENT_API_KEY. Accepted for backward compatibility."
    requires:
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---

# Boss AI Agent

Your AI management middleware — connects you to all systems through mentor wisdom.

## Features

- **7 automated scenarios**: daily check-in cycle, project health patrol, smart briefing, 1:1 meeting prep, signal scanning, knowledge base, emergency response
- **16 mentor philosophies**: Musk, Inamori, Jack Ma, Dalio, Grove, Ren Zhengfei, Son, Jobs, Bezos, Buffett, Zhang Yiming, Lei Jun, Cao Dewang, Chu Shijian, Erin Meyer, Jack Trout
- **6 AI C-Suite seats**: CEO, CFO, CMO, CTO, CHRO, COO — virtual board discussions for strategic decisions
- **9 culture packs**: default, Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India
- **Multi-client MCP**: works with Claude Code (stdio), ChatGPT (HTTP), and Gemini (HTTP)
- **23+ messaging platforms**: works with any channel connected to OpenClaw
- **Zero external dependency**: fully functional without any cloud account

## How It Works

Boss AI Agent connects to your team through **OpenClaw's existing integrations** (Telegram, Slack, GitHub, etc.). It does NOT store or manage tokens for external services — all service access is inherited from OpenClaw's configured connections. If a service is not connected in OpenClaw, the corresponding feature is simply skipped.

**Data flow**: All 13 MCP tools are hosted on `manageaibrain.com/mcp`. Tool parameters (employee names, discussion topics, message content) are sent to the cloud server for processing. The server stores team data (check-ins, reports, employee profiles) in PostgreSQL. Write tools (`send_checkin`, `chase_employee`, `send_summary`, `send_message`) deliver messages to employees via Telegram/Slack/Lark/Signal. Local files (`config.json`, chat history, memory) are never sent. The optional cloud API key (`BOSS_AI_AGENT_API_KEY`) adds read-only access to mentor configs and analytics dashboards — removing the key stops that communication. All 7 scenarios work fully without it.

**Persistent behavior**: The skill registers up to 5 cron jobs (checkin, chase, summary, briefing, signalScan) via OpenClaw's cron API. Solo founder mode (team=0) only registers 2 (briefing + signalScan). Manage jobs:
- View: `cron list`
- Remove one: `cron remove <job-id>`
- Remove all: `cron remove --skill boss-ai-agent`
- Uninstall: `clawhub uninstall boss-ai-agent` cleans up all jobs automatically.

## Install

```
clawhub install boss-ai-agent
```

## Quick Start

```
/boss-ai-agent
> How many people do you manage? 5
> What communication tools does your team use? Telegram and GitHub
> Done! Musk mode activated. First check-in at 9 AM tomorrow.
```

## Mentors

| ID | Name | Style |
|----|------|-------|
| musk | Elon Musk | First principles, urgency, 10x thinking |
| inamori | Kazuo Inamori (稻盛和夫) | Altruism, respect, team harmony |
| ma | Jack Ma (马云) | Embrace change, teamwork, customer-first |
| dalio | Ray Dalio | Radical transparency, principles-driven |
| grove | Andy Grove | OKR-driven, data-focused, high output |
| ren | Ren Zhengfei (任正非) | Wolf culture, self-criticism, striver-oriented |
| son | Masayoshi Son (孙正义) | 300-year vision, bold bets |
| jobs | Steve Jobs | Simplicity, excellence pursuit |
| bezos | Jeff Bezos | Day 1 mentality, customer obsession |
| buffett | Warren Buffett | Long-term value, patience |
| zhangyiming | Zhang Yiming (张一鸣) | Delayed gratification, data-driven |
| leijun | Lei Jun (雷军) | Extreme value, user participation |
| caodewang | Cao Dewang (曹德旺) | Industrial spirit, craftsmanship |
| chushijian | Chu Shijian (褚时健) | Ultimate focus, resilience |
| meyer | Erin Meyer (艾琳·梅耶尔) | Cross-cultural communication, culture map |
| trout | Jack Trout (杰克·特劳特) | Positioning theory, brand strategy |

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

Ask: *"Should we expand to Japan?"* → All 6 seats analyze from their perspective, followed by a unified synthesis.

## MCP Server

13 tools accessible via MCP for Claude Code, ChatGPT, and Gemini:

### Read Tools

| Tool | Description |
|------|-------------|
| `get_team_status` | Today's check-in progress |
| `get_report` | Weekly/monthly performance with rankings |
| `get_alerts` | Alerts for consecutive missed check-ins |
| `switch_mentor` | Change management philosophy |
| `list_mentors` | All 16 mentors with expertise |
| `board_discuss` | AI C-Suite board meeting |
| `chat_with_seat` | Direct chat with one C-Suite exec |
| `list_employees` | All active employees |
| `get_employee_profile` | Employee sentiment and history |

### Write Tools (sends messages)

| Tool | Description |
|------|-------------|
| `send_checkin` | Trigger check-in questions for all or one employee |
| `chase_employee` | Chase reminders for non-submitters |
| `send_summary` | Generate and send daily summary to boss |
| `send_message` | Send custom message to an employee |

**MCP endpoint**: All 13 tools are hosted at `https://manageaibrain.com/mcp`. Claude Code connects via stdio; ChatGPT/Gemini connect via MCP HTTP to this URL. Tool parameters are processed on the server.

## Scenarios

1. **Daily Management Cycle** — AI sends check-ins, chases non-responders, generates end-of-day summary
2. **Project Health Patrol** — scans GitHub/Linear/Jira for stale PRs, failed CI, overdue tasks
3. **Smart Daily Briefing** — cross-channel morning briefing sorted by mentor priority
4. **1:1 Meeting Assistant** — auto-generates prep document with employee data and suggested topics
5. **Periodic Signal Scanning** — monitors team channels every 30 min for urgent/warning signals
6. **Knowledge Base Management** — records decisions and notes to Notion, Google Sheets, or local files
7. **Emergency Response** — detects critical signals, alerts boss immediately, gathers rapid intel

## Web Dashboard (v2.0)

Professional management dashboard at [manageaibrain.com](https://manageaibrain.com), built with NaiveUI + ECharts:

- **Health Gauge** — real-time team health score (red/yellow/green)
- **Check-in Status Panel** — live submitted/pending/missed with chase count
- **Submission Trend Chart** — 7-day bar+line dual-axis (count + rate%)
- **Sentiment Heatmap** — employee × sentiment color matrix
- **Alert Center** — active alerts with severity badges + alert rules
- **Employee Activity Table** — sortable/filterable 7-day activity with missed highlight
- **Report Browser** — date-navigable daily summaries with Q&A expand
- **Settings** — tenant, channels, scheduler, API keys, billing (5 tabs)

## Cloud Platform (Optional)

Connect to manageaibrain.com for additional features:

- Web dashboard with ECharts analytics
- Full mentor configs for all 16 mentors
- AI C-Suite virtual board discussions
- Cross-team benchmarking

Set `BOSS_AI_AGENT_API_KEY` to enable. All 7 scenarios work without it.

## 中文说明

Boss AI Agent 是老板的 AI 管理中间件。通过 OpenClaw 连接已有的沟通工具（Telegram/Slack/飞书等），零外部依赖即可管理团队。

**核心功能：**
- 7 大自动化场景 — 签到、巡检、早报、1:1、信号扫描、知识库、紧急响应
- 16 位导师哲学 — 马斯克、稻盛和夫、马云、达利欧、格鲁夫、任正非、孙正义、乔布斯、贝索斯、巴菲特、张一鸣、雷军、曹德旺、褚时健、梅耶尔、特劳特
- 6 位 AI C-Suite 高管 — CEO/CFO/CMO/CTO/CHRO/COO 虚拟董事会
- 9 套文化包 — 适配菲律宾、新加坡、中国、美国、印度等文化差异
- 多客户端 — Claude Code (stdio) + ChatGPT/Gemini (MCP HTTP)
- 全新管理台 (v2.0) — NaiveUI + ECharts 仪表盘，健康仪表、提交趋势、情绪热力图、预警中心

**数据安全：** 所有场景无需云端即可运行。可选的 API Key 仅从云端拉取导师配置和分析数据，不会上传任何本地数据（消息、文件、记忆）。外部服务（GitHub/Jira/Notion）的访问通过 OpenClaw 已有的集成，本技能不存储或管理任何外部服务令牌。

安装：`clawhub install boss-ai-agent`

**从 management-brain 迁移？** Boss AI Agent 是全新技能，旧数据不受影响。安装后运行 `/boss-ai-agent` 即可。旧环境变量 `MANAGEMENT_BRAIN_API_KEY` 仍然兼容。

## Links

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
- ClawHub: https://clawhub.ai/tonypk/boss-ai-agent
