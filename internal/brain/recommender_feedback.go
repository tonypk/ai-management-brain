package brain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

// RecommendationFeedback records recommendation outcomes as strategy_result memories.
type RecommendationFeedback struct {
	memStore *memory.MemoryStore
}

// NewRecommendationFeedback creates a new RecommendationFeedback.
func NewRecommendationFeedback(memStore *memory.MemoryStore) *RecommendationFeedback {
	return &RecommendationFeedback{memStore: memStore}
}

// RecordFeedback creates a strategy_result memory from a recommendation outcome.
func (f *RecommendationFeedback) RecordFeedback(ctx context.Context, tenantID, title, action, outcome string) {
	if f.memStore == nil {
		return
	}

	content := fmt.Sprintf("Recommendation '%s' was %s. Action: %s", title, outcome, action)
	summary := fmt.Sprintf("%s: %s", outcome, title)

	importance := float64(0.6)
	if outcome == "executed" {
		importance = 0.7
	}

	_, err := f.memStore.Create(ctx, memory.Memory{
		TenantID:   tenantID,
		MemoryType: memory.TypeStrategyResult,
		MemoryTier: memory.TierShortTerm,
		Content:    content,
		Summary:    summary,
		Importance: importance,
	})
	if err != nil {
		slog.Error("recommendation_feedback: create memory failed", "error", err, "title", title)
	}
}
