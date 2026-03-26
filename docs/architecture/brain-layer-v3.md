# Boss AI Agent v6 — Brain Layer v3 架构方案

> 基于 brain-layer-v2.md 持续迭代。v2 实现率 ~85%，v3 补齐缺口并新增混合存储层。

## 1. 目标

1. **补齐 v2 MCP 工具缺口**：已实现的 Brain 引擎暴露为 6 个 MCP 工具（24 → 30）
2. **混合存储层**：manageaibrain.com 保留核心数据，双向同步到用户的 Notion / Google Sheets
3. **Skill v6.0**：更新 SKILL.md 反映新工具 + 同步场景

## 2. v2 实现状态审计

### 已实现 ✅

| 类别 | 状态 | 备注 |
|------|------|------|
| 数据库（15 张核心表 + HalaOS） | 100% | migration 000001-000017 |
| SQLC 查询（11+ 文件） | 100% | |
| Brain 引擎（5 个核心模块） | 100% | context_service, state_engine, execution_planner, incentive_engine, recommender |
| HTTP Handler（8 组） | 100% | metrics, projects, tasks, reporting_lines, workflows, incentives, state, recommendations |
| 路由注册 | 100% | 所有 7 个资源组 |
| 前端视图（5 个新页面） | 100% | Metrics, Projects, Tasks, Incentives, State |
| MCP 工具 | 63% | 24/30 已实现 |

### v2 之外的额外实现 🎁

- **AI 推荐引擎**：recommender.go + 4 个模板生成器 + RecommendationsView.vue + 2 个 MCP 工具
- **HalaOS 集成**：webhook 接收 + 信号映射 + migration 000016

### 未实现 ❌（v3 Phase 1 补齐）

| MCP 工具 | 已有引擎 | 缺什么 |
|----------|----------|--------|
| `get_company_context` | ContextService.GetCompanyContext | MCP 包装 |
| `get_goal_state` | goals + key_results + metrics 查询 | MCP 包装 |
| `create_execution_plan` | ExecutionPlanner.GeneratePlan | MCP 包装 |
| `ingest_metric` | handleIngestMetricValue | MCP 包装 |
| `calculate_incentives` | IncentiveEngine.Calculate | MCP 包装 |
| `update_context` | org/employee update handlers | MCP 包装 + 聚合端点 |

## 3. 架构设计

### 3.1 整体架构（v3 更新）

```
用户环境 (OpenClaw Runtime)
  ├── MCP Connectors (用户自装)
  │    ├── Storage: Notion / Google Sheets
  │    ├── Development: GitHub / Linear / Calendar
  │    └── Communication: Telegram / Slack / Discord / Lark
  │
  └── Boss AI Agent Skill (brain layer + sync orchestrator)
       │
       │ ┌─────────────── 同步编排 ───────────────┐
       │ │  get_sync_manifest → 变更列表           │
       │ │  读 Notion/Sheets (OpenClaw connector)  │
       │ │  对比 + 冲突解决                         │
       │ │  写回两边                                │
       │ │  report_sync_result → 记录结果           │
       │ └──────────────────────────────────────────┘
       │
       └── manageaibrain.com API (核心数据 + 计算引擎)
            ├── Company Context Layer  ← 地基
            ├── Execution Intelligence ← 信号 + 风险
            ├── Communication Parser   ← 消息 → 事件
            ├── Incentive Engine       ← 激励评分
            ├── AI Recommendation Engine ← 主动建议
            └── Sync Service (NEW)     ← 同步状态追踪
```

### 3.2 同步架构

**核心原则**：
- manageaibrain.com 是**计算引擎**（跑 AI 分析、生成信号、计算激励）
- Notion/Sheets 是**用户视图**（用户在这里查看和编辑数据）
- Boss AI Agent skill 是**编排者**（协调两边数据同步）
- OpenClaw 连接器是**工具**（实际读写 Notion/Sheets）
- manageaibrain.com **不持有**用户的 Notion/Sheets token

**数据流**：

```
manageaibrain.com ←──── Skill 编排 ────→ Notion / Sheets
   (计算+存储)          (读两边,           (用户视图)
                        比较,
                        写回两边)
```

## 4. 数据库变更 — Migration 000018

### 4.1 新增表

