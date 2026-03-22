package roles

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
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

func (a *AlertCheckerAdapter) CheckAll(ctx context.Context, tenantID string, bossChannelType string, bossChannelID string) ([]AlertResult, error) {
	// Convert channel info to EmployeeInfo for the boss
	bossInfo := toBossEmployeeInfo(bossChannelType, bossChannelID)

	alerts, err := a.A.CheckAll(ctx, tenantID, bossInfo)
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

// ActionExecAdapter wraps report.ActionExecutor to implement ActionExecIface.
type ActionExecAdapter struct {
	A *report.ActionExecutor
}

func (a *ActionExecAdapter) RunWeekly(ctx context.Context, tenantID, mentorID string, bossChannelType string, bossChannelID string) error {
	bossInfo := toBossEmployeeInfo(bossChannelType, bossChannelID)
	return a.A.RunWeekly(ctx, tenantID, mentorID, bossInfo)
}

// ReportDBAdapter wraps report.DBAdapter to implement ReportDBIface.
type ReportDBAdapter struct {
	DB *report.DBAdapter
}

func (a *ReportDBAdapter) GetTenantIDByBossChatID(ctx context.Context, bossChatID int64) (string, error) {
	return a.DB.GetTenantIDByBossChatID(ctx, bossChatID)
}

// toBossEmployeeInfo creates a minimal EmployeeInfo for the boss from channel info.
// This is used by adapters to bridge between the roles package (which uses string channel types)
// and the report package (which uses EmployeeInfo).
func toBossEmployeeInfo(channelType string, channelID string) report.EmployeeInfo {
	info := report.EmployeeInfo{
		ID:               "boss",
		Name:             "Boss",
		PreferredChannel: channelType,
	}

	switch channel.Type(channelType) {
	case channel.TypeTelegram:
		chatID, err := strconv.ParseInt(channelID, 10, 64)
		if err != nil {
			slog.Warn("invalid telegram chat ID", "channelID", channelID, "error", err)
		}
		info.TelegramID = chatID
	case channel.TypeSignal:
		info.SignalPhone = channelID
	case channel.TypeSlack:
		info.SlackID = channelID
	case channel.TypeLark:
		info.LarkID = channelID
	}

	return info
}
