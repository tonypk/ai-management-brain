---
description: Send daily check-in questions to all active team members
allowed-tools: [Read, Write, Bash, WebFetch]
---

# Daily Check-in

Send personalized check-in questions to all active employees.

1. Read `~/.boss-ai-agent/config.json` for mentor, culture, and team settings.
2. Read `~/.boss-ai-agent/data/employees.json` for the team list.
3. If team is empty, respond: "No team members configured. Use `/team add` to add employees."
4. For each active employee:
   - Read their recent reports (last 7 days) from `~/.boss-ai-agent/data/reports/`.
   - Generate check-in questions based on active mentor's question set.
   - Adapt questions for the employee's culture setting.
   - Send via their configured channel (Slack MCP, Telegram curl, or display in conversation).
5. Create/update today's report file at `~/.boss-ai-agent/data/reports/{YYYY-MM-DD}.json` with check-in status.
6. Confirm: "Check-in sent to {N} employees."
