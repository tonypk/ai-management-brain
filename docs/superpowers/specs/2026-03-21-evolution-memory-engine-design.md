# Evolution Memory Engine — Design Spec

**Date:** 2026-03-21
**Project:** AI Management Brain
**Status:** Approved

## Overview

The Evolution Memory Engine adds long-term memory capabilities to the AI Management Brain. Instead of each mentor interaction being stateless, the system remembers employee patterns, strategy effectiveness, and organizational knowledge — enabling increasingly personalized and effective management guidance over time.

## Goals

1. **Employee Personal Memory** — Remember each employee's behavioral patterns, strengths, weaknesses, emotional trends, and historical context
2. **Strategy Effectiveness Tracking** — Track which management strategies work for which employees, enabling data-driven approach optimization
3. **Organizational Knowledge Base** — Accumulate collective team knowledge: project history, decisions, recurring problems and solutions

## Non-Goals

- Real-time memory updates during a conversation turn (batch extraction is sufficient)
- Complex graph-based knowledge representation (flat vector search is enough for now)
- Cross-tenant memory sharing (strict tenant isolation)

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Vector storage | pgvector on existing PostgreSQL | No new service needed, simple deployment on t3a.small |
| Embedding model | Voyage AI (voyage-3-lite, 1024 dims) | Anthropic-recommended, best compatibility with Claude |
| Memory lifecycle | Layered (short/long/profile) + AI consolidation | Balances freshness with stability |
| Memory usage | Transparent injection into mentor prompts | Seamless UX, mentor naturally references history |

---

## Data Model

### `memories` Table

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),

    -- Classification
    memory_type  VARCHAR(30) NOT NULL,    -- employee_insight | strategy_result | org_knowledge
    memory_tier  VARCHAR(20) NOT NULL DEFAULT 'short_term', -- short_term | long_term | profile

    -- Associations
    employee_id  UUID REFERENCES employees(id),
    source_type  VARCHAR(30),             -- report | chase | summary | conversation | manual
    source_id    UUID,

    -- Content
    content      TEXT NOT NULL,
    summary      TEXT,
    embedding    vector(1024),            -- Nullable: allows storage when Voyage API unavailable

    -- Scoring & lifecycle
    importance   FLOAT DEFAULT 0.5,       -- 0.0-1.0
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,             -- NULL for long_term/profile
    merged_into  UUID REFERENCES memories(id),

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- NOTE: Skip vector index on initial deployment (empty table).
-- Add IVFFlat index via migration once row count exceeds 1,000:
--   CREATE INDEX idx_memories_embedding ON memories
--       USING ivfflat (embedding vector_cosine_ops) WITH (lists = 30);
-- Until then, exact cosine distance on small datasets is fast enough.

-- Query indexes
CREATE INDEX idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
```

### Vector Search Query (with tenant isolation)

All vector similarity searches MUST filter by `tenant_id` to enforce tenant isolation:

```sql
-- name: SearchMemoriesBySimilarity :many
SELECT id, content, summary, importance, memory_type, memory_tier, employee_id, created_at,
       1 - (embedding <=> @query_embedding::vector) AS similarity
FROM memories
WHERE tenant_id = @tenant_id
  AND embedding IS NOT NULL
  AND merged_into IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY embedding <=> @query_embedding::vector