```sql
-- 同步配置（每租户每存储类型一条记录）
CREATE TABLE sync_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  storage_type TEXT NOT NULL,                    -- 'notion' | 'sheets'
  is_enabled BOOLEAN NOT NULL DEFAULT false,
  entity_types TEXT[] NOT NULL DEFAULT '{}',     -- {'tasks','goals','projects','metrics'}
  sync_frequency_minutes INT NOT NULL DEFAULT 30,
  last_sync_at TIMESTAMPTZ,
  last_sync_status TEXT,                         -- 'success' | 'partial' | 'failed'
  config JSONB NOT NULL DEFAULT '{}',            -- 存储特定配置（Notion database IDs, Sheet IDs 等）
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, storage_type)
);

-- 同步日志
CREATE TABLE sync_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  sync_config_id UUID NOT NULL REFERENCES sync_configs(id),
  direction TEXT NOT NULL,                       -- 'push' | 'pull' | 'bidirectional'
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'running',        -- 'running' | 'success' | 'partial' | 'failed'
  items_pushed INT NOT NULL DEFAULT 0,
  items_pulled INT NOT NULL DEFAULT 0,
  conflicts INT NOT NULL DEFAULT 0,
  errors JSONB NOT NULL DEFAULT '[]',
  summary TEXT
);

CREATE INDEX idx_sync_configs_tenant ON sync_configs(tenant_id);
CREATE INDEX idx_sync_logs_config ON sync_logs(sync_config_id);
CREATE INDEX idx_sync_logs_started ON sync_logs(started_at DESC);
```

### 4.2 扩展现有表

```sql
-- 为支持同步追踪，给核心表加外部引用字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_source TEXT;  -- 'notion' | 'sheets' | 'jira'
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_url TEXT;
```

## 5. SQLC 查询设计

### 5.1 sync.sql

```sql
-- name: CreateSyncConfig :one
INSERT INTO sync_configs (tenant_id, storage_type, is_enabled, entity_types, sync_frequency_minutes, config)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, storage_type) DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  entity_types = EXCLUDED.entity_types,
  sync_frequency_minutes = EXCLUDED.sync_frequency_minutes,
  config = EXCLUDED.config,
  updated_at = now()
RETURNING *;

-- name: GetSyncConfig :one
SELECT * FROM sync_configs WHERE tenant_id = $1 AND storage_type = $2;

-- name: ListSyncConfigs :many
SELECT * FROM sync_configs WHERE tenant_id = $1 ORDER BY storage_type;

-- name: UpdateLastSync :exec
UPDATE sync_configs SET last_sync_at = $2, last_sync_status = $3, updated_at = now()
WHERE id = $1;

-- name: CreateSyncLog :one
INSERT INTO sync_logs (tenant_id, sync_config_id, direction)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CompleteSyncLog :exec
UPDATE sync_logs SET
  completed_at = now(),
  status = $2,
  items_pushed = $3,
  items_pulled = $4,
  conflicts = $5,
  errors = $6,
  summary = $7
WHERE id = $1;

-- name: ListSyncLogs :many
SELECT * FROM sync_logs WHERE sync_config_id = $1 ORDER BY started_at DESC LIMIT $2;
```

### 5.2 sync_manifest 查询

```sql
-- name: GetChangedTasks :many
SELECT id, title, status, priority, assignee_name, due_date, external_id, external_source, updated_at
FROM tasks
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedGoals :many
SELECT id, title, level, goal_type, status, external_id, external_source, updated_at
FROM goals
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedProjects :many
SELECT id, name, status, priority, external_id, external_source, updated_at
FROM projects
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedMetrics :many
SELECT id, name, unit, current_value, target_value, external_id, external_source, updated_at
FROM metrics
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;
```

## 6. MCP 工具设计

### 6.1 Phase 1 — 补齐 v2 工具（6 个）

#### get_company_context

```typescript
server.tool(
  "get_company_context",
  "Get the complete company context: organization profile, strategic priorities, key risks, team composition, HR insights. This is the foundation for all management reasoning — call this before making recommendations.",
  {},
  async () => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getCompanyContext(client);
  },
);
```

**后端端点**: `GET /api/v1/state/company`（已存在：handleGetCompanyState）

#### get_goal_state

