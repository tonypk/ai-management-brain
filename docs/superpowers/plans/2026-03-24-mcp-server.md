# MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a TypeScript MCP server (`@tonypk/management-brain-mcp`) that exposes 9 tools for AI tools (Claude Code, Cursor, Windsurf) to interact with the management platform via cloud API, plus two new Go backend endpoints to support it.

**Architecture:** TypeScript MCP server communicates via stdio (JSON-RPC) with AI tools. It calls the manageaibrain.com REST API authenticated with `MANAGEMENT_BRAIN_API_KEY`. Two new Go backend endpoints (`POST /api/v1/seats/chat`, `GET /api/v1/employees/:name/profile`) must be added first. The MCP server is stateless — all state lives on the cloud.

**Tech Stack:** TypeScript, `@modelcontextprotocol/sdk`, Go (Gin+sqlc), PostgreSQL

**Spec:** `docs/superpowers/specs/2026-03-24-mcp-server-design.md`

---

## File Structure

### Backend Changes (Go)

- **Modify:** `sql/queries/employees.sql` — add `GetEmployeeByNameFuzzy` query
- **Modify:** `sql/queries/reports.sql` — add `GetEmployeeRecentReportsWithBlockers` query
- **Regenerate:** `internal/db/sqlc/employees.sql.go`, `internal/db/sqlc/reports.sql.go` (via `sqlc generate`)
- **Modify:** `internal/api/seat_handlers.go` — add `handleSeatChat()` handler
- **Create:** `internal/api/employee_handlers.go` — add `handleEmployeeProfile()` handler
- **Modify:** `internal/api/router.go` — register 4 new routes under API Key auth group (2 new endpoints + 2 existing endpoints exposed via API key)

### MCP Server (TypeScript, new directory)

- **Create:** `mcp-server/package.json`
- **Create:** `mcp-server/tsconfig.json`
- **Create:** `mcp-server/src/api-client.ts` — HTTP client with auth, error handling, 10s timeout
- **Create:** `mcp-server/src/types.ts` — API response type definitions
- **Create:** `mcp-server/src/tools/team.ts` — get_team_status, get_report, get_alerts
- **Create:** `mcp-server/src/tools/mentor.ts` — switch_mentor, list_mentors
- **Create:** `mcp-server/src/tools/csuite.ts` — board_discuss, chat_with_seat
- **Create:** `mcp-server/src/tools/employee.ts` — list_employees, get_employee_profile
- **Create:** `mcp-server/src/index.ts` — MCP server entry, register all tools, stdio transport
- **Create:** `mcp-server/README.md`

---

## Task 1: New sqlc Queries

Add two new SQL queries needed by the employee profile endpoint, then regenerate Go code.

**Files:**
- Modify: `sql/queries/employees.sql`
- Modify: `sql/queries/reports.sql`
- Regenerate: `internal/db/sqlc/employees.sql.go`, `internal/db/sqlc/reports.sql.go`

- [ ] **Step 1: Add GetEmployeeByNameFuzzy query**

Append to `sql/queries/employees.sql`:

```sql
-- name: GetEmployeeByNameFuzzy :one
SELECT * FROM employees
WHERE tenant_id = $1 AND is_active = true AND name ILIKE '%' || $2 || '%'
ORDER BY name
LIMIT 1;
```

- [ ] **Step 2: Add GetEmployeeRecentReportsWithBlockers query**

Append to `sql/queries/reports.sql`:

```sql
-- name: GetEmployeeRecentReportsWithBlockers :many
SELECT report_date, sentiment, blockers FROM reports
WHERE employee_id = $1
ORDER BY report_date DESC
LIMIT 7;
```

- [ ] **Step 3: Run sqlc generate**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`
Expected: No errors. New methods appear in `internal/db/sqlc/employees.sql.go` and `internal/db/sqlc/reports.sql.go`.

- [ ] **Step 4: Verify generated code compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add sql/queries/employees.sql sql/queries/reports.sql internal/db/sqlc/
git commit -m "feat: add sqlc queries for employee fuzzy search and recent reports"
```

---

## Task 2: handleSeatChat Handler

Add a new handler for `POST /api/v1/seats/chat` that lets the MCP server chat with a specific C-Suite seat.

**Files:**
- Modify: `internal/api/seat_handlers.go`

**Context:**
- Existing `handleBoardDiscuss(seatSvc *seats.SeatService)` pattern at line ~201 of `seat_handlers.go`
- `SeatService.Chat(ctx, tenantID, seatType, cultureCode, message)` returns `(string, error)` — returns soft error string for inactive seats, not a hard error
- `GetSeatByType` exists in sqlc: params `{TenantID pgtype.UUID, SeatType string}`
- The spec wants response: `{ "data": { "seat_type", "title", "persona_id", "response" } }`
- For inactive seats: `{ "data": { "message": "The {title} seat is currently inactive." } }`
- Handler needs both `seatSvc` and `q` (Queries) — `seatSvc` for Chat(), `q` for seat info lookup

- [ ] **Step 1: Add handleSeatChat handler**

Add to `internal/api/seat_handlers.go`, after the `handleBoardDiscuss` function:

