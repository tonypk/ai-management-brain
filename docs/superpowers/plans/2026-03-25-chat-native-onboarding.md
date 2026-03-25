# Chat-Native Deep Onboarding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the minimal `/start` onboarding with an LLM-driven consultative dialogue that collects company context, designs org structure, and generates a complete management plan with 4-step confirmation.

**Architecture:** New `internal/onboarding/` package handles the full onboarding state machine. Boss identity resolution is extended to Slack/Lark. The existing raw text handler in `main.go` and `UnifiedHandler` in `message_handler.go` route to `OnboardingService` when onboarding is incomplete. All LLM calls go through the existing `brain.LLMClient` interface.

**Tech Stack:** Go 1.25, PostgreSQL 16 (pgx/v5), Redis 7, sqlc, Claude API (via existing `brain.AnthropicClient`)

**Spec:** `docs/superpowers/specs/2026-03-25-chat-native-onboarding-design.md`

---

### Task 1: Database Migration

**Files:**
- Create: `sql/migrations/000011_onboarding.up.sql`
- Create: `sql/migrations/000011_onboarding.down.sql`
- Modify: `cmd/brain/main.go` (add migration011 to `runMigrations()`)

- [ ] **Step 1: Write up migration**

Create `sql/migrations/000011_onboarding.up.sql` with the exact SQL from spec Section 6:
- Drop `wizard_sessions`
- Create `onboarding_sessions` with unique index
- Create `org_units` with tenant and parent indexes
- Add `org_unit_id` to `employees`
- Relax `NOT NULL` on `organizations.industry/size/stage`
- Add 6 new nullable columns to `organizations`
- Add `onboarding_completed_at`, `boss_slack_id`, `boss_lark_id` to `tenants` with partial indexes

- [ ] **Step 2: Write down migration**

Create `sql/migrations/000011_onboarding.down.sql` — recreate empty `wizard_sessions`, drop new tables/columns. NOT NULL constraints on `industry/size/stage` intentionally not restored.

- [ ] **Step 3: Add inline migration to `runMigrations()`**

In `cmd/brain/main.go`, add `const migration011 = ...` block after `migration010`, following the existing pattern. Execute via `pool.Exec(ctx, migration011)`.

- [ ] **Step 4: Verify migration applies**

Run: `go build ./cmd/brain/` — must compile.
If local DB available: run the binary briefly to verify migration applies without errors.

- [ ] **Step 5: Commit**

```bash
git add sql/migrations/000011_onboarding.up.sql sql/migrations/000011_onboarding.down.sql cmd/brain/main.go
git commit -m "feat: add migration 000011 — onboarding_sessions, org_units, schema extensions"
```

---

### Task 2: sqlc Queries + Code Generation

**Files:**
- Create: `sql/queries/onboarding.sql`
- Create: `sql/queries/org_units.sql`
- Regenerate: `internal/db/sqlc/` (via `sqlc generate`)

- [ ] **Step 1: Write onboarding queries**

Create `sql/queries/onboarding.sql`:

```sql
-- name: CreateOnboardingSession :one
INSERT INTO onboarding_sessions (tenant_id, status, channel_type)
VALUES ($1, 'onboarding', $2)
RETURNING *;

-- name: GetOnboardingSession :one
SELECT * FROM onboarding_sessions WHERE tenant_id = $1;

-- name: UpdateOnboardingSession :exec
UPDATE onboarding_sessions
SET status = $2, confirm_step = $3, collected_data = $4,
    proposed_plan = $5, message_count = $6, channel_type = $7,
    updated_at = now()
WHERE tenant_id = $1;

-- name: DeleteOnboardingSession :exec
DELETE FROM onboarding_sessions WHERE tenant_id = $1;

-- name: GetTenantByBossSlackID :one
SELECT * FROM tenants WHERE boss_slack_id = $1 AND boss_slack_id IS NOT NULL;

-- name: GetTenantByBossLarkID :one
SELECT * FROM tenants WHERE boss_lark_id = $1 AND boss_lark_id IS NOT NULL;

-- name: UpdateOrganizationFromOnboarding :exec
UPDATE organizations
SET industry = $2, size = $3, stage = $4, business_model = $5,
    management_pain_points = $6, current_projects = $7,
    target_framework = $8, team_structure = $9,
    communication_tools = $10, culture_preferences = $11,
    updated_at = now()
WHERE tenant_id = $1;

-- name: SetTenantOnboardingCompleted :exec
UPDATE tenants SET onboarding_completed_at = now() WHERE id = $1;
```

- [ ] **Step 2: Write org_units queries**

Create `sql/queries/org_units.sql`:

```sql
-- name: CreateOrgUnit :one
INSERT INTO org_units (tenant_id, parent_id, name, unit_type, head_role, responsibilities, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListOrgUnits :many
SELECT * FROM org_units WHERE tenant_id = $1 AND is_active = true ORDER BY sort_order, name;

-- name: GetOrgUnit :one
SELECT * FROM org_units WHERE id = $1;

-- name: UpdateOrgUnit :exec
UPDATE org_units
SET name = $2, unit_type = $3, parent_id = $4, head_role = $5,
    head_employee_id = $6, responsibilities = $7, updated_at = now()
WHERE id = $1;

-- name: SoftDeleteOrgUnit :exec
UPDATE org_units SET is_active = false, updated_at = now() WHERE id = $1;

-- name: DeleteOrgUnitsByTenant :exec
DELETE FROM org_units WHERE tenant_id = $1;

-- name: AssignEmployeeToUnit :exec
UPDATE employees SET org_unit_id = $2 WHERE id = $1;

-- name: ListEmployeesByUnit :many
SELECT * FROM employees WHERE org_unit_id = $1 AND is_active = true ORDER BY name;
```

- [ ] **Step 3: Run sqlc generate**

```bash
cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate
```

