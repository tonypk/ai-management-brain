# Employee Profile Fields Design Spec

## Problem

Employees currently have minimal profile data (name, culture_code, role). The AI mentor has no visibility into what an employee does, where they're based, or what language they prefer. This limits the mentor's ability to give relevant, targeted advice. The boss also cannot see job titles or responsibilities in the employee list.

## Solution

Add 4 new fields to the `employees` table: `job_title`, `responsibilities`, `country`, `language`. Inject these into the AI mentor prompt for context-aware coaching. Expose in all entry points (bot command, API, frontend).

## New Fields

| Field | DB Type | Default | Example |
|-------|---------|---------|---------|
| `job_title` | TEXT NOT NULL | `''` | "Frontend Developer" |
| `responsibilities` | TEXT NOT NULL | `''` | "Handles UI/UX design and React development" |
| `country` | TEXT NOT NULL | `''` | "Philippines" |
| `language` | TEXT NOT NULL | `''` | "Chinese" |

All use empty string default (not nullable). Empty = not set.

## Database Migration

```sql
ALTER TABLE employees ADD COLUMN IF NOT EXISTS job_title       TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS country         TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS language        TEXT NOT NULL DEFAULT '';
```

Added as migration 000008 in `cmd/brain/main.go` inline migrations (following existing pattern).

## AI Prompt Injection

### Employee Chat Prompt

`BuildEmployeeChatPrompt` in `engine.go` receives the new fields and injects an `<employee_context>` block before the role instruction:

```
{base_prompt_with_memory}

<employee_context>
Job Title: Frontend Developer
Responsibilities: Handles UI/UX design and React development
Country: Philippines
</employee_context>

You are {mentor_name}, acting as CEO and management coach.
The employee "Alice" is asking you for guidance.
Respond based on your management philosophy. Keep responses concise and actionable.
Reply in Chinese.
```

