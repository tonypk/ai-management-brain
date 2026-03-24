---
description: Generate a smart morning briefing with prioritized action items
allowed-tools: [Read, Write, Bash, WebFetch, WebSearch]
---

# Morning Briefing

Generate a prioritized morning briefing from all connected sources.

1. Read `~/.boss-ai-agent/config.json` for enabled integrations and mentor.
2. Gather information from enabled sources:
   - **Slack** (if MCP enabled): Read unread messages from last 12 hours
   - **Telegram** (if configured): Get recent updates via Bot API
   - **GitHub** (if enabled): WebFetch open PRs, CI status, issues assigned to boss
   - **Google Calendar** (if MCP enabled): Check today's meetings
   - **Gmail** (if MCP enabled): Check high-priority unread emails
3. Read yesterday's summary from `~/.boss-ai-agent/data/summaries/` for context.
4. Sort all items by the active mentor's priority framework:
   - Musk: blockers and delays first
   - Inamori: people concerns first
   - Ma: customer impact first
5. Present a numbered briefing, most important first, tagged by severity.
