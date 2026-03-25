# Web Organization Setup Wizard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 3-step web wizard to the Organization page so users can create an organization plan without the Telegram Bot.

**Architecture:** New `POST /org/setup` endpoint accepts structured form data, calls `OrgEngine.GeneratePlan()` to produce a `ManagementPlan`, and upserts to the `organizations` table. Frontend shows a wizard (profile form → plan review → activate) when no plan exists. Existing `PUT /org/plan` and `POST /org/plan/activate` endpoints are reused for adjustment and activation.

**Tech Stack:** Go 1.25 (Gin, sqlc, pgx), Vue 3 + TypeScript + NaiveUI, PostgreSQL 16

**Spec:** `docs/superpowers/specs/2026-03-25-web-org-setup-wizard-design.md`

---

## File Map

### Backend — New/Modified

| File | Action | Responsibility |
|------|--------|---------------|
| `sql/queries/organizations.sql` | Modify | Add `UpsertOrganization` query |
| `internal/db/sqlc/*` | Regenerate | sqlc generates new Go code |
| `internal/brain/org_engine.go` | Modify | Extend `CompanyProfile` + `buildOrgUserPrompt()` |
| `internal/api/org_handlers.go` | Modify | Add `handleSetupOrg` handler + request type |
| `internal/api/router.go` | Modify | Register `POST /org/setup` route |

### Frontend — New

| File | Responsibility |
|------|---------------|
| `frontend/src/components/organization/SetupWizard.vue` | 3-step wizard container with NSteps |
| `frontend/src/components/organization/CompanyProfileForm.vue` | Step 1: form with validation |
| `frontend/src/components/organization/PlanReviewPanel.vue` | Step 2: plan display + feedback |
| `frontend/src/components/organization/ActivateStep.vue` | Step 3: summary + activate |

### Frontend — Modified

| File | Change |
|------|--------|
| `frontend/src/types/organization.ts` | Add `SetupOrgRequest` interface |
| `frontend/src/types/index.ts` | Export `SetupOrgRequest` |
| `frontend/src/api/org.ts` | Add `setupOrg()` function |
| `frontend/src/views/OrganizationView.vue` | Show wizard when no plan |

---

## Task 1: Add UpsertOrganization SQL Query

**Files:**
- Modify: `sql/queries/organizations.sql`
- Regenerate: `internal/db/sqlc/organizations.sql.go`

- [ ] **Step 1: Add upsert query to organizations.sql**

Append to `sql/queries/organizations.sql`:

```sql
-- name: UpsertOrganization :one
INSERT INTO organizations (
    tenant_id, industry, size, stage, business_model, mentor_id,
    management_plan, status, management_pain_points, current_projects,
    target_framework, team_structure, communication_tools, culture_preferences
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'draft', $8, $9, $10, $11, $12, $13
)
ON CONFLICT (tenant_id) DO UPDATE SET
    industry = EXCLUDED.industry,
    size = EXCLUDED.size,
    stage = EXCLUDED.stage,
    business_model = EXCLUDED.business_model,
    mentor_id = EXCLUDED.mentor_id,
    management_plan = EXCLUDED.management_plan,
    plan_version = organizations.plan_version + 1,
    status = 'draft',
    management_pain_points = EXCLUDED.management_pain_points,
    current_projects = EXCLUDED.current_projects,
    target_framework = EXCLUDED.target_framework,
    team_structure = EXCLUDED.team_structure,
    communication_tools = EXCLUDED.communication_tools,
    culture_preferences = EXCLUDED.culture_preferences,
    updated_at = NOW()
RETURNING *;
```

