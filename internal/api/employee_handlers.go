package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

func handleEmployeeProfile(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter is required"})
			return
		}

		tenantID := TenantFromContext(c)
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		ctx := c.Request.Context()

		// Fuzzy match employee by name
		emp, err := q.GetEmployeeByNameFuzzy(ctx, sqlc.GetEmployeeByNameFuzzyParams{
			TenantID: tenantUUID,
			Column2:  pgtype.Text{String: name, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No employee found matching '%s'.", name)})
			return
		}

		// Get submission history (last 30 days) for rate calculation
		history, err := q.GetEmployeeSubmissionHistory(ctx, emp.ID)
		if err != nil {
			history = nil
		}
		submissionRate := fmt.Sprintf("%.1f%%", float64(len(history))/30.0*100)

		// Get recent reports with blockers (last 7)
		recentReports, err := q.GetEmployeeRecentReportsWithBlockers(ctx, emp.ID)
		if err != nil {
			recentReports = nil
		}

		reports := make([]gin.H, 0, len(recentReports))
		for _, r := range recentReports {
			reports = append(reports, gin.H{
				"date":      r.ReportDate.Time.Format("2006-01-02"),
				"sentiment": textVal(r.Sentiment),
				"blockers":  textVal(r.Blockers),
			})
		}

		// Get consecutive missed days
		missedDays, err := q.GetConsecutiveMissDays(ctx, emp.ID)
		if err != nil {
			missedDays = 0
		}

		// Compute sentiment trend from last 7 days
		sentiments, err := q.GetRecentSentiments(ctx, sqlc.GetRecentSentimentsParams{
			EmployeeID: emp.ID,
			Limit:      7,
		})
		if err != nil {
			sentiments = nil
		}
		trend := computeSentimentTrend(sentiments)

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"employee": gin.H{
				"id":        formatUUID(emp.ID),
				"name":      emp.Name,
				"role":      emp.Role,
				"job_title": emp.JobTitle,
				"country":   emp.Country,
			},
			"submission_rate":    submissionRate,
			"recent_reports":     reports,
			"sentiment_trend":    trend,
			"consecutive_missed": missedDays,
		}})
	}
}

func textVal(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func computeSentimentTrend(sentiments []pgtype.Text) string {
	if len(sentiments) < 2 {
		return "stable"
	}

	scoreMap := map[string]int{
		"positive": 2,
		"neutral":  1,
		"negative": 0,
	}

	// Compare first half (recent) vs second half (older)
	mid := len(sentiments) / 2
	var recentSum, olderSum int
	var recentCount, olderCount int

	for i, s := range sentiments {
		if !s.Valid {
			continue
		}
		score, ok := scoreMap[s.String]
		if !ok {
			continue
		}
		if i < mid {
			recentSum += score
			recentCount++
		} else {
			olderSum += score
			olderCount++
		}
	}

	if recentCount == 0 || olderCount == 0 {
		return "stable"
	}

	recentAvg := float64(recentSum) / float64(recentCount)
	olderAvg := float64(olderSum) / float64(olderCount)
	diff := recentAvg - olderAvg

	if diff > 0.3 {
		return "improving"
	}
	if diff < -0.3 {
		return "declining"
	}
	return "stable"
}
