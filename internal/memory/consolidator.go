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
	assigned := make([]int, n)
	for i := range assigned {
		assigned[i] = -1
	}

	clusterID := 0
	for i := 0; i < n; i++ {
		if assigned[i] != -1 {
			continue
		}
		assigned[i] = clusterID
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
