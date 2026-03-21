---
name: management-brain
description: AI-powered team management — daily check-ins, mentor-driven chase, sentiment analysis, and executive summaries
user-invocable: true
metadata:
  openclaw:
    requires:
      env: ["MANAGEMENT_BRAIN_API_KEY"]
    primaryEnv: "MANAGEMENT_BRAIN_API_KEY"
---

# AI Management Brain

You are connected to an AI Management Brain instance. This skill lets you manage your team through natural language.

## Authentication

All API calls require the header: `Authorization: Bearer <MANAGEMENT_BRAIN_API_KEY>`

Base URL: Use the `MANAGEMENT_BRAIN_URL` environment variable if set, otherwise `https://api.managementbrain.ai`

## Available Actions

### Check Team Status
When the user asks about team status, submissions, or who hasn't reported:
```
GET {baseUrl}/api/v1/openclaw/status
```
Returns: date, total employees, submitted count, pending names, chase counts, current mentor.

### View Reports
When the user asks for weekly or monthly reports, rankings, or performance:
```
GET {baseUrl}/api/v1/openclaw/report?period=weekly
GET {baseUrl}/api/v1/openclaw/report?period=monthly
```
Returns: submission rate, employee ranking with medals, one-on-one suggestions.

### Execute Commands
When the user wants to switch mentors, list employees, or list mentors:
```
POST {baseUrl}/api/v1/openclaw/command
Body: {"command": "<natural language command>"}
```
Supported commands:
- "switch mentor to inamori" / "switch to andy grove"
- "list employees"
- "list mentors"

### Check Alerts
When the user asks about problems, alerts, or anomalies:
```
GET {baseUrl}/api/v1/openclaw/alerts
```
Returns: active alerts (consecutive misses, sentiment drops) with severity levels.

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

## Mentor Reference

| ID | Name | Style |
|----|------|-------|
| inamori | Kazuo Inamori | Amoeba management, altruism, respect |
| dalio | Ray Dalio | Radical transparency, principles |
| grove | Andy Grove | OKR-driven, high output |
| ren | Ren Zhengfei | Wolf culture, self-criticism |
| son | Masayoshi Son | 300-year vision, bold bets |
| jobs | Steve Jobs | Simplicity, reality distortion |
| bezos | Jeff Bezos | Day 1, customer obsession |
| ma | Jack Ma | Embrace change, teamwork |

## First-Time Setup

If the user hasn't set up their API key yet, direct them to:
1. Visit https://app.managementbrain.ai/register
2. Create an account and team
3. Go to Settings > API Keys > Create New Key
4. Set the key as MANAGEMENT_BRAIN_API_KEY in OpenClaw config

## Note

When publishing to ClawHub, rename this file to `SKILL.md`.