```go
func handleSeatChat(seatSvc *seats.SeatService, q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SeatType string `json:"seat_type" binding:"required"`
			Message  string `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "seat_type and message are required"})
			return
		}

		tenantID := TenantFromContext(c)
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Look up seat for metadata (title, persona_id)
		seat, err := q.GetSeatByType(c.Request.Context(), sqlc.GetSeatByTypeParams{
			TenantID: tenantUUID,
			SeatType: req.SeatType,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown seat type"})
			return
		}

		if !seat.IsActive.Bool {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{
				"message": "The " + seat.Title + " seat is currently inactive.",
			}})
			return
		}

		response, err := seatSvc.Chat(c.Request.Context(), tenantID, req.SeatType, "default", req.Message)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "limited") {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": errMsg})
				return
			}
			slog.Error("seat chat error", "error", errMsg, "seat_type", req.SeatType)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"seat_type":  req.SeatType,
			"title":      seat.Title,
			"persona_id": seat.PersonaID,
			"response":   response,
		}})
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/api/seat_handlers.go
git commit -m "feat: add handleSeatChat handler for MCP seat chat endpoint"
```

---

## Task 3: handleEmployeeProfile Handler

Add a new handler for `GET /api/v1/employees/:name/profile` that returns an employee's profile with submission history, sentiment trends, and recent reports.

**Files:**
- Create: `internal/api/employee_handlers.go`

**Context:**
- Uses new sqlc queries from Task 1: `GetEmployeeByNameFuzzy`, `GetEmployeeRecentReportsWithBlockers`
- Uses existing queries: `GetEmployeeSubmissionHistory` (30-day history for rate), `GetConsecutiveMissDays`, `GetRecentSentiments`
- Response format: `{ "data": { employee, submission_rate, recent_reports, sentiment_trend, consecutive_missed } }`
- Sentiment trend computed from last 7 days: "stable", "improving", "declining"
- `TenantFromContext(c)` returns tenant ID string from auth context
- `parseUUID(id)` converts string to `pgtype.UUID`

- [ ] **Step 1: Create employee_handlers.go**

Create `internal/api/employee_handlers.go`:

```go
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

func handleEmployeeProfile(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter is required"})
			return
		}

		tenantID := TenantFromContext(c)
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		ctx := c.Request.Context()

		// Fuzzy match employee by name
		emp, err := q.GetEmployeeByNameFuzzy(ctx, sqlc.GetEmployeeByNameFuzzyParams{
			TenantID: tenantUUID,
			Column2:  name,
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No employee found matching '%s'.", name)})
			return
		}

		// Get submission history (last 30 days) for rate calculation
		history, err := q.GetEmployeeSubmissionHistory(ctx, emp.ID)
		if err != nil {
			history = nil
		}
		submissionRate := fmt.Sprintf("%.1f%%", float64(len(history))/30.0*100)

		// Get recent reports with blockers (last 7)
		recentReports, err := q.GetEmployeeRecentReportsWithBlockers(ctx, emp.ID)
		if err != nil {
			recentReports = nil
		}

		reports := make([]gin.H, 0, len(recentReports))
		for _, r := range recentReports {
			reports = append(reports, gin.H{
				"date":      r.ReportDate.Time.Format("2006-01-02"),
				"sentiment": textVal(r.Sentiment),
				"blockers":  textVal(r.Blockers),
			})
		}

		// Get consecutive missed days
		missedDays, err := q.GetConsecutiveMissDays(ctx, emp.ID)
		if err != nil {
			missedDays = 0
		}

		// Compute sentiment trend from last 7 days
		sentiments, err := q.GetRecentSentiments(ctx, sqlc.GetRecentSentimentsParams{
			EmployeeID: emp.ID,
			Limit:      7,
		})
		if err != nil {
			sentiments = nil
		}
		trend := computeSentimentTrend(sentiments)

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"employee": gin.H{
				"id":        formatUUID(emp.ID),
				"name":      emp.Name,
				"role":      emp.Role,
				"job_title": emp.JobTitle,
				"country":   emp.Country,
			},
			"submission_rate":    submissionRate,
			"recent_reports":     reports,
			"sentiment_trend":    trend,
			"consecutive_missed": missedDays,
		}})
	}
}

