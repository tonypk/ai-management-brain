# Hybrid Skill Architecture Design — boss-ai-agent

**Target version**: 3.0.0

## Problem

The current `boss-ai-agent` skill requires `manageaibrain.com/mcp` for ALL functionality. Without the cloud connection, the skill is useless. This creates:

1. **High friction**: install → register → configure MCP → configure Bot → use (vs competitors: install → use)
2. **Trust barrier**: ClawHub scan flags "all data goes to external server" as a risk
3. **Single point of failure**: server down = skill broken
4. **Poor discoverability**: users can't try before committing to a cloud service

Meanwhile, most popular ClawHub skills are **pure instruction** — no external dependencies.

## Solution

Split the skill into two modes within the **same SKILL.md**:

- **Advisor Mode** (instruction-only, zero dependency): AI uses embedded mentor frameworks to answer management questions, generate check-in questions, prepare 1:1s, simulate C-Suite discussions, and provide cultural communication guidance.
- **Team Operations Mode** (MCP-connected): Everything in Advisor Mode plus real team automation — push check-ins to employees, track responses, generate reports from actual data, auto-chase non-responders.

The AI auto-detects which mode to use based on whether MCP tools are available.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    SKILL.md                          │
│                                                      │
│  ┌────────────────────────────────┐                  │
│  │   Advisor Mode (always works)  │                  │
│  │                                │                  │
│  │  • 16 mentor frameworks        │                  │
│  │    (3 full matrices +          │                  │
│  │     6 standard + 7 tag-based)  │                  │
│  │  • 9 culture packs             │                  │
│  │  • C-Suite simulation          │                  │
│  │    (stateless, no history)     │                  │
│  │  • 1:1 prep framework          │                  │
│  │  • Check-in question design    │                  │
│  │  • Management decision advice  │                  │
│  │  • Report templates            │                  │
│  └───────────────┬────────────────┘                  │
│                  │                                    │
│        MCP tools available?                          │
│        (sentinel: get_team_status)                   │
│            ┌─────┴─────┐                             │
│           No          Yes                            │
│            │     ┌─────┴──────────────────┐          │
│            │     │ Team Operations Mode   │          │
│            │     │                        │          │
│            │     │ • 13 MCP tools         │          │
│            │     │ • 5 cron jobs          │          │
│            │     │ • Persistent storage   │          │
│            │     │ • Message delivery     │          │
│            │     │   (23+ platforms)      │          │
│            │     │ • Web Dashboard        │          │
│            │     └────────────────────────┘          │
│            │                                         │
│         Use mentor                                   │
│         knowledge to                                 │
│         answer directly                              │
└─────────────────────────────────────────────────────┘
```

## Mode Detection

The SKILL.md instructs the AI:

```
Check if the `get_team_status` MCP tool is available in your tool list.
- If YES → Team Operations Mode: use MCP tools for all team operations.
- If NO → Advisor Mode: use embedded mentor frameworks to answer directly.
Always announce your mode on first interaction.
```

Using `get_team_status` as the sentinel tool (not "etc.") — if this tool is available, all 13 tools are assumed available since they come from the same MCP server.

No code changes needed — this is purely an instruction change.

## First Run

### Advisor Mode First Run

When `/boss-ai-agent` is invoked without MCP tools available:

1. Greet: "Hi! I'm Boss AI Agent, your AI management advisor. I'm running in **Advisor Mode** — no setup needed."
2. Ask ONE question: "Which mentor philosophy resonates with you?" (present top 3: Musk, Inamori, Ma — with brief descriptions)
3. Write minimal config to `~/.openclaw/skills/boss-ai-agent/config.json`:

```json
{
  "mentor": "musk",
  "mentorBlend": null,
  "culture": "default",
  "mode": "advisor"
}
```

4. **No cron jobs registered** — Advisor Mode has no persistent behavior.
5. No team size question — irrelevant in Advisor Mode.
6. Mention upgrade path: "Want automated team management? Connect to manageaibrain.com/mcp to unlock check-ins, tracking, and reports."

### Team Operations Mode First Run

Same as current behavior (unchanged):

1. Greet + announce Team Operations Mode
2. Ask 3 questions (team size, comm tools, project management)
3. Write full config with schedules
4. Register cron jobs
5. Recommend mentor based on team size

## Advisor Mode: Detailed Scenarios

### 1. Management Decision Advice

User asks a management question → AI applies current mentor's decision framework:

**Example**: "Should I promote Alex to team lead?"

- **Musk mode** (Fully-Embedded): "Does Alex push for 10x? Can they eliminate blockers for the team? First principles: what's the expected output increase?"
- **Inamori mode** (Fully-Embedded): "Does Alex care about the team's wellbeing? Do others respect and trust them? Consider: who did Alex help grow?"
- **Dalio mode** (Standard): Uses radical-transparency and principles-driven tags to frame analysis — "What do the principles say? Has Alex demonstrated radical honesty and mistake-learning?"
- **Buffett mode** (Light-touch): Infers from long-term-value and patience tags — "Is this a long-term investment in the team? What's the margin of safety?"

The 3 Fully-Embedded mentors have complete 7-point decision matrices. The 6 Standard mentors have check-in questions + core tags. The 7 Light-touch mentors have tags only — the AI infers behavior from those tags.

### 2. Check-in Question Design

User: "Generate today's check-in questions"

AI generates 3 questions per the active mentor style. User sends them through their own channels (email, Slack, WhatsApp group, etc.).

**Musk**: What did you push forward today? / What blocker can we eliminate? / If you had half the time, what would you cut?

**Inamori**: What did you contribute to the team? / Any difficulties you need help with? / What did you learn today?

### 3. 1:1 Meeting Prep

User: "I have a 1:1 with Sarah tomorrow. She's been quiet lately and missed a deadline last week."

AI generates using mentor framework + culture pack:
- Opening questions (warm-up, adapted to culture)
- Key discussion topics (based on context provided)
- Difficult conversation guidance (culture-appropriate)
- Action items template
- Follow-up schedule suggestion

### 4. C-Suite Board Simulation

User: "Should we enter the Japan market?"

**Advisor Mode**: AI simulates 6 perspectives stateless (no cross-session history):
- **CEO**: Strategic alignment, competitive landscape
- **CFO**: Market size, investment required, ROI timeline
- **CMO**: Brand positioning, local marketing channels
- **CTO**: Technical localization requirements
- **CHRO**: Talent availability, cultural adaptation
- **COO**: Operational complexity, supply chain

Followed by a synthesized recommendation.

**Team Operations Mode**: Uses `board_discuss` MCP tool for persistent discussion history stored on server and responses enriched with actual team data (workload, capacity, past decisions).

### 5. Daily/Weekly Report Templates

AI generates report frameworks based on mentor's priorities:
- **Musk**: Velocity metrics, blocker list, 10x opportunities
- **Dalio**: Principle violations, mistake log, transparency score
- **Bezos**: Customer impact metrics, Day 1 indicators

### 6. Conflict Resolution Framework

User describes a team conflict → AI applies mentor philosophy + relevant culture packs to give step-by-step resolution guidance.

### 7. Cultural Communication Guide

User: "How do I give negative feedback to my Indonesian team member?"

AI applies Indonesia culture pack: relationship-first, group harmony, never public criticism, use indirect framing, suggest private 1:1 setting.

## Mentor Switching

- **Advisor Mode**: User says "switch to Inamori" → AI updates `config.json` mentor field and applies new framework immediately. No MCP tool needed — handled via conversation + file write.
- **Team Operations Mode**: Uses `switch_mentor` MCP tool (persists on server, affects cron job behavior).

## Team Operations Mode: Existing Functionality

When MCP tools are detected, the skill operates as it does today:

- 13 MCP tools for real team operations
- 5 cron jobs for automated management cycle
- Persistent data storage on `manageaibrain.com`
- Message delivery via Telegram/Slack/Lark/Signal (23+ platforms)
- Web Dashboard at manageaibrain.com

All existing documentation for MCP tools, cron management, and automated scenarios remains unchanged.

## Files to Modify

### `openclaw-skill/SKILL.md` (~100 lines added, ~40 lines restructured)

1. **Frontmatter**: Update `description` to reflect hybrid positioning: "Works instantly as management advisor. Connect MCP for full team automation." Update `version` to `3.0.0`. Change `requires.config` to `optional.config` (config is created at first run, not required to exist beforehand).
2. **Identity section**: Add dual-mode explanation.
3. **Permissions & Data**: Remove phrase "which is always active" from MCP/Cloud API descriptions. Split into "Advisor Mode" (local config only) and "Team Operations Mode" (MCP + cron + cloud).
4. **Data Flow**: Add Advisor Mode path (no cloud communication). Rewrite existing flow as "Team Operations Mode" path.
5. **New section: "Mode Detection"**: Sentinel tool check instruction.
6. **New section: "First Run (Advisor Mode)"**: Minimal setup flow.
7. **Existing First Run section**: Rename to "First Run (Team Operations Mode)".
8. **New section: "Advisor Mode"**: 7 scenario templates with mentor-specific examples across all 3 tiers.
9. **Rename existing sections**: Current MCP/Cron/Scenarios sections become subsections under "Team Operations Mode".
10. **C-Suite Board section**: Update to describe dual behavior — stateless simulation (Advisor) vs persistent `board_discuss` (Team Ops).
11. **Mentor switching**: Add note about conversation-based switching in Advisor Mode.

### `openclaw-skill/README.md` (~30 lines changed)

1. **Frontmatter**: Update `description` to match SKILL.md. Update `version` to `3.0.0`. Change `requires.config` to `optional.config`.
2. **Features list**: Replace "Cloud-powered MCP" with "Works instantly as management advisor — connect MCP for full team automation". Move "23+ messaging platforms" under Team Operations features.
3. **How It Works**: Rewrite to describe dual-mode. Remove "Cloud Architecture" section.
4. **Quick Start**: Show both paths — Advisor Mode (just mentor selection) and Team Operations (full setup).
5. **Chinese section**: Update to reflect hybrid architecture.

### No Backend Changes

The Go server, MCP tools, database, and API remain unchanged. This is purely a SKILL.md + README.md update.

### MEMORY.md Update

Update culture count from "6 cultures" to "9 culture packs: default, philippines, singapore, indonesia, srilanka, malaysia, china, usa, india".

## Marketing Positioning Change

**Before**: "Cloud management system that requires manageaibrain.com"

**After**: "AI management advisor powered by 16 mentor philosophies. Connect your team for full automation."

### Frontmatter Description (both SKILL.md and README.md)

```
Boss AI Agent — your AI management advisor. 16 mentor philosophies, 9 culture packs,
C-Suite board simulation. Works instantly after install. Connect manageaibrain.com MCP
for full team automation: auto check-ins, tracking, reports, 23+ platform messaging.
```

### ClawHub Description

```
Boss AI Agent — your AI management advisor. 16 mentor philosophies (Musk, Inamori, Ma, Dalio...),
9 culture packs, C-Suite board simulation. Works instantly after install — no account needed.
Connect manageaibrain.com MCP for full team automation: auto check-ins, tracking, reports,
message delivery across 23+ platforms.
```

## User Journey

### Path A: Casual User (majority)

```
clawhub install boss-ai-agent
/boss-ai-agent
→ "Hi! I'm Boss AI Agent, your AI management advisor. Running in Advisor Mode."
→ "Which mentor resonates with you? Musk (first principles), Inamori (人心经营),
   Ma (customer-first)..."
