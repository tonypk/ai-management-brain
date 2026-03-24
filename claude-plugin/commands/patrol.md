---
description: Run project health patrol — scan GitHub, Slack, and other tools for issues
allowed-tools: [Read, Write, Bash, WebFetch]
---

# Project Health Patrol

Scan all connected project management tools for health issues.

1. Read `~/.boss-ai-agent/config.json` for enabled integrations.
2. Dispatch parallel sub-agents (via Task tool) for each enabled integration:
   - **github-scanner**: Scan configured repos for stale PRs (>3 days), failed CI, stale issues (>7 days)
   - **signal-scanner**: Scan Slack channels for blockers, help requests, negative signals
3. Collect all sub-agent results. Skip any that time out or fail.
4. Deduplicate findings across sources.
5. Apply the active mentor's risk framework to prioritize findings.
6. Present a structured report with severity levels:
   - Critical: immediate action needed
   - Warning: monitor closely
   - Info: positive signals
7. Include recommended actions for each finding.
