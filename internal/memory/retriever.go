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

	// 1. Try to get the profile
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

	// 3. Search for similar memories
	maxSearch := r.maxResults
	if maxSearch < 10 {
		maxSearch = 10
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

		total := len(result.Insights) + len(result.Strategies) + len(result.Knowledge)
		if total >= r.maxResults-1 {
			break
		}
	}

	// 5. Estimate token count
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

	// 6. Trim if over token budget
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
