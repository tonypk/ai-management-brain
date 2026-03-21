---
name: management-brain
description: "AI Management OS — zero-setup via your existing Telegram/Slack/Lark bot. 9 mentor philosophies (Inamori/Musk/Jobs...), smart chase, sentiment analysis, 6 culture packs. Data to Notion/Google Sheets/Obsidian. Optional cloud dashboard."
user-invocable: true
---

# AI Management Brain

Your AI-powered Management Operating System. Select a world-class mentor philosophy, and the AI handles daily check-ins, proactive employee chase, sentiment analysis, cultural adaptation, and executive summaries — all through your existing OpenClaw channels.

**Zero setup for basic use.** If you already have Telegram, Slack, or Lark connected to OpenClaw, this skill works immediately. No external account needed.

## How It Works

This skill operates in two modes:

### Mode 1: OpenClaw Native (Default — No Registration Needed)

Uses your existing OpenClaw channels to communicate with employees directly. Data stored locally.

**Just install and start:**
```
/management-brain
> Add employee John, telegram ID 123456, culture: philippines
> Set mentor to Elon Musk
> Send daily check-in to all employees
> Who hasn't reported today?
> Generate today's summary
```

### Mode 2: Cloud Platform (Optional — For Advanced Features)

Connect to manageaibrain.com for web dashboard, analytics, multi-tenant support, and automated scheduling.

Set `MANAGEMENT_BRAIN_API_KEY` to enable. See "Cloud Platform Setup" section below.

---

## What It Does

- **Proactive Check-ins**: AI sends daily questions to employees via your OpenClaw channels (Telegram/Slack/Lark) — using the selected mentor's question style
- **Smart Chase**: When employees don't report, auto-escalates following mentor philosophy (private DM → manager notify → skip)
- **Sentiment Analysis**: Detects mood changes, burnout signals, and engagement drops from daily reports
- **Cultural Adaptation**: Auto-adjusts communication style per employee's cultural background
- **Executive Summaries**: AI-generated daily/weekly summaries focused on what the mentor considers most important
- **Anomaly Alerts**: Proactive warnings for consecutive misses, sentiment drops, and team health issues

## 中文说明

AI Management Brain 是一个 AI 管理操作系统。安装后直接通过你的 OpenClaw 频道（Telegram/Slack/飞书）管理团队，无需注册外部账号。

**两种模式：**
- **原生模式（默认）**：直接使用你已配置好的 OpenClaw 频道与员工沟通，数据存储到你已有的 Notion/Google Sheets/Obsidian（或本地），零注册即用
- **云平台模式（可选）**：连接 manageaibrain.com 获得 Web Dashboard、数据分析、自动调度等高级功能

**核心功能：**
- AI 通过你的 Telegram/Slack/飞书 主动联系员工，按导师风格提问
- 员工未提交时自动追踪升级（私信 → 通知经理 → 标记跳过）
- 情绪分析、文化适配、异常告警
- AI 生成管理层日报/周报

## OpenClaw Native Mode

### First Run — Storage Setup

On first use, ask the user where to store management data. Detect available integrations and recommend accordingly:

| Storage | Best For | Data Format |
|---------|----------|-------------|
| **Notion** (recommended if available) | Teams, searchable, shared access | Database tables: Employees, Reports, Summaries |
| **Google Sheets** | Simple, familiar, easy sharing | Spreadsheet with tabs: Employees, Reports, Summaries |
| **Obsidian** | Personal knowledge base, markdown | Vault folder: `Management Brain/` with daily notes |
| **Google Drive** | File-based, backup-friendly | JSON/Markdown files in a dedicated folder |
| **Local files** (fallback) | No integrations available | `~/.openclaw/skills/management-brain/data/` |

Ask: "Where would you like to store your team management data? I detected you have [Notion/Google Sheets/Obsidian/...] connected."