func textVal(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func computeSentimentTrend(sentiments []pgtype.Text) string {
	if len(sentiments) < 2 {
		return "stable"
	}

	scoreMap := map[string]int{
		"positive": 2,
		"neutral":  1,
		"negative": 0,
	}

	// Compare first half (recent) vs second half (older)
	mid := len(sentiments) / 2
	var recentSum, olderSum int
	var recentCount, olderCount int

	for i, s := range sentiments {
		if !s.Valid {
			continue
		}
		score, ok := scoreMap[s.String]
		if !ok {
			continue
		}
		if i < mid {
			recentSum += score
			recentCount++
		} else {
			olderSum += score
			olderCount++
		}
	}

	if recentCount == 0 || olderCount == 0 {
		return "stable"
	}

	recentAvg := float64(recentSum) / float64(recentCount)
	olderAvg := float64(olderSum) / float64(olderCount)
	diff := recentAvg - olderAvg

	if diff > 0.3 {
		return "improving"
	}
	if diff < -0.3 {
		return "declining"
	}
	return "stable"
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS. If `Column2` is not the correct generated param name for the fuzzy query, check `internal/db/sqlc/employees.sql.go` and adjust.

- [ ] **Step 3: Commit**

```bash
git add internal/api/employee_handlers.go
git commit -m "feat: add handleEmployeeProfile handler for MCP employee profile endpoint"
```

---

## Task 4: Route Registration + Backend Build + Deploy

Register both new endpoints under API Key auth, build, and deploy to server.

**Files:**
- Modify: `internal/api/router.go`

**Context:**
- Existing patterns in `router.go`:
  - OpenClaw group at line 189: `openclaw := v1.Group("/openclaw")` with `APIKeyMiddleware` + `AuthMiddleware`
  - Seats group at line 95: `protected.GET("/seats", ...)` under JWT `protected` group
  - Optional feature guard: `if cfg.SeatService != nil { ... }`
- All MCP-accessible endpoints need `APIKeyMiddleware` (not JWT), same as OpenClaw
- The seat chat endpoint needs `cfg.SeatService` (may be nil) and `cfg.Queries`
- The employee profile endpoint needs `cfg.Queries` only
- **CRITICAL**: Existing `board/discuss` and `mentors` routes are under JWT-only `protected` group. The MCP server needs API key access to these. Register them at spec paths (`/seats/board/discuss`, `/seats/mentors`) under the new API key group. This creates separate routes from the existing JWT ones — no conflict.

- [ ] **Step 1: Add route registration**

In `internal/api/router.go`, after the openclaw group block (after line ~199), add:

```go
	// API Key-accessible endpoints for MCP server
	mcpAPI := v1.Group("")
	mcpAPI.Use(APIKeyMiddleware(cfg.Queries))
	mcpAPI.Use(AuthMiddleware(cfg.JWTSecret))
	{
		if cfg.SeatService != nil {
			mcpAPI.POST("/seats/chat", handleSeatChat(cfg.SeatService, cfg.Queries))
			mcpAPI.POST("/seats/board/discuss", handleBoardDiscuss(cfg.SeatService))
		}
		mcpAPI.GET("/seats/mentors", handleListMentorsWithDomain())
		mcpAPI.GET("/employees/:name/profile", handleEmployeeProfile(cfg.Queries))
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./...`
Expected: All existing tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/api/router.go
git commit -m "feat: register MCP API key routes for seat chat, board discuss, mentors, employee profile"
```

- [ ] **Step 5: Cross-compile for Linux**

Run: `cd /Users/anna/Documents/ai-management-brain && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/brain ./cmd/brain`
Expected: Binary at `bin/brain`

- [ ] **Step 6: Upload to server**

Run: `scp -i ~/.ssh/opentoke.pem bin/brain ubuntu@18.141.251.99:/home/ubuntu/ai-management-brain/brain`

- [ ] **Step 7: Rebuild Docker and verify**

Run:
```bash
ssh ai-brain "cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build brain && sleep 3 && curl -s localhost/healthz"
```
Expected: Health check returns OK

- [ ] **Step 8: Test new endpoints with curl**

Test seat chat (requires valid API key):
```bash
ssh ai-brain 'curl -s -X POST http://localhost:8080/api/v1/seats/chat \
  -H "Authorization: Bearer mb_<your-api-key>" \
  -H "Content-Type: application/json" \
  -d "{\"seat_type\":\"ceo\",\"message\":\"test\"}"'
```

Test employee profile:
```bash
ssh ai-brain 'curl -s http://localhost:8080/api/v1/employees/john/profile \
  -H "Authorization: Bearer mb_<your-api-key>"'
```

Expected: Both return JSON responses (200 or 404 depending on data)

---

## Task 5: MCP Server Project Scaffolding

Set up the TypeScript project structure for the MCP server.

**Files:**
- Create: `mcp-server/package.json`
- Create: `mcp-server/tsconfig.json`
- Create: `mcp-server/.gitignore`

- [ ] **Step 1: Create package.json**

Create `mcp-server/package.json`:

```json
{
  "name": "@tonypk/management-brain-mcp",
  "version": "1.0.0",
  "description": "MCP server for AI Management Brain — 9 tools for team management, C-Suite discussions, and employee insights",
  "type": "module",
  "bin": {
    "management-brain-mcp": "./dist/index.js"
  },
  "files": [
    "dist"
  ],
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js",
    "dev": "tsc --watch",
    "test": "vitest run",
    "test:watch": "vitest",
    "prepublishOnly": "npm run build"
  },
  "keywords": [
    "mcp",
    "management",
    "ai",
    "claude",
    "model-context-protocol"
  ],
  "license": "MIT",
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.12.0"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "typescript": "^5.7.0",
    "vitest": "^3.0.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

Create `mcp-server/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist"]
}
```

- [ ] **Step 3: Create .gitignore**

Create `mcp-server/.gitignore`:

```
node_modules/
dist/
```

- [ ] **Step 4: Install dependencies**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npm install`
Expected: `node_modules` created, `package-lock.json` generated

- [ ] **Step 5: Commit**

```bash
git add mcp-server/package.json mcp-server/package-lock.json mcp-server/tsconfig.json mcp-server/.gitignore
git commit -m "feat: scaffold MCP server TypeScript project"
```

---

## Task 6: API Client + Types

Create the HTTP client that handles auth, error handling, response normalization, and TypeScript type definitions.

**Files:**
- Create: `mcp-server/src/api-client.ts`
- Create: `mcp-server/src/types.ts`

**Context:**
- Base URL: `https://manageaibrain.com` (configurable via `MANAGEMENT_BRAIN_BASE_URL` env var for testing)
- Auth: `Authorization: Bearer {MANAGEMENT_BRAIN_API_KEY}` header
- Two response formats: OpenClaw endpoints return flat JSON; seats/employees endpoints wrap in `{ "data": ... }`
- 10 second timeout, one automatic retry on 5xx with 500ms backoff
- Error handling: 401 → "Invalid API key", timeout → "Cloud API unreachable", 5xx → retry then "Server error"

- [ ] **Step 1: Create types.ts**

Create `mcp-server/src/types.ts`:

```typescript
// Team tools
export interface TeamStatus {
  date: string;
  total_employees: number;
  submitted: number;
  pending: Array<{
    id: string;
    name: string;
    chase_count: number;
  }>;
  chase_count: number;
  mentor: string;
  mentor_name: string;
}

export interface TeamReport {
  period: string;
  date_range: { start: string; end: string };
  submission_rate: string;
  ranking: Array<{
    id: string;
    name: string;
    days: number;
    medal?: string;
  }>;
  one_on_one_suggestions: Array<{
    id: string;
    name: string;
    days: number;
  }>;
}

export interface Alerts {
  alerts: Array<{
    employee_id: string;
    employee_name: string;
    missed_days: number;
    severity: string;
  }>;
  total: number;
}

// Mentor tools
export interface SwitchMentorSuccess {
  result: string;
  mentor_id: string;
  name: string;
}

export interface SwitchMentorError {
  error: string;
  available_mentors: string[];
}

export interface Mentor {
  id: string;
  name: string;
  name_en: string;
  company: string;
  philosophy: string;
  domain: string;
  tags: string[];
  recommended_seats: string[];
}

// C-Suite tools
export interface BoardDiscussResponse {
  topic: string;
  responses: Array<{
    seat_type: string;
    title: string;
    persona_id: string;
    content: string;
  }>;
  synthesis: string;
}

export interface SeatChatResponse {
  seat_type: string;
  title: string;
  persona_id: string;
  response: string;
}

export interface SeatChatInactiveResponse {
  message: string;
}

// Employee tools
export interface CommandResult {
  result: string;
  employees: Array<{
    id: string;
    name: string;
    role: string;
  }>;
}

export interface EmployeeProfile {
  employee: {
    id: string;
    name: string;
    role: string;
    job_title: string;
    country: string;
  };
  submission_rate: string;
  recent_reports: Array<{
    date: string;
    sentiment: string;
    blockers: string;
  }>;
  sentiment_trend: string;
  consecutive_missed: number;
}
```

- [ ] **Step 2: Create api-client.ts**

Create `mcp-server/src/api-client.ts`:

```typescript
export class APIError extends Error {
  constructor(
    message: string,
    public readonly statusCode: number,
  ) {
    super(message);
    this.name = "APIError";
  }
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly timeoutMs = 10_000;

  constructor(baseUrl: string, apiKey: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.apiKey = apiKey;
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  async post<T>(path: string, body: Record<string, unknown>): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private async request<T>(
    method: string,
    path: string,
    body?: Record<string, unknown>,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      Authorization: `Bearer ${this.apiKey}`,
      "Content-Type": "application/json",
    };

    const options: RequestInit = {
      method,
      headers,
      signal: AbortSignal.timeout(this.timeoutMs),
      ...(body && { body: JSON.stringify(body) }),
    };

    // First attempt
    let response = await this.fetchWithErrorHandling(url, options);

    // Retry once on 5xx
    if (response.status >= 500) {
      await this.sleep(500);
      response = await this.fetchWithErrorHandling(url, options);
      if (response.status >= 500) {
        throw new APIError("Server error. Please try again.", response.status);
      }
    }

    if (response.status === 401) {
      throw new APIError(
        "Invalid API key. Check your MANAGEMENT_BRAIN_API_KEY.",
        401,
      );
    }

    if (response.status === 429) {
      throw new APIError(
        "Board discussions are limited to once per 5 minutes.",
        429,
      );
    }

    if (response.status >= 400) {
      const errorBody = await response.json().catch(() => ({}));
      const message =
        (errorBody as Record<string, string>).error ||
        `API error (${response.status})`;
      throw new APIError(message, response.status);
    }

    const json = await response.json();

    // Normalize: extract .data if present, pass through otherwise
    if (json && typeof json === "object" && "data" in json) {
      return json.data as T;
    }
    return json as T;
  }

  private async fetchWithErrorHandling(
    url: string,
    options: RequestInit,
  ): Promise<Response> {
    try {
      return await fetch(url, options);
    } catch (error) {
      if (error instanceof DOMException && error.name === "TimeoutError") {
        throw new APIError(
          "Cloud API unreachable. Please check your network.",
          0,
        );
      }
      if (
        error instanceof TypeError &&
        error.message.includes("abort")
      ) {
        throw new APIError(
          "Cloud API unreachable. Please check your network.",
          0,
        );
      }
      throw new APIError(
        "Cloud API unreachable. Please check your network.",
        0,
      );
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add mcp-server/src/types.ts mcp-server/src/api-client.ts
git commit -m "feat: add MCP server API client and type definitions"
```

---

## Task 7: Team + Mentor Tools

Implement 5 tools: get_team_status, get_report, get_alerts, switch_mentor, list_mentors.

**Files:**
- Create: `mcp-server/src/tools/team.ts`
- Create: `mcp-server/src/tools/mentor.ts`

**Context:**
- Each tool function takes an `ApiClient` and returns a `CallToolResult` object from the MCP SDK
- Team tools use OpenClaw endpoints (flat JSON, no wrapper — but api-client normalizes)
- Mentor tools: `switch_mentor` uses `POST /api/v1/openclaw/command` (flat), `list_mentors` uses `GET /api/v1/seats/mentors` (wrapped in `{data}`)
- api-client already normalizes both formats, so tools just call and format

- [ ] **Step 1: Create team.ts**

Create `mcp-server/src/tools/team.ts`:

```typescript
import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type { TeamStatus, TeamReport, Alerts } from "../types.js";

export async function getTeamStatus(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<TeamStatus>("/api/v1/openclaw/status");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getReport(
  client: ApiClient,
  period: string,
): Promise<CallToolResult> {
  if (period !== "weekly" && period !== "monthly") {
    return {
      content: [
        { type: "text", text: 'Period must be "weekly" or "monthly".' },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.get<TeamReport>(
      `/api/v1/openclaw/report?period=${period}`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getAlerts(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Alerts>("/api/v1/openclaw/alerts");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
```

- [ ] **Step 2: Create mentor.ts**

Create `mcp-server/src/tools/mentor.ts`:

```typescript
import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  SwitchMentorSuccess,
  SwitchMentorError,
  Mentor,
} from "../types.js";

export async function switchMentor(
  client: ApiClient,
  mentor: string,
): Promise<CallToolResult> {
  if (!mentor.trim()) {
    return {
      content: [{ type: "text", text: "Mentor name cannot be empty." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<SwitchMentorSuccess | SwitchMentorError>(
      "/api/v1/openclaw/command",
      { command: `switch mentor ${mentor}` },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function listMentors(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Mentor[]>("/api/v1/seats/mentors");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add mcp-server/src/tools/team.ts mcp-server/src/tools/mentor.ts
git commit -m "feat: add MCP team and mentor tools (5 tools)"
```

---

## Task 8: C-Suite + Employee Tools

Implement 4 tools: board_discuss, chat_with_seat, list_employees, get_employee_profile.

**Files:**
- Create: `mcp-server/src/tools/csuite.ts`
- Create: `mcp-server/src/tools/employee.ts`

**Context:**
- `board_discuss`: `POST /api/v1/seats/board/discuss` (existing endpoint, wrapped in `{data}`)
- `chat_with_seat`: `POST /api/v1/seats/chat` (new endpoint from Task 2, wrapped in `{data}`)
- `list_employees`: `POST /api/v1/openclaw/command` with `{ "command": "list employees" }` (flat JSON)
- `get_employee_profile`: `GET /api/v1/employees/:name/profile` (new endpoint from Task 3, wrapped in `{data}`)
- Input validation: topic/message max 4000 chars, cannot be empty

- [ ] **Step 1: Create csuite.ts**

Create `mcp-server/src/tools/csuite.ts`:

```typescript
import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  BoardDiscussResponse,
  SeatChatResponse,
  SeatChatInactiveResponse,
} from "../types.js";

const MAX_INPUT_LEN = 4000;

export async function boardDiscuss(
  client: ApiClient,
  topic: string,
): Promise<CallToolResult> {
  if (!topic.trim()) {
    return {
      content: [{ type: "text", text: "Topic cannot be empty." }],
      isError: true,
    };
  }
  if (topic.length > MAX_INPUT_LEN) {
    return {
      content: [
        {
          type: "text",
          text: `Topic too long (max ${MAX_INPUT_LEN} characters).`,
        },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.post<BoardDiscussResponse>(
      "/api/v1/seats/board/discuss",
      { topic },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function chatWithSeat(
  client: ApiClient,
  seatType: string,
  message: string,
): Promise<CallToolResult> {
  if (!message.trim()) {
    return {
      content: [{ type: "text", text: "Message cannot be empty." }],
      isError: true,
    };
  }
  if (message.length > MAX_INPUT_LEN) {
    return {
      content: [
        {
          type: "text",
          text: `Message too long (max ${MAX_INPUT_LEN} characters).`,
        },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.post<
      SeatChatResponse | SeatChatInactiveResponse
    >("/api/v1/seats/chat", { seat_type: seatType, message });

    // Check if seat is inactive (response has "message" field instead of "response")
    if ("message" in data) {
      return {
        content: [{ type: "text", text: data.message }],
      };
    }

    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    if (error instanceof APIError && error.statusCode === 400) {
      return {
        content: [
          {
            type: "text",
            text: "Unknown seat type. Valid types: ceo, cfo, cmo, cto, chro, coo",
          },
        ],
        isError: true,
      };
    }
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
```

- [ ] **Step 2: Create employee.ts**

Create `mcp-server/src/tools/employee.ts`:

```typescript
import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type { CommandResult, EmployeeProfile } from "../types.js";

export async function listEmployees(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.post<CommandResult>(
      "/api/v1/openclaw/command",
      { command: "list employees" },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getEmployeeProfile(
  client: ApiClient,
  name: string,
): Promise<CallToolResult> {
  if (!name.trim()) {
    return {
      content: [{ type: "text", text: "Employee name cannot be empty." }],
      isError: true,
    };
  }
  try {
    const data = await client.get<EmployeeProfile>(
      `/api/v1/employees/${encodeURIComponent(name)}/profile`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    if (error instanceof APIError && error.statusCode === 404) {
      return {
        content: [
          {
            type: "text",
            text: `No employee found matching '${name}'.`,
          },
        ],
        isError: true,
      };
    }
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add mcp-server/src/tools/csuite.ts mcp-server/src/tools/employee.ts
git commit -m "feat: add MCP C-Suite and employee tools (4 tools)"
```

---

## Task 9: MCP Server Entry Point

Create `index.ts` that registers all 9 tools, handles configuration, and starts the stdio transport.

**Files:**
- Create: `mcp-server/src/index.ts`

**Context:**
- MCP SDK pattern: create `McpServer`, call `server.tool(name, description, schema, handler)` for each tool
- Transport: `StdioServerTransport` from `@modelcontextprotocol/sdk/server/stdio.js`
- Server: `McpServer` from `@modelcontextprotocol/sdk/server/mcp.js`
- Schema: use `zod` (bundled with MCP SDK) for tool parameter validation
- API key: from `MANAGEMENT_BRAIN_API_KEY` env var. If missing, tools return error message.
- Base URL: from `MANAGEMENT_BRAIN_BASE_URL` env var, default `https://manageaibrain.com`
- Add `#!/usr/bin/env node` shebang for `npx` execution

- [ ] **Step 1: Create index.ts**

Create `mcp-server/src/index.ts`:

```typescript
#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { ApiClient } from "./api-client.js";
import { getTeamStatus, getReport, getAlerts } from "./tools/team.js";
import { switchMentor, listMentors } from "./tools/mentor.js";
import { boardDiscuss, chatWithSeat } from "./tools/csuite.js";
import { listEmployees, getEmployeeProfile } from "./tools/employee.js";

const apiKey = process.env.MANAGEMENT_BRAIN_API_KEY ?? "";
const baseUrl =
  process.env.MANAGEMENT_BRAIN_BASE_URL ?? "https://manageaibrain.com";

const NO_KEY_MSG =
  "Please set MANAGEMENT_BRAIN_API_KEY environment variable.";

function makeClient(): ApiClient | null {
  if (!apiKey) return null;
  return new ApiClient(baseUrl, apiKey);
}

const server = new McpServer({
  name: "management-brain",
  version: "1.0.0",
});

// --- Group 1: Core Management ---

server.tool(
  "get_team_status",
  "Get today's team check-in status — submission rate, pending employees, chase counts",
  {},
  async () => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getTeamStatus(client);
  },
);

server.tool(
  "get_report",
  "Get team performance report with ranking and 1:1 suggestions",
  { period: z.enum(["weekly", "monthly"]).describe("Report period") },
  async ({ period }) => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getReport(client, period);
  },
);

server.tool(
  "get_alerts",
  "Get active alerts for employees with consecutive missed check-in days",
  {},
  async () => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getAlerts(client);
  },
);

// --- Group 2: Mentors ---

server.tool(
  "switch_mentor",
  'Switch the active management mentor philosophy (e.g., "musk", "inamori")',
  {
    mentor: z
      .string()
      .describe('Mentor ID or name, e.g. "musk", "inamori", "dalio"'),
  },
  async ({ mentor }) => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return switchMentor(client, mentor);
  },
);

server.tool(
  "list_mentors",
  "List all available mentors with domain expertise and recommended C-Suite seats",
  {},
  async () => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return listMentors(client);
  },
);

// --- Group 3: C-Suite ---

server.tool(
  "board_discuss",
  "Run a board discussion across all active C-Suite seats on a topic. Each seat responds from their expertise, followed by a synthesis.",
  {
    topic: z
      .string()
      .min(1)
      .max(4000)
      .describe("The topic for the board to discuss"),
  },
  async ({ topic }) => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return boardDiscuss(client, topic);
  },
);

server.tool(
  "chat_with_seat",
  "Chat directly with a specific C-Suite seat (e.g., ask the CFO about budget)",
  {
    seat_type: z
      .string()
      .describe('C-Suite seat type, e.g. "ceo", "cfo", "cmo", "cto", "chro", "coo"'),
    message: z
      .string()
      .min(1)
      .max(4000)
      .describe("Your message to the C-Suite seat"),
  },
  async ({ seat_type, message }) => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return chatWithSeat(client, seat_type, message);
  },
);

// --- Group 4: Employees ---

server.tool(
  "list_employees",
  "List all active employees with their roles",
  {},
  async () => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return listEmployees(client);
  },
);

server.tool(
  "get_employee_profile",
  "Get an employee's profile with submission history, sentiment trends, and recent reports",
  {
    name: z
      .string()
      .describe("Employee name (case-insensitive fuzzy match)"),
  },
  async ({ name }) => {
    const client = makeClient();
    if (!client)
      return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getEmployeeProfile(client, name);
  },
);

// --- Start ---

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((error) => {
  console.error("MCP server failed to start:", error);
  process.exit(1);
});
```

- [ ] **Step 2: Add zod dependency**

The MCP SDK re-exports zod, but we import it directly. Check if it's included:

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && node -e "require('zod')" 2>/dev/null && echo "zod available" || npm install zod`

If zod is not bundled, add it:
Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npm install zod`

- [ ] **Step 3: Build the project**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npm run build`
Expected: `dist/` directory created with compiled JS files, no errors

- [ ] **Step 4: Verify the server starts**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}' | node dist/index.js 2>/dev/null | head -c 500`

Expected: JSON response containing `"serverInfo":{"name":"management-brain","version":"1.0.0"}` and tool capabilities

- [ ] **Step 5: Commit**

```bash
git add mcp-server/src/index.ts mcp-server/package.json mcp-server/package-lock.json
git commit -m "feat: add MCP server entry point with all 9 tools registered"
```

---

## Task 10: Tests + README + npm Publish Config

Add unit tests for the API client and tool functions, write README, and configure for npm publishing.

**Files:**
- Create: `mcp-server/src/__tests__/api-client.test.ts`
- Create: `mcp-server/src/__tests__/tools.test.ts`
- Create: `mcp-server/README.md`
- Modify: `mcp-server/tsconfig.json` (exclude tests from build output)
- Create: `mcp-server/vitest.config.ts`

- [ ] **Step 1: Create vitest config**

Create `mcp-server/vitest.config.ts`:

```typescript
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["src/__tests__/**/*.test.ts"],
  },
});
```

- [ ] **Step 2: Update tsconfig to exclude tests from build**

In `mcp-server/tsconfig.json`, update the `exclude` array:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist", "src/__tests__"]
}
```

- [ ] **Step 3: Create API client tests**

Create `mcp-server/src/__tests__/api-client.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiClient, APIError } from "../api-client.js";