```typescript
server.tool(
  "get_goal_state",
  "Get OKR and KPI progress: all goals with linked key results, current metric values vs targets, completion percentages, and owners. Use this to understand strategic alignment and goal health.",
  {},
  async () => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getGoalState(client);
  },
);
```

**后端端点**: `GET /api/v1/goals`（已存在） + `GET /api/v1/metrics?with_values=true`

#### create_execution_plan

```typescript
server.tool(
  "create_execution_plan",
  "Generate a prioritized action plan based on current company context, goals, signals, and metrics. Returns recommended next actions with owners, priorities, deadlines, and evidence-based reasoning. Use this for proactive management planning.",
  {
    focus_area: z.string().optional().describe("Optional focus area: 'risks', 'goals', 'tasks', or 'overall'"),
  },
  async ({ focus_area }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return createExecutionPlan(client, { focus_area });
  },
);
```

**后端端点**: `POST /api/v1/state/execution-plan`（需新增，调用 ExecutionPlanner）

#### ingest_metric

```typescript
server.tool(
  "ingest_metric",
  "Record a KPI data point. Use this to import metric values from external sources (spreadsheets, reports, dashboards). Specify the metric name and observed value.",
  {
    metric_id: z.string().describe("Metric UUID"),
    value: z.number().describe("The observed value"),
    observed_at: z.string().optional().describe("ISO timestamp, defaults to now"),
    source: z.string().optional().describe("Data source, e.g. 'sheets', 'manual'"),
  },
  async ({ metric_id, value, observed_at, source }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return ingestMetric(client, { metric_id, value, observed_at, source });
  },
);
```

**后端端点**: `POST /api/v1/metrics/:id/values`（已存在：handleIngestMetricValue）

#### calculate_incentives

```typescript
server.tool(
  "calculate_incentives",
  "Calculate incentive scores for all employees in a given period. Uses execution data, goal attribution, communication quality, and active incentive rules. Returns per-employee scores with breakdowns and human-review flags.",
  {
    period: z.string().describe("Period in YYYY-MM format"),
  },
  async ({ period }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return calculateIncentives(client, { period });
  },
);
```

**后端端点**: `POST /api/v1/incentives/calculate`（已存在：handleCalculateIncentives）

#### update_context

```typescript
server.tool(
  "update_context",
  "Update company context: strategic priorities, key risks, management style weights, or employee-level fields (strengths, risk flags, work scope). Use this during onboarding or when the boss shares new strategic information.",
  {
    updates: z.object({
      strategic_priorities: z.array(z.string()).optional(),
      key_risks: z.array(z.string()).optional(),
      management_style_weights: z.record(z.number()).optional(),
    }).describe("Fields to update on the organization"),
  },
  async ({ updates }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return updateContext(client, updates);
  },
);
```

**后端端点**: `PUT /api/v1/org/context`（需新增）

### 6.2 Phase 2 — 同步工具（3 个）

#### get_sync_manifest

```typescript
server.tool(
  "get_sync_manifest",
  "Get a list of data changes since the last sync. Returns changed tasks, goals, projects, and metrics with their current values and timestamps. The skill uses this to know what to push to or pull from Notion/Sheets.",
  {
    storage_type: z.enum(["notion", "sheets"]).describe("Which storage to get manifest for"),
  },
  async ({ storage_type }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return getSyncManifest(client, storage_type);
  },
);
```

**后端端点**: `GET /api/v1/sync/manifest?storage_type=notion`（新增）

返回格式：
```json
{
  "since": "2026-03-26T10:00:00Z",
  "changes": {
    "tasks": [
      { "id": "uuid", "title": "...", "status": "in_progress", "external_id": "notion-page-id", "action": "update", "updated_at": "..." }
    ],
    "goals": [...],
    "projects": [...],
    "metrics": [...]
  },
  "export_only": {
    "signals": [...],
    "recommendations": [...],
    "working_memory": {...}
  }
}
```

#### report_sync_result