Store the user's choice in `~/.openclaw/skills/management-brain/config.json`:
```json
{
  "storage": "notion",
  "mentor": "inamori",
  "timezone": "Asia/Singapore"
}
```

### Storage Formats

**Notion**: Create a workspace with 3 databases:
- **Employees** — Name, Channel, Channel ID, Culture, Role, Active
- **Daily Reports** — Date, Employee, Answers, Sentiment, Blockers
- **Summaries** — Date, Period, Mentor, Content, Submission Rate

**Google Sheets**: Create a spreadsheet "Management Brain" with 3 tabs:
- **Employees** — columns: Name | Channel | Channel ID | Culture | Role | Active
- **Reports** — columns: Date | Employee | Q1 | Q2 | Q3 | Sentiment | Blockers
- **Summaries** — columns: Date | Period | Submission Rate | Summary

**Obsidian**: Create folder `Management Brain/` in vault:
- `Employees.md` — table of all employees
- `Reports/YYYY-MM-DD.md` — daily report with all submissions
- `Summaries/YYYY-MM-DD-weekly.md` — summary documents

**Local files** (JSON fallback):
- `data/employees.json`
- `data/reports-{date}.json`
- `data/summaries-{date}.json`

### Managing Employees

When the user wants to add, list, or remove employees:

Supported commands:
- "add employee [name], [channel] ID [id], culture: [code]"
- "list employees"
- "remove employee [name]"
- "set [name]'s culture to [code]"

Write to the configured storage backend. Example employee record:
```json
{
  "name": "John Santos",
  "channel": "telegram",
  "channelId": "123456789",
  "culture": "philippines",
  "role": "member",
  "active": true
}
```

### Sending Check-ins

When the user asks to send check-ins or it's time for daily reports:

1. Load the current mentor's check-in questions
2. For each active employee, use OpenClaw's `sendMessage` action to send questions via their configured channel:
   - Telegram: send DM with check-in questions
   - Slack: send DM with check-in questions
   - Lark: send message with check-in questions
3. Record check-in status in storage

Example message for Inamori mentor:
> Hi John! Daily check-in time. Please answer when you can:
> 1. What did you contribute to the team today?
> 2. Any difficulties you need help with?
> 3. What did you learn from today's work?

### Collecting Reports

When an employee responds, or the user pastes employee responses:

1. Parse answers and map to check-in questions
2. Analyze sentiment (positive/neutral/negative) from the response
3. Extract blockers if mentioned
4. Save report to configured storage

### Chase Logic

When the user asks who hasn't reported, or wants to chase employees:

Follow the current mentor's chase strategy:
1. **Step 1**: Send private reminder via OpenClaw channel (tone per mentor)
2. **Step 2**: After delay, escalate (notify manager or send stronger reminder)
3. **Step 3**: Mark as skipped for the day

Always respect cultural overrides:
- Filipino employees: never name publicly, warmth required
- Singaporean employees: direct but polite
- Chinese employees: face-saving, collective framing

### Generating Summaries

When the user asks for a daily or weekly summary:

1. Read all reports for the period from storage
2. Apply the current mentor's summary focus:
   - Inamori: morale, collaboration, support needs
   - Musk: velocity, blockers removed, breakthrough progress
   - Jobs: product quality, simplicity, innovation
   - etc.
3. Generate a structured summary with: submission rate, key highlights, concerns, recommended 1:1s
4. Save summary to storage

---

## Mentor Reference

Selecting a mentor changes the entire management strategy: check-in questions, chase escalation, summary focus, and AI personality.

