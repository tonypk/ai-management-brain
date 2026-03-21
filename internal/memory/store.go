package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// MemoryStore handles database operations for memories.
type MemoryStore struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// NewMemoryStore creates a new MemoryStore with the given sqlc queries and connection pool.
func NewMemoryStore(q *sqlc.Queries, pool *pgxpool.Pool) *MemoryStore {
	return &MemoryStore{q: q, pool: pool}
}

// --- pgtype conversion helpers ---

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

func optionalUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	u, _ := parseUUID(s)
	return u
}

func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgfloat8(f float64) pgtype.Float8 {
	return pgtype.Float8{Float64: f, Valid: true}
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

func toVector(embedding []float32) pgvector.Vector {
	if len(embedding) == 0 {
		return pgvector.Vector{}
	}
	return pgvector.NewVector(embedding)
}

// --- Conversion: sqlc row → Memory struct ---

func rowToMemory(row sqlc.Memory) Memory {
	var meta map[string]any
	if row.Metadata != nil {
		_ = json.Unmarshal(row.Metadata, &meta)
	}

	var importance float64
	if row.Importance.Valid {
		importance = row.Importance.Float64
	}

	var accessCount int
	if row.AccessCount.Valid {
		accessCount = int(row.AccessCount.Int32)
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
		Importance:  importance,
		AccessCount: accessCount,
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

	if row.Embedding.Slice() != nil {
		m.Embedding = row.Embedding.Slice()
	}

	return m
}

// --- Public methods ---

// Create inserts a new memory record and returns the created Memory.
func (s *MemoryStore) Create(ctx context.Context, m Memory) (Memory, error) {
	tenantID, err := parseUUID(m.TenantID)
	if err != nil {
		return Memory{}, fmt.Errorf("tenant_id: %w", err)
	}

	metaJSON, _ := json.Marshal(m.Metadata)
	if m.Metadata == nil {
		metaJSON = []byte("{}")
	}

	row, err := s.q.CreateMemory(ctx, sqlc.CreateMemoryParams{
		TenantID:   tenantID,
		MemoryType: m.MemoryType,
		MemoryTier: m.MemoryTier,
		EmployeeID: optionalUUID(m.EmployeeID),
		SourceType: pgtext(m.SourceType),
		SourceID:   optionalUUID(m.SourceID),
		Content:    m.Content,
		Summary:    pgtext(m.Summary),
		Embedding:  toVector(m.Embedding),
		Importance: pgfloat8(m.Importance),
		Metadata:   metaJSON,
		ExpiresAt:  pgtimestamp(m.ExpiresAt),
	})
	if err != nil {
		return Memory{}, fmt.Errorf("create memory: %w", err)
	}

	return rowToMemory(row), nil
}

// Get retrieves a single memory by ID and tenant.
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

// List retrieves memories for a tenant with optional filters and pagination.
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

// Count returns the count of non-merged memories for a tenant.
func (s *MemoryStore) Count(ctx context.Context, tenantID string) (int64, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return 0, err
	}
	return s.q.CountMemoriesByTenant(ctx, tid)
}

// Delete removes a memory by ID and tenant.
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

// DeleteExpired removes all expired memories and returns the count deleted.
func (s *MemoryStore) DeleteExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredMemories(ctx)
}

// MarkMerged marks a memory as merged into another memory.
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

// ListShortTermByEmployee retrieves short-term memories for a specific employee.
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

// ListLongTermByEmployee retrieves long-term memories for a specific employee.
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

// GetProfile retrieves the employee profile memory (most recent profile tier entry).
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

// IncrementAccess increments the access count for a memory.
func (s *MemoryStore) IncrementAccess(ctx context.Context, id string) error {
	mid, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.q.IncrementAccessCount(ctx, mid)
}

// BackfillEmbedding sets the embedding for a memory that currently has none.
func (s *MemoryStore) BackfillEmbedding(ctx context.Context, id string, embedding []float32) error {
	mid, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.q.BackfillEmbedding(ctx, sqlc.BackfillEmbeddingParams{
		ID:        mid,
		Embedding: pgvector.NewVector(embedding),
	})
}

// SearchSimilar performs a raw pgx vector similarity search using cosine distance (<=>).
// sqlc does not support pgvector operators, so this uses the pool directly.
func (s *MemoryStore) SearchSimilar(ctx context.Context, tenantID string, embedding []float32, employeeFilter string, maxResults int) ([]Memory, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, tenant_id, memory_type, memory_tier, employee_id, source_type, source_id,
		       content, summary, embedding, importance, access_count, metadata, expires_at,
		       merged_into, created_at, updated_at,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM memories
		WHERE tenant_id = $2
		  AND embedding IS NOT NULL
		  AND merged_into IS NULL
	`
	args := []any{pgvector.NewVector(embedding), tid}
	argIdx := 3

	if employeeFilter != "" {
		eid, err := parseUUID(employeeFilter)
		if err != nil {
			return nil, err
		}
		query += fmt.Sprintf("  AND employee_id = $%d\n", argIdx)
		args = append(args, eid)
		argIdx++
	}

	query += fmt.Sprintf("ORDER BY embedding <=> $1::vector\nLIMIT $%d", argIdx)
	args = append(args, maxResults)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search similar: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var row sqlc.Memory
		var similarity float64
		if err := rows.Scan(
			&row.ID, &row.TenantID, &row.MemoryType, &row.MemoryTier,
			&row.EmployeeID, &row.SourceType, &row.SourceID,
			&row.Content, &row.Summary, &row.Embedding, &row.Importance,
			&row.AccessCount, &row.Metadata, &row.ExpiresAt,
			&row.MergedInto, &row.CreatedAt, &row.UpdatedAt,
			&similarity,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		m := rowToMemory(row)
		m.Similarity = similarity
		memories = append(memories, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return memories, nil
}

// ListTenantsWithMemories returns distinct tenant IDs that have memories.
func (s *MemoryStore) ListTenantsWithMemories(ctx context.Context) ([]string, error) {
	uuids, err := s.q.ListTenantsWithMemories(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(uuids))
	for i, u := range uuids {
		ids[i] = formatUUID(u)
	}
	return ids, nil
}

// ListEmployeesWithShortTermMemories returns employee IDs with short-term memories for a tenant.
func (s *MemoryStore) ListEmployeesWithShortTermMemories(ctx context.Context, tenantID string) ([]string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	uuids, err := s.q.ListEmployeesWithShortTermMemories(ctx, tid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(uuids))
	for i, u := range uuids {
		ids[i] = formatUUID(u)
	}
	return ids, nil
}

// ListEmployeesWithLongTermMemories returns employee IDs with long-term memories for a tenant.
func (s *MemoryStore) ListEmployeesWithLongTermMemories(ctx context.Context, tenantID string) ([]string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	uuids, err := s.q.ListEmployeesWithLongTermMemories(ctx, tid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(uuids))
	for i, u := range uuids {
		ids[i] = formatUUID(u)
	}
	return ids, nil
}
