# Web Organization Setup Wizard

## Problem

Organization plan setup only works through Telegram Bot. Users who haven't configured the bot, or prefer web-based workflows, cannot create an organization plan. The web Organization page shows EmptyState with no actionable path forward.

## Solution

Add a 3-step form wizard to the web Organization page. When no plan exists, show the wizard instead of EmptyState. Reuse `OrgEngine.GeneratePlan()` for AI plan generation (produces `ManagementPlan`, compatible with all existing handlers). Skip the bot's conversational extraction layer since form data is already structured.

## Design

### User Flow

```
No plan exists → Show Setup Wizard (instead of EmptyState)
  Step 1: Company Profile form (structured fields)
  Step 2: AI generates plan → user reviews → optional text feedback to adjust
  Step 3: Confirm & Activate
Plan exists → Show current Organization page (tabs: Overview, Plan Details, AI Roles, Suggestions)
```

### Step 1: Company Profile

Single form with grouped fields:

**Basic Info**
- Industry (text input with common suggestions: Tech, Manufacturing, Retail, Finance, Healthcare, Education)
- Company Stage (select: Startup / Growth / Mature)
- Business Model (select: B2B / B2C / Marketplace / SaaS / Other + text)
- Team Size (number input, min 1, max 100000)

**Organization**
- Org Structure (select: Flat / Hierarchical / Matrix / Team-based)
- Current Projects (textarea, brief description)

**Preferences**
- Pain Points (multi-select tags: Poor communication, No metrics, Low engagement, Unclear roles, Slow decisions, High turnover)
- Communication Tools (multi-select: Telegram / Slack / Lark / Email / WhatsApp)
- Culture Preferences (textarea, optional)
- Goal Framework (select: OKR / KPI / Scrum / MBO / BSC)

Required fields: Industry, Company Stage, Team Size, Org Structure, at least 1 Pain Point, at least 1 Comm Tool.

**Mentor**: Not shown in the form. The backend uses the tenant's current `mentor_id` (set via Mentor page or default "inamori"). If the user wants a different mentor, they change it on the Mentor page first, then run setup.

### Step 2: Review Plan

- Show loading spinner while AI generates (~5-15s)
- On AI failure: show error message with "Retry" button; "Back" button preserves form data
- Display generated ManagementPlan using existing components (OrgDesignPanel, PlanDetailsPanel patterns)
- Textarea for optional feedback → "Regenerate" button calls existing `PUT /org/plan` to adjust
- "Confirm" button proceeds to Step 3

### Step 3: Activate

- Summary of what will happen: "This will create AI roles and start automated management"
- "Activate" button → calls existing `POST /org/plan/activate`
- On success → reload page, which now shows the Organization tabs view

## API

### New Endpoint

```
POST /org/setup
Body: {
  industry: string,          // required
  company_stage: string,     // required
  business_model: string,    // optional
  team_size: number,         // required, 1-100000
  org_structure: string,     // required
  current_projects: string,  // optional
  pain_points: string[],     // required, min 1
  comm_tools: string[],      // required, min 1
  culture_prefs: string,     // optional
  goal_framework: string     // optional
}
Response: {
  data: {
    id: string,
    industry: string,
    size: number,
    stage: string,
    mentor_id: string,
    plan: ManagementPlan,
    plan_version: number,
    status: string            // "draft"
  }
}
```

This endpoint:
1. Validates required fields (400 on missing/invalid)
2. Loads tenant's current `mentor_id` → loads `MentorConfig`
3. Builds `CompanyProfile` from input (extended struct with all form fields)
4. Calls `OrgEngine.GeneratePlan(ctx, mentorConfig, profile, nil)` → returns `*ManagementPlan`
5. Upserts to `organizations` table (INSERT or UPDATE if row exists for tenant)
6. Returns full `OrgPlan` shape matching `GET /org/plan` response

### Existing Endpoints (unchanged)

```
PUT  /org/plan              — Adjust plan with feedback (Step 2 "Regenerate" calls this)
POST /org/plan/activate     — Activate plan (Step 3 calls this)
GET  /org/plan              — Get current plan (page load calls this)
```

## Backend

### Extend `CompanyProfile`

Add fields to `internal/brain/org_engine.go` `CompanyProfile`:

```go
type CompanyProfile struct {
    Industry        string   `json:"industry"`
    Size            int      `json:"size"`
    Stage           string   `json:"stage"`
    BusinessModel   string   `json:"business_model,omitempty"`
    Region          string   `json:"region,omitempty"`
    PainPoints      []string `json:"pain_points,omitempty"`
    OrgStructure    string   `json:"org_structure,omitempty"`      // new
    CurrentProjects string   `json:"current_projects,omitempty"`   // new
    CommTools       []string `json:"comm_tools,omitempty"`         // new
    CulturePrefs    string   `json:"culture_prefs,omitempty"`      // new
    GoalFramework   string   `json:"goal_framework,omitempty"`     // new
}
```

Update `buildOrgUserPrompt()` to include these fields in the LLM prompt.

### New sqlc Query