Verify no errors. Check that `internal/db/sqlc/` has new generated files for onboarding and org_units queries.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add sql/queries/onboarding.sql sql/queries/org_units.sql internal/db/sqlc/
git commit -m "feat: add sqlc queries for onboarding_sessions and org_units"
```

---

### Task 3: ProposedPlan Types

**Files:**
- Create: `internal/onboarding/types.go`

- [ ] **Step 1: Write types**

Create `internal/onboarding/types.go` with all typed structs from spec Section 2:

```go
package onboarding

// ProposedPlan is the AI-generated management plan, validated before storage.
type ProposedPlan struct {
    Mentor    MentorPlan    `json:"mentor"`
    Board     []SeatPlan    `json:"board"`
    OrgDesign OrgDesignPlan `json:"org_design"`
    Policies  PolicyPlan    `json:"policies"`
    Schedule  SchedulePlan  `json:"schedule"`
    Reasoning string        `json:"reasoning"`
}

type MentorPlan struct {
    PrimaryID   string  `json:"primary_id"`
    SecondaryID string  `json:"secondary_id,omitempty"`
    BlendWeight float64 `json:"blend_weight,omitempty"`
    Reasoning   string  `json:"reasoning"`
}

type SeatPlan struct {
    SeatType  string `json:"seat_type"`
    PersonaID string `json:"persona_id"`
    Reasoning string `json:"reasoning"`
}

type OrgDesignPlan struct {
    Units     []OrgUnitPlan `json:"units"`
    Reasoning string        `json:"reasoning"`
}

type OrgUnitPlan struct {
    RefID            string `json:"ref_id"`
    ParentRefID      string `json:"parent_ref_id"`
    Name             string `json:"name"`
    UnitType         string `json:"unit_type"`
    HeadRole         string `json:"head_role"`
    Responsibilities string `json:"responsibilities"`
}

type PolicyPlan struct {
    Framework        string   `json:"framework"`
    CheckinQuestions []string `json:"checkin_questions"`
    TrackingFocus    []string `json:"tracking_focus"`
    RiskRules        RiskRules `json:"risk_rules"`
    Cadence          Cadence   `json:"cadence"`
    Reasoning        string   `json:"reasoning"`
}

type RiskRules struct {
    ConsecutiveMisses      int      `json:"consecutive_misses"`
    SentimentDropThreshold float64  `json:"sentiment_drop_threshold"`
    UrgentKeywords         []string `json:"urgent_keywords"`
}

type Cadence struct {
    DailyActions   []string `json:"daily_actions"`
    WeeklyActions  []string `json:"weekly_actions"`
    WeeklyDay      string   `json:"weekly_day"`
    MonthlyActions []string `json:"monthly_actions"`
    MonthlyDay     int      `json:"monthly_day"`
}

type SchedulePlan struct {
    Checkin    string `json:"checkin"`
    Chase      string `json:"chase"`
    Summary    string `json:"summary"`
    Briefing   string `json:"briefing"`
    SignalScan string `json:"signal_scan"`
    Timezone   string `json:"timezone"`
}

// CollectedData tracks what info has been extracted from the onboarding dialogue.
type CollectedData struct {
    Industry        string   `json:"industry,omitempty"`
    CompanyStage    string   `json:"company_stage,omitempty"`
    BusinessModel   string   `json:"business_model,omitempty"`
    TeamSize        int      `json:"team_size,omitempty"`
    OrgStructure    string   `json:"org_structure,omitempty"`
    CurrentProjects string   `json:"current_projects,omitempty"`
    PainPoints      []string `json:"pain_points,omitempty"`
    CommTools       []string `json:"comm_tools,omitempty"`
    CulturePrefs    string   `json:"culture_prefs,omitempty"`
    GoalFramework   string   `json:"goal_framework,omitempty"`
}

// RequiredFieldsCovered returns true when all required onboarding info has been collected.
func (c *CollectedData) RequiredFieldsCovered() bool {
    return c.Industry != "" &&
        c.CompanyStage != "" &&
        c.BusinessModel != "" &&
        c.TeamSize > 0 &&
        c.OrgStructure != "" &&
        c.CurrentProjects != "" &&
        len(c.PainPoints) > 0 &&
        len(c.CommTools) > 0
}

// Validate checks that a ProposedPlan has all required fields populated.
func (p *ProposedPlan) Validate() error {
    if p.Mentor.PrimaryID == "" {
        return fmt.Errorf("mentor primary_id is required")
    }
    if len(p.Board) == 0 {
        return fmt.Errorf("at least one board seat is required")
    }
    if len(p.OrgDesign.Units) == 0 {
        return fmt.Errorf("at least one org unit is required")
    }
    if p.Policies.Framework == "" {
        return fmt.Errorf("policy framework is required")
    }
    if p.Schedule.Timezone == "" {
        return fmt.Errorf("schedule timezone is required")
    }
    return nil
}
```

Add `import "fmt"` at the top.

- [ ] **Step 2: Write tests for types**

Create `internal/onboarding/types_test.go`:

```go
package onboarding

import (
    "encoding/json"
    "testing"
)

func TestCollectedData_RequiredFieldsCovered(t *testing.T) {
    // Empty — not covered
    cd := &CollectedData{}
    if cd.RequiredFieldsCovered() {
        t.Error("empty data should not be covered")
    }

    // All filled — covered
    cd = &CollectedData{
        Industry: "SaaS", CompanyStage: "startup", BusinessModel: "B2B",
        TeamSize: 10, OrgStructure: "flat", CurrentProjects: "API platform",
        PainPoints: []string{"hiring"}, CommTools: []string{"telegram"},
    }
    if !cd.RequiredFieldsCovered() {
        t.Error("all required fields should be covered")
    }
}

func TestProposedPlan_Validate(t *testing.T) {
    // Valid plan
    plan := validTestPlan()
    if err := plan.Validate(); err != nil {
        t.Errorf("valid plan should pass: %v", err)
    }

    // Missing mentor
    bad := validTestPlan()
    bad.Mentor.PrimaryID = ""
    if err := bad.Validate(); err == nil {
        t.Error("missing mentor should fail")
    }
}

