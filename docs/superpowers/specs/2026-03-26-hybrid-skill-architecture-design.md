# Hybrid Skill Architecture Design — boss-ai-agent

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
│  │  • 16 mentor decision matrices │                  │
│  │  • 9 culture communication     │                  │
│  │  • C-Suite simulation          │                  │
│  │  • 1:1 prep framework          │                  │
│  │  • Check-in question design    │                  │
│  │  • Management decision advice  │                  │
│  │  • Report templates            │                  │
│  └───────────────┬────────────────┘                  │
│                  │                                    │
│           MCP available?                             │
│            ┌─────┴─────┐                             │
│           No          Yes                            │
│            │     ┌─────┴──────────────────┐          │
│            │     │ Team Operations Mode   │          │
│            │     │                        │          │
│            │     │ • 13 MCP tools         │          │
│            │     │ • 5 cron jobs          │          │
│            │     │ • Persistent storage   │          │
│            │     │ • Message delivery     │          │
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
Check if MCP tools (get_team_status, get_report, etc.) are available in your tool list.
- If YES → Team Operations Mode: use MCP tools for all team operations.
- If NO → Advisor Mode: use embedded mentor frameworks to answer directly.
Always announce your mode on first interaction.
```

No code changes needed — this is purely an instruction change.

## Advisor Mode: Detailed Scenarios

### 1. Management Decision Advice

User asks a management question → AI applies current mentor's decision framework:

**Example**: "Should I promote Alex to team lead?"

- **Musk mode**: "Does Alex push for 10x? Can they eliminate blockers for the team? First principles: what's the expected output increase?"
- **Inamori mode**: "Does Alex care about the team's wellbeing? Do others respect and trust them? Consider: who did Alex help grow?"
- **Ma mode**: "Is Alex customer-focused? Can they embrace change and inspire the team?"

The mentor's complete decision matrix (check-in style, chase intensity, risk assessment, patrol focus, info priority, 1:1 advice, emergency style) is already embedded in SKILL.md.

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

AI simulates 6 perspectives WITHOUT MCP tools:
- **CEO**: Strategic alignment, competitive landscape
- **CFO**: Market size, investment required, ROI timeline
- **CMO**: Brand positioning, local marketing channels
- **CTO**: Technical localization requirements
- **CHRO**: Talent availability, cultural adaptation
- **COO**: Operational complexity, supply chain

Followed by a synthesized recommendation.

When MCP IS connected: uses `board_discuss` tool for persistent history and richer data-backed analysis.

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

## Team Operations Mode: Existing Functionality

When MCP tools are detected, the skill operates as it does today:

- 13 MCP tools for real team operations
- 5 cron jobs for automated management cycle
- Persistent data storage on `manageaibrain.com`
- Message delivery via Telegram/Slack/Lark/Signal
- Web Dashboard at manageaibrain.com

All existing documentation for MCP tools, cron management, and automated scenarios remains unchanged.

## Files to Modify

### `openclaw-skill/SKILL.md` (~100 lines added, ~30 lines restructured)

1. **Identity section**: Add dual-mode explanation
2. **New section: "Mode Detection"**: Instructions for AI to detect available mode
3. **New section: "Advisor Mode"**: 7 scenario templates with mentor-specific examples
4. **Rename existing sections**: Current MCP/Cron/Scenarios sections become "Team Operations Mode"
5. **C-Suite Board section**: Update to work in both modes

### `openclaw-skill/README.md` (~20 lines changed)

1. **Features list**: Replace "Cloud-powered MCP" with "Works instantly as management advisor — connect MCP for full team automation"
2. **How It Works**: Add advisor mode description
3. **Quick Start**: Show both paths (advisor-only vs full setup)
4. **Remove**: "Cloud Architecture" section (replaced by clearer dual-mode description)

### No Backend Changes

The Go server, MCP tools, database, and API remain unchanged. This is purely a SKILL.md + README.md update.

## Marketing Positioning Change

**Before**: "Cloud management system that requires manageaibrain.com"

**After**: "AI management advisor powered by 16 mentor philosophies. Connect your team for full automation."

### ClawHub Description Update

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
→ "Hi! I'm Boss AI Agent. I'm running in Advisor Mode — I'll use mentor
   frameworks to help with management decisions, 1:1 prep, team communication,
   and more. No setup needed."
→ "Which mentor resonates with you? Musk (first principles), Inamori (人心经营),
   Ma (customer-first)..."
→ User picks Musk
→ User: "How should I handle a consistently late employee?"
→ AI gives Musk-flavored advice with specific action steps
```

### Path B: Power User (upgrade path)

```
User uses Advisor Mode for a week, finds it valuable
→ "Want real-time team management? Connect to manageaibrain.com for auto
   check-ins, tracking, and reports."
→ User sets up MCP connection + Telegram Bot
→ Skill seamlessly upgrades to Team Operations Mode
→ Same mentor philosophy now powers actual team interactions
```

## Validation

1. Install without MCP → skill responds in Advisor Mode
2. Ask management questions → mentor-appropriate answers
3. Request check-in questions → mentor-styled questions generated
4. Request 1:1 prep → structured prep doc with cultural adaptation
5. Ask for C-Suite analysis → 6-perspective simulation
6. Connect MCP → skill announces Team Operations Mode
7. All existing MCP functionality works unchanged
8. ClawHub scan: no more "zero dependency" contradiction
