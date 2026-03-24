# Boss AI Agent — Skill Redesign Spec

**Date**: 2026-03-24
**Status**: Draft → Reviewed (R1 fixes applied)
**Scope**: Rewrite SKILL.md + README.md only (no backend changes)
**Publish to**: ClawHub as `boss-ai-agent`

---

## 1. Identity

**Name**: `boss-ai-agent`
**Display**: Boss AI Agent
**Emoji**: 🤖
**Homepage**: https://manageaibrain.com

> You are Boss AI Agent — the boss's AI management middleware. You connect the boss to all systems (messaging platforms, project management, knowledge bases, email) and make management decisions through a mentor philosophy framework. You don't just answer questions passively — you proactively patrol, discover issues, and drive action.

## 2. Core Positioning

- **What**: Full-stack AI management middleware for small team bosses/founders
- **Runtime**: OpenClaw native-first; cloud platform (manageaibrain.com) optional for enhanced mentor configs and web dashboard
- **Approach**: Tool-driven — every scenario maps to specific OpenClaw tool invocations

## 3. OpenClaw Tool Mapping

| Tool | Boss AI Agent Usage |
|------|---------------------|
| `message` | Send check-in questions, chase reminders, collect replies, monitor group signals (sentiment/conflict/help requests) across 23+ platforms |
| `cron` | Scheduled tasks: daily check-in (9AM), chase (5:30PM), summary (7PM), weekly review (Mon 9AM), daily briefing (8AM), signal scanning (every 30min during work hours) |
| `memory_search` / `memory_get` | Persist employee profiles, management decisions, project status, boss preferences |
| `sessions_spawn` | Dispatch sub-agents for parallel system scanning (GitHub + Notion + Slack simultaneously) |
| `web_fetch` | Pull GitHub PR/Issue lists, Linear sprint status, Jira board data, Notion pages |
| `web_search` | Industry trends, competitor intel (when boss asks) |
| `browser` | Screenshot project dashboards when boss asks "show me the board" (optional, not used in core scenarios) |
| `read` / `write` | Read/write local config, export reports, generate markdown summaries |
| `exec` | Run scripts (deployment status checks, log analysis, database queries) |
| `nodes` | Push urgent alerts to boss's phone |
| `image` | Analyze screenshots (bug reports, kanban board captures) |

## 4. Seven Core Scenarios

### Scenario 1: Daily Management Cycle (Core)
```
9:00 AM  → [cron] trigger check-in
         → [message send] send mentor-styled questions to each employee
         → [memory_search] inject employee history for personalized questions
5:30 PM  → [cron] trigger chase
         → [message read] check who replied
         → [message send] chase non-responders per mentor+culture strategy
7:00 PM  → [cron] trigger summary
         → [memory_search] pull historical trends for comparison
         → [message send] send mentor-perspective summary to boss
```

### Scenario 2: Project Health Patrol
```
Boss says "check project status" OR [cron] weekly Monday auto-trigger
→ [sessions_spawn] dispatch parallel sub-agents:
   → agent-1: [web_fetch] GitHub — open PRs, stale issues, failed CI
   → agent-2: [web_fetch] Linear/Jira — sprint progress, overdue tasks
   → agent-3: [message search] Slack/TG — project discussion hotspots
→ Aggregate all signals → evaluate risk via mentor framework
→ [message send] structured report + recommended actions to boss
```

### Scenario 3: Smart Daily Briefing
```
Boss says "what's important today" OR [cron] daily morning trigger
→ [message read] scan unread important messages across all channels
→ [web_fetch] check email summaries, calendar events
→ [web_search] optional — industry/competitor news
→ [memory_search] correlate with historical context
→ Sort by mentor priority framework → generate briefing
→ [message send] push to boss
```

### Scenario 4: 1:1 Meeting Assistant
```
Boss says "I need to do a 1:1 with John"
→ [memory_search] pull John's last 30 days of data
→ [web_fetch] check John's GitHub/Linear contributions
→ [message search] John's group channel sentiment
→ Generate prep materials: performance trends, mood shifts, suggested topics
→ Suggest conversation strategy per mentor framework
```

### Scenario 5: Periodic Signal Scanning
```
[cron] every 30 minutes during work hours (e.g., "*/30 9-18 * * 1-5")
→ [message read] poll recent messages from team channels (last 30 min)
→ Detect key signals via keyword + sentiment analysis:
   - 🔴 Conflict/complaint ("this requirement is unreasonable", negative sentiment)
   - 🟡 Help/blocked ("blocked by xxx", "need help")
   - 🟢 Breakthrough/good news ("feature shipped", "deployed")
→ [memory] record significant signals with timestamp
→ When threshold reached (e.g., 2+ red signals in 1 hour) → [message send] alert boss
→ Fallback: boss can manually trigger "scan team channels" at any time
```

