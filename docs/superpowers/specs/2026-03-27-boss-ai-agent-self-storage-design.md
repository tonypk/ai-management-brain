# Boss AI Agent — Self-Storage Mode Design

## Goal

Enable the `boss-ai-agent` OpenClaw skill to work **without** manageaibrain.com by storing data in the user's own storage: local files, Notion, or Google Sheets. Users install community MCP servers for Notion/Sheets; the skill auto-detects available tools and adapts.

## Architecture

```
boss-ai-agent SKILL.md (instruction-only, published on OpenClaw)
    │
    ├── Mentor Frameworks (9 mentors, 6 cultures) — unchanged
    ├── Management Logic (check-in analysis, reports, advice) — unchanged
    │
    └── Storage Layer — NEW
        ├── Storage Detection (auto-detect at startup)
        └── Storage Adapters
            ├── Local File Adapter (default fallback)
            ├── Notion Adapter (requires community Notion MCP)
            └── Sheets Adapter (requires community Sheets MCP)
```

### Key Principle

The skill's core value — mentor frameworks, management analysis, check-in question design — lives in the prompt. Storage is just where data lands. By decoupling storage from logic, users control their data without losing any management intelligence.

## Storage Detection

On skill activation, first check for a config override, then auto-detect:

```
0. Read config at ~/.openclaw/skills/boss-ai-agent/config.json
   → If "storage" field is set AND the required MCP tools are available:
     use the configured mode. Skip auto-detection.

1. Check for manageaibrain MCP tools (any tool containing "management_brain")
   → Found: Cloud Mode (existing behavior, no changes)

2. Check for Notion MCP tools (any tool containing "notion" in its name)
   → Found: Notion Mode

3. Check for Sheets MCP tools (any tool containing "sheets" or "spreadsheet" in its name)
   → Found: Sheets Mode

4. None found: Local Mode (Claude Code Read/Write tools)
```

Default priority (when no config override): **Cloud > Notion > Sheets > Local**

Config override takes precedence — if user sets `"storage": "notion"`, Notion mode activates even when Cloud MCP tools are present.

### Fuzzy Tool Detection

MCP tool names vary across community servers. Detection must be **fuzzy** — match by substring, not exact name:

| Mode | Match pattern | Example tools |
|------|--------------|---------------|
| Cloud | tool name contains `management_brain` | `mcp__management-brain__get_team_status` |
| Notion | tool name contains `notion` | `notion_search`, `search_notion`, `notion-query` |
| Sheets | tool name contains `sheets` or `spreadsheet` | `sheets_read`, `google_sheets_append`, `read_spreadsheet` |

After detection, the skill discovers exact tool names by listing available tools and stores the mapping in config for subsequent sessions.

## Core Data Entities (5)

Only the 5 essential entities are stored. AI-derived data (signals, recommendations, working memory) is computed in real-time from raw data — never persisted.

### 1. Employees

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier (adapter-specific — see ID Strategy) |
| name | string | Full name |
| role | string | Team role (frontend, backend, etc.) |
| job_title | string | Job title |
| responsibilities | string | Key responsibilities |
| country | string | Country |
| language | string | Preferred language |
| strengths | string | Known strengths |
| risk_flags | string | Current risk indicators |
| current_load | number | Workload 1-10 |
| is_active | boolean | Whether employee is active (default: true) |

### 2. Check-ins (Reports)

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| employee_name | string | Who submitted |
| date | date | Check-in date |
| answers | array | Structured Q&A (see schema below) |
| blockers | string | Current blockers |
| sentiment | string | positive/neutral/anxious/frustrated |

**Answers schema** (must always serialize to this structure):
```json
{
  "answers": [
    { "question": "What did you accomplish today?", "answer": "Finished the login page." },
    { "question": "Any blockers?", "answer": "Waiting on API docs from backend team." }
  ]
}
```

### 3. Tasks

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| title | string | Task title |
| owner | string | Assigned employee name |
| status | string | todo/in_progress/done/blocked/archived |
| priority | string | low/medium/high/critical |
| due_date | date | Deadline |
| project | string | Parent project name |