LIMIT @max_results;
```

### Memory Types

| Type | `memory_type` | Example |
|------|---------------|---------|
| Employee insight | `employee_insight` | "Xiao Ming reported increased customer complaints for 3 consecutive weeks" |
| Strategy result | `strategy_result` | "Using Inamori's gratitude-style chase improved Xiao Ming's reply rate from 60% to 90%" |
| Org knowledge | `org_knowledge` | "Q1 product launch delayed due to insufficient test environments" |

### Memory Tiers

| Tier | `memory_tier` | Lifecycle | Description |
|------|---------------|-----------|-------------|
| Short-term | `short_term` | 30-day auto-expiry | Raw insights from daily reports |
| Long-term | `long_term` | Permanent | AI-consolidated stable insights |
| Profile | `profile` | Permanent, periodically refreshed | Core employee/org characteristic summaries |

### Per-Tenant Limits

To prevent resource exhaustion on t3a.small:

| Tier | Max per tenant | Enforcement |
|------|----------------|-------------|
| Free | 5,000 memories | Prune oldest short-term when exceeded |
| Pro | 20,000 memories | Same |
| Enterprise | 100,000 memories | Same |

---

## Data Types

### Memory Struct

```go
type Memory struct {
    ID          string
    TenantID    string
    MemoryType  string     // employee_insight | strategy_result | org_knowledge
    MemoryTier  string     // short_term | long_term | profile
    EmployeeID  string     // empty if not employee-specific
    SourceType  string
    SourceID    string
    Content     string
    Summary     string
    Embedding   []float32  // nil when pending backfill
    Importance  float64
    AccessCount int
    Metadata    map[string]any
    ExpiresAt   *time.Time
    MergedInto  string     // empty if not merged
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Note: Service-layer uses `string` IDs (matching existing codebase pattern in `report.DBAdapter`). Conversion to `pgtype.UUID` happens at the database adapter boundary.

### Consolidation Task Type

```go
type ConsolidationTask string

const (
    ConsolidationClean     ConsolidationTask = "clean"
    ConsolidationMerge     ConsolidationTask = "merge"
    ConsolidationRebuild   ConsolidationTask = "rebuild"
)
```

---

## Component Architecture

### Package Structure

```
internal/memory/
├── engine.go            # MemoryEngine — unified entry point
├── store.go             # MemoryStore — database adapter (pgtype conversion)
├── embedder.go          # Embedder — Voyage AI client
├── retriever.go         # Retriever — semantic search + ranking
├── extractor.go         # Extractor — extract memories from reports/chases/summaries
├── consolidator.go      # Consolidator — periodic merge/upgrade/decay
├── profile.go           # ProfileBuilder — employee profile generation
├── embedder_test.go     # Tests per component
├── retriever_test.go
├── extractor_test.go
├── consolidator_test.go
├── profile_test.go
└── engine_test.go
```

### Component Responsibilities

#### Embedder (`embedder.go`)

Wraps the Voyage AI API for generating text embeddings.

- Model: `voyage-3-lite` (1024 dimensions, fast and cost-effective)
- Supports batch embedding (up to 128 texts per call)
- In-memory cache for identical texts within a session
- Graceful degradation: if Voyage API unavailable, return nil embedding (memory stored without vector, backfilled later)

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
```

#### Extractor (`extractor.go`)

Listens to system events and extracts memories from various sources.

- **`report.submitted`** → Extract employee insights (sentiment, blockers, achievements)
- **`chase.completed`** (new event type, must be added to event bus) → Extract strategy results
- **`summary.generated`** → Extract organizational knowledge (team patterns, project status)
- Uses Claude for structured extraction: "Extract memorable insights from this report"

Note: The existing event bus defines `ChaseTriggered` but not `ChaseCompleted`. A new `ChaseCompleted EventType = "chase.completed"` must be added to `internal/events/bus.go`, and the chaser module must emit it after completing a chase sequence.

```go
type Extractor interface {
    FromReport(ctx context.Context, report *Report) ([]Memory, error)
    FromChase(ctx context.Context, chase *ChaseLog) ([]Memory, error)
    FromSummary(ctx context.Context, summary *Summary) ([]Memory, error)
}
```

#### Retriever (`retriever.go`)

Performs semantic search to find relevant memories for a given context.

- Input: conversation context + employee ID + tenant ID
- Flow: generate embedding → pgvector cosine similarity search (filtered by tenant_id) → importance-weighted ranking → return top-K
- Budget: max 5 memories, max 800 tokens total

```go
type Retriever interface {
    Recall(ctx context.Context, query RecallQuery) (*RecallResult, error)
}

type RecallQuery struct {
    TenantID   string
    EmployeeID string
    QueryText  string   // Current conversation context
    MaxResults int      // Default: 5
    MaxTokens  int      // Default: 800
}

type RecallResult struct {
    Profile    *Memory   // Employee profile (always included if exists)
    Insights   []Memory  // Employee insights (2-3)
    Strategies []Memory  // Strategy results (1)
    Knowledge  []Memory  // Org knowledge (0-1)
    TokenCount int       // Total tokens used
}
```

#### Consolidator (`consolidator.go`)

Periodic maintenance of the memory store.

- **Daily (02:00)**: Clean expired short-term memories
- **Weekly (Sunday 03:00)**: Merge similar short-term memories into long-term insights
- **Monthly (1st, 04:00)**: Rebuild employee profiles from all long-term memories

Merge logic:
1. Group short-term memories by employee
2. Cap at 200 short-term memories per employee per run (process oldest first if exceeded)
3. Cluster similar memories using embedding cosine similarity (threshold: 0.85)
4. For each cluster, call Claude: "Consolidate these observations into one higher-level insight"
5. Create new `long_term` memory, mark originals with `merged_into`

#### ProfileBuilder (`profile.go`)

Generates and maintains employee characteristic summaries.

- Input: all long-term memories for an employee
- Output: 200-token profile summary covering personality, work patterns, communication style, growth areas
- Updated monthly or when significant long-term memories are added

```go
type ProfileBuilder interface {
    Build(ctx context.Context, employeeID string) (*Memory, error)
    Refresh(ctx context.Context, employeeID string) (*Memory, error)
}
```

#### MemoryEngine (`engine.go`)

Unified entry point that wires all components together.

```go
type MemoryEngine struct {
    store        MemoryStore
    embedder     Embedder
    retriever    Retriever
    extractor    Extractor
    consolidator Consolidator
    profiler     ProfileBuilder
}

// Called by brain engine before generating mentor response
func (e *MemoryEngine) RecallForMentor(ctx context.Context, tenantID, employeeID, queryText string) (*RecallResult, error)

// Called by event handlers after report/chase/summary (typed methods, no interface{})
func (e *MemoryEngine) ExtractFromReport(ctx context.Context, report *Report) error
func (e *MemoryEngine) ExtractFromChase(ctx context.Context, chase *ChaseLog) error
func (e *MemoryEngine) ExtractFromSummary(ctx context.Context, summary *Summary) error

// Called by scheduler for periodic maintenance
func (e *MemoryEngine) RunConsolidation(ctx context.Context, task ConsolidationTask) error
```

---

## Prompt Injection Format

When the mentor generates a response, relevant memories are injected into the prompt:

```
You are {mentor_name}, a management mentor.

<employee_context>
Name: {name}
Position: {position}
Culture: {culture}
</employee_context>

<memory>
## Employee Profile
{profile_summary}

## Relevant Memories (by relevance)
1. [{date}] {memory_content} (importance: {score})
2. [{date}] {memory_content} (importance: {score})
3. [{date}] {memory_content} (importance: {score})

## Strategy Insights
- {strategy_description} (effectiveness: {metric})
</memory>

<today_report>
{report_content}
</today_report>

Based on your management philosophy and the memories above, provide a personalized response.
```

### Retrieval Strategy

Per interaction, inject at most **5 memories** (token budget: 800):

| Slot | Type | Count | Priority |
|------|------|-------|----------|
| Profile | `profile` | 1 | Always included if exists |
| Insights | `employee_insight` | 2-3 | By cosine similarity to current context |
| Strategy | `strategy_result` | 1 | Most relevant strategy recommendation |
| Knowledge | `org_knowledge` | 0-1 | Only if relevant to current topic |

When total tokens exceed 800, trim by lowest importance score first.

---

## Scheduled Tasks

Integrated into existing `internal/scheduler/`:

| Task | Schedule | Logic |
|------|----------|-------|
| `CleanExpiredMemories` | Daily 02:00 | Delete memories where `expires_at < NOW()` and `merged_into IS NULL` |
| `ConsolidateMemories` | Weekly Sunday 03:00 | AI-merge similar short-term → long-term memories |
| `RebuildProfiles` | Monthly 1st 04:00 | Regenerate all employee profiles from long-term memories |

### Consolidation Algorithm

```
For each tenant:
  For each employee with short_term memories:
    1. Fetch short_term memories not yet merged (cap: 200 per employee, oldest first)
    2. Compute pairwise cosine similarity (max 200*200 = 40,000 comparisons)
    3. Cluster memories with similarity > 0.85
    4. For each cluster (size >= 2):
       a. Call Claude: "Merge these observations into one insight"
       b. Create new long_term memory with merged content
       c. Generate embedding for merged content
       d. Set importance = max(cluster importances)
       e. Mark originals: merged_into = new_memory_id
    5. Short-term memories not in any cluster remain until expiry
```

---

## sqlc Queries

New file: `sql/queries/memories.sql`

```sql
-- name: CreateMemory :one
INSERT INTO memories (tenant_id, memory_type, memory_tier, employee_id, source_type, source_id,
    content, summary, embedding, importance, metadata, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetMemory :one
SELECT * FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: ListMemoriesByTenant :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND ($2 = '' OR memory_type = $2)
  AND ($3 = '' OR memory_tier = $3)
  AND ($4 = '' OR employee_id::text = $4)
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: SearchMemoriesBySimilarity :many
SELECT id, tenant_id, memory_type, memory_tier, employee_id, content, summary,
       importance, metadata, created_at,
       1 - (embedding <=> @query_embedding::vector) AS similarity
FROM memories
WHERE tenant_id = @tenant_id
  AND embedding IS NOT NULL
  AND merged_into IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
  AND (@employee_filter = '' OR employee_id::text = @employee_filter)
ORDER BY embedding <=> @query_embedding::vector
LIMIT @max_results;

-- name: UpdateMemoryMergedInto :exec
UPDATE memories SET merged_into = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteExpiredMemories :execrows
DELETE FROM memories WHERE expires_at < NOW() AND merged_into IS NULL;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: CountMemoriesByTenant :one
SELECT COUNT(*) FROM memories WHERE tenant_id = $1 AND merged_into IS NULL;

-- name: ListShortTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id::text = $2
  AND memory_tier = 'short_term'
  AND merged_into IS NULL
ORDER BY created_at ASC
LIMIT 200;

-- name: ListLongTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id::text = $2
  AND memory_tier = 'long_term'
  AND merged_into IS NULL
ORDER BY importance DESC, created_at DESC;

-- name: GetProfileByEmployee :one
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id::text = $2
  AND memory_tier = 'profile'
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: BackfillEmbedding :exec
UPDATE memories SET embedding = $2, updated_at = NOW()
WHERE id = $1 AND embedding IS NULL;
```

Note: pgvector `vector` type may need a custom type override in `sqlc.yaml`. If sqlc does not natively support the `vector` type, the `SearchMemoriesBySimilarity` and `BackfillEmbedding` queries can use raw `pgx` queries alongside sqlc for standard CRUD operations.

---

## REST API Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/api/v1/memories` | List memories (filter by type, tier, employee_id, date range) | `AuthMiddleware` |
| GET | `/api/v1/memories/:id` | Get single memory detail | `AuthMiddleware` |
| POST | `/api/v1/memories/search` | Semantic search (body: `{query: string, limit: int}`) | `AuthMiddleware`, rate limit 10 req/min |
| DELETE | `/api/v1/memories/:id` | Delete a memory manually | `RequireRole("boss")` |
| GET | `/api/v1/employees/:id/profile` | Get employee AI profile | `AuthMiddleware` |
| POST | `/api/v1/memories/consolidate` | Manually trigger consolidation (admin only) | `RequireRole("boss")` |

All endpoints are under the `protected` route group with `AuthMiddleware`.

### Pagination

All list endpoints support page-based pagination (matching existing codebase pattern):

```json
{
  "data": [...],
  "meta": {
    "total": 150,
    "page": 1,
    "limit": 20,
    "has_more": true
  }
}
```

---

## Integration Points

| Existing Module | Integration |
|-----------------|-------------|
| `internal/brain/engine.go` | Call `MemoryEngine.RecallForMentor()` before generating response |
| `internal/brain/llm.go` | Add `<memory>` section to prompt template |
| `internal/report/` | After report submission, call `MemoryEngine.ExtractFromReport()` |
| `internal/scheduler/` | Add 3 new consolidation cron jobs |
| `internal/events/bus.go` | Add `ChaseCompleted EventType = "chase.completed"` constant |
| `internal/bot/` (or chaser) | Emit `ChaseCompleted` event after finishing a chase sequence |
| `internal/api/router.go` | Register new `/api/v1/memories` routes under `protected` group |

---

## Error Handling

| Failure | Behavior |
|---------|----------|
| Voyage AI unavailable | Store memory with `embedding = NULL`; background job backfills via `BackfillEmbedding` query |
| Claude API unavailable | Skip current consolidation round; retry next schedule |
| pgvector query slow | Log warning, reduce top-K from 5 to 3 |
| Embedding dimension mismatch | Reject and log; do not store corrupted vectors |
| Memory extraction returns empty | Normal — not all reports contain notable insights |
| Tenant memory limit exceeded | Prune oldest short-term memories (FIFO) |

---

## Configuration

New environment variables:

```env
# Voyage AI
VOYAGE_API_KEY=voyage-...
VOYAGE_MODEL=voyage-3-lite         # Default model
VOYAGE_BATCH_SIZE=128              # Max texts per batch

# Memory Engine
MEMORY_MAX_RECALL=5                # Max memories per retrieval
MEMORY_MAX_TOKENS=800              # Token budget for memory injection
MEMORY_SHORT_TERM_DAYS=30          # Short-term expiry
MEMORY_CONSOLIDATION_THRESHOLD=0.85 # Cosine similarity for merging
MEMORY_MAX_PER_TENANT=20000        # Per-tenant memory limit
```

---

## Frontend (Minimal Viable)

Three new pages in the Vue3 SPA:

1. **Memory Browser** — Table view of all memories, filterable by type/tier/employee/date
2. **Employee Profile Card** — AI-generated profile summary for each employee
3. **Memory Timeline** — Chronological view of an employee's memory evolution

---

## Migration Plan

Two migration files:

**`sql/migrations/000005_memories.up.sql`:**
1. Enable pgvector extension
2. Create `memories` table with query indexes (no vector index yet)
3. No data migration needed (new feature)

**`sql/migrations/000005_memories.down.sql`:**
```sql
DROP TABLE IF EXISTS memories;
-- Note: do NOT drop the vector extension as other tables may use it in the future
```

---

## Testing Strategy

- **Unit tests per component**: `embedder_test.go`, `retriever_test.go`, `extractor_test.go`, `consolidator_test.go`, `profile_test.go`, `engine_test.go`
- **Integration tests**: Full flow — report → extract → store → recall → inject
- **Mock Voyage API**: Use recorded responses for deterministic tests
- **Mock Claude API**: Use canned responses for extractor and consolidator tests
- **Target**: 80%+ coverage for `internal/memory/` package
