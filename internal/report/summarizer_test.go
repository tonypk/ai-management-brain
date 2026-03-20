package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

type mockSummarizerDB struct {
	reports     []report.ReportRow
	activeCount int64
	createdSumm *report.SummaryEntry
}

func (m *mockSummarizerDB) GetReportsByTenantDate(ctx context.Context, tenantID, date string) ([]report.ReportRow, error) {
	return m.reports, nil
}

func (m *mockSummarizerDB) CountActiveEmployees(ctx context.Context, tenantID string) (int64, error) {
	return m.activeCount, nil
}

func (m *mockSummarizerDB) CreateSummary(ctx context.Context, entry report.SummaryEntry) error {
	m.createdSumm = &entry
	return nil
}

func TestSummarizer_GenerateWithReports(t *testing.T) {
	db := &mockSummarizerDB{
		reports: []report.ReportRow{
			{EmployeeName: "Alice", Answers: `{"q1":"did X","q2":"no blockers","q3":"learned Y"}`},
			{EmployeeName: "Bob", Answers: `{"q1":"did Z","q2":"blocked on API","q3":"learned Go"}`},
		},
		activeCount: 5,
	}
	llm := &mockLLM{response: "## Summary\nTeam is making progress. Alice and Bob submitted. Blocker: API dependency."}
	engine, _ := brain.NewEngine("inamori", "philippines")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.SubmissionRate != 0.4 {
		t.Errorf("submission rate = %f, want 0.4", result.SubmissionRate)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if db.createdSumm == nil {
		t.Error("summary should be saved to DB")
	}
}

func TestSummarizer_PartialData(t *testing.T) {
	db := &mockSummarizerDB{
		reports:     []report.ReportRow{},
		activeCount: 10,
	}
	llm := &mockLLM{response: "No reports submitted today."}
	engine, _ := brain.NewEngine("inamori", "default")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.SubmissionRate != 0 {
		t.Errorf("rate should be 0 with no reports, got %f", result.SubmissionRate)
	}
}

func TestSummarizer_LLMFallback(t *testing.T) {
	db := &mockSummarizerDB{
		reports: []report.ReportRow{
			{EmployeeName: "Alice", Answers: `{"q1":"did X"}`},
		},
		activeCount: 3,
	}
	llm := &mockLLM{err: errors.New("api down")}
	engine, _ := brain.NewEngine("inamori", "default")

	summarizer := report.NewSummarizer(db, brain.NewLLMService(llm), engine)
	result, err := summarizer.Generate(context.Background(), "tenant-1", "2026-03-20")
	if err != nil {
		t.Fatalf("should not error with fallback: %v", err)
	}
	if result.Content == "" {
		t.Error("fallback should produce non-empty content")
	}
}