| ID | Name | Philosophy | Style |
|----|------|-----------|-------|
| inamori | Kazuo Inamori (稻盛和夫) | Amoeba management | Altruism, respect, team harmony |
| dalio | Ray Dalio | Radical transparency | Principles-driven, mistake analysis |
| grove | Andy Grove | High output management | OKR-driven, data-focused |
| ren | Ren Zhengfei (任正非) | Wolf culture | Self-criticism, striver-oriented |
| son | Masayoshi Son (孙正义) | 300-year vision | Bold bets, time machine theory |
| jobs | Steve Jobs | Insanely great | Simplicity, excellence pursuit |
| bezos | Jeff Bezos | Day 1 mentality | Customer obsession, long-term thinking |
| ma | Jack Ma (马云) | 102-year company | Embrace change, teamwork |
| musk | Elon Musk (马斯克) | First principles | Urgency, 10x thinking, rapid iteration |

### Mentor Check-in Questions

**Inamori**: "What did you contribute to the team today?" / "Any difficulties you need help with?" / "What did you learn?"

**Musk**: "What did you push forward today? Any breakthroughs?" / "What process or blocker can we eliminate?" / "If you had half the time, what would you do?"

**Jobs**: "What did you ship today that you're proud of?" / "What can be made simpler?" / "How far is your work from 'insanely great'?"

**Dalio**: "What decision did you make today? What was your reasoning?" / "What mistake did you make and what did you learn?" / "What principle applies here?"

**Grove**: "What's your OKR progress this week?" / "What's your biggest bottleneck?" / "What output did you deliver?"

**Ren**: "What goal did you accomplish today?" / "What challenge did you overcome?" / "How did you push your limits?"

**Bezos**: "What did you do for the customer today?" / "What would you do differently on Day 1?" / "What data informed your decision?"

**Ma**: "How did you help a teammate today?" / "What change did you embrace?" / "What's your biggest learning?"

**Son**: "What progress did you make toward the big vision?" / "What bold bet are you considering?" / "What did you learn from other industries?"

## Cultural Adaptation

Culture packs override mentor strategy when there's a conflict (e.g., Dalio wants public feedback but Filipino culture requires private-first).

| Culture | Directness | Hierarchy | Key Rules |
|---------|-----------|-----------|-----------|
| philippines | Low | High | Never name in group, warmth required, acknowledge effort |
| singapore | High | Medium | Direct but polite, efficiency-focused |
| indonesia | Low | High | Relationship-first, group harmony |
| srilanka | Low | High | Respectful tone, private feedback |
| malaysia | Medium | Medium | Multicultural sensitivity, balanced approach |
| china | Medium | High | Face-saving, collective achievement |

## Response Formatting

- Present team status as a concise summary, highlighting pending employees and alerts
- Show rankings in a table format with medals (gold, silver, bronze)
- For alerts, indicate severity: warning or critical
- When mentor is switched, explain what changes (questions, chase style, summary focus)

---

## Cloud Platform Setup (Optional)

For advanced features (web dashboard, analytics, automated scheduling, multi-tenant), connect to the cloud platform:

1. Visit https://app.manageaibrain.com/register
2. Create an account and team
3. Go to Settings > API Keys > Create New Key
4. Set `MANAGEMENT_BRAIN_API_KEY` in your OpenClaw config

### Cloud API Endpoints

All API calls require: `Authorization: Bearer <MANAGEMENT_BRAIN_API_KEY>`

Base URL: `MANAGEMENT_BRAIN_URL` env var, or `https://api.manageaibrain.com`

```
GET  {baseUrl}/api/v1/openclaw/status     — Team status, submissions, pending
GET  {baseUrl}/api/v1/openclaw/report?period=weekly  — Rankings, metrics
POST {baseUrl}/api/v1/openclaw/command    — Switch mentor, list employees
GET  {baseUrl}/api/v1/openclaw/alerts     — Anomaly alerts
POST {baseUrl}/api/v1/auth/api-keys       — Create API key
GET  {baseUrl}/api/v1/auth/api-keys       — List API keys
DELETE {baseUrl}/api/v1/auth/api-keys/{id} — Revoke API key
```

When `MANAGEMENT_BRAIN_API_KEY` is set, prefer using cloud API endpoints for all queries. When not set, use local data files.

## Links

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
