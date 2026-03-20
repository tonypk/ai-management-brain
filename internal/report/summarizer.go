package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// ReportRow holds one report row from the database.
type ReportRow struct {
	EmployeeName string
	Answers      string // JSON string
}

// SummaryEntry holds data for inserting into the summaries table.
type SummaryEntry struct {
	TenantID       string
	SummaryDate    string
	Content        string
	SubmissionRate float64
	BlockersCount  int
	KeyMetrics     string // JSON
}

// SummaryResult is the output of summary generation.
type SummaryResult struct {
	Content        string
	SubmissionRate float64
	BlockersCount  int
}

// SummarizerDB defines the database operations needed by the summarizer.
type SummarizerDB interface {
	GetReportsByTenantDate(ctx context.Context, tenantID, date string) ([]ReportRow, error)
	CountActiveEmployees(ctx context.Context, tenantID string) (int64, error)
	CreateSummary(ctx context.Context, entry SummaryEntry) error
}

// Summarizer generates team summaries from daily reports.
type Summarizer struct {
	db     SummarizerDB
	llm    *brain.LLMService
	engine *brain.Engine
}

// NewSummarizer creates a new summarizer.
func NewSummarizer(db SummarizerDB, llm *brain.LLMService, engine *brain.Engine) *Summarizer {
	return &Summarizer{db: db, llm: llm, engine: engine}
}

// Generate creates a summary for the given tenant and date.
func (s *Summarizer) Generate(ctx context.Context, tenantID, date string) (*SummaryResult, error) {
	// Get reports
	reports, err := s.db.GetReportsByTenantDate(ctx, tenantID, date)
	if err != nil {
		return nil, fmt.Errorf("get reports: %w", err)
	}

	// Count active employees
	activeCount, err := s.db.CountActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("count employees: %w", err)
	}

	// Calculate submission rate
	var submissionRate float64
	if activeCount > 0 {
		submissionRate = float64(len(reports)) / float64(activeCount)
	}

	// Convert reports to ReportData for LLM
	var reportData []brain.ReportData
	for _, r := range reports {
		var answers map[string]string
		json.Unmarshal([]byte(r.Answers), &answers) //nolint:errcheck
		reportData = append(reportData, brain.ReportData{
			EmployeeName: r.EmployeeName,
			Answers:      answers,
		})
	}

	// Build system prompt with mentor's summary config
	summaryConfig := s.engine.GetSummaryConfig()
	systemPrompt := s.engine.BuildSystemPrompt()
	systemPrompt += fmt.Sprintf("\n\nSummary Focus: %s\nHighlight: %s\nFlag: %s",
		strings.Join(summaryConfig.Focus, ", "),
		summaryConfig.Highlight,
		summaryConfig.Flag,
	)

	// Generate via LLM
	content, err := s.llm.GenerateSummary(ctx, systemPrompt, reportData)
	if err != nil {
		// Fallback: bullet-point summary
		slog.Warn("LLM failed for summary, using fallback", "error", err)
		content = s.buildFallbackSummary(reports, len(reports), int(activeCount))
	}

	result := &SummaryResult{
		Content:        content,
		SubmissionRate: submissionRate,
	}

	// Save to DB
	entry := SummaryEntry{
		TenantID:       tenantID,
		SummaryDate:    date,
		Content:        content,
		SubmissionRate: submissionRate,
	}
	if err := s.db.CreateSummary(ctx, entry); err != nil {
		slog.Error("save summary", "error", err)
	}

	return result, nil
}

func (s *Summarizer) buildFallbackSummary(reports []ReportRow, submitted, total int) string {
	var sb strings.Builder
	pct := 0
	if total > 0 {
		pct = submitted * 100 / total
	}
	sb.WriteString(fmt.Sprintf("Daily Report Summary (AI unavailable)\nSubmitted: %d/%d (%d%%)\n\n", submitted, total, pct))

	for _, r := range reports {
		var answers map[string]string
		json.Unmarshal([]byte(r.Answers), &answers) //nolint:errcheck
		sb.WriteString(fmt.Sprintf("- %s:", r.EmployeeName))
		for _, v := range answers {
			sb.WriteString(fmt.Sprintf(" %s", v))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