```typescript
server.tool(
  "report_sync_result",
  "Report the result of a sync operation. The skill calls this after completing a sync to record what was pushed, pulled, and any conflicts encountered.",
  {
    storage_type: z.enum(["notion", "sheets"]),
    items_pushed: z.number().describe("Number of items pushed to external storage"),
    items_pulled: z.number().describe("Number of items pulled from external storage"),
    conflicts: z.number().describe("Number of conflicts detected"),
    errors: z.array(z.string()).optional().describe("Error messages if any"),
    pulled_items: z.array(z.object({
      entity_type: z.string(),
      external_id: z.string(),
      data: z.record(z.any()),
    })).optional().describe("Items pulled from external storage to update in manageaibrain"),
  },
  async ({ storage_type, items_pushed, items_pulled, conflicts, errors, pulled_items }) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return reportSyncResult(client, { storage_type, items_pushed, items_pulled, conflicts, errors, pulled_items });
  },
);
```

**后端端点**: `POST /api/v1/sync/result`（新增）

#### configure_sync

```typescript
server.tool(
  "configure_sync",
  "Configure sync settings for Notion or Google Sheets. Set which data types to sync, frequency, and storage-specific config (Notion database IDs, Sheet IDs). Call this during first-time setup.",
  {
    storage_type: z.enum(["notion", "sheets"]),
    is_enabled: z.boolean(),
    entity_types: z.array(z.enum(["tasks", "goals", "projects", "metrics"])),
    sync_frequency_minutes: z.number().optional().describe("Sync interval in minutes, default 30"),
    config: z.record(z.any()).optional().describe("Storage-specific config: Notion database IDs or Sheet IDs"),
  },
  async (params) => {
    const client = makeClient();
    if (!client) return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
    return configureSync(client, params);
  },
);
```

**后端端点**: `PUT /api/v1/sync/config`（新增）

## 7. Handler 设计

### 7.1 新增 Handler

```go
// internal/api/sync_handlers.go

func (s *Server) handleGetSyncManifest(c *gin.Context) {
    // 1. 获取 sync_config 的 last_sync_at
    // 2. 查询各表 updated_at > last_sync_at 的记录
    // 3. 查询 export_only 数据（signals, recommendations, working_memory）
    // 4. 返回 manifest JSON
}

func (s *Server) handleReportSyncResult(c *gin.Context) {
    // 1. 创建 sync_log 记录
    // 2. 如果有 pulled_items: 逐条 upsert 到对应表（通过 external_id 匹配）
    // 3. 更新 sync_config.last_sync_at
    // 4. 返回结果
}

func (s *Server) handleConfigureSync(c *gin.Context) {
    // 1. Upsert sync_config（ON CONFLICT tenant_id + storage_type）
    // 2. 返回配置
}

func (s *Server) handleCreateExecutionPlan(c *gin.Context) {
    // 1. 获取 company context
    // 2. 调用 ExecutionPlanner.GeneratePlan
    // 3. 返回行动计划
}

func (s *Server) handleUpdateContext(c *gin.Context) {
    // 1. 更新 organizations 表字段
    // 2. 返回更新后的 context
}
```

### 7.2 路由注册

```go
// 新增路由组
sync := api.Group("/sync")
{
    sync.GET("/manifest", s.handleGetSyncManifest)
    sync.POST("/result", s.handleReportSyncResult)
    sync.PUT("/config", s.handleConfigureSync)
    sync.GET("/configs", s.handleListSyncConfigs)
    sync.GET("/logs", s.handleListSyncLogs)
}

// state 组新增
state.POST("/execution-plan", s.handleCreateExecutionPlan)

// org 组新增
org.PUT("/context", s.handleUpdateContext)
```

## 8. 同步协议详解

### 8.1 同步流程

```
┌─────────────────────────────────────────────────┐
│  OpenClaw Cron (每 30 分钟) 或 用户手动触发       │
│  → 调用 Boss AI Agent Skill                      │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────┐
│  Step 1: get_sync_manifest(storage_type)         │
│  → 获取 manageaibrain.com 自上次同步后的变更     │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────┐
│  Step 2: 通过 OpenClaw Notion/Sheets 连接器      │
│  读取对应 database/sheet 的当前数据               │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────┐
│  Step 3: 对比两边数据                            │
│  - 匹配: external_id ↔ Notion page ID / Sheet row │
│  - 新增: 一边有另一边没有 → 创建                  │
│  - 更新: 两边都有 → 比较 updated_at              │
│  - 冲突: 两边都改且时间差 < 5min → 标记冲突       │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────┐
│  Step 4: 执行同步                                │
│  - Push: 写变更到 Notion/Sheets (via connector)  │
│  - Pull: 收集需写回 manageaibrain 的变更          │
│  - Export: 写信号/推荐/memory 到 Notion (只读页面) │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────┐
│  Step 5: report_sync_result(...)                 │
│  → 回报结果 + 将 pulled_items 写回后端            │
│  → 更新 last_sync_at                             │
└─────────────────────────────────────────────────┘
```

