package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// DBAdapter bridges sqlc.Queries to report interfaces (ChaserDB, SummarizerDB, etc).
type DBAdapter struct {
	q *sqlc.Queries
}

// NewDBAdapter creates a new report DB adapter.
func NewDBAdapter(q *sqlc.Queries) *DBAdapter {
	return &DBAdapter{q: q}
}

// --- ChaserDB ---

func (a *DBAdapter) ListEmployeesWithoutReport(ctx context.Context, tenantID, date string) ([]EmployeeInfo, error) {
	uid, err := rParseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	d, err := rParseDate(date)
	if err != nil {
		return nil, err
	}
	emps, err := a.q.ListEmployeesWithoutReport(ctx, sqlc.ListEmployeesWithoutReportParams{
		TenantID:   uid,
		ReportDate: d,
	})
	if err != nil {
		return nil, err
	}
	result := make([]EmployeeInfo, 0, len(emps))
	for _, e := range emps {
		if !e.TelegramID.Valid {
			continue
		}
		result = append(result, EmployeeInfo{
			ID:          rFormatUUID(e.ID),
			Name:        e.Name,
			TelegramID:  e.TelegramID.Int64,
			CultureCode: e.CultureCode,
		})
	}
	return result, nil
}

func (a *DBAdapter) GetLastChaseStep(ctx context.Context, employeeID, date string) (int, error) {
	uid, err := rParseUUID(employeeID)
	if err != nil {
		return 0, err
	}
	d, err := rParseDate(date)
	if err != nil {
		return 0, err
	}
	result, err := a.q.GetLastChaseStep(ctx, sqlc.GetLastChaseStepParams{
		EmployeeID: uid,
		ReportDate: d,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	switch v := result.(type) {
	case int64:
		return int(v), nil
	case int32:
		return int(v), nil
	default:
		return 0, nil
	}
}

func (a *DBAdapter) CreateChaseLog(ctx context.Context, entry ChaseLogEntry) error {
	tid, err := rParseUUID(entry.TenantID)
	if err != nil {
		return err
	}
	eid, err := rParseUUID(entry.EmployeeID)
	if err != nil {
		return err
	}
	d, err := rParseDate(entry.ReportDate)
	if err != nil {
		return err
	}
	_, err = a.q.CreateChaseLog(ctx, sqlc.CreateChaseLogParams{
		TenantID:   tid,
		EmployeeID: eid,
		ReportDate: d,
		Step:       int32(entry.Step),
		Action:     entry.Action,
		Message:    entry.Message,
	})
	return err
}

// --- SummarizerDB ---

func (a *DBAdapter) GetReportsByTenantDate(ctx context.Context, tenantID, date string) ([]ReportRow, error) {
	uid, err := rParseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	d, err := rParseDate(date)
	if err != nil {
		return nil, err
	}
	rows, err := a.q.GetReportsByTenantDate(ctx, sqlc.GetReportsByTenantDateParams{
		TenantID:   uid,
		ReportDate: d,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ReportRow, len(rows))
	for i, r := range rows {
		result[i] = ReportRow{
			EmployeeName: r.EmployeeName,
			Answers:      string(r.Answers),
		}
	}
	return result, nil
}

func (a *DBAdapter) CountActiveEmployees(ctx context.Context, tenantID string) (int64, error) {
	uid, err := rParseUUID(tenantID)
	if err != nil {
		return 0, err
	}
	emps, err := a.q.ListActiveEmployees(ctx, uid)
	if err != nil {
		return 0, err
	}
	return int64(len(emps)), nil
}

func (a *DBAdapter) CreateSummary(ctx context.Context, entry SummaryEntry) error {
	uid, err := rParseUUID(entry.TenantID)
	if err != nil {
		return err
	}
	d, err := rParseDate(entry.SummaryDate)
	if err != nil {
		return err
	}
	km := []byte("{}")
	if entry.KeyMetrics != "" {
		km = []byte(entry.KeyMetrics)
	}
	_, err = a.q.CreateSummary(ctx, sqlc.CreateSummaryParams{
		TenantID:       uid,
		SummaryDate:    d,
		Content:        entry.Content,
		SubmissionRate: entry.SubmissionRate,
		BlockersCount:  int32(entry.BlockersCount),
		KeyMetrics:     km,
	})
	return err
}

// --- Report creation (used by conversation flow) ---

// CreateReport saves a completed report to the database.
func (a *DBAdapter) CreateReport(ctx context.Context, tenantID, employeeID, date string, answers map[string]string) error {
	tid, err := rParseUUID(tenantID)
	if err != nil {
		return err
	}
	eid, err := rParseUUID(employeeID)
	if err != nil {
		return err
	}
	d, err := rParseDate(date)
	if err != nil {
		return err
	}
	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	_, err = a.q.CreateReport(ctx, sqlc.CreateReportParams{
		TenantID:   tid,
		EmployeeID: eid,
		ReportDate: d,
		Answers:    answersJSON,
	})
	return err
}

// --- Employee queries (used by remind job) ---

// ListActiveEmployeesWithTelegram returns active employees who have Telegram linked.
func (a *DBAdapter) ListActiveEmployeesWithTelegram(ctx context.Context, tenantID string) ([]EmployeeInfo, error) {
	uid, err := rParseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	emps, err := a.q.ListActiveEmployees(ctx, uid)
	if err != nil {
		return nil, err
	}
	var result []EmployeeInfo
	for _, e := range emps {
		if e.TelegramID.Valid {
			result = append(result, EmployeeInfo{
				ID:          rFormatUUID(e.ID),
				Name:        e.Name,
				TelegramID:  e.TelegramID.Int64,
				CultureCode: e.CultureCode,
			})
		}
	}
	return result, nil
}

// --- TriggerDB ---

// GetMissedDaysLast7 returns the number of missed report days in the last 7 days.
func (a *DBAdapter) GetMissedDaysLast7(ctx context.Context, employeeID string) (int, error) {
	uid, err := rParseUUID(employeeID)
	if err != nil {
		return 0, err
	}
	missed, err := a.q.GetEmployeeReportStreak(ctx, uid)
	if err != nil {
		return 0, err
	}
	return int(missed), nil
}

// GetSubmittedDaysLast7 returns the number of submitted report days in the last 7 days.
func (a *DBAdapter) GetSubmittedDaysLast7(ctx context.Context, employeeID string) (int, error) {
	uid, err := rParseUUID(employeeID)
	if err != nil {
		return 0, err
	}
	count, err := a.q.GetSubmittedDaysLast7(ctx, uid)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetTenantIDByBossChatID returns the tenant UUID for the given boss chat ID.
func (a *DBAdapter) GetTenantIDByBossChatID(ctx context.Context, bossChatID int64) (string, error) {
	t, err := a.q.GetTenantByBossChatID(ctx, bossChatID)
	if err != nil {
		return "", err
	}
	return rFormatUUID(t.ID), nil
}

// --- AnalyzerDB ---

// GetLatestReportByEmployee returns the report ID and answers JSON for analysis.
func (a *DBAdapter) GetLatestReportByEmployee(ctx context.Context, employeeID, date string) (string, string, error) {
	eid, err := rParseUUID(employeeID)
	if err != nil {
		return "", "", err
	}
	d, err := rParseDate(date)
	if err != nil {
		return "", "", err
	}
	r, err := a.q.GetLatestReportByEmployee(ctx, sqlc.GetLatestReportByEmployeeParams{
		EmployeeID: eid,
		ReportDate: d,
	})
	if err != nil {
		return "", "", err
	}
	return rFormatUUID(r.ID), string(r.Answers), nil
}

// UpdateReportAnalysis saves extracted blockers and sentiment to the report.
func (a *DBAdapter) UpdateReportAnalysis(ctx context.Context, reportID, blockers, sentiment string) error {
	rid, err := rParseUUID(reportID)
	if err != nil {
		return err
	}
	return a.q.UpdateReportAnalysis(ctx, sqlc.UpdateReportAnalysisParams{
		ID:        rid,
		Blockers:  pgtext(blockers),
		Sentiment: pgtext(sentiment),
	})
}

// --- Profile queries ---

// SubmissionHistoryRow represents one day's submission status.
type SubmissionHistoryRow struct {
	ReportDate string
	Sentiment  string
}

// GetEmployeeSubmissionHistory returns the last 30 days of submissions.
func (a *DBAdapter) GetEmployeeSubmissionHistory(ctx context.Context, employeeID string) ([]SubmissionHistoryRow, error) {
	uid, err := rParseUUID(employeeID)
	if err != nil {
		return nil, err
	}
	rows, err := a.q.GetEmployeeSubmissionHistory(ctx, uid)
	if err != nil {
		return nil, err
	}
	result := make([]SubmissionHistoryRow, len(rows))
	for i, r := range rows {
		date := ""
		if r.ReportDate.Valid {
			date = r.ReportDate.Time.Format("2006-01-02")
		}
		sentiment := ""
		if r.Sentiment.Valid {
			sentiment = r.Sentiment.String
		}
		result[i] = SubmissionHistoryRow{ReportDate: date, Sentiment: sentiment}
	}
	return result, nil
}

// pgtext creates a pgtype.Text from a string (valid if non-empty).
func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// --- helpers (prefixed with r to avoid collision with bot/adapter.go) ---

func rFormatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func rParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}

func rParseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("parse date %q: %w", s, err)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
