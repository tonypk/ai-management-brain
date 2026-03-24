# MCP Server Design Spec

## Goal

Add a Model Context Protocol (MCP) server to AI Management Brain, enabling AI tools (Claude Code, Cursor, Windsurf) to interact with the management platform directly through 9 tools covering team management, C-Suite board discussions, and employee insights.

## Architecture

```
AI Tool (Claude Code / Cursor / Windsurf)
       │
       │ stdio (JSON-RPC)
       │
  ┌────▼─────────────────────┐
  │  MCP Server (TypeScript)  │
  │  @tonypk/                 │
  │    management-brain-mcp   │
  │                           │
  │  9 tools, stateless       │
  │  api-client → HTTP        │
  └────────┬─────────────────┘
           │
           │ HTTPS + Bearer token
           │
  ┌────────▼─────────────────┐
  │  manageaibrain.com        │
  │  /api/v1/openclaw/*       │
  │  /api/v1/seats/*          │
  │  /api/v1/employees/*      │
  └──────────────────────────┘
```

### Key Decisions

- **Transport**: stdio (standard input/output), the standard local MCP transport
- **SDK**: `@modelcontextprotocol/sdk` (official TypeScript SDK)
- **Auth**: Environment variable `MANAGEMENT_BRAIN_API_KEY`, validated at startup
- **Data source**: Cloud API only (manageaibrain.com REST endpoints). No local database.
- **Stateless**: MCP server holds no state. All data lives on the cloud.
- **Distribution**: npm package `@tonypk/management-brain-mcp`, users run via `npx`

## Tools (9 total)

### Group 1: Core Management (5 tools)

#### `get_team_status`

- **Description**: Get today's team check-in status — submission rate, pending employees, chase counts.
- **Parameters**: None
- **API**: `GET /api/v1/openclaw/status`
- **Returns**: `{ date, total_employees, submitted, pending[], chase_count, mentor, mentor_name }`

#### `get_report`

- **Description**: Get team performance report with ranking and 1:1 suggestions.
- **Parameters**: `period` (required, enum: `"weekly"` | `"monthly"`)
- **API**: `GET /api/v1/openclaw/report?period={period}`
- **Returns**: `{ period, date_range, submission_rate, ranking[], one_on_one_suggestions[] }`

#### `get_alerts`

- **Description**: Get active alerts for employees with consecutive missed check-in days.
- **Parameters**: None
- **API**: `GET /api/v1/openclaw/alerts`
- **Returns**: `{ alerts[{ employee_id, employee_name, missed_days, severity }], total }`

#### `switch_mentor`

- **Description**: Switch the active management mentor philosophy.
- **Parameters**: `mentor` (required, string — mentor ID or name, e.g. "musk", "inamori")
- **API**: `POST /api/v1/openclaw/command` with body `{ "command": "switch mentor {mentor}" }`
- **Returns**: `{ result, mentor_id, name }`

#### `list_mentors`

- **Description**: List all available mentors with domain expertise and recommended C-Suite seats.
- **Parameters**: None
- **API**: `GET /api/v1/seats/mentors`
- **Returns**: `{ data[{ id, name, name_en, company, philosophy, domain, tags[], recommended_seats[] }] }`

### Group 2: C-Suite (2 tools)

#### `board_discuss`

- **Description**: Run a board discussion across all active C-Suite seats on a topic. Each seat responds from their expertise, followed by a synthesis.
- **Parameters**: `topic` (required, string, max 4000 chars)
- **API**: `POST /api/v1/seats/board/discuss` with body `{ "topic": "{topic}" }`
- **Returns**: `{ data: { topic, responses[{ seat_type, title, persona_id, content }], synthesis } }`

#### `chat_with_seat`

- **Description**: Chat directly with a specific C-Suite seat (e.g., ask the CFO about budget).
- **Parameters**: `seat_type` (required, string — e.g. "ceo", "cfo", "cmo"), `message` (required, string, max 4000 chars)
- **API**: `POST /api/v1/seats/chat` with body `{ "seat_type": "{seat_type}", "message": "{message}" }` **(NEW endpoint)**
- **Returns**: `{ data: { seat_type, title, persona_id, response } }`

### Group 3: Employee Management (2 tools)

#### `list_employees`

- **Description**: List all active employees with their roles.
- **Parameters**: None
- **API**: `POST /api/v1/openclaw/command` with body `{ "command": "list employees" }`
- **Returns**: `{ result, employees[{ id, name, role }] }`