func TestProposedPlan_JSONRoundtrip(t *testing.T) {
    plan := validTestPlan()
    data, err := json.Marshal(plan)
    if err != nil {
        t.Fatal(err)
    }
    var decoded ProposedPlan
    if err := json.Unmarshal(data, &decoded); err != nil {
        t.Fatal(err)
    }
    if decoded.Mentor.PrimaryID != plan.Mentor.PrimaryID {
        t.Error("roundtrip mismatch")
    }
}

func validTestPlan() ProposedPlan {
    return ProposedPlan{
        Mentor:  MentorPlan{PrimaryID: "musk", Reasoning: "test"},
        Board:   []SeatPlan{{SeatType: "ceo", PersonaID: "musk", Reasoning: "test"}},
        OrgDesign: OrgDesignPlan{
            Units: []OrgUnitPlan{{RefID: "eng", Name: "Engineering", UnitType: "department"}},
        },
        Policies: PolicyPlan{Framework: "okr", CheckinQuestions: []string{"q1"}},
        Schedule: SchedulePlan{Checkin: "0 9 * * 1-5", Timezone: "Asia/Manila"},
    }
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/onboarding/ -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/onboarding/types.go internal/onboarding/types_test.go
git commit -m "feat: add ProposedPlan and CollectedData types with validation"
```

---

### Task 4: Boss Identity Resolution

**Files:**
- Create: `internal/channel/boss_resolve.go`
- Create: `internal/channel/boss_resolve_test.go`

- [ ] **Step 1: Write the test**

Create `internal/channel/boss_resolve_test.go`. Use the existing `CommandQuerier` interface pattern — define a mock DB that returns a tenant for known boss IDs. Test:
- Telegram boss resolves via `GetTenantByBossChatID`
- Slack boss resolves via `GetTenantByBossSlackID`
- Lark boss resolves via `GetTenantByBossLarkID`
- Unknown channel returns error
- Unknown user returns error

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/channel/ -run TestResolveBoss -v
```

Expected: FAIL — `ResolveBoss` not defined.

- [ ] **Step 3: Implement ResolveBoss**

Create `internal/channel/boss_resolve.go`:

```go
package channel

import (
    "context"
    "fmt"
    "strconv"

    "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// BossResolver provides DB queries needed for boss identity resolution.
type BossResolver interface {
    GetTenantByBossChatID(ctx context.Context, bossChatID int64) (*sqlc.Tenant, error)
    GetTenantByBossSlackID(ctx context.Context, slackID string) (sqlc.Tenant, error)
    GetTenantByBossLarkID(ctx context.Context, larkID string) (sqlc.Tenant, error)
}

// ResolveBoss checks if the sender is a boss on any channel.
// Returns the tenant if found, or an error if not a boss.
func ResolveBoss(ctx context.Context, db BossResolver, channelType Type, userID string) (*sqlc.Tenant, error) {
    switch channelType {
    case TypeTelegram:
        id, err := strconv.ParseInt(userID, 10, 64)
        if err != nil {
            return nil, fmt.Errorf("invalid telegram ID: %w", err)
        }
        return db.GetTenantByBossChatID(ctx, id)
    case TypeSlack:
        t, err := db.GetTenantByBossSlackID(ctx, userID)
        if err != nil {
            return nil, err
        }
        return &t, nil
    case TypeLark:
        t, err := db.GetTenantByBossLarkID(ctx, userID)
        if err != nil {
            return nil, err
        }
        return &t, nil
    default:
        return nil, fmt.Errorf("unsupported channel type for boss resolution: %s", channelType)
    }
}
```

Note: check the exact return types from sqlc-generated code — `GetTenantByBossChatID` may return `(sqlc.Tenant, error)` not `(*sqlc.Tenant, error)`. Adapt accordingly (the existing codebase wraps it in `commands.go`).

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/channel/ -run TestResolveBoss -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/channel/boss_resolve.go internal/channel/boss_resolve_test.go
git commit -m "feat: add ResolveBoss for multi-channel boss identity resolution"
```

---

### Task 5: Utility Functions

**Files:**
- Create: `internal/onboarding/util.go`
- Create: `internal/onboarding/util_test.go`

These utility functions are used by Tasks 6-9. Must be created first.

- [ ] **Step 1: Implement shared utilities**

```go
package onboarding

import (
    "encoding/json"
    "strings"
)

// toJSON marshals a value to a JSON string, returning "{}" on error.
func toJSON(v interface{}) string {
    data, err := json.Marshal(v)
    if err != nil {
        return "{}"
    }
    return string(data)
}

// cleanJSON strips markdown code fences from LLM output.
func cleanJSON(s string) string {
    s = strings.TrimSpace(s)
    s = strings.TrimPrefix(s, "```json")
    s = strings.TrimPrefix(s, "```")
    s = strings.TrimSuffix(s, "```")
    return strings.TrimSpace(s)
}
```

- [ ] **Step 2: Write tests**

Create `internal/onboarding/util_test.go`:

```go
package onboarding

import "testing"

func TestCleanJSON(t *testing.T) {
    tests := []struct{in, want string}{
        {`{"a":1}`, `{"a":1}`},
        {"```json\n{\"a\":1}\n```", `{"a":1}`},
        {"```\n{\"a\":1}\n```", `{"a":1}`},
        {"  {\"a\":1}  ", `{"a":1}`},
    }
    for _, tc := range tests {
        if got := cleanJSON(tc.in); got != tc.want {
            t.Errorf("cleanJSON(%q) = %q, want %q", tc.in, got, tc.want)
        }
    }
}

func TestToJSON(t *testing.T) {
    got := toJSON(map[string]int{"x": 1})
    if got != `{"x":1}` {
        t.Errorf("toJSON = %q", got)
    }
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/onboarding/ -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/onboarding/util.go internal/onboarding/util_test.go
git commit -m "feat: add onboarding utility functions (toJSON, cleanJSON)"
```

---

### Task 6: Prompt Builder

**Files:**
- Create: `internal/onboarding/prompt.go`
- Create: `internal/onboarding/prompt_test.go`

- [ ] **Step 1: Write tests**

Test that `BuildConsultantPrompt` returns a system prompt containing:
- The consultant role definition
- The collection targets table
- Already-collected info injected correctly
- Missing required fields listed
- Turn count awareness (wraps up at 15+)

Test that `BuildExtractionPrompt` returns a prompt for Haiku-class extraction with the latest user message and expected JSON output format.

- [ ] **Step 2: Run tests — expect FAIL**

- [ ] **Step 3: Implement prompt.go**

```go
package onboarding

import (
    "fmt"
    "strings"
)

// BuildConsultantPrompt constructs the system prompt for the onboarding dialogue LLM.
func BuildConsultantPrompt(collected *CollectedData, messageCount int) string {
    var sb strings.Builder

    sb.WriteString(`You are an experienced management consultant conducting an initial assessment for a new client.
Your goal is to understand their company deeply so you can design a complete management system.

RULES:
- Ask ONE question at a time
- Be conversational, not interrogative — follow up on interesting points
- Respond in the boss's language (auto-detect from their messages)
- Do NOT list all questions upfront
`)

    // Inject collected info
    sb.WriteString("\n## Already Collected\n")
    any := false
    if collected.Industry != "" { fmt.Fprintf(&sb, "- Industry: %s\n", collected.Industry); any = true }
    if collected.CompanyStage != "" { fmt.Fprintf(&sb, "- Company stage: %s\n", collected.CompanyStage); any = true }
    if collected.BusinessModel != "" { fmt.Fprintf(&sb, "- Business model: %s\n", collected.BusinessModel); any = true }
    if collected.TeamSize > 0 { fmt.Fprintf(&sb, "- Team size: %d\n", collected.TeamSize); any = true }
    if collected.OrgStructure != "" { fmt.Fprintf(&sb, "- Org structure: %s\n", collected.OrgStructure); any = true }
    if collected.CurrentProjects != "" { fmt.Fprintf(&sb, "- Current projects: %s\n", collected.CurrentProjects); any = true }
    if len(collected.PainPoints) > 0 { fmt.Fprintf(&sb, "- Pain points: %s\n", strings.Join(collected.PainPoints, ", ")); any = true }
    if len(collected.CommTools) > 0 { fmt.Fprintf(&sb, "- Comm tools: %s\n", strings.Join(collected.CommTools, ", ")); any = true }
    if collected.CulturePrefs != "" { fmt.Fprintf(&sb, "- Culture prefs: %s\n", collected.CulturePrefs); any = true }
    if collected.GoalFramework != "" { fmt.Fprintf(&sb, "- Goal framework: %s\n", collected.GoalFramework); any = true }
    if !any {
        sb.WriteString("- (Nothing collected yet)\n")
    }

    // Inject missing required fields
    sb.WriteString("\n## Still Need (Required)\n")
    missing := missingRequired(collected)
    if len(missing) == 0 {
        sb.WriteString("ALL REQUIRED INFO COLLECTED. Wrap up now.\n")
    } else {
        for _, m := range missing {
            fmt.Fprintf(&sb, "- %s\n", m)
        }
    }

    // Turn awareness
    if messageCount >= 15 {
        sb.WriteString("\nYou are at turn " + fmt.Sprint(messageCount) + "/20. Start wrapping up — summarize what you know and ask about remaining gaps directly.\n")
    }
    if messageCount >= 20 {
        sb.WriteString("\nTURN LIMIT REACHED. Summarize all collected info and tell the boss you'll proceed with what you have. Ask them to fill in any critical gaps.\n")
    }

    return sb.String()
}

// BuildExtractionPrompt constructs the prompt for lightweight info extraction.
func BuildExtractionPrompt(currentData *CollectedData, userMessage string) string {
    return fmt.Sprintf(`Extract structured information from this message. Return ONLY valid JSON matching the CollectedData schema. Only include fields that are NEW or UPDATED — omit unchanged fields. If no new info, return {}.

Current data: %s

User message: %s`, toJSON(currentData), userMessage)
}

func missingRequired(c *CollectedData) []string {
    var missing []string
    if c.Industry == "" { missing = append(missing, "Industry") }
    if c.CompanyStage == "" { missing = append(missing, "Company stage") }
    if c.BusinessModel == "" { missing = append(missing, "Business model") }
    if c.TeamSize == 0 { missing = append(missing, "Team size") }
    if c.OrgStructure == "" { missing = append(missing, "Organizational structure") }
    if c.CurrentProjects == "" { missing = append(missing, "Current projects") }
    if len(c.PainPoints) == 0 { missing = append(missing, "Management pain points") }
    if len(c.CommTools) == 0 { missing = append(missing, "Communication tools") }
    return missing
}
```

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/onboarding/prompt.go internal/onboarding/prompt_test.go
git commit -m "feat: add onboarding prompt builder for consultant dialogue and extraction"
```

---

### Task 7: Extractor

**Files:**
- Create: `internal/onboarding/extractor.go`
- Create: `internal/onboarding/extractor_test.go`

- [ ] **Step 1: Write tests**

Test `ExtractInfo`:
- Given current `CollectedData` and a user message mentioning "We're a SaaS startup with 15 people", extractor should return updated data with `Industry: "SaaS"`, `TeamSize: 15`, `CompanyStage: "startup"`
- Given a message with no new info, returns unchanged data
- Use a mock `LLMClient` that returns predefined JSON

- [ ] **Step 2: Run tests — expect FAIL**

- [ ] **Step 3: Implement extractor.go**

```go
package onboarding

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"

    "github.com/tonypk/ai-management-brain/internal/brain"
)

// Extractor extracts structured info from onboarding dialogue using a lightweight LLM call.
type Extractor struct {
    llm brain.LLMClient // single-turn, Haiku-class
}

func NewExtractor(llm brain.LLMClient) *Extractor {
    return &Extractor{llm: llm}
}

// ExtractInfo takes the current collected data and a new user message,
// returns updated collected data with any new info merged in.
func (e *Extractor) ExtractInfo(ctx context.Context, current *CollectedData, userMessage string) (*CollectedData, error) {
    prompt := BuildExtractionPrompt(current, userMessage)
    resp, err := e.llm.Chat(ctx, "You are a JSON extraction assistant. Return ONLY valid JSON.", prompt)
    if err != nil {
        slog.Warn("extraction LLM call failed", "error", err)
        return current, nil // non-fatal: continue with what we have
    }

    var delta CollectedData
    if err := json.Unmarshal([]byte(cleanJSON(resp)), &delta); err != nil {
        slog.Warn("extraction JSON parse failed", "error", err, "response", resp)
        return current, nil
    }

    return mergeCollectedData(current, &delta), nil
}

// mergeCollectedData merges new non-zero fields from delta into base.
func mergeCollectedData(base, delta *CollectedData) *CollectedData {
    result := *base // copy
    if delta.Industry != "" { result.Industry = delta.Industry }
    if delta.CompanyStage != "" { result.CompanyStage = delta.CompanyStage }
    if delta.BusinessModel != "" { result.BusinessModel = delta.BusinessModel }
    if delta.TeamSize > 0 { result.TeamSize = delta.TeamSize }
    if delta.OrgStructure != "" { result.OrgStructure = delta.OrgStructure }
    if delta.CurrentProjects != "" { result.CurrentProjects = delta.CurrentProjects }
    if len(delta.PainPoints) > 0 { result.PainPoints = delta.PainPoints }
    if len(delta.CommTools) > 0 { result.CommTools = delta.CommTools }
    if delta.CulturePrefs != "" { result.CulturePrefs = delta.CulturePrefs }
    if delta.GoalFramework != "" { result.GoalFramework = delta.GoalFramework }
    return &result
}
```

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/onboarding/extractor.go internal/onboarding/extractor_test.go
git commit -m "feat: add onboarding info extractor with LLM-based extraction"
```

---

### Task 8: Planner

**Files:**
- Create: `internal/onboarding/planner.go`
- Create: `internal/onboarding/planner_test.go`

- [ ] **Step 1: Write tests**

Test `GeneratePlan`:
- Given complete `CollectedData`, calls LLM and returns a valid `ProposedPlan`
- Use mock LLM that returns valid JSON plan
- Test validation: malformed JSON triggers retry (up to 2 retries)
- Test that after 3 failures, returns error

- [ ] **Step 2: Run tests — expect FAIL**

- [ ] **Step 3: Implement planner.go**

```go
package onboarding

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/tonypk/ai-management-brain/internal/brain"
)

type Planner struct {
    llm brain.LLMClient
}

func NewPlanner(llm brain.LLMClient) *Planner {
    return &Planner{llm: llm}
}

const maxPlanRetries = 3

// GeneratePlan creates a complete management plan from collected onboarding data.
func (p *Planner) GeneratePlan(ctx context.Context, data *CollectedData) (*ProposedPlan, error) {
    systemPrompt := buildPlanGenerationPrompt()
    userPrompt := fmt.Sprintf("Generate a management plan based on this company profile:\n%s", toJSON(data))

    for attempt := 0; attempt < maxPlanRetries; attempt++ {
        resp, err := p.llm.Chat(ctx, systemPrompt, userPrompt)
        if err != nil {
            return nil, fmt.Errorf("LLM call failed: %w", err)
        }

        var plan ProposedPlan
        if err := json.Unmarshal([]byte(cleanJSON(resp)), &plan); err != nil {
            userPrompt = fmt.Sprintf("Your previous response was not valid JSON. Error: %s\nPlease try again with valid JSON only.", err)
            continue
        }

        if err := plan.Validate(); err != nil {
            userPrompt = fmt.Sprintf("Your plan was missing required fields: %s\nPlease include all required fields.", err)
            continue
        }

        return &plan, nil
    }

    return nil, fmt.Errorf("failed to generate valid plan after %d attempts", maxPlanRetries)
}

func buildPlanGenerationPrompt() string {
    return `You are a management systems architect. Given a company profile, generate a complete management plan as JSON.

The JSON must match this exact structure:
{
  "mentor": {"primary_id": "...", "secondary_id": "...", "blend_weight": 0.7, "reasoning": "..."},
  "board": [{"seat_type": "ceo|cfo|cmo|cto|chro|coo", "persona_id": "mentor_id", "reasoning": "..."}],
  "org_design": {
    "units": [{"ref_id": "...", "parent_ref_id": "", "name": "...", "unit_type": "department|team|squad", "head_role": "...", "responsibilities": "..."}],
    "reasoning": "..."
  },
  "policies": {
    "framework": "okr|kpi|scrum|mbo|bsc",
    "checkin_questions": ["..."],
    "tracking_focus": ["..."],
    "risk_rules": {"consecutive_misses": 3, "sentiment_drop_threshold": -0.3, "urgent_keywords": ["urgent"]},
    "cadence": {"daily_actions": [...], "weekly_actions": [...], "weekly_day": "friday", "monthly_actions": [...], "monthly_day": 1},
    "reasoning": "..."
  },
  "schedule": {"checkin": "0 9 * * 1-5", "chase": "30 17 * * 1-5", "summary": "0 19 * * 1-5", "briefing": "0 8 * * 1-5", "signal_scan": "*/30 9-18 * * 1-5", "timezone": "..."},
  "reasoning": "..."
}

Available mentor IDs: musk, inamori, ma, dalio, grove, ren, son, jobs, bezos, buffett, zhangyiming, leijun, caodewang, chushijian, meyer, trout

RESPOND WITH JSON ONLY. No markdown, no explanation.`
}
```

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/onboarding/planner.go internal/onboarding/planner_test.go
git commit -m "feat: add onboarding planner with retry and validation"
```

---

### Task 9: Confirmer

**Files:**
- Create: `internal/onboarding/confirmer.go`
- Create: `internal/onboarding/confirmer_test.go`

- [ ] **Step 1: Write tests**

Test `Confirmer`:
- `FormatStep1` renders mentor + board plan as readable text
- `FormatStep2` renders org structure as an ASCII tree
- `FormatStep3` renders management policies
- `FormatStep4` renders schedule and group setup instructions
- `HandleConfirmResponse` with "ok"/"yes"/confirm advances `confirm_step`
- `HandleConfirmResponse` with modification request (e.g., "change mentor to inamori") calls LLM to modify the plan

- [ ] **Step 2: Run tests — expect FAIL**

- [ ] **Step 3: Implement confirmer.go**

The confirmer formats each step of the `ProposedPlan` as human-readable text, handles boss responses (accept or modify), and manages the `confirm_step` state.

Key functions:
- `FormatStep(plan *ProposedPlan, step int) string` — renders the appropriate step
- `IsConfirmation(text string) bool` — detects "ok", "yes", "confirm", "good", etc.
- `HandleModification(ctx context.Context, plan *ProposedPlan, step int, request string) (*ProposedPlan, string, error)` — uses LLM to modify the plan based on boss's feedback, returns updated plan + formatted response

For Step 2 (org structure), format as ASCII tree:
```
CEO (You)
├── Engineering (VP Eng)
│   ├── Frontend Team (Team Lead)
│   └── Backend Team (Team Lead)
└── Product (Product Manager)
```

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/onboarding/confirmer.go internal/onboarding/confirmer_test.go
git commit -m "feat: add onboarding confirmer with 4-step formatting and modification"
```

---

### Task 10: Main Service (State Machine)

**Files:**
- Create: `internal/onboarding/service.go`
- Create: `internal/onboarding/service_test.go`

This is the central piece — the state machine that orchestrates everything.

- [ ] **Step 1: Write tests**

Test the `HandleMessage` state machine:
- `onboarding` state: message → LLM dialogue + extraction → response
- `onboarding` → `configuring` transition when `RequiredFieldsCovered()`
- `configuring` state: generates plan → moves to `confirming` → returns Step 1 formatted
- `confirming` state step 1: "ok" → writes mentor/board to DB → advances to step 2 → returns Step 2 formatted
- `confirming` state step 4: "ok" → writes to DB → sets `active` → returns completion message
- Concurrency: second message while locked returns "thinking..." message
- Redis cache miss: rebuilds context from `collected_data`

Use mock DB (`sqlc.Querier` interface), mock Redis, mock LLM.

- [ ] **Step 2: Run tests — expect FAIL**

- [ ] **Step 3: Implement service.go**

```go
package onboarding

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"

    "github.com/tonypk/ai-management-brain/internal/brain"
    "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

type Service struct {
    db        Querier           // subset of sqlc.Queries needed
    redis     RedisClient
    llm       brain.LLMClient      // single-turn (extraction, planning)
    chatLLM   brain.ChatLLMClient   // multi-turn with history (onboarding dialogue)
    extractor *Extractor
    planner   *Planner
    confirmer *Confirmer
}

type Querier interface {
    // Onboarding session CRUD
    GetOnboardingSession(ctx context.Context, tenantID uuid.UUID) (sqlc.OnboardingSession, error)
    CreateOnboardingSession(ctx context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error)
    UpdateOnboardingSession(ctx context.Context, arg sqlc.UpdateOnboardingSessionParams) error
    DeleteOnboardingSession(ctx context.Context, tenantID uuid.UUID) error

    // Confirmation step writes
    UpdateOrganizationFromOnboarding(ctx context.Context, arg sqlc.UpdateOrganizationFromOnboardingParams) error
    SetTenantOnboardingCompleted(ctx context.Context, id uuid.UUID) error

    // Org units
    CreateOrgUnit(ctx context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error)
    DeleteOrgUnitsByTenant(ctx context.Context, tenantID uuid.UUID) error
}

type RedisClient interface {
    SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
    Del(ctx context.Context, keys ...string) *redis.IntCmd
    Get(ctx context.Context, key string) *redis.StringCmd
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

func NewService(db Querier, rdb RedisClient, llm brain.LLMClient, chatLLM brain.ChatLLMClient) *Service {
    return &Service{
        db:        db,
        redis:     rdb,
        llm:       llm,
        chatLLM:   chatLLM,
        extractor: NewExtractor(llm),
        planner:   NewPlanner(llm),
        confirmer: NewConfirmer(llm),
    }
}

func (s *Service) HandleMessage(ctx context.Context, tenantID uuid.UUID, channelType, userID, text string) (string, error) {
    // 1. Acquire processing lock
    lockKey := fmt.Sprintf("onboarding:lock:%s", tenantID)
    acquired, _ := s.redis.SetNX(ctx, lockKey, "1", 60*time.Second).Result()
    if !acquired {
        return "I'm still thinking about your last message, one moment...", nil
    }
    defer s.redis.Del(ctx, lockKey)

    // 2. Get or create session
    session, err := s.getOrCreateSession(ctx, tenantID, channelType)
    if err != nil {
        return "", fmt.Errorf("get session: %w", err)
    }

    // 3. Route by state
    switch session.Status {
    case "onboarding":
        return s.handleOnboarding(ctx, session, text)
    case "configuring":
        return s.handleConfiguring(ctx, session)
    case "confirming":
        return s.handleConfirming(ctx, session, text)
    default:
        return "Onboarding is already complete.", nil
    }
}

func (s *Service) handleOnboarding(ctx context.Context, session *sqlc.OnboardingSession, text string) (string, error) {
    // 1. Load collected data
    var collected CollectedData
    json.Unmarshal(session.CollectedData, &collected)

    // 2. Extract new info from message
    updated, _ := s.extractor.ExtractInfo(ctx, &collected, text)

    // 3. Check if all required fields covered
    if updated.RequiredFieldsCovered() {
        // Transition to configuring
        return s.transitionToConfiguring(ctx, session, updated)
    }

    // 4. Load chat history from Redis (or rebuild from collected data)
    history := s.loadChatHistory(ctx, session.TenantID)

    // 5. Build prompt and get LLM response (multi-turn with history)
    // ChatWithHistory signature: (ctx, systemPrompt, history []ChatMessage, userMessage) -> (string, error)
    prompt := BuildConsultantPrompt(updated, int(session.MessageCount)+1)
    resp, err := s.chatLLM.ChatWithHistory(ctx, prompt, history, text)
    if err != nil {
        return "I'm having a technical issue. Please try again.", nil
    }

    // 6. Save state — append user + assistant messages to history
    history = append(history,
        brain.ChatMessage{Role: "user", Content: text},
        brain.ChatMessage{Role: "assistant", Content: resp})
    s.saveChatHistory(ctx, session.TenantID, history)
    s.updateSession(ctx, session.TenantID, "onboarding", int(session.ConfirmStep),
        updated, nil, int(session.MessageCount)+1, session.ChannelType)

    return resp, nil
}
```

Implement `handleConfiguring` and `handleConfirming`:

**`handleConfiguring`**: calls `s.planner.GeneratePlan(ctx, &collected)`, stores plan in session as `proposed_plan` JSONB, transitions status to `"confirming"` with `confirm_step=1`, returns `s.confirmer.FormatStep(plan, 1)`.

**`handleConfirming`**: routes by `session.ConfirmStep`:

```go
func (s *Service) handleConfirming(ctx context.Context, session *sqlc.OnboardingSession, text string) (string, error) {
    var plan ProposedPlan
    json.Unmarshal(session.ProposedPlan, &plan)
    step := int(session.ConfirmStep)

    if s.confirmer.IsConfirmation(text) {
        // Apply this step's changes to DB
        if err := s.applyStep(ctx, session.TenantID, &plan, step); err != nil {
            return "Failed to save changes. Please try again.", nil
        }
        if step >= 4 {
            // Final step — mark onboarding complete
            s.db.SetTenantOnboardingCompleted(ctx, session.TenantID)
            s.db.DeleteOnboardingSession(ctx, session.TenantID)
            return "✅ Onboarding complete! Your management system is now active.", nil
        }
        // Advance to next step
        s.updateSession(ctx, session.TenantID, "confirming", step+1,
            nil, &plan, int(session.MessageCount)+1, session.ChannelType)
        return s.confirmer.FormatStep(&plan, step+1), nil
    }

    // Modification request — use LLM to adjust plan
    updated, resp, err := s.confirmer.HandleModification(ctx, &plan, step, text)
    if err != nil {
        return "I couldn't understand your modification. Please try again.", nil
    }
    s.updateSession(ctx, session.TenantID, "confirming", step,
        nil, updated, int(session.MessageCount)+1, session.ChannelType)
    return resp, nil
}
```

**`applyStep` — per-step DB writes** (from spec Section 2):

```go
func (s *Service) applyStep(ctx context.Context, tenantID uuid.UUID, plan *ProposedPlan, step int) error {
    switch step {
    case 1: // Mentor + Board
        // Write mentor config to organizations table (mentor_id, mentor_blend)
        // Insert board seats (up to 6 rows in seats table)
        return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
            TenantID: tenantID,
            // ... map plan.Mentor and plan.Board fields
        })
    case 2: // Org Structure
        // Delete existing org_units for tenant, then bulk-insert from plan
        s.db.DeleteOrgUnitsByTenant(ctx, tenantID)
        for i, unit := range plan.OrgDesign.Units {
            s.db.CreateOrgUnit(ctx, sqlc.CreateOrgUnitParams{
                TenantID:         tenantID,
                Name:             unit.Name,
                UnitType:         unit.UnitType,
                HeadRole:         &unit.HeadRole,
                Responsibilities: &unit.Responsibilities,
                SortOrder:        int32(i),
                // ParentID resolved from unit.ParentRefID via refID→UUID map
            })
        }
        return nil
    case 3: // Policies
        // Write framework, checkin_questions, risk_rules, cadence to organizations
        return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
            TenantID:        tenantID,
            TargetFramework: &plan.Policies.Framework,
            // ... map remaining policy fields
        })
    case 4: // Schedule
        // Write schedule cron expressions to organizations table
        // The scheduler reads these at runtime — no dynamic job registration needed
        return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
            TenantID: tenantID,
            // ... map schedule fields
        })
    }
    return nil
}
```

Note: The exact `UpdateOrganizationFromOnboarding` param mapping depends on sqlc-generated types from Task 2. The implementer should check the generated params struct and map fields accordingly. If a single query can't handle all step-specific writes, add additional sqlc queries (e.g., `UpdateTenantMentor`, `CreateSeat`).

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/onboarding/service.go internal/onboarding/service_test.go
git commit -m "feat: add OnboardingService with full state machine"
```