- [ ] **Step 2: Regenerate sqlc**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`

Expected: No errors. New `UpsertOrganization` and `UpsertOrganizationParams` appear in `internal/db/sqlc/organizations.sql.go`.

- [ ] **Step 3: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: Build passes.

- [ ] **Step 4: Commit**

```bash
git add sql/queries/organizations.sql internal/db/sqlc/
git commit -m "feat: add UpsertOrganization sqlc query for web setup"
```

---

## Task 2: Extend CompanyProfile and buildOrgUserPrompt

**Files:**
- Modify: `internal/brain/org_engine.go:11-18` (CompanyProfile struct)
- Modify: `internal/brain/org_engine.go:229-247` (buildOrgUserPrompt function)

- [ ] **Step 1: Extend CompanyProfile struct**

In `internal/brain/org_engine.go`, replace the existing `CompanyProfile` (lines 11-18):

```go
// CompanyProfile holds the collected information about a company.
type CompanyProfile struct {
	Industry        string   `json:"industry"`
	Size            int      `json:"size"`
	Stage           string   `json:"stage"`
	BusinessModel   string   `json:"business_model,omitempty"`
	Region          string   `json:"region,omitempty"`
	PainPoints      []string `json:"pain_points,omitempty"`
	OrgStructure    string   `json:"org_structure,omitempty"`
	CurrentProjects string   `json:"current_projects,omitempty"`
	CommTools       []string `json:"comm_tools,omitempty"`
	CulturePrefs    string   `json:"culture_prefs,omitempty"`
	GoalFramework   string   `json:"goal_framework,omitempty"`
}
```

- [ ] **Step 2: Update buildOrgUserPrompt to include new fields**

Replace `buildOrgUserPrompt` (lines 229-247):

```go
// buildOrgUserPrompt creates the user prompt from a company profile.
func buildOrgUserPrompt(profile CompanyProfile) string {
	var sb strings.Builder
	sb.WriteString("请为以下公司设计管理体系：\n\n")
	sb.WriteString(fmt.Sprintf("行业：%s\n", profile.Industry))
	sb.WriteString(fmt.Sprintf("团队规模：%d 人\n", profile.Size))
	sb.WriteString(fmt.Sprintf("公司阶段：%s\n", profile.Stage))

	if profile.BusinessModel != "" {
		sb.WriteString(fmt.Sprintf("商业模式：%s\n", profile.BusinessModel))
	}
	if profile.Region != "" {
		sb.WriteString(fmt.Sprintf("地区：%s\n", profile.Region))
	}
	if profile.OrgStructure != "" {
		sb.WriteString(fmt.Sprintf("组织结构偏好：%s\n", profile.OrgStructure))
	}
	if profile.CurrentProjects != "" {
		sb.WriteString(fmt.Sprintf("当前项目：%s\n", profile.CurrentProjects))
	}
	if len(profile.PainPoints) > 0 {
		sb.WriteString(fmt.Sprintf("痛点/挑战：%s\n", strings.Join(profile.PainPoints, "、")))
	}
	if len(profile.CommTools) > 0 {
		sb.WriteString(fmt.Sprintf("沟通工具：%s\n", strings.Join(profile.CommTools, "、")))
	}
	if profile.CulturePrefs != "" {
		sb.WriteString(fmt.Sprintf("文化偏好：%s\n", profile.CulturePrefs))
	}
	if profile.GoalFramework != "" {
		sb.WriteString(fmt.Sprintf("目标管理框架偏好：%s\n", profile.GoalFramework))
	}

	return sb.String()
}
```

- [ ] **Step 3: Verify build and existing tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./... && go test ./internal/brain/...`

Expected: Build passes, existing tests pass (new fields are optional, backward-compatible).

- [ ] **Step 4: Commit**

```bash
git add internal/brain/org_engine.go
git commit -m "feat: extend CompanyProfile with org structure and preference fields"
```

---

## Task 3: Add handleSetupOrg Handler

**Files:**
- Modify: `internal/api/org_handlers.go` — add request type + handler function

- [ ] **Step 1: Add request type and handler**

In `internal/api/org_handlers.go`, add after the existing request types (after line 29):

```go
type setupOrgRequest struct {
	Industry        string   `json:"industry" binding:"required"`
	CompanyStage    string   `json:"company_stage" binding:"required"`
	BusinessModel   string   `json:"business_model"`
	TeamSize        int      `json:"team_size" binding:"required,min=1,max=100000"`
	OrgStructure    string   `json:"org_structure" binding:"required"`
	CurrentProjects string   `json:"current_projects"`
	PainPoints      []string `json:"pain_points" binding:"required,min=1"`
	CommTools       []string `json:"comm_tools" binding:"required,min=1"`
	CulturePrefs    string   `json:"culture_prefs"`
	GoalFramework   string   `json:"goal_framework"`
}
```