describe("ApiClient", () => {
  let client: ApiClient;
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    client = new ApiClient("https://example.com", "test-key");
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("sends authorization header", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ result: "ok" }),
    });

    await client.get("/api/v1/test");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "https://example.com/api/v1/test",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-key",
        }),
      }),
    );
  });

  it("extracts .data from wrapped responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ data: { name: "test" } }),
    });

    const result = await client.get("/api/v1/wrapped");
    expect(result).toEqual({ name: "test" });
  });

  it("passes through flat responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ date: "2026-03-24", submitted: 4 }),
    });

    const result = await client.get<{ date: string; submitted: number }>(
      "/api/v1/flat",
    );
    expect(result).toEqual({ date: "2026-03-24", submitted: 4 });
  });

  it("throws APIError on 401", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 401,
      json: () => Promise.resolve({ error: "unauthorized" }),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(APIError);
    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "Invalid API key",
    );
  });

  it("throws APIError on 429", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 429,
      json: () => Promise.resolve({ error: "rate limited" }),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "Board discussions are limited",
    );
  });

  it("retries once on 5xx", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ status: 500, json: () => Promise.resolve({}) })
      .mockResolvedValueOnce({
        status: 200,
        json: () => Promise.resolve({ result: "ok" }),
      });
    globalThis.fetch = fetchMock;

    const result = await client.get("/api/v1/test");
    expect(result).toEqual({ result: "ok" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("throws after two 5xx failures", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 500,
      json: () => Promise.resolve({}),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "Server error. Please try again.",
    );
  });

  it("sends POST body as JSON", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ result: "ok" }),
    });

    await client.post("/api/v1/test", { command: "switch mentor musk" });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "https://example.com/api/v1/test",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ command: "switch mentor musk" }),
      }),
    );
  });
});
```

- [ ] **Step 4: Create tool tests**

Create `mcp-server/src/__tests__/tools.test.ts`:

```typescript
import { describe, it, expect, vi, afterEach } from "vitest";
import { ApiClient } from "../api-client.js";
import { getTeamStatus, getReport } from "../tools/team.js";
import { switchMentor } from "../tools/mentor.js";
import { boardDiscuss, chatWithSeat } from "../tools/csuite.js";
import { getEmployeeProfile } from "../tools/employee.js";

