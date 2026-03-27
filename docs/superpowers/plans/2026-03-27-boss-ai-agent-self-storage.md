# Boss AI Agent Self-Storage Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the `boss-ai-agent` OpenClaw skill (SKILL.md) to support 4 storage modes: Cloud, Notion, Sheets, Local — with auto-detection and user privacy control.

**Architecture:** The SKILL.md is instruction-only (no code). We rewrite sections of the existing v4.0.0 file to add Storage Detection, Storage Adapters, and session-driven Team Operations for non-cloud modes. Unchanged sections (Mentor System, Cultural Adaptation, C-Suite Board) stay as-is.

**Tech Stack:** Markdown (SKILL.md), OpenClaw skill format, clawhub CLI for publishing

**Spec:** `docs/superpowers/specs/2026-03-27-boss-ai-agent-self-storage-design.md`

---

## File Structure

```
claude-plugin/skills/boss-ai-agent/
└── SKILL.md                     — The only file. Rewrite from v4.0.0 to v5.0.0
```

No new files. No code. Pure SKILL.md rewrite.

## Current SKILL.md Structure (v4.0.0, 396 lines)

```
Lines 1-19:    YAML Front Matter
Lines 21-29:   Identity
Lines 31-38:   Mode Detection (simple: check get_team_status)
Lines 40-147:  Permissions & Data + Cron + MCP Tools
Lines 149-212: First Run (Advisor + Team Ops)
Lines 214-279: Advisor Mode
Lines 281-300: Team Operations Mode
Lines 302-353: Mentor System
Lines 355-371: Cultural Adaptation
Lines 373-388: C-Suite Board
Lines 390-396: Links
```

## Target SKILL.md Structure (v5.0.0)

```
YAML Front Matter          — version bump 4.0.0 → 5.0.0, updated description
Identity                   — unchanged
Storage & Mode Detection   — NEW: replaces old Mode Detection
Permissions & Data         — rewrite for 4 modes
Storage Adapters           — NEW: Local, Notion, Sheets sections
First Run                  — rewrite: 4 initialization flows
Advisor Mode               — update: add data-aware advice
Team Operations Mode       — update: session-driven for self-storage
MCP Tools                  — keep (Cloud mode reference)
Cron Job Management        — keep (Cloud mode reference)
Mentor System              — unchanged
Cultural Adaptation        — unchanged
C-Suite Board              — unchanged
Data & Privacy             — NEW section
Links                      — unchanged
```

---

### Task 1: Front Matter + Identity + Storage Detection + Permissions

**Files:**
- Modify: `claude-plugin/skills/boss-ai-agent/SKILL.md:1-105`

**What changes:**
- Version: `"4.0.0"` → `"5.0.0"`
- Description: add "4 storage modes: Cloud, Notion, Sheets, Local"
- Replace "Mode Detection" (lines 31-38) with "Storage & Mode Detection" (fuzzy detection, config override, priority chain)
- Rewrite "Permissions & Data" (lines 40-82) for 4 modes
- Keep Cron Job Management and MCP Tools sections (lines 84-147) — these apply to Cloud mode only, add a note

- [ ] **Step 1: Update YAML front matter**

Replace lines 1-19:

```yaml
---
name: boss-ai-agent
title: "Boss AI Agent"
version: "5.0.0"
description: "Boss AI Agent — your AI management advisor. 16 mentor philosophies, 9 culture packs, C-Suite board simulation. 4 storage modes: Cloud (manageaibrain.com), Notion, Google Sheets, or Local files — your data, your choice. Works instantly after install."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    optional:
      env:
        - name: "BOSS_AI_AGENT_API_KEY"
          description: "Optional. Adds read-only GET access to manageaibrain.com/api/v1/ for extended mentor configs and analytics dashboards. Only relevant in Cloud Mode."
        - name: "MANAGEMENT_BRAIN_API_KEY"
          description: "Legacy fallback for BOSS_AI_AGENT_API_KEY."
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---
```

- [ ] **Step 2: Keep Identity section unchanged (lines 21-29)**

No changes needed.

- [ ] **Step 3: Replace Mode Detection with Storage & Mode Detection**

