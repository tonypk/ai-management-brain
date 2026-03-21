package roles

import (
	"context"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// SummarizerAdapter wraps report.Summarizer to implement SummarizerIface.
type SummarizerAdapter struct {
	S *report.Summarizer
}

func (a *SummarizerAdapter) Generate(ctx context.Context, tenantID, date string, engine *brain.Engine) (*SummaryResult, error) {
	r, err := a.S.Generate(ctx, tenantID, date, engine)
	if err != nil {
		return nil, err
	}
	return &SummaryResult{
		Content:        r.Content,
		SubmissionRate: r.SubmissionRate,
		BlockersCount:  r.BlockersCount,
	}, nil
}

// AlertCheckerAdapter wraps report.AlertChecker to implement AlertCheckerIface.
type AlertCheckerAdapter struct {
	A *report.AlertChecker
}

func (a *AlertCheckerAdapter) CheckAll(ctx context.Context, tenantID string, bossChatID int64) ([]AlertResult, error) {
	alerts, err := a.A.CheckAll(ctx, tenantID, bossChatID)
	if err != nil {
		return nil, err
	}
	result := make([]AlertResult, len(alerts))
	for i, al := range alerts {
		result[i] = AlertResult{
			EmployeeID:   al.EmployeeID,
			EmployeeName: al.EmployeeName,
			AlertType:    al.AlertType,
			Message:      al.Message,
			Severity:     al.Severity,
		}
	}
	return result, nil
}

// ReportDBAdapter wraps report.DBAdapter to implement ReportDBIface.
type ReportDBAdapter struct {
	DB *report.DBAdapter
}

func (a *ReportDBAdapter) GetTenantIDByBossChatID(ctx context.Context, bossChatID int64) (string, error) {
	return a.DB.GetTenantIDByBossChatID(ctx, bossChatID)
}