Add the handler function at end of file (after `handleActivatePlan`):

```go
// handleSetupOrg creates a new organization plan from structured form data.
// Uses OrgEngine.GeneratePlan() to produce a ManagementPlan via AI.
func handleSetupOrg(queries *sqlc.Queries, engine *brain.OrgEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if engine == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI features not available"})
			return
		}

		var req setupOrgRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Load tenant to get mentor_id
		tenant, err := queries.GetTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		mentor, err := brain.LoadMentor(tenant.MentorID)
		if err != nil {
			slog.Error("load mentor", "mentor_id", tenant.MentorID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mentor"})
			return
		}

		profile := brain.CompanyProfile{
			Industry:        req.Industry,
			Size:            req.TeamSize,
			Stage:           req.CompanyStage,
			BusinessModel:   req.BusinessModel,
			PainPoints:      req.PainPoints,
			OrgStructure:    req.OrgStructure,
			CurrentProjects: req.CurrentProjects,
			CommTools:       req.CommTools,
			CulturePrefs:    req.CulturePrefs,
			GoalFramework:   req.GoalFramework,
		}

		// Optional: match industry template for richer context
		industry := brain.MatchIndustry(req.Industry)

		plan, err := engine.GeneratePlan(c.Request.Context(), mentor, profile, industry)
		if err != nil {
			slog.Error("generate plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate plan"})
			return
		}

		planJSON, err := json.Marshal(plan)
		if err != nil {
			slog.Error("marshal plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Persist form metadata as JSON for JSONB columns
		currentProjectsJSON, _ := json.Marshal(req.CurrentProjects)
		teamStructureJSON, _ := json.Marshal(req.OrgStructure)
		culturePrefsJSON, _ := json.Marshal(req.CulturePrefs)

		org, err := queries.UpsertOrganization(c.Request.Context(), sqlc.UpsertOrganizationParams{
			TenantID:             tenantID,
			Industry:             pgtype.Text{String: req.Industry, Valid: true},
			Size:                 pgtype.Int4{Int32: int32(req.TeamSize), Valid: true},
			Stage:                pgtype.Text{String: req.CompanyStage, Valid: true},
			BusinessModel:        pgtype.Text{String: req.BusinessModel, Valid: req.BusinessModel != ""},
			MentorID:             tenant.MentorID,
			ManagementPlan:       planJSON,
			ManagementPainPoints: req.PainPoints,
			CurrentProjects:      currentProjectsJSON,
			TargetFramework:      pgtype.Text{String: req.GoalFramework, Valid: req.GoalFramework != ""},
			TeamStructure:        teamStructureJSON,
			CommunicationTools:   req.CommTools,
			CulturePreferences:   culturePrefsJSON,
		})
		if err != nil {
			slog.Error("upsert organization", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":           formatUUID(org.ID),
				"industry":     org.Industry,
				"size":         org.Size,
				"stage":        org.Stage,
				"mentor_id":    org.MentorID,
				"plan":         plan,
				"plan_version": org.PlanVersion,
				"status":       org.Status,
			},
		})
	}
}
```

Note: Add `"github.com/jackc/pgx/v5/pgtype"` to the imports. The full import block should be:

```go
import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/roles"
)
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: Build passes. Check the `pgtype` import is present.

- [ ] **Step 3: Commit**

```bash
git add internal/api/org_handlers.go
git commit -m "feat: add handleSetupOrg handler for web-based org setup"
```

---

## Task 4: Register Route

**Files:**
- Modify: `internal/api/router.go:120-121`

- [ ] **Step 1: Add POST /org/setup route**

In `internal/api/router.go`, in the org group block (after line 121), add:

```go
			org.POST("/setup", handleSetupOrg(cfg.Queries, cfg.OrgEngine))
```

Place it after the wizard stubs and before the plan routes, so the block looks like:

```go
		{
			org.POST("/wizard/start", handleStartWizard(cfg.Queries, cfg.OrgWizard))
			org.POST("/wizard/answer", handleWizardAnswer(cfg.Queries, cfg.OrgWizard))
			org.POST("/setup", handleSetupOrg(cfg.Queries, cfg.OrgEngine))
			org.GET("/plan", handleGetPlan(cfg.Queries))
			...
		}
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: Build passes.