Replace lines 31-38 with:

```markdown
## Storage & Mode Detection

On activation, detect available storage. **Config override takes precedence** — if `~/.openclaw/skills/boss-ai-agent/config.json` has a `"storage"` field, use that mode (after verifying required tools are available). Otherwise, auto-detect in priority order:

### Detection Logic

```
0. Read config → if "storage" field set AND required tools available → use configured mode

1. Any tool name containing "management_brain" → Cloud Mode
   Announce: "Running in Cloud Mode — connected to manageaibrain.com."

2. Any tool name containing "notion" → Notion Mode
   Announce: "Running in Notion Mode — data stored in your Notion workspace."

3. Any tool name containing "sheets" or "spreadsheet" → Sheets Mode
   Announce: "Running in Sheets Mode — data stored in your Google Sheet."

4. None matched → Local Mode
   Announce: "Running in Local Mode — data stored in ~/.boss-ai-agent/data/."
```

After detection, discover exact tool names and cache in config under `tool_mapping` for subsequent sessions.

### Mode Capabilities

| Capability | Cloud | Notion | Sheets | Local |
|-----------|-------|--------|--------|-------|
| Mentor frameworks | ✅ | ✅ | ✅ | ✅ |
| C-Suite board | ✅ | ✅ | ✅ | ✅ |
| Store employees | ✅ | ✅ | ✅ | ✅ |
| Store check-ins | ✅ | ✅ | ✅ | ✅ |
| Store tasks/goals/metrics | ✅ | ✅ | ✅ | ✅ |
| Employee self-fill check-ins | ✅ | ✅ | ✅ | ❌ (boss dictates) |
| Automated cron jobs | ✅ | ❌ | ❌ | ❌ |
| Push notifications | ✅ | ❌ | ❌ | ❌ |
| 22 MCP tools | ✅ | ❌ | ❌ | ❌ |
| Real-time AI analysis | ✅ | ✅ | ✅ | ✅ |
| Data location | manageaibrain.com | Your Notion | Your Google Drive | Your machine |
```

- [ ] **Step 4: Rewrite Permissions & Data section**

Replace lines 40-82 with:

```markdown
## Permissions & Data

### All Modes

- **Config file**: writes `~/.openclaw/skills/boss-ai-agent/config.json` during first run (storage mode, mentor preference, culture setting, tool mapping). User can read, edit, or delete this file at any time.
- **No secrets stored**: Config contains settings and IDs only — no API keys, passwords, or tokens.

### Local Mode

- **File writes**: creates and updates JSON files in `~/.boss-ai-agent/data/`. Data never leaves your machine.
- **No network access**: zero HTTP requests.
- **No cron jobs**: all interactions are user-initiated.

### Notion Mode

- **Notion MCP tools**: uses the user's installed Notion MCP server to create/read/update Notion databases. Data is stored in the user's own Notion workspace.
- **No direct API calls**: the skill does NOT make HTTP requests to Notion — it delegates to the MCP server.
- **No cron jobs**: all interactions are user-initiated.

### Sheets Mode

- **Sheets MCP tools**: uses the user's installed Google Sheets MCP server to read/write spreadsheet data. Data is stored in the user's own Google Drive.
- **No direct API calls**: the skill does NOT make HTTP requests to Google — it delegates to the MCP server.
- **No cron jobs**: all interactions are user-initiated.

### Cloud Mode

All permissions from the previous version (v4.0.0):

- **MCP tools**: 22 tools hosted on `manageaibrain.com/mcp`. Tool parameters (employee name, topic, period) are sent to the cloud server. 18 read-only queries; 4 write tools (`send_checkin`, `chase_employee`, `send_summary`, `send_message`) send messages to employees via Telegram/Slack/Lark/Signal.
- **Cron jobs**: registers up to 5 recurring jobs via OpenClaw's cron API.
- **Cloud API** (optional): when `BOSS_AI_AGENT_API_KEY` is set, makes read-only GET requests to `manageaibrain.com/api/v1/`.
```

- [ ] **Step 5: Add "Cloud Mode only" note to Cron and MCP sections**

Keep lines 84-147 (Cron Job Management + MCP Tools) but add a header note to each:

At the start of the Cron section (line 84), add:
```markdown
> **Cloud Mode only.** Cron jobs require the manageaibrain.com MCP connection. Self-storage modes (Notion/Sheets/Local) use session-driven automation instead — see [Session-Driven Automation](#session-driven-automation).
```

At the start of the MCP Tools section (line 106), add:
```markdown
> **Cloud Mode only.** These 22 MCP tools are available when connected to manageaibrain.com. Self-storage modes use their own storage adapters instead — see [Storage Adapters](#storage-adapters).
```

- [ ] **Step 6: Commit**

```bash
cd /Users/anna/Documents/ai-management-brain
git add claude-plugin/skills/boss-ai-agent/SKILL.md
git commit -m "feat(skill): v5.0 — storage detection + 4-mode permissions"
```

---

### Task 2: Storage Adapters Section

**Files:**
- Modify: `claude-plugin/skills/boss-ai-agent/SKILL.md` (insert new section after Permissions & Data, before First Run)

**What changes:**
- Add complete "Storage Adapters" section with Local, Notion, Sheets subsections
- Each adapter defines: data structure, operations, and check-in input method

- [ ] **Step 1: Insert Storage Adapters section**

Insert after the MCP Tools section (end of Cloud-only reference material), before "First Run":

```markdown
## Storage Adapters

> Applies to **Notion Mode**, **Sheets Mode**, and **Local Mode**. Cloud Mode uses MCP tools instead.

### Core Data Entities

5 entities are stored. AI-derived analysis (risk signals, recommendations, sentiment trends) is computed in real-time from this raw data — never persisted.

| Entity | Fields | Notes |
|--------|--------|-------|
| Employees | name, role, job_title, responsibilities, country, language, strengths, risk_flags, current_load (1-10), is_active | Soft delete: set is_active=false |
| Check-ins | employee_name, date, answers (JSON array), blockers, sentiment | Append-only. Answers format: `[{"question":"...","answer":"..."}]` |
| Tasks | title, owner (employee name), status, priority, due_date, project | Statuses: todo/in_progress/done/blocked/archived |
| Goals | title, owner, status, cycle (e.g. "Q1 2026"), key_results (JSON array) | key_results: `[{"title":"...","target":100,"current":60,"unit":"%"}]` |
| Metrics | name, value, target, unit, updated_at | Append-only |

Cross-references use **employee name** (not ID) — team sizes are small enough for name matching.

### Local File Adapter

**Directory**: `~/.boss-ai-agent/data/`

```
~/.boss-ai-agent/data/
├── employees.json       # [{name, role, ...}, ...]
├── tasks.json           # [{title, owner, ...}, ...]
├── goals.json           # [{title, owner, ...}, ...]
├── metrics.json         # [{name, value, ...}, ...]
└── reports/
    ├── 2026-03-27.json  # [{employee_name, answers, ...}, ...]
    └── 2026-03-28.json