### 4. Goals

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| title | string | Goal title |
| owner | string | Responsible person |
| status | string | on_track/at_risk/behind/completed |
| cycle | string | Q1 2026, Q2 2026, etc. |
| key_results | array | List of {title, target, current, unit} |

**key_results** is stored as a native JSON array in Local mode. In Notion (text property) and Sheets (single cell), it is serialized as a JSON string. The skill must parse it back to an array on read.

### 5. Metrics

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Metric name |
| value | number | Current value |
| target | number | Target value |
| unit | string | Unit (%, count, $, etc.) |
| updated_at | date | Last updated |

### ID Strategy

IDs are adapter-specific and not portable across storage modes:
- **Local**: UUID v4 generated by Claude
- **Notion**: Notion page ID (assigned by Notion on create)
- **Sheets**: UUID v4 generated by Claude (stored in column A)

Cross-references (e.g., Tasks.owner) use **employee name** (not ID) for portability. This is acceptable because team sizes in this tool are typically small (<50).

### Soft Delete

No entity is physically deleted. Instead, all entities support soft delete:
- **Employees**: `is_active` field set to `false`
- **Tasks**: `status` set to `archived`
- **Goals**: `status` set to `completed` (no archive — completed goals are historical records)
- **Metrics/Check-ins**: Never deleted (append-only)

On read, the skill filters out inactive employees and archived tasks by default. All data remains accessible for historical analysis.

## Storage Adapters

### Local File Adapter

**Directory structure:**
```
~/.boss-ai-agent/data/
├── employees.json       # [{...}, {...}]
├── tasks.json           # [{...}, {...}]
├── goals.json           # [{...}, {...}]
├── metrics.json         # [{...}, {...}]
└── reports/
    ├── 2026-03-27.json  # per-day file
    └── 2026-03-28.json
```

Config is stored at `~/.openclaw/skills/boss-ai-agent/config.json` (shared with existing skill).

**Operations use Claude Code native tools:**
- Read: `Read` tool on JSON files
- Write: parse JSON → modify in-memory → `Write` tool
- **Constraint**: Writes must be sequential — complete one file write before beginning the next. Never write to two files in parallel.

**Check-in input:** Boss dictates to Claude ("Alice said she finished the API, Bob is blocked on docs"), Claude parses into structured data and writes.

### Notion Adapter

**Databases (5):**
- "Team" — Employees
- "Check-ins" — Daily reports
- "Tasks" — Work items
- "Goals" — OKRs
- "Metrics" — KPIs

**Property mappings:**

| Entity | Notion Properties |
|--------|------------------|
| Employee | Title: name, Select: role, Text: job_title, Text: responsibilities, Select: country, Select: language, Text: strengths, Text: risk_flags, Number: current_load, Checkbox: is_active |
| Check-in | Relation→Team: employee, Date: date, Text: answers (JSON string), Text: blockers, Select: sentiment |
| Task | Title: title, Relation→Team: owner, Status: status, Select: priority, Date: due_date, Text: project |
| Goal | Title: title, Relation→Team: owner, Status: status, Select: cycle, Text: key_results (JSON string) |
| Metric | Title: name, Number: value, Number: target, Text: unit, Date: updated_at |

**Operations:** Use whichever Notion MCP tools are available. The skill discovers exact tool names at detection time and maps them to these operations:
- **List**: query/search a database by ID
- **Create**: create a page in a database with properties
- **Update**: update page properties by page ID
- **Archive**: update page `archived` property to true (soft delete)

Database IDs stored in `~/.openclaw/skills/boss-ai-agent/config.json`.

**Check-in input:** Employees fill in Notion database directly (collaborative). Skill reads on session start.

**Existing data detection:** On first run, search for databases matching expected names. If found, ask user whether to reuse or create new.

### Sheets Adapter

**Spreadsheet structure (1 spreadsheet, 5 tabs):**

Spreadsheet: "Boss AI Agent"

