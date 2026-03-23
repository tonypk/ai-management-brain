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

// extractedInsight is the JSON structure Claude returns.
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

func (e *Extractor) FromChat(ctx context.Context, input ChatInput) ([]Memory, error) {
	return e.extract(ctx, input.TenantID, input.EmployeeID, SourceConversation, "", input.Transcript)
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