- Only non-empty fields are included in `<employee_context>`
- If all 4 fields are empty, the `<employee_context>` block is omitted entirely
- `language` field, if non-empty, appends `"Reply in {language}."` to force the AI to respond in that language
- If `language` is empty, existing behavior is preserved (AI mirrors user's language)

### Boss Prompt — Roster Enhancement

The employee roster in `HandleBoss` is enriched:
```
1. Alice - Frontend Developer (member, active)
2. Bob - Sales Manager (manager, active)
3. Charlie (member, inactive)
```

Job title appears after the name if non-empty. Other fields (responsibilities, country, language) are not included in the roster to keep it compact.

## ChatService Interface Change

Replace the long parameter list in `HandleEmployee` with a struct:

```go
type EmployeeChatRequest struct {
    EmployeeID       string
    TenantID         string
    Name             string
    JobTitle         string
    Responsibilities string
    Country          string
    Language         string
    MentorID         string
    CultureCode      string
    Text             string
}

func (s *ChatService) HandleEmployee(ctx context.Context, req EmployeeChatRequest) (string, error)
```

All callers (Telegram handler, UnifiedHandler OnText) are updated to pass the struct.

`BuildEmployeeChatPrompt` signature also changes to accept the new fields:

```go
func (e *Engine) BuildEmployeeChatPrompt(ctx context.Context, tenantID, employeeID string, profile EmployeeProfile, queryText string) string

type EmployeeProfile struct {
    Name             string
    JobTitle         string
    Responsibilities string
    Country          string
    Language         string
}
```

## Bot Command Changes

### `/addemployee` — New Format

```
/addemployee Alice | Frontend Developer | Handles UI/UX | Philippines | Chinese
```

Pipe-separated fields: `name | job_title | responsibilities | country | language`

- Only `name` is required
- Remaining fields are optional (left empty if not provided)
- `culture_code` moves to a separate `/culture` command (already exists)
- Backward compatible: `/addemployee Alice` still works (all profile fields default to empty)

Response:
```
Employee added!
Name: Alice
Job: Frontend Developer
Country: Philippines
Language: Chinese
Invite code: A1B2C3D4

Share this code with Alice to join via /join A1B2C3D4
```

### `/profile <name>` — Enhanced Display

Add the new fields to the existing profile display:
```
Employee Profile: Alice
Status: Active (linked)
Job Title: Frontend Developer
Responsibilities: Handles UI/UX design
Country: Philippines
Language: Chinese
Culture: philippines
Reports (7d): 5/7
Reports (30d): 22/30
Streak: 5 days
```

## API Changes

### `POST /api/v1/employees` — Create Employee

Request body adds optional fields:
```json
{
    "name": "Alice",
    "culture_code": "philippines",
    "job_title": "Frontend Developer",
    "responsibilities": "Handles UI/UX design",
    "country": "Philippines",
    "language": "Chinese"
}
```

### `GET /api/v1/employees` and `GET /api/v1/employees/:id`

Response adds the 4 fields:
```json
{
    "id": "...",
    "name": "Alice",
    "culture_code": "philippines",
    "role": "member",
    "is_active": true,
    "has_telegram": true,
    "invite_code": "A1B2C3D4",
    "job_title": "Frontend Developer",
    "responsibilities": "Handles UI/UX design",
    "country": "Philippines",
    "language": "Chinese"
}
```

### `PUT /api/v1/employees/:id/profile` — New Endpoint (Boss Only)

Updates the 4 profile fields:
```json
{
    "job_title": "Senior Frontend Developer",
    "responsibilities": "Leads frontend team",
    "country": "Philippines",
    "language": "Chinese"
}
```

All fields optional — only provided fields are updated (patch semantics).

## Frontend Changes

### EmployeesView.vue — Add Employee Form

Add 4 input fields to the create form:
- Job Title (text input, optional)
- Responsibilities (textarea, optional)
- Country (text input, optional)
- Language (text input, optional)

### EmployeesView.vue — Table

Add `Job Title` column to the employees table (between Name and Culture).

### Employee TypeScript Interface

```typescript
export interface Employee {
    id: string;
    name: string;
    culture_code: string;
    role: string;
    is_active: boolean;
    has_telegram: boolean;
    invite_code: string;
    job_title: string;
    responsibilities: string;
    country: string;
    language: string;
}
```

## sqlc Changes

### New Query: CreateEmployee (updated)

```sql
-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code, job_title, responsibilities, country, language)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;
```

### New Query: UpdateEmployeeProfile

```sql
-- name: UpdateEmployeeProfile :exec
UPDATE employees
SET job_title = $2, responsibilities = $3, country = $4, language = $5
WHERE id = $1;
```

### Existing Queries

`ListActiveEmployees`, `GetEmployee`, `GetEmployeeByTelegramID` etc. all automatically pick up the new columns since they use `SELECT *` or return the full `Employee` struct. The sqlc models.go `Employee` struct gains 4 new fields after regeneration.

## Files Changed

### New Files
- None

### Modified Files
- `cmd/brain/main.go` — Add migration 000008
- `sql/queries/employees.sql` — Update CreateEmployee, add UpdateEmployeeProfile
- `internal/db/sqlc/` — Regenerated (models.go, employees.sql.go)
- `internal/bot/commands.go` — Update `/addemployee` parser, update `/profile` display
- `internal/bot/adapter.go` — Add new fields to bot.Employee, CreateEmployeeParams
- `internal/bot/middleware.go` — Add new fields to bot.Employee struct
- `internal/brain/engine.go` — Update BuildEmployeeChatPrompt with EmployeeProfile
- `internal/brain/chat.go` — Change HandleEmployee to EmployeeChatRequest struct, update HandleBoss roster
- `internal/api/handlers.go` — Update create/list/get employee handlers, add updateProfile
- `internal/api/router.go` — Add PUT /employees/:id/profile route
- `frontend/src/composables/api.ts` — Update Employee interface
- `frontend/src/views/EmployeesView.vue` — Add form fields, table column

### Unchanged
- Report collection flow
- Chase/summary logic
- Memory system
- Event bus
- Scheduler

## Out of Scope

- Dropdown/autocomplete for country/language (free text for V1)
- Employee self-service profile editing (boss-only for now)
- Translating the AI response (language field controls AI output language, not translation)
- Validating country/language against a list (free text, any value accepted)