- [ ] **Step 3: Commit**

```bash
git add internal/api/router.go
git commit -m "feat: register POST /org/setup route"
```

---

## Task 5: Frontend — Types and API

**Files:**
- Modify: `frontend/src/types/organization.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/org.ts`

- [ ] **Step 1: Add SetupOrgRequest type**

In `frontend/src/types/organization.ts`, append after the `AISuggestion` interface:

```typescript
export interface SetupOrgRequest {
  industry: string
  company_stage: string
  business_model: string
  team_size: number
  org_structure: string
  current_projects: string
  pain_points: string[]
  comm_tools: string[]
  culture_prefs: string
  goal_framework: string
}
```

- [ ] **Step 2: Export new type**

In `frontend/src/types/index.ts`, update the organization export line to include `SetupOrgRequest`:

```typescript
export type {
  WizardSession, WizardAnswer, OrgProfile, OrgPlan, ManagementPlan,
  OrgDesign, OrgUnit, SupportRole, KpiItem, MeetingItem, AlertRule,
  AIRole, AISuggestion, SuggestionStatus, SetupOrgRequest,
} from './organization'
```

- [ ] **Step 3: Add setupOrg API function**

In `frontend/src/api/org.ts`, **replace** the existing import on line 2:

```typescript
// Replace this line:
// import type { OrgPlan, ManagementPlan, AIRole, AISuggestion } from '@/types'
// With:
import type { OrgPlan, ManagementPlan, AIRole, AISuggestion, SetupOrgRequest } from '@/types'
```

Add the function:

```typescript
export async function setupOrg(data: SetupOrgRequest): Promise<OrgPlan> {
  const res = await post<{ data: OrgPlan }>('/org/setup', data)
  return res.data
}
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build passes.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/organization.ts frontend/src/types/index.ts frontend/src/api/org.ts
git commit -m "feat: add SetupOrgRequest type and setupOrg API function"
```

---

## Task 6: Frontend — CompanyProfileForm Component

**Files:**
- Create: `frontend/src/components/organization/CompanyProfileForm.vue`

- [ ] **Step 1: Create CompanyProfileForm.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import {
  NForm, NFormItem, NInput, NInputNumber, NSelect, NButton,
  NSpace, NDivider, NTag, type FormRules, type FormInst,
} from 'naive-ui'
import type { SetupOrgRequest } from '@/types'

const emit = defineEmits<{ submit: [data: SetupOrgRequest] }>()
const formRef = ref<FormInst | null>(null)

const form = ref<SetupOrgRequest>({
  industry: '',
  company_stage: '',
  business_model: '',
  team_size: 10,
  org_structure: '',
  current_projects: '',
  pain_points: [],
  comm_tools: [],
  culture_prefs: '',
  goal_framework: '',
})

const stageOptions = [
  { label: 'Startup', value: 'Startup' },
  { label: 'Growth', value: 'Growth' },
  { label: 'Mature', value: 'Mature' },
]

const modelOptions = [
  { label: 'B2B', value: 'B2B' },
  { label: 'B2C', value: 'B2C' },
  { label: 'Marketplace', value: 'Marketplace' },
  { label: 'SaaS', value: 'SaaS' },
]

const structureOptions = [
  { label: 'Flat', value: 'Flat' },
  { label: 'Hierarchical', value: 'Hierarchical' },
  { label: 'Matrix', value: 'Matrix' },
  { label: 'Team-based', value: 'Team-based' },
]

const painPointOptions = [
  { label: 'Poor communication', value: 'Poor communication' },
  { label: 'No metrics', value: 'No metrics' },
  { label: 'Low engagement', value: 'Low engagement' },
  { label: 'Unclear roles', value: 'Unclear roles' },
  { label: 'Slow decisions', value: 'Slow decisions' },
  { label: 'High turnover', value: 'High turnover' },
]

const commToolOptions = [
  { label: 'Telegram', value: 'Telegram' },
  { label: 'Slack', value: 'Slack' },
  { label: 'Lark', value: 'Lark' },
  { label: 'Email', value: 'Email' },
  { label: 'WhatsApp', value: 'WhatsApp' },
]