### Scenario 6: Knowledge Base Management
```
Boss says "record today's decision" / "update project doc in Notion"
→ [web_fetch] or MCP connect to Notion/Google Sheets
→ [write] generate structured content
→ Auto-update knowledge base
→ [memory] index for future reference
```

### Scenario 7: Emergency Response
```
Urgent event detected (via signal scanning or employee direct message):
  deploy failure, customer complaint, attrition signal, etc.
→ [message send] IMMEDIATELY alert boss on preferred channel (Telegram/Slack/etc.)
   Fallback chain: preferred channel → all configured channels → [nodes notify] if available
→ [sessions_spawn] rapid intel gathering (see sub-agent spec below)
→ [message send] emergency brief + mentor-recommended response plan to boss
→ After boss approves → [message send] execute (notify people, dispatch resources)
```

### Sub-Agent Specification (for sessions_spawn)

Used in Scenarios 2, 5, and 7. Each sub-agent receives a focused prompt and returns structured JSON.

**Prompt template**:
```
You are a {role} sub-agent for Boss AI Agent.
Task: {task_description}
Tools available: {tool_list}
Output format: JSON with fields: status (ok/warning/critical), findings (array of {title, detail, severity}), summary (1-2 sentences)
Timeout: 60 seconds
```

**Scenario 2 sub-agents**:
| Sub-agent | Role | Task | Tools |
|-----------|------|------|-------|
| github-scanner | Code reviewer | Scan repos for: open PRs > 3 days, failed CI, stale issues > 7 days | `web_fetch` |
| pm-scanner | Project tracker | Check sprint progress, overdue tasks, unassigned items | `web_fetch` |
| chat-scanner | Signal analyst | Scan team channels for project-related discussions, blockers, sentiment | `message read` |

**Aggregation**: Parent agent collects all sub-agent results, deduplicates findings, applies mentor risk framework to prioritize, generates unified report.

**Failure handling**: If a sub-agent times out or errors, skip that source and note "⚠️ {source} unavailable" in the report. Never block the entire report for one failed source.

## 5. Mentor System

### Mentor Architecture

- **3 fully-embedded mentors** with complete decision matrices: Musk, Inamori, Ma (Jack Ma)
- **6 standard mentors** with check-in questions + core trait tags (carried from v2.x): Dalio, Grove, Ren, Son, Jobs, Bezos
- **5 light-touch mentors** (new additions) with brief descriptions + tags — agent infers behavior: Buffett, Zhang Yiming, Lei Jun, Cao Dewang, Chu Shijian
- **Cloud extension**: If `BOSS_AI_AGENT_API_KEY` is configured, fetch full configs from `POST /api/v1/openclaw/command` (`{"command": "list mentors"}`) via manageaibrain.com, overriding all light-touch and standard versions with complete decision matrices

### Mentor Decision Matrix (3 Exemplars)

| Decision Point | Musk | Inamori (稻盛和夫) | Ma (马云) |
|---------------|------|-------------------|----------|
| Check-in questions | "What's blocking your 10x progress?" | "Who did you help today?" | "Which customer did you help? What change did you embrace?" |
| Chase intensity | Aggressive — chase after 2h | Gentle — warm reminder before EOD | Moderate — encouraging, emphasize team responsibility |
| Risk assessment | First principles decomposition | Impact on people | Reason backwards from customer/market |
| Project patrol focus | Speed, delivery, blocker removal | Team morale, collaboration quality | Customer value, team collaboration, adaptability |
| Info priority | 🔴 Blockers and delays | 🔴 Employee mood anomalies | 🔴 Customer issues and team collaboration breakdown |
| 1:1 advice | "Challenge them to think bigger" | "Care about their wellbeing first" | "Discuss their understanding of team and customers" |
| Emergency style | Act immediately, fast decisions | Stabilize people first, then fix | Embrace change, turn crisis into opportunity |

### Standard Mentors (6 — with check-in questions)

