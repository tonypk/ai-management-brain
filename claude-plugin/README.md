# Boss AI Agent — Claude Code Plugin

AI Management OS for Claude Code / Claude Cowork. 14 mentor philosophies, 9 culture packs, 7 automated management scenarios.

## Install

```
/plugin marketplace add tonypk/boss-ai-agent-claude-plugin
/plugin install boss-ai-agent@boss-ai-agent
```

## Quick Start

```
/setup                          # First-run onboarding
/team add John --channel slack --id U123456 --culture philippines
/mentor musk                    # Switch to Musk management style
/checkin                        # Send daily check-in to team
/chase                          # Chase non-responders
/summary                        # Generate daily summary
/briefing                       # Smart morning briefing
/patrol                         # Project health check
/1on1 John                      # 1:1 meeting prep
```

## Features

**7 Automated Scenarios:**
1. Daily Management Cycle — check-in, chase, summary
2. Project Health Patrol — GitHub/Linear scanning
3. Smart Daily Briefing — cross-channel prioritized briefing
4. 1:1 Meeting Assistant — data-driven meeting prep
5. Periodic Signal Scanning — detect issues in real-time
6. Knowledge Base Management — record decisions
7. Emergency Response — auto-alert + investigation

**14 Mentor Philosophies:**

| Tier | Mentors |
|------|---------|
| Fully-embedded | Musk, Inamori, Ma |
| Standard | Dalio, Grove, Ren, Son, Jobs, Bezos |
| Light-touch | Buffett, Zhang Yiming, Lei Jun, Cao Dewang, Chu Shijian |

**9 Culture Packs:** Default, Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India

## Integrations

Connects via MCP servers (all optional, enable what you use):

| Tool | MCP Server | Purpose |
|------|-----------|---------|
| Slack | `@anthropic/mcp-slack` | Team messaging |
| Gmail | `@anthropic/mcp-gmail` | Email communication |
| Google Calendar | `@anthropic/mcp-google-calendar` | Meeting awareness |
| Linear | `@anthropic/mcp-linear` | Project tracking |
| Notion | `@anthropic/mcp-notion` | Knowledge base |
| Telegram | Direct Bot API (curl) | Team messaging |
| GitHub | WebFetch (API) | Code health monitoring |

## Scheduling

Set up recurring tasks with Claude's `/schedule`:

```
/schedule "Send daily check-in" every weekday at 9am
/schedule "Chase non-responders" every weekday at 5:30pm
/schedule "Generate daily summary" every weekday at 7pm
/schedule "Morning briefing" every weekday at 8am
/schedule "Project health patrol" every Monday at 9am
```

## Cloud Platform (Optional)

Set `BOSS_AI_AGENT_API_KEY` to connect to [manageaibrain.com](https://manageaibrain.com) for:
- Full mentor configs (all 14 complete)
- Web dashboard and analytics
- Cross-team benchmarking

All features work fully without it.

## Links

- Website: [manageaibrain.com](https://manageaibrain.com)
- OpenClaw version: [clawhub.ai/tonypk/boss-ai-agent](https://clawhub.ai/tonypk/boss-ai-agent)
- GitHub: [github.com/tonypk/ai-management-brain](https://github.com/tonypk/ai-management-brain)