| Tab | Columns |
|-----|---------|
| Employees | A: ID, B: Name, C: Role, D: Job Title, E: Responsibilities, F: Country, G: Language, H: Strengths, I: Risk Flags, J: Load, K: Active |
| Check-ins | A: ID, B: Employee, C: Date, D: Answers, E: Blockers, F: Sentiment |
| Tasks | A: ID, B: Title, C: Owner, D: Status, E: Priority, F: Due Date, G: Project |
| Goals | A: ID, B: Title, C: Owner, D: Status, E: Cycle, F: Key Results |
| Metrics | A: ID, B: Name, C: Value, D: Target, E: Unit, F: Updated |

Row 1 is always the header row.

**Operations:** Use whichever Sheets MCP tools are available:
- **List**: read range "TabName!A:Z"
- **Create**: append row to tab
- **Update**: update specific cell range
- **Soft delete**: update `Active` column to `FALSE` (employees) or `Status` column to `archived` (tasks)

On read, filter rows where Active=FALSE or Status=archived.

Spreadsheet ID stored in `~/.openclaw/skills/boss-ai-agent/config.json`.

**Check-in input:** Employees fill in "Check-ins" tab directly (shared spreadsheet). Skill reads on session start.

## First-Run Initialization

### Local Mode

1. `mkdir -p ~/.boss-ai-agent/data/reports/`
2. Write config: `{ "storage": "local", "mentor": "inamori" }` to `~/.openclaw/skills/boss-ai-agent/config.json`
3. Write empty arrays: `employees.json`, `tasks.json`, `goals.json`, `metrics.json`
4. Prompt user: "Local storage initialized. Tell me about your team to get started."

### Notion Mode

1. Search for existing databases matching names ("Team", "Check-ins", etc.)
2. If found: ask user "Found existing 'Team' database with N members. Use it?"
3. If not found or user declines: attempt to create 5 new databases with property schemas
4. **If database creation fails** (MCP server may not support it): inform user that manual creation is needed. Provide exact database names and property schemas. Then search for the user-created databases and store their IDs.
5. Save database IDs to `~/.openclaw/skills/boss-ai-agent/config.json`
6. Prompt user: "Notion databases ready. Share 'Check-ins' with your team for daily submissions."

### Sheets Mode

1. Attempt to create spreadsheet "Boss AI Agent"
2. **If creation fails**: ask user to create a Google Sheet manually, name it "Boss AI Agent", and share the URL. Extract spreadsheet ID from URL.
3. Create 5 tabs with header rows
4. Save spreadsheet ID to `~/.openclaw/skills/boss-ai-agent/config.json`
5. Prompt user: "Google Sheet ready. Share it with your team for daily check-ins."

## Automation: Session-Driven (No Cron)

Without a backend server, cron jobs are replaced by session-start scans and user commands.

### Session Start Scan

Every time the skill activates, it reads **recent** data and generates a briefing:

**Data windows:**
- Check-ins: last **7 days** only
- Tasks: only **open** tasks (status != done/archived)
- Goals: **current cycle** only
- Metrics: all (typically small dataset)
- Employees: only **active** (is_active = true)

For Local mode, check-in files are read by date-filtered filename (e.g., only files from the last 7 days), not by scanning all files.

```
Good morning! Today's status:
- 3/5 check-ins submitted
- Alice: completed login page, sentiment: positive
- Bob: blocked by CI issue (day 3), sentiment: anxious
- Charlie, Dave, Eve: no submission
- 2 overdue tasks
- Q1 OKR deadline in 4 days, progress 60%

What would you like to focus on?
```

### Command Mapping

| Cloud Mode (cron) | Self-Storage Mode (user command) |
|-------------------|----------------------------------|
| 9am send check-in reminders | Session start: "Who hasn't submitted?" |
| 5:30pm chase non-responders | "Chase [name]" or "remind team" |
| 7pm daily summary | "Today's summary" |
| Friday weekly report | "Weekly report" |
| 10:30am risk scan | Session start: auto-scan |
| 11am project patrol | "Check projects" |
| KPI dashboard refresh | "Show KPIs" |
| Goal progress check | "Goal progress" |
| Monthly incentive calc | "Calculate incentives" |

