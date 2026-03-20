package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleAnalyticsOverview returns team health analytics for the authenticated tenant.
func handleAnalyticsOverview(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		today := time.Now().Format("2006-01-02")
		todayDate, _ := parseDate(today)

		// Get today's report count
		todayReports, _ := queries.CountReportsByTenantDate(c.Request.Context(), sqlc.CountReportsByTenantDateParams{
			TenantID:   tenantID,
			ReportDate: todayDate,
		})

		// Get active employee count
		employees, _ := queries.ListActiveEmployees(c.Request.Context(), tenantID)
		employeeCount := int64(len(employees))

		// Calculate submission rate
		var submissionRate float64
		if employeeCount > 0 {
			submissionRate = float64(todayReports) / float64(employeeCount)
		}

		// Get 7-day submission trend
		trend := make([]gin.H, 7)
		for i := 6; i >= 0; i-- {
			d := time.Now().AddDate(0, 0, -i)
			dateStr := d.Format("2006-01-02")
			dt, _ := parseDate(dateStr)
			count, _ := queries.CountReportsByTenantDate(c.Request.Context(), sqlc.CountReportsByTenantDateParams{
				TenantID:   tenantID,
				ReportDate: dt,
			})
			trend[6-i] = gin.H{
				"date":  dateStr,
				"count": count,
				"rate":  safeRate(count, employeeCount),
			}
		}

		// Get sentiment distribution from today's reports
		reports, _ := queries.GetReportsByTenantDate(c.Request.Context(), sqlc.GetReportsByTenantDateParams{
			TenantID:   tenantID,
			ReportDate: todayDate,
		})
		sentimentDist := map[string]int{
			"positive": 0,
			"neutral":  0,
			"negative": 0,
			"stressed": 0,
		}
		for _, r := range reports {
			if r.Sentiment.Valid {
				sentimentDist[r.Sentiment.String]++
			}
		}

		// Team health score (0-100)
		healthScore := calculateHealthScore(submissionRate, sentimentDist)

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"today": gin.H{
					"date":            today,
					"reports":         todayReports,
					"employees":       employeeCount,
					"submission_rate":  submissionRate,
				},
				"trend_7d":          trend,
				"sentiment":         sentimentDist,
				"health_score":      healthScore,
			},
		})
	}
}

// handleEmployeeActivity returns per-employee activity data.
func handleEmployeeActivity(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		employees, err := queries.ListActiveEmployees(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list employees"})
			return
		}

		activity := make([]gin.H, 0, len(employees))
		for _, emp := range employees {
			submitted, _ := queries.GetSubmittedDaysLast7(c.Request.Context(), emp.ID)
			missed, _ := queries.GetEmployeeReportStreak(c.Request.Context(), emp.ID)

			var lastSentiment string
			sentiments, _ := queries.GetRecentSentiments(c.Request.Context(), emp.ID, 1)
			if len(sentiments) > 0 && sentiments[0].Valid {
				lastSentiment = sentiments[0].String
			}

			activity = append(activity, gin.H{
				"id":              formatUUID(emp.ID),
				"name":            emp.Name,
				"submitted_7d":    submitted,
				"missed_7d":       missed,
				"last_sentiment":  lastSentiment,
				"culture_code":    emp.CultureCode,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": activity})
	}
}

// GetRecentSentiments signature helper - queries.GetRecentSentiments takes pgtype.UUID
func safeRate(count, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total)
}

// calculateHealthScore produces a 0-100 team health score.
func calculateHealthScore(submissionRate float64, sentimentDist map[string]int) int {
	// Base: 40% from submission rate
	score := submissionRate * 40

	// 40% from positive sentiment ratio
	total := 0
	for _, c := range sentimentDist {
		total += c
	}
	if total > 0 {
		positiveRatio := float64(sentimentDist["positive"]) / float64(total)
		neutralRatio := float64(sentimentDist["neutral"]) / float64(total)
		score += (positiveRatio*40 + neutralRatio*20)
	} else {
		score += 30 // neutral when no data
	}

	// 20% base (team exists and is active)
	score += 20

	if score > 100 {
		score = 100
	}
	return int(score)
}

