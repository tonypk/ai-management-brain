package report

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

// AlertDB defines the database queries needed by the alert agent.
type AlertDB interface {
	ListActiveEmployees(ctx context.Context, tenantID string) ([]EmployeeInfo, error)
	GetConsecutiveMissDays(ctx context.Context, employeeID string) (int, error)
	GetRecentSentiments(ctx context.Context, employeeID string, days int) ([]string, error)
}

// Alert represents a detected anomaly.
type Alert struct {
	EmployeeID   string
	EmployeeName string
	AlertType    string // "consecutive_miss", "sentiment_drop", "blocker_surge"
	Message      string
	Severity     string // "warning", "critical"
}

// AlertChecker detects employee anomalies and alerts the boss.
type AlertChecker struct {
	db     AlertDB
	sender channel.Sender
}

// NewAlertChecker creates a new alert checker.
func NewAlertChecker(db AlertDB, sender channel.Sender) *AlertChecker {
	return &AlertChecker{db: db, sender: sender}
}

// CheckAll scans all employees for anomalies and returns triggered alerts.
func (a *AlertChecker) CheckAll(ctx context.Context, tenantID string, bossInfo EmployeeInfo) ([]Alert, error) {
	employees, err := a.db.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	var alerts []Alert
	for _, emp := range employees {
		empAlerts := a.checkEmployee(ctx, emp)
		alerts = append(alerts, empAlerts...)
	}

	// Send alerts to boss
	if len(alerts) > 0 {
		chType, chID := resolveEmployeeChannel(bossInfo)
		if chType == "" {
			slog.Error("boss has no channel configured, cannot send alerts")
		} else {
			msg := formatAlerts(alerts)
			if err := a.sender.Send(ctx, chType, chID, msg); err != nil {
				slog.Error("send alerts to boss", "error", err)
			}
		}
	}

	return alerts, nil
}

func (a *AlertChecker) checkEmployee(ctx context.Context, emp EmployeeInfo) []Alert {
	var alerts []Alert

	// Check consecutive misses
	missedDays, err := a.db.GetConsecutiveMissDays(ctx, emp.ID)
	if err != nil {
		slog.Error("check consecutive miss", "employee_id", emp.ID, "error", err)
	} else if missedDays >= 3 {
		severity := "warning"
		if missedDays >= 5 {
			severity = "critical"
		}
		alerts = append(alerts, Alert{
			EmployeeID:   emp.ID,
			EmployeeName: emp.Name,
			AlertType:    "consecutive_miss",
			Message:      fmt.Sprintf("%s has missed %d consecutive days of check-in", emp.Name, missedDays),
			Severity:     severity,
		})
	}

	// Check sentiment drop
	sentiments, err := a.db.GetRecentSentiments(ctx, emp.ID, 5)
	if err != nil {
		slog.Error("check sentiment", "employee_id", emp.ID, "error", err)
	} else if hasSentimentDrop(sentiments) {
		alerts = append(alerts, Alert{
			EmployeeID:   emp.ID,
			EmployeeName: emp.Name,
			AlertType:    "sentiment_drop",
			Message:      fmt.Sprintf("%s shows persistent negative sentiment (3+ consecutive negative reports)", emp.Name),
			Severity:     "warning",
		})
	}

	return alerts
}

// hasSentimentDrop checks if the last 3+ sentiments are negative.
func hasSentimentDrop(sentiments []string) bool {
	if len(sentiments) < 3 {
		return false
	}
	negativeCount := 0
	for _, s := range sentiments {
		if s == "negative" {
			negativeCount++
		} else {
			negativeCount = 0
		}
		if negativeCount >= 3 {
			return true
		}
	}
	return false
}

func formatAlerts(alerts []Alert) string {
	today := time.Now().Format("2006-01-02")
	msg := fmt.Sprintf("Alert Report (%s)\n%d anomalies detected:\n\n", today, len(alerts))

	for i, a := range alerts {
		icon := "⚠️"
		if a.Severity == "critical" {
			icon = "🚨"
		}
		msg += fmt.Sprintf("%d. %s [%s] %s\n", i+1, icon, a.Severity, a.Message)
	}

	return msg
}