### 8.2 冲突解决策略

| 场景 | 处理 |
|------|------|
| 只有 manageaibrain 变更 | Push 到 Notion/Sheets |
| 只有 Notion/Sheets 变更 | Pull 到 manageaibrain |
| 两边都变更，时间差 ≥ 5min | Last-write-wins（以较新的 updated_at 为准） |
| 两边都变更，时间差 < 5min | 标记为冲突，生成 AI 推荐让 boss 决定 |
| 一边新增（无 external_id） | 在另一边创建，建立映射 |
| 一边删除 | 标记 soft-delete，不自动删另一边 |

### 8.3 Notion workspace 结构

首次连接时，skill 在用户 Notion 中创建以下 database：

```
📁 Boss AI Agent (Notion page)
  ├── 📋 Tasks (database) — bidirectional
  │    Properties: Title, Status, Priority, Assignee, Due Date, Project
  │
  ├── 🎯 Goals & KPIs (database) — bidirectional
  │    Properties: Title, Type (OKR/KPI), Level, Status, Target, Current
  │
  ├── 📊 Projects (database) — bidirectional
  │    Properties: Name, Status, Priority, Start Date, End Date, Blockers
  │
  ├── 📈 Metrics (database) — bidirectional
  │    Properties: Name, Unit, Current Value, Target, Last Updated
  │
  ├── 🔔 Execution Signals (page) — export only
  │    Updated daily: top risks, overdue items, engagement alerts
  │
  ├── 💡 AI Recommendations (page) — export only
  │    Updated daily: pending suggestions with priority and evidence
  │
  └── 👥 Team Overview (page) — export only
       Updated daily: team status, sentiment, check-in rates
```

### 8.4 Google Sheets 结构

```
📊 Boss AI Agent Spreadsheet
  ├── Sheet: Metrics — bidirectional (用户在这里填 KPI 数据)
  │    Columns: Metric Name | Unit | Current Value | Target | Date
  │
  ├── Sheet: Tasks — bidirectional
  │    Columns: Title | Status | Priority | Assignee | Due Date
  │
  ├── Sheet: Team Status — export only
  │    Columns: Name | Check-in Rate | Sentiment | Risk Flags
  │
  └── Sheet: Signals — export only
       Columns: Signal Type | Subject | Severity | Evidence | Date
```

## 9. 前端更新

### 9.1 Settings 页面 — Sync 配置 Tab

在 SettingsView.vue 新增 "Sync" tab：

```
┌────────────────────────────────────────────────┐
│ Settings > [General] [Sync] [Notifications]     │
├────────────────────────────────────────────────┤
│                                                 │
│  ┌─ Notion Sync ──────────────────────────────┐│
│  │ Status: ● Connected (last sync: 5 min ago) ││
│  │ Frequency: [30 min ▼]                      ││
│  │ Sync: ☑ Tasks  ☑ Goals  ☑ Projects  □ Metrics ││
│  │                          [Sync Now] [Disconnect] ││
│  └────────────────────────────────────────────┘│
│                                                 │
│  ┌─ Google Sheets Sync ──────────────────────┐ │
│  │ Status: ○ Not connected                    │ │
│  │                          [Connect Sheets]   │ │
│  └────────────────────────────────────────────┘ │
│                                                 │
│  ┌─ Sync History ─────────────────────────────┐│
│  │ 2026-03-26 10:30 | ✅ Success | ↑12 ↓3 ⚠0 ││
│  │ 2026-03-26 10:00 | ✅ Success | ↑8  ↓1 ⚠0 ││
│  │ 2026-03-26 09:30 | ⚠ Partial | ↑5  ↓0 ⚠2 ││
│  └────────────────────────────────────────────┘│
└────────────────────────────────────────────────┘
```

### 9.2 新增文件

