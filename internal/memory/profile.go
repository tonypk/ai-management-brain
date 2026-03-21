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

	var sb strings.Builder
	for i, m := range memories {
		fmt.Fprintf(&sb, "%d. %s (importance: %.1f)\n", i+1, m.Content, m.Importance)
	}

	profileContent, err := b.llm.Chat(ctx, profileSystemPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("generate profile: %w", err)
	}

	embedding, err := b.embedder.Embed(ctx, profileContent)
	if err != nil {
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
		Importance: 1.0,
	}

	created, err := b.store.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("save profile: %w", err)
	}

	return &created, nil
}

func (b *ProfileBuilder) Refresh(ctx context.Context, tenantID, employeeID string) (*Memory, error) {
	newProfile, err := b.Build(ctx, tenantID, employeeID)
	if err != nil {
		return nil, err
	}

	existing, err := b.store.GetProfile(ctx, tenantID, employeeID)
	if err == nil && existing != nil && existing.ID != newProfile.ID {
		_ = b.store.MarkMerged(ctx, existing.ID, newProfile.ID)
	}

	return newProfile, nil
}