function mockClient(response: unknown, status = 200): ApiClient {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = vi.fn().mockResolvedValue({
    status,
    json: () => Promise.resolve(response),
  });
  return new ApiClient("https://example.com", "test-key");
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("team tools", () => {
  it("getTeamStatus returns formatted response", async () => {
    const client = mockClient({ date: "2026-03-24", submitted: 4 });
    const result = await getTeamStatus(client);
    expect(result.isError).toBeUndefined();
    expect(result.content[0].type).toBe("text");
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.submitted).toBe(4);
  });

  it("getReport validates period", async () => {
    const client = mockClient({});
    const result = await getReport(client, "invalid");
    expect(result.isError).toBe(true);
  });
});

describe("mentor tools", () => {
  it("switchMentor rejects empty input", async () => {
    const client = mockClient({});
    const result = await switchMentor(client, "  ");
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("empty");
  });
});

describe("csuite tools", () => {
  it("boardDiscuss rejects empty topic", async () => {
    const client = mockClient({});
    const result = await boardDiscuss(client, "");
    expect(result.isError).toBe(true);
  });

  it("boardDiscuss rejects oversized topic", async () => {
    const client = mockClient({});
    const result = await boardDiscuss(client, "x".repeat(4001));
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("4000");
  });

  it("chatWithSeat rejects empty message", async () => {
    const client = mockClient({});
    const result = await chatWithSeat(client, "ceo", "");
    expect(result.isError).toBe(true);
  });

  it("chatWithSeat handles inactive seat", async () => {
    const client = mockClient({
      data: { message: "The CEO seat is currently inactive." },
    });
    const result = await chatWithSeat(client, "ceo", "hello");
    expect(result.isError).toBeUndefined();
    expect((result.content[0] as { text: string }).text).toContain("inactive");
  });
});