Add `UpsertOrganization` to `sql/queries/organizations.sql`:
```sql
-- name: UpsertOrganization :one
INSERT INTO organizations (tenant_id, industry, size, stage, business_model, mentor_id,
    management_plan, plan_version, status, management_pain_points, current_projects,
    target_framework, team_structure, communication_tools, culture_preferences)
VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (tenant_id) DO UPDATE SET
    industry = EXCLUDED.industry, size = EXCLUDED.size, stage = EXCLUDED.stage,
    business_model = EXCLUDED.business_model, mentor_id = EXCLUDED.mentor_id,
    management_plan = EXCLUDED.management_plan, plan_version = organizations.plan_version + 1,
    status = EXCLUDED.status, management_pain_points = EXCLUDED.management_pain_points,
    current_projects = EXCLUDED.current_projects, target_framework = EXCLUDED.target_framework,
    team_structure = EXCLUDED.team_structure, communication_tools = EXCLUDED.communication_tools,
    culture_preferences = EXCLUDED.culture_preferences, updated_at = NOW()
RETURNING *;
```

This persists all form fields to their corresponding DB columns (not just to `CompanyProfile` for the LLM).

**Field mapping** (form → CompanyProfile → DB column):
| Form field | CompanyProfile | DB column |
|------------|---------------|-----------|
| `org_structure` | `OrgStructure` | `team_structure` |
| `goal_framework` | `GoalFramework` | `target_framework` |
| `comm_tools` | `CommTools` | `communication_tools` |
| `culture_prefs` | `CulturePrefs` | `culture_preferences` |
| `current_projects` | `CurrentProjects` | `current_projects` |
| `pain_points` | `PainPoints` | `management_pain_points` |

### New Handler

Add `handleSetupOrg` in `internal/api/org_handlers.go`:
1. Parse and validate request body (400 on missing required fields)
2. Load tenant via `queries.GetTenantByID(ctx, tenantID)` to get `tenant.MentorID`
3. Load mentor config via `brain.LoadMentor(tenant.MentorID)` (matches existing pattern in `handleUpdatePlan`)
4. Build `CompanyProfile` from input
5. Call `OrgEngine.GeneratePlan(ctx, mentorConfig, profile, nil)`
6. Upsert to organizations table via `queries.UpsertOrganization()` — atomic, no race condition
7. Return OrgPlan

### New Route

Add to `internal/api/routes.go`:
```go
orgGroup.POST("/setup", handleSetupOrg(queries, cfg.OrgEngine))
```

Note: No `MentorEngine` parameter needed — use `brain.LoadMentor()` package-level function inside the handler (matching existing pattern).

### Reused Components

| Component | Source | Purpose |
|-----------|--------|---------|
| `OrgEngine.GeneratePlan()` | `internal/brain/org_engine.go` | AI plan generation (returns `ManagementPlan`) |
| `OrgEngine.AdjustPlan()` | `internal/brain/org_engine.go` | Plan adjustment via feedback |
| `organizations` table | existing | Plan storage |
| `org_units` table | existing | Org structure storage |
| Activate logic | `org_handlers.go` | Role creation |

### What We Skip

- `onboarding.Service` state machine (not needed for structured input)
- `Extractor` LLM (form data is already structured)
- `onboarding.Planner` (produces `ProposedPlan`, incompatible with existing handlers)
- Redis chat history (no conversation to track)
- 4-step confirmation flow (web shows everything at once in Step 2)

## Frontend

### New Files

```
components/organization/SetupWizard.vue       — 3-step wizard container with NSteps (~100 LOC)
components/organization/CompanyProfileForm.vue — Step 1 form with NaiveUI validation rules (~130 LOC)
components/organization/PlanReviewPanel.vue    — Step 2: plan display + feedback textarea + regenerate (~80 LOC)
components/organization/ActivateStep.vue       — Step 3: summary + activate button (~40 LOC)
```

### Modified Files

```
api/org.ts                  — Add setupOrg() function
types/organization.ts       — Add SetupOrgRequest interface
types/index.ts              — Export new type
views/OrganizationView.vue  — Show SetupWizard when plan is null instead of EmptyState
```

### Component Hierarchy

```
OrganizationView
  ├── (plan exists) → PageHeader + NTabs (existing, no changes)
  └── (no plan)     → SetupWizard
                        ├── Step 1: CompanyProfileForm
                        ├── Step 2: PlanReviewPanel
                        │     └── reuses PlanDetailsPanel + OrgDesignPanel
                        └── Step 3: ActivateStep
```

## Validation

1. `go build ./...` — no compile errors
2. `go test ./...` — existing tests pass
3. `npm run build` — no TS errors
4. `/organization` with no plan → shows SetupWizard with 3 steps
5. Step 1: validation prevents submit with missing required fields
6. Step 1 → Step 2: AI generates plan, loading spinner shown, plan displayed
7. Step 2: AI failure → error message + Retry button; Back preserves form data
8. Step 2: feedback textarea + Regenerate → calls `PUT /org/plan` → updated plan shown
9. Step 2 → Step 3: Activate → roles created → page reloads to tabs view
10. `/organization` with existing plan → shows tabs (no regression)
11. `POST /org/setup` with missing required fields → 400 error
12. `POST /org/setup` with existing organization → upserts (no duplicate key error)
