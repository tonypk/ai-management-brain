# Evolution Memory Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a three-layer memory system (short-term → long-term → profile) to AI Management Brain so mentors can remember employee history, track strategy effectiveness, and accumulate organizational knowledge.

**Architecture:** pgvector on existing PostgreSQL for vector storage. Voyage AI (voyage-3-lite) for 1024-dim embeddings. Event-driven extraction from reports/chases/summaries. Transparent injection into mentor prompts via semantic retrieval. Periodic AI consolidation (daily cleanup, weekly merge, monthly profile rebuild).

**Tech Stack:** Go 1.25, pgvector, Voyage AI API, Claude API (anthropic-sdk-go), pgx/v5, sqlc, Gin, gocron/v2, Redis pub/sub, Vue3+TS

**Spec:** `docs/superpowers/specs/2026-03-21-evolution-memory-engine-design.md`

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `sql/migrations/000005_memories.up.sql` | pgvector extension + memories table + indexes |
| `sql/migrations/000005_memories.down.sql` | Drop memories table |
| `sql/queries/memories.sql` | sqlc queries for memories CRUD + vector search |
| `internal/memory/types.go` | Memory struct, RecallQuery, RecallResult, ConsolidationTask, constants |
| `internal/memory/embedder.go` | Voyage AI HTTP client for text embeddings |
| `internal/memory/embedder_test.go` | Tests with mock HTTP server |
| `internal/memory/store.go` | MemoryStore DB adapter (string↔pgtype conversion) |
| `internal/memory/store_test.go` | Tests with mock DBTX |
| `internal/memory/extractor.go` | Extract memories from reports/chases/summaries via Claude |
| `internal/memory/extractor_test.go` | Tests with mock LLM |
| `internal/memory/retriever.go` | Semantic search + ranking + token budgeting |
| `internal/memory/retriever_test.go` | Tests with mock store + embedder |
| `internal/memory/consolidator.go` | Periodic merge/cleanup/profile rebuild |
| `internal/memory/consolidator_test.go` | Tests with mock store + LLM |
| `internal/memory/profile.go` | Employee profile generation via Claude |
| `internal/memory/profile_test.go` | Tests with mock store + LLM |
| `internal/memory/engine.go` | MemoryEngine wiring all components |
| `internal/memory/engine_test.go` | Integration tests |
| `internal/api/memory_handlers.go` | REST API handlers for memories |

### Modified Files

| File | Change |
|------|--------|
| `internal/events/bus.go` | Add `ChaseCompleted` event type constant |
| `internal/config/config.go` | Add Voyage AI + memory config fields |
| `internal/brain/engine.go` | Call `MemoryEngine.RecallForMentor()` before response generation |
| `internal/brain/llm.go` | Add `<memory>` section to prompt template |
| `internal/scheduler/scheduler.go` | Register 3 new consolidation cron jobs |
| `internal/api/router.go` | Register `/api/v1/memories` routes |
| `sqlc.yaml` | Add vector type override (if needed) |
| `go.mod` | Add pgvector dependency |
| `cmd/brain/main.go` | Wire MemoryEngine into app startup; add migration005 inline |

### Implementation Notes

- **Inline migrations**: This project runs migrations inline in `main.go`'s `runMigrations()` function (not from `sql/migrations/` files). The migration files are kept for reference but the SQL must also be added to `runMigrations()`.
- **Event payloads**: `ReportSubmittedPayload` only contains `EmployeeID`, `EmployeeName`, `ReportDate` — no content. The memory extractor must fetch the report content from the database.
- **sqlc `Queries` unexported `db` field**: A `*pgxpool.Pool` must be passed separately to `MemoryStore` for raw pgx vector queries.
- **UUID comparison**: The plan uses direct UUID comparison (`employee_id = $2`) instead of the spec's `employee_id::text = $2` — this is an intentional improvement for type safety and performance.

---

## Task 1: Database Migration

**Files:**
- Create: `sql/migrations/000005_memories.up.sql`
- Create: `sql/migrations/000005_memories.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- sql/migrations/000005_memories.up.sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),

    memory_type  VARCHAR(30) NOT NULL,
    memory_tier  VARCHAR(20) NOT NULL DEFAULT 'short_term',

    employee_id  UUID REFERENCES employees(id),
    source_type  VARCHAR(30),
    source_id    UUID,

    content      TEXT NOT NULL,
    summary      TEXT,
    embedding    vector(1024),

    importance   FLOAT DEFAULT 0.5,
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    merged_into  UUID REFERENCES memories(id),

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
```

- [ ] **Step 2: Write the down migration**

```sql
-- sql/migrations/000005_memories.down.sql
DROP TABLE IF EXISTS memories;
```

- [ ] **Step 3: Add inline migration to main.go**

Read `cmd/brain/main.go` and find the `runMigrations()` function. Add a `migration005` block following the pattern of migrations 002-004:

