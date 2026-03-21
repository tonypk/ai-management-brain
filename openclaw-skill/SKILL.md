---
name: management-brain
description: "AI Management OS: 9 mentor philosophies (Inamori/Musk/Jobs...), proactive Telegram/Slack/Lark chase, sentiment analysis, cultural adaptation, executive summaries"
user-invocable: true
metadata:
  openclaw:
    requires:
      env: ["MANAGEMENT_BRAIN_API_KEY"]
    primaryEnv: "MANAGEMENT_BRAIN_API_KEY"
---

# AI Management Brain

Your AI-powered Management Operating System. Connect Telegram, Slack, or Lark — select a world-class mentor philosophy — and the AI handles daily check-ins, proactive employee chase, sentiment analysis, cultural adaptation, and executive summaries for you.

**Invisible but indispensable.** The brain quietly takes over management logic without replacing any existing tools.

## What It Does

- **Proactive Communication**: AI actively reaches out to employees via Telegram, Slack, or Lark for daily check-ins — no manual follow-up needed
- **Smart Chase**: When employees don't report, the system escalates automatically (private DM → manager notify → skip) following the selected mentor's philosophy
- **Sentiment Analysis**: AI detects mood changes, burnout signals, and engagement drops from daily reports
- **Cultural Adaptation**: Auto-adjusts communication style per employee's cultural background (Philippines, Singapore, Indonesia, etc.)
- **Executive Summaries**: Boss receives AI-generated daily/weekly summaries focused on what matters most per the chosen mentor
- **Anomaly Alerts**: Proactive warnings for consecutive misses, sentiment drops, and team health issues

## 中文说明

AI Management Brain 是一个 AI 管理操作系统。连接 Telegram/Slack/Lark，选择一位世界级管理导师的哲学体系，AI 就会自动处理每日汇报收集、主动追踪未提交员工、情绪分析、文化适配和管理层摘要。

**核心功能：**
- **主动沟通**：AI 通过 Telegram/Slack/Lark 主动联系员工进行每日汇报，无需人工跟进
- **智能追踪**：员工未提交时，系统按导师哲学自动升级（私信提醒 → 通知经理 → 标记跳过）
- **情绪分析**：从每日汇报中检测情绪变化、倦怠信号和参与度下降
- **文化适配**：根据员工文化背景自动调整沟通风格（菲律宾、新加坡、印尼等）
- **管理层摘要**：老板收到 AI 生成的每日/每周总结，聚焦导师关注的关键指标
- **异常告警**：连续缺勤、情绪骤降、团队健康问题的主动预警

**9位管理导师：** 稻盛和夫 · 达利欧 · 格鲁夫 · 任正非 · 孙正义 · 乔布斯 · 贝佐斯 · 马云 · 马斯克

## Authentication

All API calls require the header: `Authorization: Bearer <MANAGEMENT_BRAIN_API_KEY>`

Base URL: Use the `MANAGEMENT_BRAIN_URL` environment variable if set, otherwise `https://api.manageaibrain.com`

## Available Actions

### Check Team Status
When the user asks about team status, submissions, or who hasn't reported:
```
GET {baseUrl}/api/v1/openclaw/status
```
Returns: date, total employees, submitted count, pending names (with chase counts), current mentor, active channel integrations.

### View Reports
When the user asks for weekly or monthly reports, rankings, or performance:
```
GET {baseUrl}/api/v1/openclaw/report?period=weekly
GET {baseUrl}/api/v1/openclaw/report?period=monthly
```
Returns: submission rate, employee ranking with medals, sentiment trends, one-on-one suggestions, mentor-specific metrics.

### Execute Commands
When the user wants to switch mentors, list employees, or list mentors:
```
POST {baseUrl}/api/v1/openclaw/command
Body: {"command": "<natural language command>"}
```
Supported commands:
- "switch mentor to inamori" / "switch to elon musk"
- "list employees"
- "list mentors"

### Check Alerts
When the user asks about problems, alerts, or anomalies:
```
GET {baseUrl}/api/v1/openclaw/alerts
```
Returns: active alerts (consecutive misses, sentiment drops, engagement anomalies) with severity levels (warning/critical).

### Manage API Keys
```
POST {baseUrl}/api/v1/auth/api-keys   (body: {"name": "my key"})
GET  {baseUrl}/api/v1/auth/api-keys
DELETE {baseUrl}/api/v1/auth/api-keys/{id}
```

## Response Formatting

- Present team status as a concise summary, highlighting pending employees and alerts
- Show rankings in a table format with medals (gold, silver, bronze)
- For alerts, indicate severity: warning or critical
- When mentor is switched, explain what changes (questions, chase style, summary focus)
- Include channel info (which employees are on Telegram vs Slack vs Lark)

## Mentor Reference

Selecting a mentor changes the entire management strategy: check-in questions, chase escalation, summary focus, proactive actions, and AI personality.

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

## Multi-Channel Support

The system proactively communicates with employees through:
- **Telegram** — Primary channel, DM-based report collection and chase
- **Slack** — Workspace integration with channel and DM support
- **Lark (飞书)** — Enterprise messaging with rich card support

Each employee can be on a different channel. The AI adapts message format per platform while maintaining consistent management strategy.

## First-Time Setup

If the user hasn't set up their API key yet, direct them to:
1. Visit https://app.manageaibrain.com/register
2. Create an account and team
3. Connect your Telegram/Slack/Lark channel
4. Go to Settings > API Keys > Create New Key
5. Set the key as MANAGEMENT_BRAIN_API_KEY in OpenClaw config

## Links

- Website: https://manageaibrain.com
- Documentation: https://docs.manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