const frameworkOptions = [
  { label: 'OKR', value: 'OKR' },
  { label: 'KPI', value: 'KPI' },
  { label: 'Scrum', value: 'Scrum' },
  { label: 'MBO', value: 'MBO' },
  { label: 'BSC', value: 'BSC' },
]

const rules: FormRules = {
  industry: { required: true, message: 'Industry is required', trigger: 'blur' },
  company_stage: { required: true, message: 'Company stage is required', trigger: 'change' },
  team_size: { required: true, type: 'number', min: 1, message: 'Team size must be at least 1', trigger: 'blur' },
  org_structure: { required: true, message: 'Org structure is required', trigger: 'change' },
  pain_points: { required: true, type: 'array', min: 1, message: 'Select at least 1 pain point', trigger: 'change' },
  comm_tools: { required: true, type: 'array', min: 1, message: 'Select at least 1 tool', trigger: 'change' },
}

function handleSubmit() {
  formRef.value?.validate((errors) => {
    if (!errors) {
      emit('submit', { ...form.value })
    }
  })
}
</script>

<template>
  <NForm ref="formRef" :model="form" :rules="rules" label-placement="top" style="max-width: 640px">
    <NDivider title-placement="left" style="margin-top: 0">Basic Info</NDivider>

    <NFormItem label="Industry" path="industry">
      <NInput v-model:value="form.industry" placeholder="e.g. Tech, Manufacturing, Finance" />
    </NFormItem>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
      <NFormItem label="Company Stage" path="company_stage">
        <NSelect v-model:value="form.company_stage" :options="stageOptions" placeholder="Select" />
      </NFormItem>
      <NFormItem label="Business Model" path="business_model">
        <NSelect v-model:value="form.business_model" :options="modelOptions" placeholder="Optional" clearable />
      </NFormItem>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
      <NFormItem label="Team Size" path="team_size">
        <NInputNumber v-model:value="form.team_size" :min="1" :max="100000" style="width: 100%" />
      </NFormItem>
      <NFormItem label="Org Structure" path="org_structure">
        <NSelect v-model:value="form.org_structure" :options="structureOptions" placeholder="Select" />
      </NFormItem>
    </div>

    <NDivider title-placement="left">Organization</NDivider>

    <NFormItem label="Current Projects" path="current_projects">
      <NInput v-model:value="form.current_projects" type="textarea" :rows="2" placeholder="Brief description of your team's current work" />
    </NFormItem>

    <NDivider title-placement="left">Preferences</NDivider>

    <NFormItem label="Pain Points" path="pain_points">
      <NSelect v-model:value="form.pain_points" :options="painPointOptions" multiple placeholder="Select pain points" />
    </NFormItem>

    <NFormItem label="Communication Tools" path="comm_tools">
      <NSelect v-model:value="form.comm_tools" :options="commToolOptions" multiple placeholder="Select tools your team uses" />
    </NFormItem>

    <NFormItem label="Culture Preferences" path="culture_prefs">
      <NInput v-model:value="form.culture_prefs" type="textarea" :rows="2" placeholder="Optional: describe your ideal team culture" />
    </NFormItem>

    <NFormItem label="Goal Framework" path="goal_framework">
      <NSelect v-model:value="form.goal_framework" :options="frameworkOptions" placeholder="Optional" clearable />
    </NFormItem>

    <NSpace justify="end" style="margin-top: 16px">
      <NButton type="primary" @click="handleSubmit">Generate Plan</NButton>
    </NSpace>
  </NForm>
