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
