package report_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

type mockAnalyzerDB struct {
	reportID   string
	answersJSON string
	getErr     error

	updatedID        string
	updatedBlockers  string
	updatedSentiment string
	updateErr        error
}

func (m *mockAnalyzerDB) GetLatestReportByEmployee(_ context.Context, _, _ string) (string, string, error) {
	return m.reportID, m.answersJSON, m.getErr
}

func (m *mockAnalyzerDB) UpdateReportAnalysis(_ context.Context, reportID, blockers, sentiment string) error {
	m.updatedID = reportID
	m.updatedBlockers = blockers
	m.updatedSentiment = sentiment
	return m.updateErr
}

func TestAnalyzer_ExtractsBlockersAndSentiment(t *testing.T) {
	db := &mockAnalyzerDB{
		reportID:    "report-1",
		answersJSON: `{"q1":"Finished API endpoint","q2":"Waiting on DB migration from DevOps","q3":"Learned about indexes"}`,
	}
	llm := &mockLLM{
		response: `{"blockers": "Waiting on DB migration from DevOps", "sentiment": "neutral"}`,
	}
	llmService := brain.NewLLMService(llm)
	analyzer := report.NewAnalyzer(db, llmService)

	err := analyzer.Analyze(context.Background(), "emp-1", "2026-03-20")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if db.updatedID != "report-1" {
		t.Errorf("expected report ID 'report-1', got %q", db.updatedID)
	}
	if db.updatedBlockers != "Waiting on DB migration from DevOps" {
		t.Errorf("expected blockers, got %q", db.updatedBlockers)
	}
	if db.updatedSentiment != "neutral" {
		t.Errorf("expected 'neutral' sentiment, got %q", db.updatedSentiment)
	}
}

func TestAnalyzer_PositiveSentiment(t *testing.T) {
	db := &mockAnalyzerDB{
		reportID:    "report-2",
		answersJSON: `{"q1":"Great progress today!","q2":"No blockers","q3":"Excited about the new feature"}`,
	}
	llm := &mockLLM{
		response: `{"blockers": "", "sentiment": "positive"}`,
	}
	llmService := brain.NewLLMService(llm)
	analyzer := report.NewAnalyzer(db, llmService)

	err := analyzer.Analyze(context.Background(), "emp-2", "2026-03-20")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if db.updatedBlockers != "" {
		t.Errorf("expected no blockers, got %q", db.updatedBlockers)
	}
	if db.updatedSentiment != "positive" {
		t.Errorf("expected 'positive' sentiment, got %q", db.updatedSentiment)
	}
}

func TestAnalyzer_NilLLM_Skips(t *testing.T) {
	db := &mockAnalyzerDB{
		reportID:    "report-3",
		answersJSON: `{"q1":"test"}`,
	}
	analyzer := report.NewAnalyzer(db, nil)

	err := analyzer.Analyze(context.Background(), "emp-3", "2026-03-20")
	if err != nil {
		t.Fatalf("Analyze should not fail with nil LLM: %v", err)
	}

	if db.updatedID != "" {
		t.Error("should not update DB when LLM is nil")
	}
}

func TestAnalyzer_MalformedJSON_FallsBack(t *testing.T) {
	db := &mockAnalyzerDB{
		reportID:    "report-4",
		answersJSON: `{"q1":"some work done"}`,
	}
	llm := &mockLLM{
		response: "not valid json response",
	}
	llmService := brain.NewLLMService(llm)
	analyzer := report.NewAnalyzer(db, llmService)

	err := analyzer.Analyze(context.Background(), "emp-4", "2026-03-20")
	if err != nil {
		t.Fatalf("Analyze should not fail on malformed JSON: %v", err)
	}

	// Should fallback to neutral sentiment
	if db.updatedSentiment != "neutral" {
		t.Errorf("expected fallback 'neutral' sentiment, got %q", db.updatedSentiment)
	}
}