```go
// Memory table
migration005 := `
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    memory_type  VARCHAR(30) NOT NULL,
    memory_tier  VARCHAR(20) NOT NULL DEFAULT 'short_term',
    employee_id  UUID REFERENCES employees(id),
    source_type  VARCHAR(30),
    source_id    UUID,
    content      TEXT NOT NULL,
    summary      TEXT,
    embedding    vector(1024),
    importance   FLOAT DEFAULT 0.5,
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    merged_into  UUID REFERENCES memories(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX IF NOT EXISTS idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
`
if _, err := pool.Exec(ctx, migration005); err != nil {
    return fmt.Errorf("migration005: %w", err)
}
```

- [ ] **Step 4: Verify migration files exist**

Run: `ls -la sql/migrations/000005_*`
Expected: Both `.up.sql` and `.down.sql` files present.

- [ ] **Step 5: Commit**

```bash
git add sql/migrations/000005_memories.up.sql sql/migrations/000005_memories.down.sql cmd/brain/main.go
git commit -m "feat: add memories table migration with pgvector"
```

---

## Task 2: sqlc Queries + Code Generation

**Files:**
- Create: `sql/queries/memories.sql`
- Modify: `sqlc.yaml` (if vector type override needed)

- [ ] **Step 1: Write sqlc query file**

```sql
-- sql/queries/memories.sql

-- name: CreateMemory :one
INSERT INTO memories (
    tenant_id, memory_type, memory_tier, employee_id, source_type, source_id,
    content, summary, embedding, importance, metadata, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetMemory :one
SELECT * FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: ListMemoriesByTenant :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND ($2::varchar = '' OR memory_type = $2)
  AND ($3::varchar = '' OR memory_tier = $3)
  AND ($4::varchar = '' OR employee_id::text = $4)
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountMemoriesByTenant :one
SELECT COUNT(*) FROM memories
WHERE tenant_id = $1 AND merged_into IS NULL;

-- name: ListShortTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'short_term'
  AND merged_into IS NULL
ORDER BY created_at ASC
LIMIT 200;

-- name: ListLongTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'long_term'
  AND merged_into IS NULL
ORDER BY importance DESC, created_at DESC;

-- name: GetProfileByEmployee :one
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'profile'
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateMemoryMergedInto :exec
UPDATE memories SET merged_into = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteExpiredMemories :execrows
DELETE FROM memories WHERE expires_at < NOW() AND merged_into IS NULL;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: BackfillEmbedding :exec
UPDATE memories SET embedding = $2, updated_at = NOW()
WHERE id = $1 AND embedding IS NULL;

-- name: IncrementAccessCount :exec
UPDATE memories SET access_count = access_count + 1, updated_at = NOW() WHERE id = $1;

-- name: ListTenantsWithMemories :many
SELECT DISTINCT tenant_id FROM memories WHERE merged_into IS NULL;

-- name: ListEmployeesWithShortTermMemories :many
SELECT DISTINCT employee_id FROM memories
WHERE tenant_id = $1
  AND memory_tier = 'short_term'
  AND merged_into IS NULL
  AND employee_id IS NOT NULL;

-- name: ListEmployeesWithLongTermMemories :many
SELECT DISTINCT employee_id FROM memories
WHERE tenant_id = $1
  AND memory_tier = 'long_term'
  AND merged_into IS NULL
  AND employee_id IS NOT NULL;
```

- [ ] **Step 2: Check if sqlc supports pgvector**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate 2>&1`

If sqlc errors on the `vector` type, add an override to `sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "vector"
            go_type: "github.com/pgvector/pgvector-go.Vector"
```

If the override is needed, add the dependency first:
```bash
cd /Users/anna/Documents/ai-management-brain && go get github.com/pgvector/pgvector-go
```

- [ ] **Step 3: Run sqlc generate**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`
Expected: No errors. New generated files in `internal/db/sqlc/`.

- [ ] **Step 4: Verify generated code compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add sql/queries/memories.sql sqlc.yaml internal/db/sqlc/ go.mod go.sum
git commit -m "feat: add sqlc queries for memories table"
```

---

## Task 3: Config — Add Voyage AI + Memory Settings

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Read the current config file**

Read `internal/config/config.go` fully to understand the exact structure.

- [ ] **Step 2: Add new config fields**

Add to the `Config` struct:

```go
// Voyage AI
VoyageAPIKey   string
VoyageModel    string
VoyageBatchSize int

// Memory Engine
MemoryMaxRecall              int
MemoryMaxTokens              int
MemoryShortTermDays          int
MemoryConsolidationThreshold float64
MemoryMaxPerTenant           int
```

Add to the `Load()` function (in the optional fields section, after AnthropicKey):

```go
// Voyage AI (optional — memory features disabled without it)
cfg.VoyageAPIKey = os.Getenv("VOYAGE_API_KEY")
cfg.VoyageModel = getEnv("VOYAGE_MODEL", "voyage-3-lite")
cfg.VoyageBatchSize = getEnvInt("VOYAGE_BATCH_SIZE", 128)

// Memory Engine defaults
cfg.MemoryMaxRecall = getEnvInt("MEMORY_MAX_RECALL", 5)
cfg.MemoryMaxTokens = getEnvInt("MEMORY_MAX_TOKENS", 800)
cfg.MemoryShortTermDays = getEnvInt("MEMORY_SHORT_TERM_DAYS", 30)
cfg.MemoryConsolidationThreshold = getEnvFloat("MEMORY_CONSOLIDATION_THRESHOLD", 0.85)
cfg.MemoryMaxPerTenant = getEnvInt("MEMORY_MAX_PER_TENANT", 20000)
```

Add helper functions (if `getEnvInt` and `getEnvFloat` don't already exist):

```go
func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add Voyage AI and memory engine config"
```

---

## Task 4: Event Bus — Add ChaseCompleted Event

**Files:**
- Modify: `internal/events/bus.go`

- [ ] **Step 1: Read the current event bus file**

Read `internal/events/bus.go` fully.

- [ ] **Step 2: Add ChaseCompleted constant**

Add to the `const` block after `ChaseTriggered`:

```go
ChaseCompleted   EventType = "chase.completed"
```

- [ ] **Step 3: Find where chases finish and emit the event**

Search for where chase sequences complete. Look in `internal/bot/` or `internal/scheduler/` for the chase handler. After the chase completes (the last chase step or employee response), add:

```go
_ = bus.PublishPayload(ctx, events.ChaseCompleted, tenantID, map[string]any{
    "employee_id": employeeID,
    "chase_log":   chaseLog,
})
```

This needs to be identified during implementation — search for `ChaseTriggered` usage to find the chase flow.

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/events/bus.go
git commit -m "feat: add ChaseCompleted event type"
```

---

## Task 5: Memory Types

**Files:**
- Create: `internal/memory/types.go`

- [ ] **Step 1: Create the types file**

```go
package memory

import "time"

// Memory type constants
const (
	TypeEmployeeInsight = "employee_insight"
	TypeStrategyResult  = "strategy_result"
	TypeOrgKnowledge    = "org_knowledge"
)

// Memory tier constants
const (
	TierShortTerm = "short_term"
	TierLongTerm  = "long_term"
	TierProfile   = "profile"
)

// Source type constants
const (
	SourceReport       = "report"
	SourceChase        = "chase"
	SourceSummary      = "summary"
	SourceConversation = "conversation"
	SourceManual       = "manual"
)

// ConsolidationTask represents the type of periodic maintenance
type ConsolidationTask string

const (
	ConsolidationClean   ConsolidationTask = "clean"
	ConsolidationMerge   ConsolidationTask = "merge"
	ConsolidationRebuild ConsolidationTask = "rebuild"
)

// Memory represents a single memory record at the service layer.
// IDs are strings (matching existing codebase pattern); converted to pgtype at DB boundary.
type Memory struct {
	ID          string
	TenantID    string
	MemoryType  string
	MemoryTier  string
	EmployeeID  string
	SourceType  string
	SourceID    string
	Content     string
	Summary     string
	Embedding   []float32
	Importance  float64
	AccessCount int
	Metadata    map[string]any
	ExpiresAt   *time.Time
	MergedInto  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Similarity  float64 // populated by search queries
}

// RecallQuery is the input for semantic memory retrieval.
type RecallQuery struct {
	TenantID   string
	EmployeeID string
	QueryText  string
	MaxResults int // default 5
	MaxTokens  int // default 800
}

// RecallResult is the output of memory retrieval, slotted by type.
type RecallResult struct {
	Profile    *Memory
	Insights   []Memory
	Strategies []Memory
	Knowledge  []Memory
	TokenCount int
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./internal/memory/...`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add internal/memory/types.go
git commit -m "feat: add memory types, constants, and data structures"
```

---

## Task 6: Voyage AI Embedder

**Files:**
- Create: `internal/memory/embedder.go`
- Create: `internal/memory/embedder_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/embedder_test.go
package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoyageEmbedder_Embed(t *testing.T) {
	// Mock Voyage API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header")
		}

		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["model"] != "voyage-3-lite" {
			t.Errorf("unexpected model: %v", req["model"])
		}

		// Return fake embedding (3 dims for testing)
		resp := map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.1, 0.2, 0.3}},
			},
			"usage": map[string]any{"total_tokens": 10},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)
	embedder.baseURL = server.URL // override for testing

	vec, err := embedder.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(vec))
	}
	if vec[0] != 0.1 || vec[1] != 0.2 || vec[2] != 0.3 {
		t.Errorf("unexpected values: %v", vec)
	}
}

func TestVoyageEmbedder_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		inputs := req["input"].([]any)
		data := make([]map[string]any, len(inputs))
		for i := range inputs {
			data[i] = map[string]any{
				"embedding": []float64{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3},
			}
		}

		resp := map[string]any{
			"data":  data,
			"usage": map[string]any{"total_tokens": 20},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)
	embedder.baseURL = server.URL

	vecs, err := embedder.EmbedBatch(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
}