```

**Operations**:
- **Read**: Use the `Read` tool on JSON files. For check-ins, read by filename date (last 7 days only).
- **Write**: Read file → parse JSON → modify array in memory → write back with `Write` tool.
- **Constraint**: Complete one file write before starting the next. Never write two files in parallel.
- **IDs**: Generate UUID v4 for each new record.

**Check-in input**: Boss dictates to you. Parse natural language into structured check-in data.
Example: "Alice finished the API, Bob is blocked on docs" → creates 2 check-in records with parsed answers, blockers, and inferred sentiment.

### Notion Adapter

**Databases** (5, created or discovered on first run):

| Database Name | Entity | Key Properties |
|--------------|--------|---------------|
| Team | Employees | Title: name, Select: role, Text: job_title, Text: responsibilities, Select: country, Checkbox: is_active |
| Check-ins | Check-ins | Relation→Team: employee, Date: date, Text: answers (JSON string), Text: blockers, Select: sentiment |
| Tasks | Tasks | Title: title, Relation→Team: owner, Status: status, Select: priority, Date: due_date |
| Goals | Goals | Title: title, Relation→Team: owner, Status: status, Select: cycle, Text: key_results (JSON string) |
| Metrics | Metrics | Title: name, Number: value, Number: target, Text: unit, Date: updated_at |

**Operations**: Use whichever Notion MCP tools are available (tool names vary by server — use the discovered `tool_mapping` from config):
- **List**: query database by ID
- **Create**: create page with properties
- **Update**: update page properties
- **Archive**: set page archived=true (soft delete)

Database IDs are stored in config under `notion.team_db`, `notion.checkins_db`, etc.

**Check-in input**: Employees fill in the "Check-ins" Notion database directly. You read new entries on each session start.

### Sheets Adapter

**Spreadsheet**: "Boss AI Agent" (1 spreadsheet, 5 tabs)

| Tab | Columns |
|-----|---------|
| Employees | A: ID, B: Name, C: Role, D: Job Title, E: Responsibilities, F: Country, G: Language, H: Strengths, I: Risk Flags, J: Load, K: Active |
| Check-ins | A: ID, B: Employee, C: Date, D: Answers, E: Blockers, F: Sentiment |
| Tasks | A: ID, B: Title, C: Owner, D: Status, E: Priority, F: Due Date, G: Project |
| Goals | A: ID, B: Title, C: Owner, D: Status, E: Cycle, F: Key Results |
| Metrics | A: ID, B: Name, C: Value, D: Target, E: Unit, F: Updated |

Row 1 is always the header row.

**Operations**: Use whichever Sheets MCP tools are available (use discovered `tool_mapping`):
- **List**: read range "TabName!A:Z"
- **Create**: append row
- **Update**: update cell range
- **Soft delete**: set Active column to FALSE, or Status to "archived"

On read, filter out rows where Active=FALSE or Status=archived.

Spreadsheet ID stored in config under `sheets.spreadsheet_id`.

**Check-in input**: Employees fill in the "Check-ins" tab directly. You read new rows on each session start.
```

- [ ] **Step 2: Commit**

```bash
git add claude-plugin/skills/boss-ai-agent/SKILL.md
git commit -m "feat(skill): add storage adapters — Local, Notion, Sheets"
```

---

### Task 3: First Run Flows

**Files:**
- Modify: `claude-plugin/skills/boss-ai-agent/SKILL.md` (rewrite First Run section, lines 149-212)

**What changes:**
- Replace 2 first-run flows (Advisor + Team Ops) with 4 flows (Local + Notion + Sheets + Cloud)
- Local/Notion/Sheets flows follow the spec's initialization steps
- Cloud flow is the existing Team Ops first run (keep as-is, just relabel)

- [ ] **Step 1: Rewrite First Run section**

Replace lines 149-212 with:

