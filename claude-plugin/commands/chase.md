---
description: Chase team members who haven't submitted their daily report
allowed-tools: [Read, Write, Bash, WebFetch]
---

# Chase Non-Responders

Follow up with employees who haven't submitted their daily report.

1. Read `~/.boss-ai-agent/config.json` for mentor and alert settings.
2. Read `~/.boss-ai-agent/data/employees.json` for the team list.
3. Read today's report file from `~/.boss-ai-agent/data/reports/{YYYY-MM-DD}.json`.
4. Identify employees who have NOT submitted a report today.
5. For each non-responder:
   - Apply the active mentor's chase strategy (aggressive for Musk, gentle for Inamori, etc.).
   - Apply cultural override for the employee's culture setting.
   - Send a culture-adapted reminder via their configured channel.
6. Update today's report file with chase events.
7. Report to boss: "{N} employees chased. {M} still pending."