---

### Task 11: Routing — Wire OnboardingService Into Bot

**Files:**
- Modify: `internal/bot/commands.go` — `HandleStart` delegates to onboarding
- Modify: `cmd/brain/main.go` — inject OnboardingService, add routing in raw text handler
- Modify: `internal/channel/message_handler.go` — add boss resolution + onboarding routing

- [ ] **Step 1: Modify CommandHandler to accept OnboardingService**

In `internal/bot/commands.go`, add:
```go
type OnboardingHandler interface {
    HandleMessage(ctx context.Context, tenantID uuid.UUID, channelType, userID, text string) (string, error)
}

func (h *CommandHandler) SetOnboardingService(svc OnboardingHandler) {
    h.onboarding = svc
}
```

Add `onboarding OnboardingHandler` field to `CommandHandler` struct.

- [ ] **Step 2: Modify HandleStart**

Change `HandleStart` to check onboarding state after tenant creation:

```go
func (h *CommandHandler) HandleStart(c BotContext) error {
    // ... existing boss check and tenant creation ...

    // If OnboardingService is set and onboarding not complete, delegate
    if h.onboarding != nil && tenant.OnboardingCompletedAt == nil {
        resp, err := h.onboarding.HandleMessage(
            context.Background(), tenant.ID,
            "telegram", strconv.FormatInt(c.SenderID(), 10), "/start")
        if err != nil {
            return fmt.Errorf("onboarding: %w", err)
        }
        return c.Send(resp)
    }

    // Original welcome for already-onboarded tenants
    return c.Send(fmt.Sprintf("Welcome to AI Management Brain!..."))
}
```