describe("employee tools", () => {
  it("getEmployeeProfile rejects empty name", async () => {
    const client = mockClient({});
    const result = await getEmployeeProfile(client, "");
    expect(result.isError).toBe(true);
  });
});
```

- [ ] **Step 5: Run tests**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npm test`
Expected: All tests pass

- [ ] **Step 6: Create README.md**

Create `mcp-server/README.md`:

```markdown
# @tonypk/management-brain-mcp

MCP server for [AI Management Brain](https://manageaibrain.com) — 9 tools for team management, C-Suite board discussions, and employee insights.

## Install

Add to your Claude Code MCP config (`~/.claude.json` or project `.mcp.json`):

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

## Tools

| Tool | Description |
|------|-------------|
| `get_team_status` | Today's check-in status — submission rate, pending employees |
| `get_report` | Team performance report (weekly/monthly) with ranking |
| `get_alerts` | Alerts for employees with consecutive missed days |
| `switch_mentor` | Switch management mentor (musk, inamori, dalio, etc.) |
| `list_mentors` | List all mentors with expertise and recommended seats |
| `board_discuss` | Board discussion across all C-Suite seats on a topic |
| `chat_with_seat` | Chat with a specific C-Suite seat (CEO, CFO, etc.) |
| `list_employees` | List all active employees |
| `get_employee_profile` | Employee profile with sentiment and submission history |

## Usage Examples

In Claude Code:
- "How's my team doing today?" → `get_team_status`
- "Show me the weekly report" → `get_report`
- "Should we expand to Japan?" → `board_discuss`
- "Ask the CFO about Q2 budget" → `chat_with_seat`
- "How is John doing?" → `get_employee_profile`
- "Switch to Inamori management style" → `switch_mentor`

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `MANAGEMENT_BRAIN_API_KEY` | Yes | API key from manageaibrain.com |
| `MANAGEMENT_BRAIN_BASE_URL` | No | Override API URL (default: `https://manageaibrain.com`) |

