---
description: First-run setup for Boss AI Agent — configure team, channels, and mentor
allowed-tools: [Read, Write, Bash, Glob]
---

# Boss AI Agent Setup

Run the first-run onboarding sequence for Boss AI Agent.

1. Check if `~/.boss-ai-agent/config.json` exists using Read tool.
2. If it exists, tell the user: "Boss AI Agent is already configured. Use `/mentor` to change mentor, `/team` to manage employees."
3. If it does NOT exist, follow the First Run sequence from the boss-ai-agent skill:
   - Create `~/.boss-ai-agent/` directory and all subdirectories
   - Ask the 3 onboarding questions one at a time
   - Generate config.json with user's answers
   - Create empty employees.json
   - Guide MCP server setup for their tools
   - Guide scheduled task setup
   - Recommend a mentor