```markdown
## First Run

### Local Mode First Run

1. Greet: "Hi! I'm Boss AI Agent. Running in **Local Mode** — your data stays on your machine."
2. Ask: "Which mentor philosophy resonates with you?" Present top 3 (Musk, Inamori, Ma) + option for full list.
3. Create data directory: `mkdir -p ~/.boss-ai-agent/data/reports/`
4. Write empty JSON files: `employees.json`, `tasks.json`, `goals.json`, `metrics.json` (each containing `[]`)
5. Write config to `~/.openclaw/skills/boss-ai-agent/config.json`:
```json
{
  "storage": "local",
  "mentor": "musk",
  "culture": "default",
  "mode": "local"
}
```
6. Ask: "Tell me about your team — names, roles, and responsibilities."
7. Parse team info → write to `employees.json`.
8. Mention: "Your team members can't self-fill check-ins in Local Mode. You'll dictate their updates to me, and I'll record them. Want to switch to Notion or Google Sheets for collaborative check-ins? Just install a Notion or Sheets MCP server."

### Notion Mode First Run

1. Greet: "Hi! I'm Boss AI Agent. Running in **Notion Mode** — data stored in your Notion workspace."
2. Ask mentor question (same as Local).
3. Search for existing Notion databases named "Team", "Check-ins", "Tasks", "Goals", "Metrics".
   - If found: "Found an existing 'Team' database with N entries. Use it?" → if yes, store DB ID; if no, create new.
   - If not found: attempt to create 5 databases.
   - **If creation fails**: "Your Notion MCP doesn't support database creation. Please create these 5 databases manually in Notion:" (list names + properties from Storage Adapters section). "Then tell me 'done' and I'll find them."
4. Save database IDs to config:
```json
{
  "storage": "notion",
  "mentor": "inamori",
  "culture": "default",
  "mode": "notion",
  "tool_mapping": { ... },
  "notion": {
    "team_db": "...",
    "checkins_db": "...",
    "tasks_db": "...",
    "goals_db": "...",
    "metrics_db": "..."
  }
}
```
5. Ask about team → create employee pages in "Team" database.
6. Mention: "Share the 'Check-ins' database with your team so they can submit daily updates directly."

### Sheets Mode First Run

1. Greet: "Hi! I'm Boss AI Agent. Running in **Sheets Mode** — data stored in your Google Sheet."
2. Ask mentor question.
3. Attempt to create spreadsheet "Boss AI Agent" with 5 tabs + header rows.
   - **If creation fails**: "Please create a Google Sheet named 'Boss AI Agent' with 5 tabs: Employees, Check-ins, Tasks, Goals, Metrics. Share the URL with me."
   - Extract spreadsheet ID from URL.
4. Save spreadsheet ID to config:
```json
{
  "storage": "sheets",
  "mentor": "ma",
  "culture": "default",
  "mode": "sheets",
  "tool_mapping": { ... },
  "sheets": {
    "spreadsheet_id": "1BxiMVs..."
  }
}
```
5. Ask about team → append rows to "Employees" tab.
6. Mention: "Share this spreadsheet with your team so they can fill in daily check-ins."

### Cloud Mode First Run

(Existing Team Operations Mode first run — no changes)

1. Greet: "Hi! I'm Boss AI Agent. Running in **Cloud Mode** — connected to manageaibrain.com."
2. Ask 3 questions: team size, communication tools, project management tools.
3. Write full config with schedule and alerts.
4. Register cron jobs.
5. If team size = 0: solo founder mode.
6. Recommend mentor based on team size and style.
```

- [ ] **Step 2: Commit**

```bash
git add claude-plugin/skills/boss-ai-agent/SKILL.md
git commit -m "feat(skill): rewrite first-run flows for 4 storage modes"
```

---

### Task 4: Update Advisor Mode + Team Operations Mode

**Files:**
- Modify: `claude-plugin/skills/boss-ai-agent/SKILL.md` (update Advisor Mode lines 214-279, Team Ops lines 281-300)

**What changes:**
- Advisor Mode: add "Data-Aware Advice" subsection (when storage data is available)
- Team Operations Mode: add "Session-Driven Automation" subsection for self-storage modes
- Keep all existing content (Cloud scenarios, mentor-specific behaviors)

- [ ] **Step 1: Add Data-Aware Advice to Advisor Mode**

After the existing Advisor Mode intro (line 216), add:

```markdown
### Data-Aware Advice (Notion/Sheets/Local Modes)

When storage data is available (employees, check-ins, tasks, goals, metrics), Advisor Mode becomes data-driven:

**Session Start Briefing** — Every session, read recent data and generate a briefing:

Data windows for reading:
- Check-ins: last **7 days** only (filter by date)
- Tasks: only **open** (status != done/archived)
- Goals: **current cycle** only
- Metrics: all
- Employees: only **active** (is_active = true)

```
Good morning! Today's status:
- 3/5 check-ins submitted yesterday
- Alice: completed login page, sentiment: positive
- Bob: blocked by CI issue (day 3), sentiment: anxious
- Charlie, Dave, Eve: no submission
- 2 overdue tasks
- Q1 OKR deadline in 4 days, progress: 60%

What would you like to focus on?
```

**Real-Time Analysis** — Compute from raw data on demand (never stored):
1. Sentiment trends: last 7-14 days of check-ins per employee
2. Blocker detection: repeated blockers across check-ins
3. Risk signals: overdue tasks + negative sentiment + off-track goals
4. Recommendations: mentor framework applied to current data

**Without data** (no storage initialized): fall back to pure conversation advice (existing behavior).
```

- [ ] **Step 2: Add Session-Driven Automation to Team Operations Mode**

