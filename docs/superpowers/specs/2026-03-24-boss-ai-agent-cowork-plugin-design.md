# Boss AI Agent — Claude Cowork Plugin Design

## Overview

Port Boss AI Agent from OpenClaw to Claude Code/Cowork as a native plugin. All 7 management scenarios, 14 mentors, 9 culture packs — adapted to Claude Code's tool ecosystem (Read/Write, WebFetch, Task sub-agents, MCP servers, `/schedule`).

## Why Cowork

Claude Cowork (launched March 2026) provides:
- **Scheduled tasks** (`/schedule`) — cron equivalent
- **Sub-agents** (Task tool) — parallel investigation
- **Persistent memory** — file-based state in project directory
- **38+ MCP connectors** — Slack, Gmail, Google Calendar, Notion, Linear, GitHub
- **Dispatch** — remote control from phone (trigger tasks away from desk)
- **Plugin marketplace** — distribution channel

## Architecture

### Plugin Structure

```
boss-ai-agent/
├── .claude-plugin/
│   └── plugin.json              # Manifest: name, version, description, author
├── .mcp.json                    # MCP server configs (Slack, Gmail — optional)
├── skills/
│   └── boss-ai-agent/
│       ├── SKILL.md             # Core management AI (main skill)
│       └── references/
│           ├── mentors.md       # Full mentor reference (14 mentors)
│           └── cultures.md      # Full culture reference (9 packs)
├── commands/
│   ├── checkin.md               # /checkin — send check-in to team
│   ├── chase.md                 # /chase — chase non-responders
│   ├── summary.md               # /summary — generate daily/weekly summary
│   ├── briefing.md              # /briefing — morning briefing
│   ├── patrol.md                # /patrol — project health patrol
│   ├── 1on1.md                  # /1on1 <name> — 1:1 meeting prep
│   ├── mentor.md                # /mentor <id> — switch mentor
│   ├── team.md                  # /team — add/list/remove employees
│   └── setup.md                 # /setup — first-run onboarding
├── agents/
│   ├── github-scanner.md        # GitHub patrol sub-agent
│   ├── signal-scanner.md        # Channel signal scanner
│   └── sentiment-analyzer.md    # Sentiment analysis sub-agent
├── hooks/
│   └── hooks.json               # SessionStart: load context
├── settings.json                # Default plugin settings
└── README.md                    # User-facing documentation
```

### Tool Mapping (OpenClaw → Claude Code)

| OpenClaw Tool | Claude Code Equivalent | Notes |
|---|---|---|
| `message send` | MCP Slack `send_message` / Bash `curl` Telegram API | MCP for Slack; curl for Telegram Bot API |
| `message read` | MCP Slack `list_messages` / Bash `curl` Telegram API | Same approach |
| `cron add/remove` | Guide user to `/schedule` | Plugin can't auto-register; SKILL.md tells user how |
| `memory_search/get/set` | Read/Write to `~/.boss-ai-agent/data/` | JSON files for state |
| `sessions_spawn` | Task tool (sub-agents) | Same concept |
| `web_fetch` | WebFetch tool | Direct equivalent |
| `web_search` | WebSearch tool | Direct equivalent |
| `read/write` | Read/Write tools | Direct equivalent |
| `browser` | Browse skill (if available) | Optional |
| `exec` | Bash tool | Direct equivalent |
| `nodes notify` | Not available | Skip — use messaging fallback |
| `image` | Read tool (multimodal) | Read images directly |

### State Management

All persistent data stored in `~/.boss-ai-agent/`:

```
~/.boss-ai-agent/
├── config.json              # Main config (mentor, culture, schedule, team)
├── data/
│   ├── employees.json       # Team member profiles
│   ├── reports/
│   │   └── 2026-03-24.json  # Daily report data
│   ├── summaries/
│   │   └── 2026-03-24.json  # Generated summaries
│   ├── signals/
│   │   └── 2026-03-24.json  # Scanned signals
│   └── incidents/
│       └── 2026-03-24.json  # Emergency incidents
└── knowledge/
    └── decisions.json       # Recorded decisions
```

### MCP Integration (.mcp.json)