| ID | Name | Check-in Questions | Core Tags |
|----|------|--------------------|-----------|
| dalio | Ray Dalio | "What decision did you make today? Reasoning?" / "What mistake did you learn from?" / "What principle applies?" | radical-transparency, principles-driven, mistake-analysis |
| grove | Andy Grove | "What's your OKR progress?" / "Biggest bottleneck?" / "What output did you deliver?" | OKR-driven, data-focused, high-output |
| ren | Ren Zhengfei (任正非) | "What goal did you accomplish?" / "What challenge did you overcome?" / "How did you push your limits?" | wolf-culture, self-criticism, striver-oriented |
| son | Masayoshi Son (孙正义) | "Progress toward the big vision?" / "What bold bet are you considering?" / "What did you learn from other industries?" | 300-year-vision, bold-bets, time-machine |
| jobs | Steve Jobs | "What did you ship that you're proud of?" / "What can be simpler?" / "How far from 'insanely great'?" | simplicity, excellence-pursuit, reality-distortion |
| bezos | Jeff Bezos | "What did you do for the customer?" / "What would you do differently on Day 1?" / "What data informed your decision?" | day-1-mentality, customer-obsession, long-term |

### Light-touch Mentors (5 — new additions, agent infers behavior from tags)

| ID | Name | Core Tags |
|----|------|-----------|
| buffett | Warren Buffett | long-term-value, margin-of-safety, patience |
| zhangyiming | Zhang Yiming (张一鸣) | delayed-gratification, context-not-control, data-driven |
| leijun | Lei Jun (雷军) | extreme-value, user-participation, focus |
| caodewang | Cao Dewang (曹德旺) | industrial-spirit, cost-control, craftsmanship |
| chushijian | Chu Shijian (褚时健) | ultimate-focus, quality-obsession, resilience |

### Mentor Blending

- Blend any two mentors with configurable primary weight (50-90%)
- Questions merged from both mentors
- Decision framework uses primary mentor, supplemented by secondary
- If cloud API available, blending uses full configs; otherwise agent infers from tags

## 6. Cultural Adaptation (6 Packs)

| Culture | Directness | Hierarchy | Key Rules |
|---------|-----------|-----------|-----------|
| default | High | Low | Neutral/Western-default, direct communication, merit-based feedback |
| philippines | Low | High | Never name in group, warmth required, acknowledge effort |
| singapore | High | Medium | Direct but polite, efficiency-focused |
| indonesia | Low | High | Relationship-first, group harmony |
| srilanka | Low | High | Respectful tone, private feedback |
| malaysia | Medium | Medium | Multicultural sensitivity, balanced approach |
| china | Medium | High | Face-saving, collective achievement |

Culture packs override mentor strategy when conflicts exist (e.g., Dalio wants public feedback but Filipino culture requires private-first).

## 7. Data Architecture

### Configuration
Stored at `~/.openclaw/skills/boss-ai-agent/config.json`:

```json
{
  "mentor": "musk",
  "mentorBlend": { "secondary": "inamori", "weight": 70 },
  "culture": "china",
  "timezone": "Asia/Shanghai",
  "team": [
    {
      "name": "John Santos",
      "channel": "telegram",
      "channelId": "123456",
      "culture": "philippines",
      "role": "engineer",
      "github": "johnsantos"
    }
  ],
  "integrations": {
    "github": { "repos": ["tonypk/myproject"], "enabled": true },
    "linear": { "team": "MY-TEAM", "enabled": false },
    "notion": { "workspace": "xxx", "enabled": true },
    "gmail": { "enabled": false }
  },
  "schedule": {
    "checkin": "0 9 * * 1-5",
    "chase": "30 17 * * 1-5",
    "summary": "0 19 * * 1-5",
    "weeklyReview": "0 9 * * 1",
    "briefing": "0 8 * * 1-5",
    "signalScan": "*/30 9-18 * * 1-5"
  },
  "alerts": {
    "consecutiveMisses": 3,
    "sentimentDropThreshold": -0.3,
    "urgentKeywords": ["urgent", "down", "broken", "紧急", "挂了"]
  }
}
```

### Memory (All via OpenClaw `memory` tool)

| Type | Key Prefix | Content |
|------|-----------|---------|
| Employee profiles | `emp:{name}` | Performance trends, personality, strengths/weaknesses |
| Management decisions | `decision:{date}` | Decisions made + outcomes |
| Project status | `project:{name}` | Milestones, risk records |
| Boss preferences | `boss:pref` | Communication style, focus areas |

### Configuration Schema

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| `mentor` | No | `"musk"` | Any valid mentor ID |
| `mentorBlend` | No | `null` | Optional secondary mentor |
| `culture` | No | `"default"` | Team-level default culture |
| `timezone` | Yes | — | IANA timezone string |
| `team` | No | `[]` | Empty = solo founder mode |
| `integrations` | No | all disabled | Each integration has `enabled: boolean` |
| `schedule` | No | defaults above | Standard 5-field cron expressions |
| `alerts` | No | defaults above | Threshold configuration |