Check that `tenant` has `OnboardingCompletedAt` field (it should after sqlc regeneration from Task 2).

- [ ] **Step 3: Modify raw text handler in main.go**

In the boss private chat section (around line 870), before the existing C-Suite seat routing, check `tenant.OnboardingCompletedAt`:

```go
if senderID == cfg.BossTelegramID {
    tenant, err := botDB.GetTenantByBossChatID(ctx, senderID)
    if err == nil && onboardingSvc != nil && !tenant.OnboardingCompletedAt.Valid {
        resp, _ := onboardingSvc.HandleMessage(ctx, tenant.ID, "telegram",
            strconv.FormatInt(senderID, 10), text)
        return sendReply(resp)
    }
    // ... existing routing (seat chat, mentor chat)
}
```

- [ ] **Step 4: Wire OnboardingService creation in main.go**

After `chatService` creation (~line 634), add:

```go
var onboardingSvc *onboarding.Service
if llmService != nil {
    onboardingSvc = onboarding.NewService(queries, &redisWrapper{client: rdb}, llmService.Client(), llmService.ChatClient())
}
if cmdHandler != nil && onboardingSvc != nil {
    cmdHandler.SetOnboardingService(onboardingSvc)
}
```

Note: `llmService.Client()` and `llmService.ChatClient()` are illustrative — check the actual `ChatService` API. Since `brain.AnthropicClient` implements both `LLMClient` and `ChatLLMClient` interfaces, you may pass the same client instance for both parameters.