func TestVoyageEmbedder_EmptyInput(t *testing.T) {
	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)

	_, err := embedder.Embed(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestVoyageEmbedder -v`
Expected: FAIL — `NewVoyageEmbedder` not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/embedder.go
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// VoyageEmbedder calls the Voyage AI embeddings API.
type VoyageEmbedder struct {
	apiKey    string
	model     string
	batchSize int
	baseURL   string
	client    *http.Client
}

func NewVoyageEmbedder(apiKey, model string, batchSize int) *VoyageEmbedder {
	return &VoyageEmbedder{
		apiKey:    apiKey,
		model:     model,
		batchSize: batchSize,
		baseURL:   "https://api.voyageai.com",
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

type voyageRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type voyageResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (e *VoyageEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty input text")
	}

	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *VoyageEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("empty input texts")
	}

	var allResults [][]float32

	// Process in batches
	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		results, err := e.callAPI(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i/e.batchSize, err)
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (e *VoyageEmbedder) callAPI(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := voyageRequest{
		Input: texts,
		Model: e.model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voyage API error %d: %s", resp.StatusCode, string(respBody))
	}

	var voyageResp voyageResponse
	if err := json.Unmarshal(respBody, &voyageResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	results := make([][]float32, len(voyageResp.Data))
	for i, d := range voyageResp.Data {
		vec := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			vec[j] = float32(v)
		}
		results[i] = vec
	}

	return results, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestVoyageEmbedder -v`
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/embedder.go internal/memory/embedder_test.go
git commit -m "feat: add Voyage AI embedder with batch support"
```

---

## Task 7: Memory Store (DB Adapter)

**Files:**
- Create: `internal/memory/store.go`
- Create: `internal/memory/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/store_test.go
package memory

import (
	"testing"
	"time"
)

func TestFormatUUID(t *testing.T) {
	// Test the UUID formatting helper
	id := "550e8400-e29b-41d4-a716-446655440000"
	u, err := parseUUID(id)
	if err != nil {
		t.Fatalf("parse UUID: %v", err)
	}
	got := formatUUID(u)
	if got != id {
		t.Errorf("expected %q, got %q", id, got)
	}
}

func TestFormatUUID_Empty(t *testing.T) {
	id := ""
	_, err := parseUUID(id)
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestMemoryFromRow(t *testing.T) {
	// Test that conversion from sqlc row to Memory struct works
	// This tests the adapter logic
	now := time.Now()
	m := Memory{
		ID:         "test-id",
		TenantID:   "tenant-id",
		MemoryType: TypeEmployeeInsight,
		MemoryTier: TierShortTerm,
		Content:    "test content",
		Importance: 0.7,
		CreatedAt:  now,
	}

	if m.MemoryType != TypeEmployeeInsight {
		t.Errorf("expected %q, got %q", TypeEmployeeInsight, m.MemoryType)
	}
	if m.MemoryTier != TierShortTerm {
		t.Errorf("expected %q, got %q", TierShortTerm, m.MemoryTier)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestFormat -v`
Expected: FAIL — `parseUUID` not defined.

- [ ] **Step 3: Write the implementation**

The store wraps sqlc-generated queries, converting between service-layer `string` IDs and database `pgtype.UUID`. Look at the actual sqlc-generated types in `internal/db/sqlc/` to match the exact parameter/return types.

```go
// internal/memory/store.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// MemoryStore handles database operations for memories.
// Implements string↔pgtype conversion at the boundary (matching report.DBAdapter pattern).
// Takes both sqlc.Queries (for CRUD) and pgxpool.Pool (for raw vector queries).
type MemoryStore struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewMemoryStore(q *sqlc.Queries, pool *pgxpool.Pool) *MemoryStore {
	return &MemoryStore{q: q, pool: pool}
}

// --- pgtype conversion helpers (same pattern as report/dbadapter.go) ---

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if s == "" {
		return u, fmt.Errorf("empty UUID")
	}
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}

func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgtimestamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgTimestamp(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// --- Conversions: sqlc row ↔ Memory struct ---
// NOTE: The exact sqlc row type depends on what sqlc generates.
// Adjust field names to match the generated code in internal/db/sqlc/memories.sql.go.
// The pattern below assumes standard sqlc naming.

func rowToMemory(row sqlc.Memory) Memory {
	var meta map[string]any
	if row.Metadata != nil {
		json.Unmarshal(row.Metadata, &meta)
	}

	m := Memory{
		ID:          formatUUID(row.ID),
		TenantID:    formatUUID(row.TenantID),
		MemoryType:  row.MemoryType,
		MemoryTier:  row.MemoryTier,
		EmployeeID:  formatUUID(row.EmployeeID),
		SourceType:  row.SourceType.String,
		SourceID:    formatUUID(row.SourceID),
		Content:     row.Content,
		Importance:  row.Importance,
		AccessCount: int(row.AccessCount),
		Metadata:    meta,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
	if row.Summary.Valid {
		m.Summary = row.Summary.String
	}
	if row.MergedInto.Valid {
		m.MergedInto = formatUUID(row.MergedInto)
	}
	m.ExpiresAt = fromPgTimestamp(row.ExpiresAt)

	// Extract embedding if the generated type supports it
	// This depends on whether pgvector-go is used or raw bytes
	// Adjust based on actual generated type

	return m
}

// --- Public methods ---

func (s *MemoryStore) Create(ctx context.Context, m Memory) (Memory, error) {
	tenantID, err := parseUUID(m.TenantID)
	if err != nil {
		return Memory{}, fmt.Errorf("tenant_id: %w", err)
	}

	var employeeID pgtype.UUID
	if m.EmployeeID != "" {
		employeeID, err = parseUUID(m.EmployeeID)
		if err != nil {
			return Memory{}, fmt.Errorf("employee_id: %w", err)
		}
	}

	var sourceID pgtype.UUID
	if m.SourceID != "" {
		sourceID, err = parseUUID(m.SourceID)
		if err != nil {
			return Memory{}, fmt.Errorf("source_id: %w", err)
		}
	}

	metaJSON, _ := json.Marshal(m.Metadata)

	// NOTE: The embedding parameter type depends on sqlc.yaml override.
	// If using pgvector-go, pass pgvector.NewVector(m.Embedding).
	// If sqlc doesn't support vector, use a raw query for Create and
	// use sqlc for other CRUD operations. Adjust during implementation.

	row, err := s.q.CreateMemory(ctx, sqlc.CreateMemoryParams{
		TenantID:   tenantID,
		MemoryType: m.MemoryType,
		MemoryTier: m.MemoryTier,
		EmployeeID: employeeID,
		SourceType: pgtext(m.SourceType),
		SourceID:   sourceID,
		Content:    m.Content,
		Summary:    pgtext(m.Summary),
		// Embedding: handle based on generated type
		Importance: m.Importance,
		Metadata:   metaJSON,
		ExpiresAt:  pgtimestamp(m.ExpiresAt),
	})
	if err != nil {
		return Memory{}, fmt.Errorf("create memory: %w", err)
	}

	return rowToMemory(row), nil
}

func (s *MemoryStore) Get(ctx context.Context, id, tenantID string) (Memory, error) {
	mid, err := parseUUID(id)
	if err != nil {
		return Memory{}, err
	}
	tid, err := parseUUID(tenantID)
	if err != nil {
		return Memory{}, err
	}

	row, err := s.q.GetMemory(ctx, sqlc.GetMemoryParams{ID: mid, TenantID: tid})
	if err != nil {
		return Memory{}, fmt.Errorf("get memory: %w", err)
	}
	return rowToMemory(row), nil
}

func (s *MemoryStore) List(ctx context.Context, tenantID, memType, memTier, employeeID string, limit, offset int32) ([]Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListMemoriesByTenant(ctx, sqlc.ListMemoriesByTenantParams{
		TenantID: tid,
		Column2:  memType,
		Column3:  memTier,
		Column4:  employeeID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}

	memories := make([]Memory, len(rows))
	for i, row := range rows {
		memories[i] = rowToMemory(row)
	}
	return memories, nil
}

func (s *MemoryStore) Count(ctx context.Context, tenantID string) (int64, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return 0, err
	}
	return s.q.CountMemoriesByTenant(ctx, tid)
}

func (s *MemoryStore) Delete(ctx context.Context, id, tenantID string) error {
	mid, err := parseUUID(id)
	if err != nil {
		return err
	}
	tid, err := parseUUID(tenantID)
	if err != nil {
		return err
	}
	return s.q.DeleteMemory(ctx, sqlc.DeleteMemoryParams{ID: mid, TenantID: tid})
}

func (s *MemoryStore) DeleteExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredMemories(ctx)
}

func (s *MemoryStore) MarkMerged(ctx context.Context, id, mergedIntoID string) error {
	mid, err := parseUUID(id)
	if err != nil {
		return err
	}
	targetID, err := parseUUID(mergedIntoID)
	if err != nil {
		return err
	}
	return s.q.UpdateMemoryMergedInto(ctx, sqlc.UpdateMemoryMergedIntoParams{
		ID:         mid,
		MergedInto: targetID,
	})
}

func (s *MemoryStore) ListShortTermByEmployee(ctx context.Context, tenantID, employeeID string) ([]Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListShortTermByEmployee(ctx, sqlc.ListShortTermByEmployeeParams{
		TenantID:   tid,
		EmployeeID: eid,
	})
	if err != nil {
		return nil, fmt.Errorf("list short-term: %w", err)
	}

	memories := make([]Memory, len(rows))
	for i, row := range rows {
		memories[i] = rowToMemory(row)
	}
	return memories, nil
}

func (s *MemoryStore) ListLongTermByEmployee(ctx context.Context, tenantID, employeeID string) ([]Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListLongTermByEmployee(ctx, sqlc.ListLongTermByEmployeeParams{
		TenantID:   tid,
		EmployeeID: eid,
	})
	if err != nil {
		return nil, fmt.Errorf("list long-term: %w", err)
	}

	memories := make([]Memory, len(rows))
	for i, row := range rows {
		memories[i] = rowToMemory(row)
	}
	return memories, nil
}

func (s *MemoryStore) GetProfile(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return nil, err
	}

	row, err := s.q.GetProfileByEmployee(ctx, sqlc.GetProfileByEmployeeParams{
		TenantID:   tid,
		EmployeeID: eid,
	})
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	m := rowToMemory(row)
	return &m, nil
}

func (s *MemoryStore) IncrementAccess(ctx context.Context, id string) error {
	mid, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.q.IncrementAccessCount(ctx, mid)
}
```

**Important note**: The exact parameter/return types in `Create`, `List`, etc. depend on what sqlc generates from the queries. During implementation, check `internal/db/sqlc/memories.sql.go` and adjust field names and types accordingly. The `Column2`/`Column3`/`Column4` names are sqlc's default for unnamed parameters — consider using named parameters (`@param_name`) in the query if sqlc supports it.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestFormat -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/store.go internal/memory/store_test.go
git commit -m "feat: add memory store with pgtype conversion"
```

---

## Task 8: Memory Extractor

**Files:**
- Create: `internal/memory/extractor.go`
- Create: `internal/memory/extractor_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/extractor_test.go
package memory

import (
	"context"
	"testing"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return m.response, m.err
}

func TestExtractor_FromReport(t *testing.T) {
	llm := &mockLLM{
		response: `[{"content":"Employee mentioned project deadline pressure","type":"employee_insight","importance":0.7}]`,
	}
	embedder := &mockEmbedder{
		vec: []float32{0.1, 0.2, 0.3},
	}

	ext := NewExtractor(llm, embedder)

	memories, err := ext.FromReport(context.Background(), ReportInput{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		ReportID:   "report-1",
		Content:    "Today was stressful, project deadline is approaching fast.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memories) == 0 {
		t.Fatal("expected at least one memory extracted")
	}
	if memories[0].MemoryType != TypeEmployeeInsight {
		t.Errorf("expected %q, got %q", TypeEmployeeInsight, memories[0].MemoryType)
	}
	if memories[0].TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %q", memories[0].TenantID)
	}
	if memories[0].SourceType != SourceReport {
		t.Errorf("expected %q, got %q", SourceReport, memories[0].SourceType)
	}
}

func TestExtractor_EmptyReport(t *testing.T) {
	llm := &mockLLM{response: "[]"}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	ext := NewExtractor(llm, embedder)
	memories, err := ext.FromReport(context.Background(), ReportInput{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		ReportID:   "report-1",
		Content:    "Nothing special today.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(memories))
	}
}

// mockEmbedder for testing
type mockEmbedder struct {
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.vec, m.err
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.vec
	}
	return result, m.err
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestExtractor -v`
Expected: FAIL — `NewExtractor` not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/extractor.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// LLMClient matches the existing brain.LLMClient interface.
type LLMClient interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// Input types for extraction (decoupled from internal/report types)
type ReportInput struct {
	TenantID   string
	EmployeeID string
	ReportID   string
	Content    string
}

type ChaseInput struct {
	TenantID   string
	EmployeeID string
	ChaseLogID string
	Step       int
	Action     string
	Message    string
	Response   string
}

type SummaryInput struct {
	TenantID  string
	SummaryID string
	Content   string
}

// extractedInsight is the JSON structure Claude returns
type extractedInsight struct {
	Content    string  `json:"content"`
	Type       string  `json:"type"`
	Importance float64 `json:"importance"`
}

// Extractor extracts memorable insights from various sources using Claude.
type Extractor struct {
	llm      LLMClient
	embedder Embedder
}

func NewExtractor(llm LLMClient, embedder Embedder) *Extractor {
	return &Extractor{llm: llm, embedder: embedder}
}

const extractSystemPrompt = `You are a memory extraction assistant. Given a piece of text from a workplace context, extract memorable insights worth remembering long-term.

Return a JSON array of objects with these fields:
- "content": the insight in one clear sentence
- "type": one of "employee_insight", "strategy_result", "org_knowledge"
- "importance": float 0.0-1.0 (how important is this to remember)

Rules:
- Only extract genuinely notable observations (not routine/boring items)
- Keep each insight concise (one sentence)
- Return empty array [] if nothing is worth remembering
- Return valid JSON only, no markdown wrapping`

func (e *Extractor) FromReport(ctx context.Context, input ReportInput) ([]Memory, error) {
	return e.extract(ctx, input.TenantID, input.EmployeeID, SourceReport, input.ReportID, input.Content)
}

func (e *Extractor) FromChase(ctx context.Context, input ChaseInput) ([]Memory, error) {
	content := fmt.Sprintf("Chase step %d: Action=%s, Message=%s, Response=%s",
		input.Step, input.Action, input.Message, input.Response)
	return e.extract(ctx, input.TenantID, input.EmployeeID, SourceChase, input.ChaseLogID, content)
}

func (e *Extractor) FromSummary(ctx context.Context, input SummaryInput) ([]Memory, error) {
	return e.extract(ctx, input.TenantID, "", SourceSummary, input.SummaryID, input.Content)
}

func (e *Extractor) extract(ctx context.Context, tenantID, employeeID, sourceType, sourceID, content string) ([]Memory, error) {
	response, err := e.llm.Chat(ctx, extractSystemPrompt, content)
	if err != nil {
		return nil, fmt.Errorf("llm extraction: %w", err)
	}

	var insights []extractedInsight
	if err := json.Unmarshal([]byte(response), &insights); err != nil {
		return nil, fmt.Errorf("parse extraction result: %w", err)
	}

	if len(insights) == 0 {
		return nil, nil
	}

	// Generate embeddings for all insights in batch
	texts := make([]string, len(insights))
	for i, ins := range insights {
		texts[i] = ins.Content
	}

	embeddings, err := e.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		// Graceful degradation: store without embeddings
		embeddings = make([][]float32, len(insights))
	}

	expiresAt := time.Now().AddDate(0, 0, 30)
	memories := make([]Memory, len(insights))
	for i, ins := range insights {
		memories[i] = Memory{
			TenantID:   tenantID,
			EmployeeID: employeeID,
			MemoryType: ins.Type,
			MemoryTier: TierShortTerm,
			SourceType: sourceType,
			SourceID:   sourceID,
			Content:    ins.Content,
			Embedding:  embeddings[i],
			Importance: ins.Importance,
			ExpiresAt:  &expiresAt,
		}
	}

	return memories, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestExtractor -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/extractor.go internal/memory/extractor_test.go
git commit -m "feat: add memory extractor with Claude-based insight extraction"
```

---

## Task 9: Memory Retriever

**Files:**
- Create: `internal/memory/retriever.go`
- Create: `internal/memory/retriever_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/retriever_test.go
package memory

import (
	"context"
	"testing"
	"time"
)

type mockSearchStore struct {
	profile  *Memory
	insights []Memory
	longTerm []Memory
}

func (m *mockSearchStore) SearchSimilar(ctx context.Context, tenantID string, embedding []float32, employeeFilter string, maxResults int) ([]Memory, error) {
	var results []Memory
	results = append(results, m.insights...)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

func (m *mockSearchStore) GetProfile(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	return m.profile, nil
}

func (m *mockSearchStore) IncrementAccess(ctx context.Context, id string) error {
	return nil
}

func TestRetriever_Recall(t *testing.T) {
	now := time.Now()
	store := &mockSearchStore{
		profile: &Memory{
			ID:         "profile-1",
			MemoryType: TypeEmployeeInsight,
			MemoryTier: TierProfile,
			Content:    "Diligent worker, sometimes stressed by deadlines.",
			CreatedAt:  now,
		},
		insights: []Memory{
			{ID: "m1", MemoryType: TypeEmployeeInsight, Content: "Reported deadline stress", Importance: 0.8, Similarity: 0.9, CreatedAt: now},
			{ID: "m2", MemoryType: TypeStrategyResult, Content: "Gratitude chase worked well", Importance: 0.7, Similarity: 0.85, CreatedAt: now},
			{ID: "m3", MemoryType: TypeOrgKnowledge, Content: "Q1 launch delayed", Importance: 0.6, Similarity: 0.8, CreatedAt: now},
		},
	}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	retriever := NewRetriever(store, embedder, 5, 800)

	result, err := retriever.Recall(context.Background(), RecallQuery{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		QueryText:  "How is the employee doing today?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Profile == nil {
		t.Error("expected profile to be included")
	}
	if len(result.Insights) == 0 {
		t.Error("expected at least one insight")
	}
}

func TestRetriever_NoProfile(t *testing.T) {
	store := &mockSearchStore{
		profile: nil,
		insights: []Memory{
			{ID: "m1", MemoryType: TypeEmployeeInsight, Content: "test", Importance: 0.5, Similarity: 0.9},
		},
	}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	retriever := NewRetriever(store, embedder, 5, 800)
	result, err := retriever.Recall(context.Background(), RecallQuery{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		QueryText:  "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Profile != nil {
		t.Error("expected no profile")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestRetriever -v`
Expected: FAIL — `NewRetriever` not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/retriever.go
package memory

import (
	"context"
	"fmt"
)

// SearchStore is the subset of MemoryStore needed by the Retriever.
type SearchStore interface {
	SearchSimilar(ctx context.Context, tenantID string, embedding []float32, employeeFilter string, maxResults int) ([]Memory, error)
	GetProfile(ctx context.Context, tenantID, employeeID string) (*Memory, error)
	IncrementAccess(ctx context.Context, id string) error
}

// Retriever performs semantic memory recall for mentor prompt injection.
type Retriever struct {
	store      SearchStore
	embedder   Embedder
	maxResults int
	maxTokens  int
}

func NewRetriever(store SearchStore, embedder Embedder, maxResults, maxTokens int) *Retriever {
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxTokens <= 0 {
		maxTokens = 800
	}
	return &Retriever{
		store:      store,
		embedder:   embedder,
		maxResults: maxResults,
		maxTokens:  maxTokens,
	}
}

func (r *Retriever) Recall(ctx context.Context, query RecallQuery) (*RecallResult, error) {
	result := &RecallResult{}

	// 1. Try to get the profile (skip if no employee specified)
	if query.EmployeeID != "" {
		profile, err := r.store.GetProfile(ctx, query.TenantID, query.EmployeeID)
		if err == nil && profile != nil {
			result.Profile = profile
		}
	}

	// 2. Generate embedding for the query text
	queryVec, err := r.embedder.Embed(ctx, query.QueryText)
	if err != nil {
		return result, fmt.Errorf("embed query: %w", err)
	}

	// 3. Search for similar memories (filtered by tenant)
	maxSearch := r.maxResults
	if maxSearch < 10 {
		maxSearch = 10 // fetch more than needed, then filter by type
	}
	memories, err := r.store.SearchSimilar(ctx, query.TenantID, queryVec, query.EmployeeID, maxSearch)
	if err != nil {
		return result, fmt.Errorf("search similar: %w", err)
	}

	// 4. Slot memories by type
	for _, m := range memories {
		switch m.MemoryType {
		case TypeEmployeeInsight:
			if len(result.Insights) < 3 {
				result.Insights = append(result.Insights, m)
			}
		case TypeStrategyResult:
			if len(result.Strategies) < 1 {
				result.Strategies = append(result.Strategies, m)
			}
		case TypeOrgKnowledge:
			if len(result.Knowledge) < 1 {
				result.Knowledge = append(result.Knowledge, m)
			}
		}

		// Check if we have enough
		total := len(result.Insights) + len(result.Strategies) + len(result.Knowledge)
		if total >= r.maxResults-1 { // -1 for profile slot
			break
		}
	}

	// 5. Estimate token count (rough: 1 token ≈ 4 chars)
	tokenCount := 0
	if result.Profile != nil {
		tokenCount += len(result.Profile.Content) / 4
	}
	for _, m := range result.Insights {
		tokenCount += len(m.Content) / 4
	}
	for _, m := range result.Strategies {
		tokenCount += len(m.Content) / 4
	}
	for _, m := range result.Knowledge {
		tokenCount += len(m.Content) / 4
	}
	result.TokenCount = tokenCount

	// 6. Trim if over token budget (remove lowest importance first)
	// Simple approach: if over budget, drop the last insight or knowledge item
	for result.TokenCount > r.maxTokens && len(result.Knowledge) > 0 {
		result.Knowledge = result.Knowledge[:len(result.Knowledge)-1]
		result.TokenCount = r.recalcTokens(result)
	}
	for result.TokenCount > r.maxTokens && len(result.Insights) > 2 {
		result.Insights = result.Insights[:len(result.Insights)-1]
		result.TokenCount = r.recalcTokens(result)
	}

	// 7. Increment access counts (fire-and-forget)
	for _, m := range result.Insights {
		_ = r.store.IncrementAccess(ctx, m.ID)
	}
	for _, m := range result.Strategies {
		_ = r.store.IncrementAccess(ctx, m.ID)
	}

	return result, nil
}

func (r *Retriever) recalcTokens(result *RecallResult) int {
	count := 0
	if result.Profile != nil {
		count += len(result.Profile.Content) / 4
	}
	for _, m := range result.Insights {
		count += len(m.Content) / 4
	}
	for _, m := range result.Strategies {
		count += len(m.Content) / 4
	}
	for _, m := range result.Knowledge {
		count += len(m.Content) / 4
	}
	return count
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestRetriever -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/retriever.go internal/memory/retriever_test.go
git commit -m "feat: add memory retriever with semantic search and token budgeting"
```

---

## Task 10: Profile Builder

**Files:**
- Create: `internal/memory/profile.go`
- Create: `internal/memory/profile_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/profile_test.go
package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type mockProfileStore struct {
	longTerm []Memory
	created  *Memory
}

func (m *mockProfileStore) ListLongTermByEmployee(ctx context.Context, tenantID, employeeID string) ([]Memory, error) {
	return m.longTerm, nil
}

func (m *mockProfileStore) GetProfile(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockProfileStore) MarkMerged(ctx context.Context, id, mergedIntoID string) error {
	return nil
}

func (m *mockProfileStore) Create(ctx context.Context, mem Memory) (Memory, error) {
	mem.ID = "new-profile"
	m.created = &mem
	return mem, nil
}

func TestProfileBuilder_Build(t *testing.T) {
	now := time.Now()
	store := &mockProfileStore{
		longTerm: []Memory{
			{Content: "Employee is diligent and detail-oriented", Importance: 0.8, CreatedAt: now},
			{Content: "Tends to get stressed under tight deadlines", Importance: 0.7, CreatedAt: now},
			{Content: "Prefers written communication over meetings", Importance: 0.6, CreatedAt: now},
		},
	}
	llm := &mockLLM{
		response: "Diligent, detail-oriented worker who prefers written communication. Gets stressed under tight deadlines.",
	}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	builder := NewProfileBuilder(store, llm, embedder)
	profile, err := builder.Build(context.Background(), "tenant-1", "emp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.MemoryTier != TierProfile {
		t.Errorf("expected tier %q, got %q", TierProfile, profile.MemoryTier)
	}
	if profile.Content == "" {
		t.Error("expected non-empty profile content")
	}
}

func TestProfileBuilder_NoMemories(t *testing.T) {
	store := &mockProfileStore{longTerm: []Memory{}}
	llm := &mockLLM{}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	builder := NewProfileBuilder(store, llm, embedder)
	_, err := builder.Build(context.Background(), "tenant-1", "emp-1")
	if err == nil {
		t.Fatal("expected error when no memories exist")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestProfileBuilder -v`
Expected: FAIL — `NewProfileBuilder` not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/profile.go
package memory

import (
	"context"
	"fmt"
	"strings"
)

// ProfileStore is the subset of MemoryStore needed by ProfileBuilder.
type ProfileStore interface {
	ListLongTermByEmployee(ctx context.Context, tenantID, employeeID string) ([]Memory, error)
	GetProfile(ctx context.Context, tenantID, employeeID string) (*Memory, error)
	Create(ctx context.Context, m Memory) (Memory, error)
	MarkMerged(ctx context.Context, id, mergedIntoID string) error
}

// ProfileBuilder generates employee characteristic summaries from long-term memories.
type ProfileBuilder struct {
	store    ProfileStore
	llm      LLMClient
	embedder Embedder
}

func NewProfileBuilder(store ProfileStore, llm LLMClient, embedder Embedder) *ProfileBuilder {
	return &ProfileBuilder{store: store, llm: llm, embedder: embedder}
}

const profileSystemPrompt = `You are a management assistant creating an employee profile summary.
Given a list of long-term observations about an employee, create a concise profile covering:
- Personality and work style
- Communication preferences
- Strengths and growth areas
- Emotional patterns and stress triggers
- Effective management approaches

Keep it under 200 words. Write in third person. Be factual and specific, not generic.`

func (b *ProfileBuilder) Build(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	memories, err := b.store.ListLongTermByEmployee(ctx, tenantID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list long-term memories: %w", err)
	}
	if len(memories) == 0 {
		return nil, fmt.Errorf("no long-term memories for employee %s", employeeID)
	}

	// Build input from memories
	var sb strings.Builder
	for i, m := range memories {
		fmt.Fprintf(&sb, "%d. %s (importance: %.1f)\n", i+1, m.Content, m.Importance)
	}

	profileContent, err := b.llm.Chat(ctx, profileSystemPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("generate profile: %w", err)
	}

	// Generate embedding for the profile
	embedding, err := b.embedder.Embed(ctx, profileContent)
	if err != nil {
		// Graceful degradation
		embedding = nil
	}

	profile := Memory{
		TenantID:   tenantID,
		EmployeeID: employeeID,
		MemoryType: TypeEmployeeInsight,
		MemoryTier: TierProfile,
		SourceType: "system",
		Content:    profileContent,
		Embedding:  embedding,
		Importance: 1.0, // profiles are always high importance
	}

	created, err := b.store.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("save profile: %w", err)
	}

	return &created, nil
}

func (b *ProfileBuilder) Refresh(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	// Build new profile first
	newProfile, err := b.Build(ctx, tenantID, employeeID)
	if err != nil {
		return nil, err
	}

	// Mark old profile as merged into the new one
	existing, err := b.store.GetProfile(ctx, tenantID, employeeID)
	if err == nil && existing != nil && existing.ID != newProfile.ID {
		_ = b.store.MarkMerged(ctx, existing.ID, newProfile.ID)
	}

	return newProfile, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestProfileBuilder -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/profile.go internal/memory/profile_test.go
git commit -m "feat: add profile builder for employee characteristic summaries"
```

---

## Task 11: Memory Consolidator

**Files:**
- Create: `internal/memory/consolidator.go`
- Create: `internal/memory/consolidator_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/consolidator_test.go
package memory

import (
	"context"
	"testing"
	"time"
)

func TestConsolidator_CosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	sim := cosineSimilarity(a, b)
	if sim < 0.99 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", sim)
	}

	c := []float32{0, 1, 0}
	sim = cosineSimilarity(a, c)
	if sim > 0.01 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", sim)
	}
}

func TestConsolidator_ClusterMemories(t *testing.T) {
	memories := []Memory{
		{ID: "m1", Content: "Stressed about deadline", Embedding: []float32{0.9, 0.1, 0.0}},
		{ID: "m2", Content: "Worried about project timeline", Embedding: []float32{0.85, 0.15, 0.0}},
		{ID: "m3", Content: "Enjoys team meetings", Embedding: []float32{0.0, 0.1, 0.9}},
	}

	clusters := clusterMemories(memories, 0.8)

	// m1 and m2 should be in the same cluster (similar vectors)
	// m3 should be alone or unclustered
	found := false
	for _, cluster := range clusters {
		if len(cluster) == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected m1 and m2 to be clustered together")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestConsolidator -v`
Expected: FAIL — functions not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/consolidator.go
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
)

// ConsolidationStore is the subset of MemoryStore needed by Consolidator.
type ConsolidationStore interface {
	DeleteExpired(ctx context.Context) (int64, error)
	ListShortTermByEmployee(ctx context.Context, tenantID, employeeID string) ([]Memory, error)
	Create(ctx context.Context, m Memory) (Memory, error)
	MarkMerged(ctx context.Context, id, mergedIntoID string) error
}

// Consolidator performs periodic memory maintenance.
type Consolidator struct {
	store     ConsolidationStore
	llm       LLMClient
	embedder  Embedder
	threshold float64
}

func NewConsolidator(store ConsolidationStore, llm LLMClient, embedder Embedder, threshold float64) *Consolidator {
	if threshold <= 0 {
		threshold = 0.85
	}
	return &Consolidator{
		store:     store,
		llm:       llm,
		embedder:  embedder,
		threshold: threshold,
	}
}

// Clean removes expired short-term memories.
func (c *Consolidator) Clean(ctx context.Context) (int64, error) {
	count, err := c.store.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete expired: %w", err)
	}
	slog.Info("cleaned expired memories", "count", count)
	return count, nil
}

// Merge consolidates similar short-term memories into long-term insights.
func (c *Consolidator) Merge(ctx context.Context, tenantID, employeeID string) (int, error) {
	memories, err := c.store.ListShortTermByEmployee(ctx, tenantID, employeeID)
	if err != nil {
		return 0, fmt.Errorf("list short-term: %w", err)
	}

	// Filter out memories without embeddings
	var withEmbeddings []Memory
	for _, m := range memories {
		if len(m.Embedding) > 0 {
			withEmbeddings = append(withEmbeddings, m)
		}
	}

	if len(withEmbeddings) < 2 {
		return 0, nil
	}

	clusters := clusterMemories(withEmbeddings, c.threshold)

	mergedCount := 0
	for _, cluster := range clusters {
		if len(cluster) < 2 {
			continue
		}

		// Build merge prompt
		var sb strings.Builder
		maxImportance := 0.0
		for i, m := range cluster {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, m.Content)
			if m.Importance > maxImportance {
				maxImportance = m.Importance
			}
		}

		merged, err := c.llm.Chat(ctx, mergeSystemPrompt, sb.String())
		if err != nil {
			slog.Error("merge cluster failed", "error", err)
			continue
		}

		// Generate embedding for merged content
		embedding, err := c.embedder.Embed(ctx, merged)
		if err != nil {
			embedding = nil
		}

		newMemory := Memory{
			TenantID:   tenantID,
			EmployeeID: employeeID,
			MemoryType: cluster[0].MemoryType,
			MemoryTier: TierLongTerm,
			SourceType: "consolidation",
			Content:    merged,
			Embedding:  embedding,
			Importance: maxImportance,
		}

		created, err := c.store.Create(ctx, newMemory)
		if err != nil {
			slog.Error("create merged memory failed", "error", err)
			continue
		}

		// Mark originals as merged
		for _, m := range cluster {
			if err := c.store.MarkMerged(ctx, m.ID, created.ID); err != nil {
				slog.Error("mark merged failed", "memory_id", m.ID, "error", err)
			}
		}

		mergedCount++
	}

	return mergedCount, nil
}

const mergeSystemPrompt = `You are a memory consolidation assistant. Given multiple related observations about an employee, merge them into a single higher-level insight.

Rules:
- Combine into ONE concise sentence
- Preserve the most important information
- Remove redundancy
- Be factual, not speculative
- Return only the merged insight text, nothing else`

// --- Clustering helpers ---

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// clusterMemories groups memories by embedding similarity using single-linkage clustering.
func clusterMemories(memories []Memory, threshold float64) [][]Memory {
	n := len(memories)
	assigned := make([]int, n) // cluster ID per memory (-1 = unassigned)
	for i := range assigned {
		assigned[i] = -1
	}

	clusterID := 0
	for i := 0; i < n; i++ {
		if assigned[i] != -1 {
			continue
		}
		assigned[i] = clusterID
		// Find all memories similar to this one
		for j := i + 1; j < n; j++ {
			if assigned[j] != -1 {
				continue
			}
			sim := cosineSimilarity(memories[i].Embedding, memories[j].Embedding)
			if sim >= threshold {
				assigned[j] = clusterID
			}
		}
		clusterID++
	}

	// Group by cluster
	groups := make(map[int][]Memory)
	for i, cid := range assigned {
		groups[cid] = append(groups[cid], memories[i])
	}

	var clusters [][]Memory
	for _, group := range groups {
		clusters = append(clusters, group)
	}
	return clusters
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestConsolidator -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/consolidator.go internal/memory/consolidator_test.go
git commit -m "feat: add memory consolidator with clustering and AI merge"
```

---

## Task 12: Memory Engine (Wiring)

**Files:**
- Create: `internal/memory/engine.go`
- Create: `internal/memory/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/engine_test.go
package memory

import (
	"context"
	"testing"
)

func TestMemoryEngine_RecallForMentor_NoEmbedder(t *testing.T) {
	// When no Voyage API key, engine should gracefully return empty result
	engine := NewMemoryEngine(nil, nil, nil, nil, nil, nil)

	result, err := engine.RecallForMentor(context.Background(), "tenant-1", "emp-1", "How are you?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Empty result is fine — memory is optional
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestMemoryEngine -v`
Expected: FAIL — `NewMemoryEngine` not defined.

- [ ] **Step 3: Write the implementation**

```go
// internal/memory/engine.go
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// MemoryEngine is the unified entry point for the memory system.
type MemoryEngine struct {
	store        *MemoryStore
	embedder     Embedder
	retriever    *Retriever
	extractor    *Extractor
	consolidator *Consolidator
	profiler     *ProfileBuilder
}

func NewMemoryEngine(
	store *MemoryStore,
	embedder Embedder,
	retriever *Retriever,
	extractor *Extractor,
	consolidator *Consolidator,
	profiler *ProfileBuilder,
) *MemoryEngine {
	return &MemoryEngine{
		store:        store,
		embedder:     embedder,
		retriever:    retriever,
		extractor:    extractor,
		consolidator: consolidator,
		profiler:     profiler,
	}
}

// Enabled returns true if the memory engine is properly configured.
func (e *MemoryEngine) Enabled() bool {
	return e.store != nil && e.embedder != nil
}

// RecallForMentor retrieves relevant memories for prompt injection.
func (e *MemoryEngine) RecallForMentor(ctx context.Context, tenantID, employeeID, queryText string) (*RecallResult, error) {
	if !e.Enabled() || e.retriever == nil {
		return &RecallResult{}, nil
	}
	return e.retriever.Recall(ctx, RecallQuery{
		TenantID:   tenantID,
		EmployeeID: employeeID,
		QueryText:  queryText,
	})
}

// ExtractFromReport extracts and stores memories from a submitted report.
func (e *MemoryEngine) ExtractFromReport(ctx context.Context, input ReportInput) error {
	if !e.Enabled() || e.extractor == nil {
		return nil
	}

	memories, err := e.extractor.FromReport(ctx, input)
	if err != nil {
		return fmt.Errorf("extract from report: %w", err)
	}

	for _, m := range memories {
		if _, err := e.store.Create(ctx, m); err != nil {
			slog.Error("store memory failed", "source", "report", "error", err)
		}
	}
	return nil
}

// ExtractFromChase extracts and stores memories from a completed chase.
func (e *MemoryEngine) ExtractFromChase(ctx context.Context, input ChaseInput) error {
	if !e.Enabled() || e.extractor == nil {
		return nil
	}

	memories, err := e.extractor.FromChase(ctx, input)
	if err != nil {
		return fmt.Errorf("extract from chase: %w", err)
	}

	for _, m := range memories {
		if _, err := e.store.Create(ctx, m); err != nil {
			slog.Error("store memory failed", "source", "chase", "error", err)
		}
	}
	return nil
}

// ExtractFromSummary extracts and stores memories from a generated summary.
func (e *MemoryEngine) ExtractFromSummary(ctx context.Context, input SummaryInput) error {
	if !e.Enabled() || e.extractor == nil {
		return nil
	}

	memories, err := e.extractor.FromSummary(ctx, input)
	if err != nil {
		return fmt.Errorf("extract from summary: %w", err)
	}

	for _, m := range memories {
		if _, err := e.store.Create(ctx, m); err != nil {
			slog.Error("store memory failed", "source", "summary", "error", err)
		}
	}
	return nil
}

// RunConsolidation executes a periodic maintenance task.
func (e *MemoryEngine) RunConsolidation(ctx context.Context, task ConsolidationTask) error {
	if !e.Enabled() {
		return nil
	}

	switch task {
	case ConsolidationClean:
		if e.consolidator == nil {
			return nil
		}
		_, err := e.consolidator.Clean(ctx)
		return err

	case ConsolidationMerge:
		if e.consolidator == nil {
			return nil
		}
		tenantIDs, err := e.store.ListTenantsWithMemories(ctx)
		if err != nil {
			return fmt.Errorf("list tenants: %w", err)
		}
		for _, tid := range tenantIDs {
			employeeIDs, err := e.store.ListEmployeesWithShortTermMemories(ctx, tid)
			if err != nil {
				slog.Error("list employees for merge", "tenant", tid, "error", err)
				continue
			}
			for _, eid := range employeeIDs {
				merged, err := e.consolidator.Merge(ctx, tid, eid)
				if err != nil {
					slog.Error("merge failed", "tenant", tid, "employee", eid, "error", err)
				} else if merged > 0 {
					slog.Info("memories merged", "tenant", tid, "employee", eid, "count", merged)
				}
			}
		}
		return nil

	case ConsolidationRebuild:
		if e.profiler == nil {
			return nil
		}
		tenantIDs, err := e.store.ListTenantsWithMemories(ctx)
		if err != nil {
			return fmt.Errorf("list tenants: %w", err)
		}
		for _, tid := range tenantIDs {
			employeeIDs, err := e.store.ListEmployeesWithLongTermMemories(ctx, tid)
			if err != nil {
				slog.Error("list employees for profile", "tenant", tid, "error", err)
				continue
			}
			for _, eid := range employeeIDs {
				_, err := e.profiler.Refresh(ctx, tid, eid)
				if err != nil {
					slog.Error("profile rebuild failed", "tenant", tid, "employee", eid, "error", err)
				} else {
					slog.Info("profile rebuilt", "tenant", tid, "employee", eid)
				}
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown consolidation task: %s", task)
	}
}

// FormatForPrompt formats a RecallResult into the <memory> XML section for prompt injection.
func FormatForPrompt(result *RecallResult) string {
	if result == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<memory>\n")

	if result.Profile != nil {
		sb.WriteString("## Employee Profile\n")
		sb.WriteString(result.Profile.Content)
		sb.WriteString("\n\n")
	}

	if len(result.Insights) > 0 || len(result.Strategies) > 0 || len(result.Knowledge) > 0 {
		sb.WriteString("## Relevant Memories (by relevance)\n")
		idx := 1
		for _, m := range result.Insights {
			fmt.Fprintf(&sb, "%d. [%s] %s (importance: %.1f)\n",
				idx, m.CreatedAt.Format("2006-01-02"), m.Content, m.Importance)
			idx++
		}
		for _, m := range result.Knowledge {
			fmt.Fprintf(&sb, "%d. [%s] %s (importance: %.1f)\n",
				idx, m.CreatedAt.Format("2006-01-02"), m.Content, m.Importance)
			idx++
		}
		sb.WriteString("\n")
	}

	if len(result.Strategies) > 0 {
		sb.WriteString("## Strategy Insights\n")
		for _, m := range result.Strategies {
			fmt.Fprintf(&sb, "- %s\n", m.Content)
		}
	}

	sb.WriteString("</memory>")
	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -run TestMemoryEngine -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/engine.go internal/memory/engine_test.go
git commit -m "feat: add memory engine with prompt formatting"
```

---

## Task 13: Brain Engine Integration

**Files:**
- Modify: `internal/brain/engine.go`
- Modify: `internal/brain/llm.go`

- [ ] **Step 1: Read the current brain engine and LLM files**

Read `internal/brain/engine.go` and `internal/brain/llm.go` fully.

- [ ] **Step 2: Add memory recall to the brain engine**

In `internal/brain/engine.go`, find the `Engine` struct and add a `memoryEngine` field. Then find the method that generates mentor responses (likely `GenerateResponse` or similar) and inject memory recall before the Claude API call.

Add the import and field:

```go
import "github.com/tonypk/ai-management-brain/internal/memory"

// In Engine struct:
memoryEngine *memory.MemoryEngine
```

Before the LLM call, add:

```go
// Recall relevant memories
var memorySection string
if e.memoryEngine != nil && e.memoryEngine.Enabled() {
    result, err := e.memoryEngine.RecallForMentor(ctx, tenantID, employeeID, reportContent)
    if err != nil {
        slog.Warn("memory recall failed", "error", err)
    } else {
        memorySection = memory.FormatForPrompt(result)
    }
}
```

- [ ] **Step 3: Inject memory into the prompt template**

In `internal/brain/llm.go`, modify the `BuildSystemPrompt` or the user prompt assembly to include the `<memory>` section:

```go
// After building the base system prompt, before the report content:
if memorySection != "" {
    prompt += "\n\n" + memorySection + "\n"
}
```

The exact insertion point depends on the current prompt structure. Insert the memory section between the employee context and the report content.

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/brain/engine.go internal/brain/llm.go
git commit -m "feat: integrate memory recall into mentor prompt pipeline"
```

---

## Task 14: Event Bus + Scheduler Integration

**Files:**
- Modify: `internal/scheduler/scheduler.go`
- Modify: `cmd/brain/main.go`

- [ ] **Step 1: Read current scheduler and main.go**

Read both files to understand the existing wiring.

- [ ] **Step 2: Add memory consolidation jobs to scheduler**

In `internal/scheduler/scheduler.go`, add the three new cron jobs. Follow the existing `AddJob` pattern:

```go
// After existing jobs are registered:
// Clean expired memories daily at 02:00
s.AddJob("memory-clean", "0 2 * * *", func(ctx context.Context) error {
    return memEngine.RunConsolidation(ctx, memory.ConsolidationClean)
})

// Consolidate memories weekly on Sunday at 03:00
s.AddJob("memory-consolidate", "0 3 * * 0", func(ctx context.Context) error {
    return memEngine.RunConsolidation(ctx, memory.ConsolidationMerge)
})

// Rebuild profiles monthly on the 1st at 04:00
s.AddJob("memory-profiles", "0 4 1 * *", func(ctx context.Context) error {
    return memEngine.RunConsolidation(ctx, memory.ConsolidationRebuild)
})
```

- [ ] **Step 3: Wire memory event subscribers in main.go**

In `cmd/brain/main.go`, where the event bus is set up:

```go
// Subscribe to events for memory extraction
// NOTE: ReportSubmittedPayload only has EmployeeID/EmployeeName/ReportDate — no content.
// Must fetch the actual report content from the database.
eventBus.Subscribe(events.ReportSubmitted, func(ctx context.Context, event events.Event) error {
    var payload struct {
        EmployeeID   string `json:"employee_id"`
        EmployeeName string `json:"employee_name"`
        ReportDate   string `json:"report_date"`
    }
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return err
    }
    // Fetch the latest report for this employee to get content
    // Use the existing report queries (e.g., queries.GetLatestReportByEmployee)
    // The exact method depends on the existing sqlc queries — check sql/queries/reports.sql
    report, err := queries.GetLatestReportByEmployee(ctx, /* params */)
    if err != nil {
        slog.Warn("fetch report for memory extraction failed", "error", err)
        return nil // non-fatal
    }
    return memEngine.ExtractFromReport(ctx, memory.ReportInput{
        TenantID:   event.TenantID,
        EmployeeID: payload.EmployeeID,
        ReportID:   formatUUID(report.ID), // convert pgtype.UUID to string
        Content:    string(report.Answers),  // JSONB answers field
    })
})

eventBus.Subscribe(events.ChaseCompleted, func(ctx context.Context, event events.Event) error {
    var payload struct {
        TenantID   string `json:"tenant_id"`
        EmployeeID string `json:"employee_id"`
        ChaseLogID string `json:"chase_log_id"`
        Step       int    `json:"step"`
        Action     string `json:"action"`
        Message    string `json:"message"`
        Response   string `json:"response"`
    }
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return err
    }
    return memEngine.ExtractFromChase(ctx, memory.ChaseInput{
        TenantID:   payload.TenantID,
        EmployeeID: payload.EmployeeID,
        ChaseLogID: payload.ChaseLogID,
        Step:       payload.Step,
        Action:     payload.Action,
        Message:    payload.Message,
        Response:   payload.Response,
    })
})

eventBus.Subscribe(events.SummaryGenerated, func(ctx context.Context, event events.Event) error {
    var payload struct {
        TenantID  string `json:"tenant_id"`
        SummaryID string `json:"summary_id"`
        Content   string `json:"content"`
    }
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return err
    }
    return memEngine.ExtractFromSummary(ctx, memory.SummaryInput{
        TenantID:  payload.TenantID,
        SummaryID: payload.SummaryID,
        Content:   payload.Content,
    })
})
```

- [ ] **Step 4: Wire MemoryEngine initialization in main.go**

```go
// Initialize memory engine (conditional on VOYAGE_API_KEY)
var memEngine *memory.MemoryEngine
var memStore *memory.MemoryStore
if cfg.VoyageAPIKey != "" {
    embedder := memory.NewVoyageEmbedder(cfg.VoyageAPIKey, cfg.VoyageModel, cfg.VoyageBatchSize)
    memStore = memory.NewMemoryStore(queries, pool) // pool is *pgxpool.Pool from DB setup
    extractor := memory.NewExtractor(llmClient, embedder)
    retriever := memory.NewRetriever(memStore, embedder, cfg.MemoryMaxRecall, cfg.MemoryMaxTokens)
    consolidator := memory.NewConsolidator(memStore, llmClient, embedder, cfg.MemoryConsolidationThreshold)
    profiler := memory.NewProfileBuilder(memStore, llmClient, embedder)
    memEngine = memory.NewMemoryEngine(memStore, embedder, retriever, extractor, consolidator, profiler)
    slog.Info("memory engine enabled")
} else {
    memEngine = memory.NewMemoryEngine(nil, nil, nil, nil, nil, nil)
    slog.Info("memory engine disabled (no VOYAGE_API_KEY)")
}
```

- [ ] **Step 5: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/scheduler/scheduler.go cmd/brain/main.go
git commit -m "feat: wire memory engine into event bus and scheduler"
```

---

## Task 15: REST API Handlers

**Files:**
- Create: `internal/api/memory_handlers.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Write the API handlers**

```go
// internal/api/memory_handlers.go
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tonypk/ai-management-brain/internal/memory"
)

func handleListMemories(memEngine *memory.MemoryEngine, store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		memType := c.Query("type")
		memTier := c.Query("tier")
		employeeID := c.Query("employee_id")

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		memories, err := store.List(c.Request.Context(), tenantID, memType, memTier, employeeID, int32(limit), int32(offset))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list memories"})
			return
		}

		total, _ := store.Count(c.Request.Context(), tenantID)

		c.JSON(http.StatusOK, gin.H{
			"data": memories,
			"meta": gin.H{
				"total":    total,
				"page":     page,
				"limit":    limit,
				"has_more": int64(offset+limit) < total,
			},
		})
	}
}

func handleGetMemory(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		id := c.Param("id")

		mem, err := store.Get(c.Request.Context(), id, tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": mem})
	}
}

func handleSearchMemories(memEngine *memory.MemoryEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")

		var req struct {
			Query      string `json:"query" binding:"required"`
			EmployeeID string `json:"employee_id"` // optional
			Limit      int    `json:"limit"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
			return
		}
		if req.Limit <= 0 || req.Limit > 20 {
			req.Limit = 10
		}

		result, err := memEngine.RecallForMentor(c.Request.Context(), tenantID, req.EmployeeID, req.Query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func handleDeleteMemory(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		id := c.Param("id")

		if err := store.Delete(c.Request.Context(), id, tenantID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete memory"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
	}
}

func handleGetEmployeeProfile(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		employeeID := c.Param("id")

		profile, err := store.GetProfile(c.Request.Context(), tenantID, employeeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": profile})
	}
}

func handleTriggerConsolidation(memEngine *memory.MemoryEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Task string `json:"task" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task is required (clean, merge, rebuild)"})
			return
		}

		task := memory.ConsolidationTask(req.Task)
		if err := memEngine.RunConsolidation(c.Request.Context(), task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": req.Task}})
	}
}
```

- [ ] **Step 2: Register routes in router.go**

In `internal/api/router.go`, add inside the `protected` group:

```go
// Memory routes
memories := protected.Group("/memories")
{
    memories.GET("", handleListMemories(memEngine, memStore))
    memories.GET("/:id", handleGetMemory(memStore))
    memories.POST("/search", handleSearchMemories(memEngine))
    memories.DELETE("/:id", RequireRole("boss"), handleDeleteMemory(memStore))
    memories.POST("/consolidate", RequireRole("boss"), handleTriggerConsolidation(memEngine))
}

// Employee profile (add alongside existing employee routes)
protected.GET("/employees/:id/profile", handleGetEmployeeProfile(memStore))
```

Update `RouterConfig` to include `MemoryEngine` and `MemoryStore`:

```go
type RouterConfig struct {
    // ... existing fields ...
    MemoryEngine *memory.MemoryEngine
    MemoryStore  *memory.MemoryStore
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/api/memory_handlers.go internal/api/router.go
git commit -m "feat: add REST API endpoints for memories"
```

---

## Task 16: Vector Search via Raw pgx (if sqlc doesn't support vector)

**Files:**
- Modify: `internal/memory/store.go`

This task is conditional — only needed if sqlc cannot handle the `vector` type properly.

- [ ] **Step 1: Check sqlc generated code for vector support**

Look at the generated `internal/db/sqlc/memories.sql.go` file. If the `SearchMemoriesBySimilarity` query was not generated or has incorrect types, implement the search using raw pgx.

- [ ] **Step 2: Add SearchSimilar method using raw pgx**

```go
// Add to store.go — uses the pool field (not sqlc Queries) for raw pgx vector queries

import (
	"github.com/jackc/pgx/v5/pgtype"
)

// SearchSimilar performs vector similarity search with tenant isolation.
// Uses raw pgx because sqlc may not support the vector type natively.
func (s *MemoryStore) SearchSimilar(ctx context.Context, tenantID string, embedding []float32, employeeFilter string, maxResults int) ([]Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}

	vecStr := float32SliceToVectorString(embedding)

	query := `
		SELECT id, tenant_id, memory_type, memory_tier, employee_id, source_type,
		       content, summary, importance, access_count, metadata, expires_at,
		       merged_into, created_at, updated_at,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM memories
		WHERE tenant_id = $2
		  AND embedding IS NOT NULL
		  AND merged_into IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND ($3::varchar = '' OR employee_id::text = $3)
		ORDER BY embedding <=> $1::vector
		LIMIT $4`

	rows, err := s.pool.Query(ctx, query, vecStr, tid, employeeFilter, maxResults)
	if err != nil {
		return nil, fmt.Errorf("search similar: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var (
			id, tenantUUID, employeeID, sourceID, mergedInto pgtype.UUID
			memType, memTier                                 string
			sourceType, summary                              pgtype.Text
			content                                          string
			importance                                       float64
			accessCount                                      int32
			metadata                                         []byte
			expiresAt                                        pgtype.Timestamptz
			createdAt, updatedAt                             pgtype.Timestamptz
			similarity                                       float64
		)

		err := rows.Scan(
			&id, &tenantUUID, &memType, &memTier, &employeeID, &sourceType,
			&content, &summary, &importance, &accessCount, &metadata, &expiresAt,
			&mergedInto, &createdAt, &updatedAt, &similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		var meta map[string]any
		if metadata != nil {
			json.Unmarshal(metadata, &meta)
		}

		m := Memory{
			ID:          formatUUID(id),
			TenantID:    formatUUID(tenantUUID),
			MemoryType:  memType,
			MemoryTier:  memTier,
			EmployeeID:  formatUUID(employeeID),
			SourceType:  sourceType.String,
			Content:     content,
			Summary:     summary.String,
			Importance:  importance,
			AccessCount: int(accessCount),
			Metadata:    meta,
			ExpiresAt:   fromPgTimestamp(expiresAt),
			MergedInto:  formatUUID(mergedInto),
			CreatedAt:   createdAt.Time,
			UpdatedAt:   updatedAt.Time,
			Similarity:  similarity,
		}
		memories = append(memories, m)
	}

	return memories, rows.Err()
}

// ListTenantsWithMemories returns distinct tenant IDs that have memories.
func (s *MemoryStore) ListTenantsWithMemories(ctx context.Context) ([]string, error) {
	rows, err := s.q.ListTenantsWithMemories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(rows))
	for i, r := range rows {
		result[i] = formatUUID(r)
	}
	return result, nil
}

// ListEmployeesWithShortTermMemories returns employee IDs with short-term memories.
func (s *MemoryStore) ListEmployeesWithShortTermMemories(ctx context.Context, tenantID string) ([]string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListEmployeesWithShortTermMemories(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(rows))
	for i, r := range rows {
		result[i] = formatUUID(r)
	}
	return result, nil
}

// ListEmployeesWithLongTermMemories returns employee IDs with long-term memories.
func (s *MemoryStore) ListEmployeesWithLongTermMemories(ctx context.Context, tenantID string) ([]string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListEmployeesWithLongTermMemories(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(rows))
	for i, r := range rows {
		result[i] = formatUUID(r)
	}
	return result, nil
}

func float32SliceToVectorString(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%f", f)
	}
	sb.WriteByte(']')
	return sb.String()
}
```

**Note:** The exact implementation depends on how the pgvector Go library works with pgx. Check `github.com/pgvector/pgvector-go` for the recommended approach — it may provide a `pgvector.NewVector()` type that works directly with pgx scanning.

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

- [ ] **Step 4: Commit (if changes were made)**

```bash
git add internal/memory/store.go
git commit -m "feat: add vector similarity search via raw pgx"
```

---

## Task 17: Full Integration Test

**Files:**
- Modify: `internal/memory/engine_test.go`

- [ ] **Step 1: Add integration test for full flow**

```go
// Add to engine_test.go
func TestFormatForPrompt(t *testing.T) {
	now := time.Now()
	result := &RecallResult{
		Profile: &Memory{
			Content: "Diligent worker, prefers written communication.",
		},
		Insights: []Memory{
			{Content: "Reported deadline stress", Importance: 0.8, CreatedAt: now},
			{Content: "Asked about learning opportunities", Importance: 0.6, CreatedAt: now},
		},
		Strategies: []Memory{
			{Content: "Gratitude-style chase improved reply rate from 60% to 90%"},
		},
		TokenCount: 150,
	}

	output := FormatForPrompt(result)

	if !strings.Contains(output, "<memory>") {
		t.Error("expected <memory> tag")
	}
	if !strings.Contains(output, "</memory>") {
		t.Error("expected </memory> tag")
	}
	if !strings.Contains(output, "Employee Profile") {
		t.Error("expected profile section")
	}
	if !strings.Contains(output, "Relevant Memories") {
		t.Error("expected memories section")
	}
	if !strings.Contains(output, "Strategy Insights") {
		t.Error("expected strategy section")
	}
	if !strings.Contains(output, "Gratitude-style") {
		t.Error("expected strategy content")
	}
}

func TestFormatForPrompt_Empty(t *testing.T) {
	output := FormatForPrompt(nil)
	if output != "" {
		t.Errorf("expected empty string for nil result, got %q", output)
	}

	output = FormatForPrompt(&RecallResult{})
	if !strings.Contains(output, "<memory>") {
		t.Error("expected <memory> tag even for empty result")
	}
}
```

- [ ] **Step 2: Run all memory package tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./internal/memory/ -v -count=1`
Expected: All tests PASS.

- [ ] **Step 3: Run full project tests**

Run: `cd /Users/anna/Documents/ai-management-brain && go test ./... -count=1`
Expected: All tests PASS.

- [ ] **Step 4: Verify full build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./cmd/brain/`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/engine_test.go
git commit -m "test: add integration tests for memory prompt formatting"
```

---

## Task 18: Update .env.example and README

**Files:**
- Modify: `.env.example` (if it exists)

- [ ] **Step 1: Add new env vars to .env.example**

```env
# Voyage AI (optional — enables memory features)
VOYAGE_API_KEY=
VOYAGE_MODEL=voyage-3-lite
VOYAGE_BATCH_SIZE=128

# Memory Engine
MEMORY_MAX_RECALL=5
MEMORY_MAX_TOKENS=800
MEMORY_SHORT_TERM_DAYS=30
MEMORY_CONSOLIDATION_THRESHOLD=0.85
MEMORY_MAX_PER_TENANT=20000
```

- [ ] **Step 2: Commit**

```bash
git add .env.example
git commit -m "docs: add memory engine env vars to .env.example"
```

---

## Deployment Checklist

After all tasks are complete:

1. **Server setup**:
   - SSH to `ai-brain` server
   - Install pgvector extension: `sudo apt install postgresql-16-pgvector` (or use Docker image with pgvector)
   - Set `VOYAGE_API_KEY` in production `.env`

2. **Deploy**:
   ```bash
   cd ~/ai-management-brain && git pull
   docker compose -f docker-compose.prod.yml up -d --build
   ```

3. **Verify**:
   - Check logs: `docker compose -f docker-compose.prod.yml logs -f brain`
   - Confirm "memory engine enabled" in logs
   - Test API: `curl localhost/api/v1/memories` (with auth header)

4. **Docker image**: Ensure the PostgreSQL Docker image includes pgvector. Use `pgvector/pgvector:pg16` or add the extension to the existing Postgres Dockerfile.