```
frontend/src/api/sync.ts               — API 调用
frontend/src/types/sync.ts             — TS 类型
frontend/src/components/settings/SyncConfigPanel.vue  — Sync 配置面板
frontend/src/components/settings/SyncHistoryTable.vue — 同步历史表
```

## 10. SKILL.md v6.0 更新要点

### 10.1 工具数更新

- 总工具数: 24 → 33
- 新增 Read Tools — Brain Context (3): `get_company_context`, `get_goal_state`, `create_execution_plan`
- 新增 Write Tools — Context (2): `ingest_metric`, `update_context`
- 新增 Write Tools — Incentives (1): `calculate_incentives`
- 新增 Sync Tools (3): `get_sync_manifest`, `report_sync_result`, `configure_sync`

### 10.2 新场景 #12: Data Sync

```
| 12 | Data Sync | Cron (every 30min) or "sync to Notion" | get_sync_manifest → read Notion/Sheets via OpenClaw connector → compare → write back → report_sync_result |
```

### 10.3 新 Cron Job

| Job | Default Schedule | Solo Mode |
|-----|-----------------|-----------|
| sync | `*/30 9-18 * * 1-5` (every 30min work hours) | Active |

### 10.4 首次设置流程更新

Team Operations Mode 首次运行增加第 4 个问题：

```
4. "Do you want to sync data with Notion or Google Sheets?"
   - Notion → check for Notion connector → create workspace → configure_sync
   - Sheets → check for Sheets connector → create spreadsheet → configure_sync
   - Both → setup both
   - Neither → skip sync, all data stays in manageaibrain.com
```

### 10.5 OpenClaw 集成架构图更新

```
OpenClaw Runtime (user environment)
  ├── MCP Connectors (user self-installs)
  │    ├── Storage: Notion / Google Sheets  ←── 双向同步目标
  │    ├── Development: GitHub / Linear
  │    └── Communication: Telegram / Slack / Discord / Lark
  │
  └── Boss AI Agent Skill (brain layer + sync orchestrator)
       └── manageaibrain.com/mcp
            ├── 33 MCP tools (24 existing + 6 brain + 3 sync)
            ├── Company Context Layer
            ├── Execution Intelligence
            ├── AI Recommendations
            └── Sync Service ← NEW
```

## 11. 实现阶段

### Phase 1: 补齐 v2 MCP 工具 (P0, 1-2 天)

| Step | 内容 | 文件 |
|------|------|------|
| 1.1 | 后端新增 execution-plan 端点 | api/state_handlers.go |
| 1.2 | 后端新增 update-context 端点 | api/org_handlers.go |
| 1.3 | MCP: get_company_context | mcp-server/src/tools/context.ts |
| 1.4 | MCP: get_goal_state | mcp-server/src/tools/goals.ts |
| 1.5 | MCP: create_execution_plan | mcp-server/src/tools/planning.ts |
| 1.6 | MCP: ingest_metric | mcp-server/src/tools/metrics-write.ts |
| 1.7 | MCP: calculate_incentives | mcp-server/src/tools/incentives-calc.ts |
| 1.8 | MCP: update_context | mcp-server/src/tools/context.ts |
| 1.9 | 注册到 server.ts | mcp-server/src/server.ts |
| 1.10 | 测试所有新工具 | |

### Phase 2: 同步后端基础 (P0, 2-3 天)

| Step | 内容 | 文件 |
|------|------|------|
| 2.1 | Migration 000018: sync_configs + sync_logs + ALTER TABLE | internal/db/migrations/ |
| 2.2 | SQLC 查询 | internal/db/queries/sync.sql |
| 2.3 | sqlc generate | |
| 2.4 | sync_handlers.go | internal/api/sync_handlers.go |
| 2.5 | 路由注册 | internal/api/router.go |
| 2.6 | MCP: get_sync_manifest | mcp-server/src/tools/sync.ts |
| 2.7 | MCP: report_sync_result | mcp-server/src/tools/sync.ts |
| 2.8 | MCP: configure_sync | mcp-server/src/tools/sync.ts |
| 2.9 | 注册到 server.ts | mcp-server/src/server.ts |
| 2.10 | 测试同步工具 | |

### Phase 3: SKILL.md v6.0 (P0, 1 天)