Add import for `"github.com/tonypk/ai-management-brain/internal/onboarding"`.

- [ ] **Step 5: Modify UnifiedHandler for non-Telegram channels**

In `internal/channel/message_handler.go`, add boss resolution before employee resolution:

```go
func (h *UnifiedHandler) HandleMessage(ctx context.Context, msg Message) error {
    // Try boss resolution first
    if h.bossResolver != nil && h.onboarding != nil {
        tenant, err := ResolveBoss(ctx, h.bossResolver, msg.ChannelType, msg.UserID)
        if err == nil {
            if !tenant.OnboardingCompletedAt.Valid {
                resp, err := h.onboarding.HandleMessage(ctx, tenant.ID, string(msg.ChannelType), msg.UserID, msg.Text)
                if err == nil && resp != "" {
                    return h.sender.Send(ctx, msg.ChannelType, msg.UserID, resp)
                }
            }
            // Boss with completed onboarding — fall through to existing handler or add boss routing
        }
    }

    // Existing employee resolution
    emp, err := h.resolveEmployee(ctx, msg.ChannelType, msg.UserID)
    // ...
}
```

Add `bossResolver BossResolver` and `onboarding OnboardingHandler` fields to `UnifiedHandler`, with setter methods following the existing pattern.

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/bot/commands.go cmd/brain/main.go internal/channel/message_handler.go
git commit -m "feat: wire OnboardingService into bot routing for all channels"
```

---

### Task 12: Scheduler Guard

**Files:**
- Modify: `cmd/brain/main.go` — add onboarding guard to scheduler callbacks

- [ ] **Step 1: Add guard to remindFn, chaseFn, summaryFn**

In each scheduler callback in `main.go`, after fetching the tenant, add:

```go
if !tenant.OnboardingCompletedAt.Valid {
    slog.Debug("skipping scheduler job for tenant still in onboarding", "tenant_id", tenant.ID)
    return
}
```

This affects:
- `remindFn` (check-in questions) — around line 1300
- `chaseFn` (chase reminders) — around line 1340
- `summaryFn` (daily summary) — around line 1380
- `groupMentorFn` (autonomous group posting) — around line 1390

- [ ] **Step 2: Verify build**

```bash
go build ./cmd/brain/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/brain/main.go
git commit -m "feat: add scheduler guard — skip jobs for tenants still in onboarding"
```

---

### Task 13: Integration Test

**Files:**
- Create: `internal/onboarding/integration_test.go`

- [ ] **Step 1: Write end-to-end test with mocks**

Test the full onboarding flow:
1. Boss sends "/start" → gets consultant greeting
2. Boss sends "We're a SaaS B2B startup, 15 people, building an API platform" → extractor fills multiple fields
3. Continue dialogue until all required fields covered
4. System generates plan → returns Step 1 (mentor + board)
5. Boss says "ok" → returns Step 2 (org structure)
6. Boss says "ok" → returns Step 3 (policies)
7. Boss says "ok" → returns Step 4 (schedule)
8. Boss says "ok" → onboarding complete
9. Verify: `onboarding_sessions.status = 'active'`, `tenants.onboarding_completed_at` set

Use mock LLM that returns scripted responses for each stage.

- [ ] **Step 2: Run integration test**

```bash
go test ./internal/onboarding/ -run TestIntegration -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/onboarding/integration_test.go
git commit -m "test: add end-to-end integration test for onboarding flow"
```

---

### Task 14: Final Verification

- [ ] **Step 1: Run all tests**

```bash
go test ./... -count=1
```

All existing tests must still pass. New onboarding tests must pass.

- [ ] **Step 2: Run build**

```bash
go build ./cmd/brain/
```

- [ ] **Step 3: Verify git status**

```bash
git status
git log --oneline -15
```

Confirm all changes committed, no untracked files.
