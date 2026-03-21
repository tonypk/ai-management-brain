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