#### `get_employee_profile`

- **Description**: Get an employee's profile with submission history, sentiment trends, and recent reports.
- **Parameters**: `name` (required, string — fuzzy match by name)
- **API**: `GET /api/v1/employees/{name}/profile` **(NEW endpoint)**
- **Returns**: `{ employee: { id, name, role, job_title, country }, submission_rate, recent_reports[], sentiment_trend, consecutive_missed }`

## File Structure

```
ai-management-brain/
  mcp-server/
    src/
      index.ts           # MCP server entry, register tools, start stdio transport
      tools/
        team.ts          # get_team_status, get_report, get_alerts
        mentor.ts        # switch_mentor, list_mentors
        csuite.ts        # board_discuss, chat_with_seat
        employee.ts      # list_employees, get_employee_profile
      api-client.ts      # HTTP client with auth, error handling, 10s timeout
      types.ts           # API response type definitions
    package.json         # @tonypk/management-brain-mcp
    tsconfig.json
    README.md
```

## User Experience

### Installation

Add to Claude Code MCP config (`~/.claude.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "management-brain": {
      "command": "npx",
      "args": ["-y", "@tonypk/management-brain-mcp"],
      "env": {
        "MANAGEMENT_BRAIN_API_KEY": "your-api-key"
      }
    }
  }
}
```

Zero local dependencies. One environment variable.

### Usage Examples

In Claude Code:
- "How's my team doing today?" → agent calls `get_team_status`
- "Show me the weekly report" → agent calls `get_report` with period="weekly"
- "Should we expand to Japan?" → agent calls `board_discuss` with that topic
- "Ask the CFO about Q2 budget" → agent calls `chat_with_seat` with seat_type="cfo"
- "How is John doing?" → agent calls `get_employee_profile` with name="John"
- "Switch to Inamori management style" → agent calls `switch_mentor` with mentor="inamori"

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No API key set | Server starts but tools return error: "Please set MANAGEMENT_BRAIN_API_KEY environment variable" |
| Invalid API key | 401 response → tool returns: "Invalid API key. Check your MANAGEMENT_BRAIN_API_KEY." |
| Network timeout | 10 second timeout → tool returns: "Cloud API unreachable. Please check your network." |
| API error (4xx/5xx) | Pass through the error message from the cloud API |
| Rate limited (board_discuss) | 429 response → tool returns: "Board discussions are limited to once per 5 minutes." |

## Backend Changes Required

Two new API endpoints must be added to the Go backend:

### 1. `POST /api/v1/seats/chat`

Single-seat chat endpoint for the `chat_with_seat` MCP tool.

- **Auth**: JWT (Bearer token)
- **Body**: `{ "seat_type": "ceo", "message": "What should our Q2 priorities be?" }`
- **Handler**: Calls existing `SeatService.Chat()` with tenant ID from JWT
- **Response**: `{ "data": { "seat_type": "ceo", "title": "Chief Executive Officer", "persona_id": "musk", "response": "..." } }`

### 2. `GET /api/v1/employees/:name/profile`

Employee profile endpoint for the `get_employee_profile` MCP tool.

- **Auth**: JWT (Bearer token)
- **Params**: `:name` — fuzzy match against employee name (case-insensitive ILIKE)
- **Handler**: Aggregates from employees, reports, and chase_logs tables
- **Response**:
```json
{
  "data": {
    "employee": { "id": "...", "name": "John Santos", "role": "employee", "job_title": "Engineer", "country": "Philippines" },
    "submission_rate": "85.7%",
    "recent_reports": [
      { "date": "2026-03-24", "sentiment": "positive", "blockers": "" },
      { "date": "2026-03-23", "sentiment": "neutral", "blockers": "waiting for API access" }
    ],
    "sentiment_trend": "stable",
    "consecutive_missed": 0
  }
}
```

## Testing

- **Unit tests**: Mock the HTTP client, test each tool's input validation and response mapping
- **Integration test**: Real HTTP calls against a test server (optional, can use recorded responses)
- **Manual test**: Install locally via `npx`, verify all 9 tools work in Claude Code

## Distribution

- **npm package**: `@tonypk/management-brain-mcp`
- **Repository**: Same repo under `mcp-server/` directory
- **Versioning**: Independent semver, starting at `1.0.0`
- **CI**: GitHub Actions to publish to npm on tag push