## License

MIT
```

- [ ] **Step 7: Build final version**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp-server && npm run build`
Expected: `dist/` directory with all compiled files

- [ ] **Step 8: Commit**

```bash
git add mcp-server/src/__tests__/ mcp-server/vitest.config.ts mcp-server/tsconfig.json mcp-server/README.md
git commit -m "feat: add MCP server tests, README, and npm publish config"
```

- [ ] **Step 9: Publish to npm (when ready)**

Run:
```bash
cd /Users/anna/Documents/ai-management-brain/mcp-server
npm login
npm publish --access public
```

Expected: Package published as `@tonypk/management-brain-mcp@1.0.0`

---

## Task 11: CI/CD Workflow for npm Publishing

Add a GitHub Actions workflow to automatically publish the npm package on tag push.

**Files:**
- Create: `.github/workflows/publish-mcp.yml`

- [ ] **Step 1: Create publish workflow**

Create `.github/workflows/publish-mcp.yml`:

```yaml
name: Publish MCP Server

on:
  push:
    tags:
      - 'mcp-v*'

jobs:
  publish:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: mcp-server
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          registry-url: 'https://registry.npmjs.org'

      - run: npm ci
      - run: npm test
      - run: npm run build
      - run: npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/publish-mcp.yml
git commit -m "ci: add GitHub Actions workflow for MCP server npm publishing"
```

- [ ] **Step 3: Usage**

To publish a new version:
```bash
cd mcp-server
# Update version in package.json
npm version patch  # or minor/major
cd ..
git push && git push --tags
```

The workflow triggers on tags matching `mcp-v*` (e.g., `mcp-v1.0.0`).

Note: The `NPM_TOKEN` secret must be configured in the GitHub repository settings before the first publish.
