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
