---
description: Generate a daily or weekly management summary
argument-hint: [daily|weekly]
allowed-tools: [Read, Write, Bash, WebFetch]
---

# Management Summary

Generate a management summary for today or this week.

**Arguments:**
- `daily` (default) — summarize today's reports
- `weekly` — summarize the full week

**Process:**

1. Read `~/.boss-ai-agent/config.json` for mentor settings.
2. Read report files from `~/.boss-ai-agent/data/reports/`:
   - For daily: today's file
   - For weekly: last 5-7 days of files
3. Read recent summaries from `~/.boss-ai-agent/data/summaries/` for trend context.
4. Generate a mentor-perspective summary:
   - Submission rate (X/Y employees reported)
   - Key highlights and concerns
   - Sentiment overview (if tracked)
   - Recommended 1:1s (declining patterns)
   - For weekly: week-over-week trends
5. Save summary to `~/.boss-ai-agent/data/summaries/{YYYY-MM-DD}.json`.
6. Send to boss via preferred channel and display in conversation.