### Analysis Computation

All AI analysis runs in real-time from raw data:

1. **Sentiment trends**: Read last 7-14 days of check-ins → compute trend per employee
2. **Blocker detection**: Scan check-ins for repeated blockers → flag persistent issues
3. **Risk signals**: Cross-reference tasks (overdue), check-ins (sentiment), goals (off-track)
4. **Recommendations**: Apply mentor framework to current data → generate actionable advice
5. **Weekly report**: Aggregate all check-ins + tasks + goals for the week

No intermediate results are stored. Each analysis is fresh from source data.

## SKILL.md Structure (Updated)

```
SKILL.md v5.0.0
├── 1. Overview
│   ├── What this skill does
│   ├── Four modes: Cloud / Notion / Sheets / Local
│   └── Privacy: your data, your storage
│
├── 2. Storage Detection (NEW)
│   ├── Config override (takes precedence)
│   ├── Fuzzy tool detection logic
│   └── First-run initialization per mode
│
├── 3. Storage Adapters (NEW)
│   ├── 3a. Local File Adapter
│   │   ├── Directory structure
│   │   ├── Sequential Read/Write operations
│   │   └── Check-in: boss dictation
│   ├── 3b. Notion Adapter
│   │   ├── Database schemas + property mappings
│   │   ├── Discovered Notion MCP operations
│   │   └── Check-in: employee self-fill
│   └── 3c. Sheets Adapter
│       ├── Tab/column schemas
│       ├── Discovered Sheets MCP operations
│       └── Check-in: employee self-fill
│
├── 4. Mentor Frameworks (unchanged)
│   └── 9 mentors, 6 cultures
│
├── 5. Advisor Mode (updated)
│   ├── Pure conversation (no data) — existing
│   └── Data-aware advice (reads local/Notion/Sheets) — NEW
│
├── 6. Team Operations Mode (updated)
│   ├── Session-start scan + briefing (with data windows)
│   ├── Commands: summary, weekly, chase, KPIs, goals, etc.
│   └── Real-time AI analysis (no stored intermediates)
│
└── 7. Data & Privacy (NEW)
    ├── Advisor Mode: no data leaves your machine
    ├── Local Mode: all data in ~/.boss-ai-agent/data/
    ├── Notion Mode: data in your Notion workspace
    ├── Sheets Mode: data in your Google Drive
    └── Cloud Mode: data on manageaibrain.com (existing)
```

## Config Schema

Config path: `~/.openclaw/skills/boss-ai-agent/config.json`

```json
{
  "storage": "notion",
  "mentor": "inamori",
  "culture": "default",
  "initialized_at": "2026-03-27T10:00:00Z",
  "tool_mapping": {
    "notion_search": "mcp__notion__search",
    "notion_create": "mcp__notion__create_page",
    "notion_update": "mcp__notion__update_page",
    "notion_query": "mcp__notion__query_database"
  },
  "notion": {
    "team_db": "abc123-...",
    "checkins_db": "def456-...",
    "tasks_db": "ghi789-...",
    "goals_db": "jkl012-...",
    "metrics_db": "mno345-..."
  },
  "sheets": {
    "spreadsheet_id": "1BxiMVs0..."
  }
}
```

The `tool_mapping` section is auto-populated during storage detection and caches the exact MCP tool names discovered for this user's setup.

## Out of Scope

- **Sync between modes**: No migrating data from Local → Notion or vice versa
- **Real-time notifications**: No push notifications without Cloud mode
- **Multi-user conflict resolution**: Notion/Sheets handle their own collaboration
- **Offline Notion/Sheets**: If MCP server is unavailable, gracefully fall back to last-known data or Local mode
- **Custom entity schemas**: Users cannot add custom fields in v1
- **Cross-adapter ID portability**: IDs are adapter-specific. Switching storage modes starts fresh.