After the existing "10 Automated Scenarios" table (line 299), add:

```markdown
### Session-Driven Automation (Notion/Sheets/Local Modes)

Without cron jobs, automation is replaced by session-start scans and user commands:

| Cloud Mode (cron) | Self-Storage Mode (user command) |
|-------------------|----------------------------------|
| 9am check-in reminders | Session start: "Who hasn't submitted?" |
| 5:30pm chase non-responders | "Chase [name]" or "Remind team" |
| 7pm daily summary | "Today's summary" |
| Friday weekly report | "Weekly report" |
| 10:30am risk scan | Session start: auto-scan |
| 11am project patrol | "Check projects" |
| KPI dashboard refresh | "Show KPIs" |
| Goal progress check | "Goal progress" |
| Monthly incentive calc | "Calculate incentives" |

All 10 Automated Scenarios above work in self-storage modes — they just require explicit user commands instead of running on cron schedules. The mentor framework shapes all outputs identically regardless of storage mode.
```

- [ ] **Step 3: Commit**

```bash
git add claude-plugin/skills/boss-ai-agent/SKILL.md
git commit -m "feat(skill): data-aware advisor + session-driven automation"
```

---

### Task 5: Data & Privacy Section + Validation + Publish

**Files:**
- Modify: `claude-plugin/skills/boss-ai-agent/SKILL.md` (add Data & Privacy before Links)

**What changes:**
- Add "Data & Privacy" section summarizing data location per mode
- Validate complete SKILL.md structure
- Publish v5.0.0 to OpenClaw

- [ ] **Step 1: Add Data & Privacy section**

Insert before the "Links" section:

```markdown
## Data & Privacy

| Mode | Data Location | Who Controls | Network |
|------|--------------|-------------|---------|
| Local | `~/.boss-ai-agent/data/` on your machine | You | None |
| Notion | Your Notion workspace | You | Notion MCP ↔ Notion API |
| Sheets | Your Google Drive | You | Sheets MCP ↔ Google API |
| Cloud | manageaibrain.com PostgreSQL | manageaibrain.com | MCP ↔ manageaibrain.com |

**Switching modes**: Change `"storage"` in `~/.openclaw/skills/boss-ai-agent/config.json` and re-run `/boss-ai-agent`. Data does not migrate between modes — each mode starts fresh.

**Deleting your data**:
- **Local**: `rm -rf ~/.boss-ai-agent/data/`
- **Notion**: Delete the 5 databases from your Notion workspace
- **Sheets**: Delete the "Boss AI Agent" spreadsheet from Google Drive
- **Cloud**: Contact manageaibrain.com or use the platform's data deletion feature
- **Config**: `rm ~/.openclaw/skills/boss-ai-agent/config.json`
```

- [ ] **Step 2: Validate complete SKILL.md**

Read the complete file and verify:
- [ ] Front matter version is `"5.0.0"`
- [ ] Storage Detection section exists with fuzzy matching
- [ ] All 4 modes documented in Permissions
- [ ] Storage Adapters section has Local, Notion, Sheets
- [ ] First Run has 4 flows
- [ ] Advisor Mode has Data-Aware subsection
- [ ] Team Ops has Session-Driven subsection
- [ ] Data & Privacy section exists
- [ ] Cron + MCP sections have "Cloud Mode only" notes
- [ ] Mentor System, Cultural Adaptation, C-Suite Board unchanged
- [ ] No broken markdown (headers, tables, code blocks)

- [ ] **Step 3: Commit**

```bash
git add claude-plugin/skills/boss-ai-agent/SKILL.md
git commit -m "feat(skill): add data privacy section, complete v5.0.0"
```

- [ ] **Step 4: Publish to OpenClaw**

```bash
cd /Users/anna/Documents/ai-management-brain
clawhub publish openclaw-skill --slug boss-ai-agent --version 5.0.0
```

Expected: `Published boss-ai-agent@5.0.0`

- [ ] **Step 5: Verify published skill**

```bash
clawhub info boss-ai-agent
```

Verify version shows 5.0.0 and description mentions 4 storage modes.

- [ ] **Step 6: Push to GitHub**

```bash
git push origin main
```