</template>
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/organization/CompanyProfileForm.vue
git commit -m "feat: add CompanyProfileForm component for org setup wizard"
```

---

## Task 7: Frontend — PlanReviewPanel and ActivateStep Components

**Files:**
- Create: `frontend/src/components/organization/PlanReviewPanel.vue`
- Create: `frontend/src/components/organization/ActivateStep.vue`

- [ ] **Step 1: Create PlanReviewPanel.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { NInput, NButton, NSpace, NDivider } from 'naive-ui'
import OrgDesignPanel from './OrgDesignPanel.vue'
import PlanDetailsPanel from './PlanDetailsPanel.vue'
import type { ManagementPlan } from '@/types'

defineProps<{ plan: ManagementPlan; adjusting: boolean }>()
const emit = defineEmits<{
  adjust: [feedback: string]
  confirm: []
  back: []
}>()

const feedback = ref('')

function handleAdjust() {
  if (!feedback.value.trim()) return
  emit('adjust', feedback.value.trim())
  feedback.value = ''
}
</script>

<template>
  <div>
    <OrgDesignPanel :design="plan.org_design" />

    <NDivider />

    <PlanDetailsPanel :plan="plan" />

    <NDivider title-placement="left">Adjust</NDivider>

    <NInput
      v-model:value="feedback"
      type="textarea"
      :rows="3"
      placeholder="Optional: describe what you'd like to change..."
      :disabled="adjusting"
      style="max-width: 640px"
    />

    <NSpace style="margin-top: 16px">
      <NButton @click="emit('back')">Back</NButton>
      <NButton :loading="adjusting" :disabled="!feedback.trim()" @click="handleAdjust">
        Regenerate
      </NButton>
      <NButton type="primary" :disabled="adjusting" @click="emit('confirm')">
        Confirm Plan
      </NButton>
    </NSpace>
  </div>
</template>
```

- [ ] **Step 2: Create ActivateStep.vue**

```vue
<script setup lang="ts">
import { NButton, NSpace, NCard, NIcon } from 'naive-ui'
import { CheckmarkCircleOutline as CheckIcon } from '@vicons/ionicons5'

defineProps<{ activating: boolean }>()
const emit = defineEmits<{ activate: []; back: [] }>()
</script>

<template>
  <NCard :bordered="false" style="max-width: 480px; box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div style="text-align: center; padding: 24px 0">
      <NIcon :size="48" color="#22c55e" :component="CheckIcon" />
      <h3 style="margin: 16px 0 8px">Ready to Activate</h3>
      <p style="color: #666; margin-bottom: 24px">
        This will create AI roles and start automated management based on your plan.
        You can adjust the plan later from the Organization page.
      </p>
      <NSpace justify="center">
        <NButton @click="emit('back')">Back</NButton>
        <NButton type="primary" :loading="activating" @click="emit('activate')">
          Activate Plan
        </NButton>
      </NSpace>
    </div>
  </NCard>
</template>
```

