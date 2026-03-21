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
    embedding    vector(1024) NOT NULL,

    -- Scoring & lifecycle
    importance   FLOAT DEFAULT 0.5,       -- 0.0-1.0
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,             -- NULL for long_term/profile
    merged_into  UUID REFERENCES memories(id),

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vector similarity search (cosine distance)
CREATE INDEX idx_memories_embedding ON memories
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Query indexes
CREATE INDEX idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
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

---

## Component Architecture

### Package Structure

```
internal/memory/
├── engine.go          # MemoryEngine — unified entry point
├── store.go           # MemoryStore — database operations (sqlc-generated)
├── embedder.go        # Embedder — Voyage AI client
├── retriever.go       # Retriever — semantic search + ranking
├── extractor.go       # Extractor — extract memories from reports/chases/summaries
├── consolidator.go    # Consolidator — periodic merge/upgrade/decay
├── profile.go         # ProfileBuilder — employee profile generation
└── engine_test.go     # Tests
```

### Component Responsibilities

#### Embedder (`embedder.go`)

Wraps the Voyage AI API for generating text embeddings.

- Model: `voyage-3-lite` (1024 dimensions, fast and cost-effective)
- Supports batch embedding (up to 128 texts per call)
- In-memory cache for identical texts within a session
- Graceful degradation: if Voyage API unavailable, store memory without embedding and backfill later

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
```

#### Extractor (`extractor.go`)

Listens to system events and extracts memories from various sources.

- **Report submitted** → Extract employee insights (sentiment, blockers, achievements)
- **Chase completed** → Extract strategy results (which approach, what response)
- **Summary generated** → Extract organizational knowledge (team patterns, project status)
- Uses Claude for structured extraction: "Extract memorable insights from this report"

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
- Flow: generate embedding → pgvector cosine similarity search → importance-weighted ranking → return top-K
- Budget: max 5 memories, max 800 tokens total

```go
type Retriever interface {
    Recall(ctx context.Context, query RecallQuery) (*RecallResult, error)
}

type RecallQuery struct {
    TenantID   uuid.UUID
    EmployeeID uuid.UUID
    Context    string   // Current conversation context
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
2. Cluster similar memories using embedding cosine similarity (threshold: 0.85)
3. For each cluster, call Claude: "Consolidate these observations into one higher-level insight"
4. Create new `long_term` memory, mark originals with `merged_into`

#### ProfileBuilder (`profile.go`)

Generates and maintains employee characteristic summaries.

- Input: all long-term memories for an employee
- Output: 200-token profile summary covering personality, work patterns, communication style, growth areas
- Updated monthly or when significant long-term memories are added

```go
type ProfileBuilder interface {
    Build(ctx context.Context, employeeID uuid.UUID) (*Memory, error)
    Refresh(ctx context.Context, employeeID uuid.UUID) (*Memory, error)
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
func (e *MemoryEngine) RecallForMentor(ctx context.Context, tenantID, employeeID uuid.UUID, context string) (*RecallResult, error)

// Called by event handlers after report/chase/summary
func (e *MemoryEngine) ExtractAndStore(ctx context.Context, source interface{}) error

// Called by scheduler for periodic maintenance
func (e *MemoryEngine) RunConsolidation(ctx context.Context, taskType string) error
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
    1. Fetch all short_term memories (not yet merged)
    2. Compute pairwise cosine similarity
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

## REST API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/memories` | List memories (filter by type, tier, employee_id, date range) |
| GET | `/api/v1/memories/:id` | Get single memory detail |
| POST | `/api/v1/memories/search` | Semantic search (body: `{query: string, limit: int}`) |
| DELETE | `/api/v1/memories/:id` | Delete a memory manually |
| GET | `/api/v1/employees/:id/profile` | Get employee AI profile |
| POST | `/api/v1/memories/consolidate` | Manually trigger consolidation (debug/admin) |

### Pagination

All list endpoints support cursor-based pagination:
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
| `internal/report/` | After report submission, call `MemoryEngine.ExtractAndStore()` |
| `internal/scheduler/` | Add 3 new consolidation cron jobs |
| `internal/events/` | Emit events for report/chase/summary completion; memory engine subscribes |
| `internal/api/router.go` | Register new `/api/v1/memories` routes |

---

## Error Handling

| Failure | Behavior |
|---------|----------|
| Voyage AI unavailable | Store memory without embedding; background job backfills later |
| Claude API unavailable | Skip current consolidation round; retry next schedule |
| pgvector query slow | Log warning, reduce top-K from 5 to 3 |
| Embedding dimension mismatch | Reject and log; do not store corrupted vectors |
| Memory extraction returns empty | Normal — not all reports contain notable insights |

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
```

---

## Frontend (Minimal Viable)

Three new pages in the Vue3 SPA:

1. **Memory Browser** — Table view of all memories, filterable by type/tier/employee/date
2. **Employee Profile Card** — AI-generated profile summary for each employee
3. **Memory Timeline** — Chronological view of an employee's memory evolution

---

## Migration Plan

Single migration file: `sql/migrations/000005_memories.up.sql`

Steps:
1. Enable pgvector extension
2. Create `memories` table with all indexes
3. No data migration needed (new feature)

---

## Testing Strategy

- **Unit tests**: Each component (Embedder, Extractor, Retriever, Consolidator, ProfileBuilder)
- **Integration tests**: Full flow — report → extract → store → recall → inject
- **Mock Voyage API**: Use recorded responses for deterministic tests
- **Target**: 80%+ coverage for `internal/memory/` package
