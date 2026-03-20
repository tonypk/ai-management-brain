package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// AnalyzerDB defines DB operations for post-submission analysis.
type AnalyzerDB interface {
	GetLatestReportByEmployee(ctx context.Context, employeeID, date string) (string, string, error) // returns reportID, answersJSON
	UpdateReportAnalysis(ctx context.Context, reportID, blockers, sentiment string) error
}

// AnalysisResult holds the extracted blockers and sentiment.
type AnalysisResult struct {
	Blockers  string `json:"blockers"`
	Sentiment string `json:"sentiment"`
}

// Analyzer extracts blockers and sentiment from submitted reports.
type Analyzer struct {
	db  AnalyzerDB
	llm *brain.LLMService
}

// NewAnalyzer creates a new report analyzer.
func NewAnalyzer(db AnalyzerDB, llm *brain.LLMService) *Analyzer {
	return &Analyzer{db: db, llm: llm}
}

// Analyze extracts blockers and sentiment from a report and updates the DB.
func (a *Analyzer) Analyze(ctx context.Context, employeeID, date string) error {
	if a.llm == nil {
		return nil // AI disabled, skip analysis
	}

	reportID, answersJSON, err := a.db.GetLatestReportByEmployee(ctx, employeeID, date)
	if err != nil {
		return fmt.Errorf("get report: %w", err)
	}

	// Parse answers for analysis
	var answers map[string]string
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return fmt.Errorf("parse answers: %w", err)
	}

	// Build combined text from all answers
	var sb strings.Builder
	for k, v := range answers {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	answerText := sb.String()

	result, err := a.llm.AnalyzeReport(ctx, answerText)
	if err != nil {
		slog.Error("LLM analysis failed", "employee_id", employeeID, "error", err)
		return nil // Don't fail the flow, just log
	}

	if err := a.db.UpdateReportAnalysis(ctx, reportID, result.Blockers, result.Sentiment); err != nil {
		return fmt.Errorf("update analysis: %w", err)
	}

	slog.Info("report analyzed",
		"employee_id", employeeID,
		"sentiment", result.Sentiment,
		"has_blockers", result.Blockers != "",
	)
	return nil
}