- [ ] **Step 3: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build passes.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/organization/PlanReviewPanel.vue frontend/src/components/organization/ActivateStep.vue
git commit -m "feat: add PlanReviewPanel and ActivateStep components"
```

---

## Task 8: Frontend — SetupWizard Container

**Files:**
- Create: `frontend/src/components/organization/SetupWizard.vue`

- [ ] **Step 1: Create SetupWizard.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { NSteps, NStep, NCard, NSpin, NAlert } from 'naive-ui'
import { useMessage } from 'naive-ui'
import CompanyProfileForm from './CompanyProfileForm.vue'
import PlanReviewPanel from './PlanReviewPanel.vue'
import ActivateStep from './ActivateStep.vue'
import { setupOrg, adjustPlan, activatePlan } from '@/api'
import type { SetupOrgRequest, OrgPlan, ManagementPlan } from '@/types'

const emit = defineEmits<{ complete: [] }>()
const message = useMessage()

const currentStep = ref(1)
const loading = ref(false)
const adjusting = ref(false)
const activating = ref(false)
const error = ref('')

const savedFormData = ref<SetupOrgRequest | null>(null)
const orgPlan = ref<OrgPlan | null>(null)
const plan = ref<ManagementPlan | null>(null)

async function handleProfileSubmit(data: SetupOrgRequest) {
  savedFormData.value = data
  loading.value = true
  error.value = ''
  try {
    orgPlan.value = await setupOrg(data)
    plan.value = orgPlan.value.plan
    currentStep.value = 2
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to generate plan'
  } finally {
    loading.value = false
  }
}

async function handleAdjust(feedback: string) {
  adjusting.value = true
  try {
    const result = await adjustPlan(feedback)
    plan.value = result.plan
    if (orgPlan.value) {
      orgPlan.value = { ...orgPlan.value, plan: result.plan, plan_version: result.plan_version }
    }
    message.success('Plan adjusted')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to adjust plan')
  } finally {
    adjusting.value = false
  }
}

function handleConfirm() {
  currentStep.value = 3
}

async function handleActivate() {
  activating.value = true
  try {
    await activatePlan()
    message.success('Plan activated!')
    emit('complete')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to activate')
  } finally {
    activating.value = false
  }
}

function handleBack() {
  if (currentStep.value > 1) {
    currentStep.value -= 1
  }
}

async function handleRetry() {
  if (savedFormData.value) {
    await handleProfileSubmit(savedFormData.value)
  }
}
</script>

<template>
  <div>
    <h2 style="font-size: 20px; font-weight: 700; margin-bottom: 8px">Set Up Your Organization</h2>
    <p style="color: #666; margin-bottom: 24px">
      Tell us about your company and AI will design a management plan tailored to your needs.
    </p>

    <NSteps :current="currentStep" style="margin-bottom: 32px; max-width: 600px">
      <NStep title="Company Profile" />
      <NStep title="Review Plan" />
      <NStep title="Activate" />
    </NSteps>

    <NSpin :show="loading">
      <template v-if="currentStep === 1">
        <NAlert v-if="error" type="error" style="margin-bottom: 16px; max-width: 640px" closable @close="error = ''">
          {{ error }}
          <template #action>
            <NButton size="small" @click="handleRetry">Retry</NButton>
          </template>
        </NAlert>
        <CompanyProfileForm @submit="handleProfileSubmit" />
      </template>

      <template v-else-if="currentStep === 2 && plan">
        <PlanReviewPanel
          :plan="plan"
          :adjusting="adjusting"
          @adjust="handleAdjust"
          @confirm="handleConfirm"
          @back="handleBack"
        />
      </template>

      <template v-else-if="currentStep === 3">
        <ActivateStep
          :activating="activating"
          @activate="handleActivate"
          @back="handleBack"
        />
      </template>
    </NSpin>
  </div>
</template>
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/organization/SetupWizard.vue
git commit -m "feat: add SetupWizard container component for 3-step org setup"
```

---

## Task 9: Frontend — Wire Wizard into OrganizationView

**Files:**
- Modify: `frontend/src/views/OrganizationView.vue`

- [ ] **Step 1: Update OrganizationView to show wizard when no plan**

Replace the `EmptyState` import and usage with `SetupWizard`. In the imports section, replace:

```typescript
import EmptyState from '@/components/shared/EmptyState.vue'
```

with:

```typescript
import SetupWizard from '@/components/organization/SetupWizard.vue'
```

In the template, replace:

```vue
    <EmptyState
      v-else-if="!loading"
      description="No organization plan found. Use the Telegram bot to set up your organization."
    />
```

with:

```vue
    <SetupWizard v-else-if="!loading" @complete="fetchData" />
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/OrganizationView.vue
git commit -m "feat: show setup wizard on Organization page when no plan exists"
```

---

## Task 10: Build, Deploy, Verify

- [ ] **Step 1: Full backend build + test**

Run:
```bash
cd /Users/anna/Documents/ai-management-brain && go build ./... && go test ./...
```

Expected: All pass.

- [ ] **Step 2: Full frontend build**

Run:
```bash
cd /Users/anna/Documents/ai-management-brain/frontend && npm run build
```

Expected: Build passes, no TS errors.

- [ ] **Step 3: Deploy frontend**

```bash
rsync -az --delete /Users/anna/Documents/ai-management-brain/frontend/dist/ ai-brain:~/ai-management-brain/frontend/dist/
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build frontend'
```

- [ ] **Step 4: Deploy backend**

Build Go binary locally, scp to server, rebuild container:

```bash
cd /Users/anna/Documents/ai-management-brain
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain
scp brain ai-brain:~/ai-management-brain/
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build brain'
```

- [ ] **Step 5: Verify on production**

1. Visit `https://manageaibrain.com/organization`
2. If no plan: wizard should appear with 3 steps
3. Fill form → submit → plan appears
4. Adjust → regenerate works
5. Activate → redirects to tabs view
6. If plan exists: tabs view (Overview, Plan Details, AI Roles, Suggestions)

- [ ] **Step 6: Final commit (if any deploy fixes needed)**