→ User picks Musk → config.json written with mentor: "musk"
→ User: "How should I handle a consistently late employee?"
→ AI gives Musk-flavored advice with specific action steps
→ User: "Switch to Inamori" → config.json updated, AI confirms
```

### Path B: Power User (upgrade path)

```
User uses Advisor Mode for a week, finds it valuable
→ "Want real-time team management? Connect to manageaibrain.com for auto
   check-ins, tracking, and reports."
→ User sets up MCP connection + Telegram Bot
→ Skill detects get_team_status tool → announces Team Operations Mode
→ Full setup flow (team size, comm tools, cron registration)
→ Same mentor philosophy now powers actual team interactions
```

## Validation

1. Install without MCP → skill responds in Advisor Mode with mentor selection
2. Ask management questions → mentor-appropriate answers (test all 3 tiers)
3. Request check-in questions → mentor-styled questions generated
4. Request 1:1 prep → structured prep doc with cultural adaptation
5. Ask for C-Suite analysis → 6-perspective stateless simulation
6. Switch mentor via conversation → config.json updated
7. Connect MCP → skill detects `get_team_status` → announces Team Operations Mode
8. All existing MCP functionality works unchanged
9. Start in Advisor Mode, connect MCP mid-session → skill announces mode upgrade
10. MCP connection drops → skill gracefully falls back to Advisor Mode
11. ClawHub scan: no more contradictions (Advisor Mode = no cloud, Team Ops = cloud)
12. `clawhub publish` with `--name "Boss AI Agent"` → correct title displayed