**Empty team guard**: If `team` is empty, skip daily check-in/chase/summary cron jobs. Daily briefing and project patrol still function for solo founders. Prompt boss to add team members.

**Env var fallback**: If `BOSS_AI_AGENT_API_KEY` is not set, check for legacy `MANAGEMENT_BRAIN_API_KEY` and use it as fallback.

**Integration auth**: GitHub/Linear/Jira access relies on OpenClaw's configured integrations (MCP servers or OAuth tokens managed by the OpenClaw gateway). The skill does NOT store auth tokens — it uses `web_fetch` which inherits the gateway's authenticated sessions. For public repos, no auth needed.

### Degradation Strategy

| Component | With API Key | Without API Key |
|-----------|-------------|----------------|
| Mentor configs | Full configs from cloud API | 3 fully-embedded + 6 with questions + 5 inferred from tags |
| Web dashboard | Available at manageaibrain.com | Not available |
| All 7 scenarios | Fully functional | Fully functional |
| Memory | OpenClaw `memory` | OpenClaw `memory` |

### First Run Onboarding

```
User: /boss-ai-agent
Agent: Hi! I'm Boss AI Agent, your AI management middleware.
       Let me get to know your team:

       1. How many people do you manage?
       2. What communication tools does your team use? (I detect you have Telegram and Slack)
       3. Do you use GitHub/Linear for project management?

→ Auto-configure config.json from answers
→ [cron add] register scheduled tasks
→ [message send] send test message to verify channels
→ Recommend mentor: "Based on your team size and industry, I recommend Musk mode (execution-oriented). Want to try it?"
```

## 8. SKILL.md File Structure

```
---
YAML frontmatter (name, version, description, metadata.openclaw)
---

# Boss AI Agent

## Identity (definition + mentor permeation principle)

## First Run (onboarding flow)

## Tool Reference (cheat sheet — how to invoke each tool)
  ### message tool
  ### cron tool
  ### memory tool
  ### sessions_spawn tool
  ### web_fetch tool
  ### other tools

## Scenarios (7 scenarios, each with workflow + tool invocations)
  ### 1. Daily Management Cycle
  ### 2. Project Health Patrol
  ### 3. Smart Daily Briefing
  ### 4. 1:1 Meeting Assistant
  ### 5. Real-time Signal Monitoring
  ### 6. Knowledge Base Management
  ### 7. Emergency Response

## Mentor System (3 complete + 11 brief)
  ### Mentor Decision Matrix (Musk / Inamori / Ma)
  ### Other Mentors Quick Reference
  ### Mentor Blending Rules

## Cultural Adaptation (6 culture packs)

## Cloud API (optional — manageaibrain.com endpoints)

## 中文说明

## Links
```

Estimated length: ~800-1200 lines of markdown (3 full decision matrices + 6 mentor questions + 7 scenarios + tool reference).

README.md: Updated with user-facing install/usage guide only, no agent instruction details.

## 9. Publishing Plan

- **ClawHub slug**: `boss-ai-agent` (new skill, not updating `management-brain`)
- **Version**: `1.0.0`
- **Env vars**: `BOSS_AI_AGENT_API_KEY` (optional)
- **Config path**: `~/.openclaw/skills/boss-ai-agent/config.json`
- Old `management-brain` skill remains on ClawHub (not deleted)

## 10. Migration from management-brain

- `boss-ai-agent` is a **new skill**, not an upgrade of `management-brain`
- Old `management-brain` remains on ClawHub, users can keep using it
- No automatic data migration — different storage model (OpenClaw `memory` vs Notion/Sheets/local JSON)
- Users who want to switch: install `boss-ai-agent`, re-run onboarding, add team members
- Legacy env var `MANAGEMENT_BRAIN_API_KEY` is accepted as fallback for `BOSS_AI_AGENT_API_KEY`

## 11. Versioning Strategy

- Follow semver: Major = breaking config changes, Minor = new scenarios/mentors, Patch = wording fixes
- Publish via `clawhub publish openclaw-skill --slug boss-ai-agent --version X.Y.Z`
- SKILL.md `version` field in frontmatter must match the published version

## 12. Out of Scope

- Backend API changes (no Go code changes)
- New API endpoints
- Database migrations
- Frontend changes
- Deleting or hiding the old `management-brain` skill