| Step | 内容 | 文件 |
|------|------|------|
| 3.1 | 更新 MCP 工具列表 (24 → 33) | openclaw-skill/SKILL.md |
| 3.2 | 新增 Sync 场景 #12 | openclaw-skill/SKILL.md |
| 3.3 | 新增 sync cron job | openclaw-skill/SKILL.md |
| 3.4 | 更新首次设置流程 | openclaw-skill/SKILL.md |
| 3.5 | 更新架构图 | openclaw-skill/SKILL.md |
| 3.6 | 更新 README.md | openclaw-skill/README.md |
| 3.7 | 发布 v6.0 到 ClawHub | |

### Phase 4: 前端 Sync 设置 (P1, 1 天)

| Step | 内容 | 文件 |
|------|------|------|
| 4.1 | Sync API 层 | frontend/src/api/sync.ts |
| 4.2 | Sync 类型 | frontend/src/types/sync.ts |
| 4.3 | SyncConfigPanel 组件 | frontend/src/components/settings/ |
| 4.4 | SyncHistoryTable 组件 | frontend/src/components/settings/ |
| 4.5 | SettingsView 新增 Sync tab | frontend/src/views/SettingsView.vue |
| 4.6 | Build + 部署 | |

### Phase 5: 部署 + 验证 (P0, 半天)

| Step | 内容 |
|------|------|
| 5.1 | Go build + 上传 |
| 5.2 | Migration 000018 |
| 5.3 | MCP server 更新 |
| 5.4 | Frontend build + rsync |
| 5.5 | Docker compose restart |
| 5.6 | 验证所有 33 个 MCP 工具 |
| 5.7 | 验证同步流程端到端 |

## 12. 文件清单

### 新增文件 (~15 个)

```
internal/db/migrations/000018_sync_tables.up.sql
internal/db/migrations/000018_sync_tables.down.sql
internal/db/queries/sync.sql
internal/api/sync_handlers.go

mcp-server/src/tools/context.ts
mcp-server/src/tools/goals.ts
mcp-server/src/tools/planning.ts
mcp-server/src/tools/metrics-write.ts
mcp-server/src/tools/incentives-calc.ts
mcp-server/src/tools/sync.ts

frontend/src/api/sync.ts
frontend/src/types/sync.ts
frontend/src/components/settings/SyncConfigPanel.vue
frontend/src/components/settings/SyncHistoryTable.vue
```

### 修改文件 (~8 个)

```
internal/api/router.go               — 新增 sync + execution-plan + context 路由
internal/api/state_handlers.go        — 新增 handleCreateExecutionPlan
internal/api/org_handlers.go          — 新增 handleUpdateContext (如无则新建)
mcp-server/src/server.ts              — 注册 9 个新工具
frontend/src/views/SettingsView.vue   — 新增 Sync tab
openclaw-skill/SKILL.md               — v6.0
openclaw-skill/README.md              — 同步更新
```

### 估算

- **~15 新文件 + ~8 修改文件**
- **~2500 行新代码**
- **总 MCP 工具数: 33**
- **总工期: ~5-7 天**

## 13. 风险与缓解

| 风险 | 严重性 | 缓解 |
|------|--------|------|
| OpenClaw Notion/Sheets 连接器尚未广泛使用 | 中 | Skill 检测连接器是否安装，未安装时优雅降级 |
| 双向同步冲突 | 中 | Last-write-wins + 小时间差冲突标记为推荐 |
| 同步频率过高导致 API 限流 | 低 | 默认 30 分钟，可配置；增量同步只查变更 |
| 用户删除 Notion database | 低 | 检测到缺失时标记同步失败，不删 manageaibrain 数据 |
| report_sync_result 的 pulled_items 数据量大 | 低 | 限制每次同步最大条目数（如 100） |

## 14. 不做

- ~~实时同步（webhook 触发）~~ — 增量轮询即可，实时增加复杂度
- ~~后端持有 Notion/Sheets token~~ — 由 OpenClaw 管理
- ~~多 workspace 同步~~ — v3 只支持单个 Notion workspace + 单个 Sheets 文件
- ~~文件/附件同步~~ — 只同步结构化数据
- ~~Jira/Linear 同步~~ — v3 只做 Notion + Sheets，其他后续
