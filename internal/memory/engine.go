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

// ExtractFromChat extracts and stores memories from a completed chat conversation.
func (e *MemoryEngine) ExtractFromChat(ctx context.Context, input ChatInput) error {
	if !e.Enabled() || e.extractor == nil {
		return nil
	}

	memories, err := e.extractor.FromChat(ctx, input)
	if err != nil {
		return fmt.Errorf("extract from chat: %w", err)
	}

	for _, m := range memories {
		if _, err := e.store.Create(ctx, m); err != nil {
			slog.Error("store memory failed", "source", "chat", "error", err)
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