```json
{
  "mcpServers": {
    "slack": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-slack"],
      "env": {
        "SLACK_BOT_TOKEN": "",
        "SLACK_TEAM_ID": ""
      },
      "disabled": true
    },
    "gmail": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-gmail"],
      "disabled": true
    },
    "google-calendar": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-google-calendar"],
      "disabled": true
    },
    "linear": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-linear"],
      "env": {
        "LINEAR_API_KEY": ""
      },
      "disabled": true
    },
    "notion": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-notion"],
      "env": {
        "NOTION_API_KEY": ""
      },
      "disabled": true
    }
  }
}
```

All MCP servers disabled by default. `/setup` command guides the user to enable the ones they need.

## Scenarios (7 — all ported)

### 1. Daily Management Cycle
- **Check-in**: `/checkin` or scheduled → Read config → Read employee list → For each employee, send check-in questions via MCP Slack / curl Telegram
- **Chase**: `/chase` or scheduled → Read today's reports → Find non-responders → Send culture-adapted reminders
- **Summary**: `/summary` or scheduled → Read all today's reports → Apply mentor lens → Generate summary → Send to boss

### 2. Project Health Patrol
- `/patrol` or scheduled weekly → Dispatch sub-agents via Task tool:
  - github-scanner: WebFetch GitHub API for stale PRs, failed CI
  - signal-scanner: Read recent Slack messages for blockers
- Collect results, prioritize by mentor framework, present report

### 3. Smart Daily Briefing
- `/briefing` or scheduled mornings → Read unread messages → Check GitHub/Linear via WebFetch → Read yesterday's summary → Prioritize by mentor → Present briefing

### 4. 1:1 Meeting Assistant
- `/1on1 <name>` → Read employee history from data files → WebFetch GitHub contributions → Analyze sentiment trend → Generate prep document

### 5. Periodic Signal Scanning
- Scheduled every 30min → Read recent messages → Analyze for critical/warning/positive signals → Store signals → Alert if 2+ red signals

### 6. Knowledge Base Management
- "Record this decision" → Write to data/knowledge/ or Notion (if MCP enabled) → Store in local state

### 7. Emergency Response
- Triggered by signal scan or manual → Immediate alert via messaging → Dispatch investigation sub-agents → Compile emergency brief

## Key Differences from OpenClaw Version

1. **No auto-cron**: Plugin can't auto-register scheduled tasks. SKILL.md guides user to manually set up via `/schedule`. Provide exact schedule strings.
2. **MCP-based messaging**: Instead of OpenClaw's unified `message` tool, use MCP servers for Slack/Gmail and Bash+curl for Telegram Bot API.
3. **File-based memory**: Instead of OpenClaw's memory API, use Read/Write to JSON files in `~/.boss-ai-agent/data/`.
4. **Dispatch-ready**: Design commands to be invocable from Dispatch (phone). Keep command output concise for mobile viewing.
5. **No cloud API dependency**: All features work locally. Optional cloud API for extended mentor configs (same as OpenClaw version).

## Mentor & Culture System

Identical to OpenClaw version:
- 14 mentors (3 fully-embedded, 6 standard, 5 light-touch)
- 9 culture packs (default, philippines, singapore, indonesia, srilanka, malaysia, china, usa, india)
- Culture overrides mentor when conflicts arise
- Mentor blending supported

## Commands Reference

| Command | Description | Key Tools Used |
|---------|-------------|---------------|
| `/setup` | First-run onboarding, create config | Read, Write |
| `/checkin` | Send check-in questions to team | MCP Slack / Bash curl, Read, Write |
| `/chase` | Chase non-responders | MCP Slack / Bash curl, Read, Write |
| `/summary [daily\|weekly]` | Generate management summary | Read, Write, MCP Slack |
| `/briefing` | Morning briefing | Read, WebFetch, MCP Slack |
| `/patrol` | Project health check | Task (sub-agents), WebFetch |
| `/1on1 <name>` | 1:1 meeting prep | Read, WebFetch |
| `/mentor <id>` | Switch mentor philosophy | Read, Write |
| `/team add\|list\|remove` | Manage team members | Read, Write |

## Hooks

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Boss AI Agent plugin loaded'",
            "async": false
          }
        ]
      }
    ]
  }
}
```

Minimal hook — just confirms plugin is loaded. SKILL.md handles context loading when invoked.

## Distribution

1. **GitHub**: Publish to `github.com/tonypk/boss-ai-agent-claude-plugin`
2. **Claude Plugin Marketplace**: Submit via `claude.ai/settings/plugins/submit`
3. **Install**: `claude plugin install github.com/tonypk/boss-ai-agent-claude-plugin`
