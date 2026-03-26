package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleOpenClawStatus returns today's team status summary.
func handleOpenClawStatus(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		ctx := c.Request.Context()

		tenant, err := queries.GetTenant(ctx, tenantID)
		if err != nil {
			slog.Error("openclaw status: get tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		employees, err := queries.ListActiveEmployees(ctx, tenantID)
		if err != nil {
			slog.Error("openclaw status: list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		today := pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true}

		reports, err := queries.GetReportsByTenantDate(ctx, sqlc.GetReportsByTenantDateParams{
			TenantID:   tenantID,
			ReportDate: today,
		})
		if err != nil {
			slog.Error("openclaw status: get reports", "error", err)
			reports = nil
		}

		// Build set of employee IDs who submitted + submitted employee list
		submittedSet := make(map[[16]byte]bool, len(reports))
		type submittedEmployee struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			SubmittedAt string `json:"submitted_at"`
		}
		submitted := make([]submittedEmployee, 0, len(reports))

		// Build a name lookup from employees
		empNameMap := make(map[[16]byte]string, len(employees))
		for _, emp := range employees {
			empNameMap[emp.ID.Bytes] = emp.Name
		}

		for _, r := range reports {
			submittedSet[r.EmployeeID.Bytes] = true
			name := empNameMap[r.EmployeeID.Bytes]
			submitted = append(submitted, submittedEmployee{
				ID:          formatUUID(r.EmployeeID),
				Name:        name,
				SubmittedAt: r.SubmittedAt.Time.Format(time.RFC3339),
			})
		}

		// Build pending list and chase counts
		type pendingEmployee struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ChaseCount int    `json:"chase_count"`
		}

		pending := make([]pendingEmployee, 0)
		missed := make([]string, 0)
		totalChaseCount := 0

		for _, emp := range employees {
			if submittedSet[emp.ID.Bytes] {
				continue
			}

			chaseLogs, err := queries.GetChaseLogsForDate(ctx, sqlc.GetChaseLogsForDateParams{
				EmployeeID: emp.ID,
				ReportDate: today,
			})
			chaseCount := 0
			if err == nil {
				chaseCount = len(chaseLogs)
			}
			totalChaseCount += chaseCount

			pending = append(pending, pendingEmployee{
				ID:         formatUUID(emp.ID),
				Name:       emp.Name,
				ChaseCount: chaseCount,
			})
		}

		// Get mentor name from mentorDescriptions
		mentorName := tenant.MentorID
		if info, ok := mentorDescriptions[tenant.MentorID]; ok {
			mentorName = info.Name
		}

		c.JSON(http.StatusOK, gin.H{
			"date":            today.Time.Format("2006-01-02"),
			"total_employees": len(employees),
			"submitted":       submitted,
			"pending":         pending,
			"missed":          missed,
			"chase_count":     totalChaseCount,
			"mentor":          tenant.MentorID,
			"mentor_name":     mentorName,
		})
	}
}

// handleOpenClawReport returns a weekly or monthly report with ranking.
func handleOpenClawReport(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		period := c.DefaultQuery("period", "weekly")
		if period != "weekly" && period != "monthly" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period must be 'weekly' or 'monthly'"})
			return
		}

		ctx := c.Request.Context()
		now := time.Now().Truncate(24 * time.Hour)

		var startDate time.Time
		var totalDays int
		if period == "weekly" {
			startDate = now.AddDate(0, 0, -6)
			totalDays = 7
		} else {
			startDate = now.AddDate(0, 0, -29)
			totalDays = 30
		}

		employees, err := queries.ListActiveEmployees(ctx, tenantID)
		if err != nil {
			slog.Error("openclaw report: list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		type employeeRanking struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Days  int    `json:"days"`
			Medal string `json:"medal,omitempty"`
		}

		rankings := make([]employeeRanking, 0, len(employees))
		totalSubmissions := 0

		for _, emp := range employees {
			days := 0
			for d := 0; d < totalDays; d++ {
				checkDate := startDate.AddDate(0, 0, d)
				pgDate := pgtype.Date{Time: checkDate, Valid: true}
				count, err := queries.CountReportsByEmployeeDate(ctx, sqlc.CountReportsByEmployeeDateParams{
					EmployeeID: emp.ID,
					ReportDate: pgDate,
				})
				if err == nil && count > 0 {
					days++
				}
			}
			totalSubmissions += days
			rankings = append(rankings, employeeRanking{
				ID:   formatUUID(emp.ID),
				Name: emp.Name,
				Days: days,
			})
		}

		// Sort by days descending
		sort.Slice(rankings, func(i, j int) bool {
			return rankings[i].Days > rankings[j].Days
		})

		// Assign medals
		medals := []string{"gold", "silver", "bronze"}
		for i := range rankings {
			if i < len(medals) {
				rankings[i].Medal = medals[i]
			}
		}

		// Calculate submission rate
		var submissionRate float64
		totalPossible := len(employees) * totalDays
		if totalPossible > 0 {
			submissionRate = float64(totalSubmissions) / float64(totalPossible) * 100
		}

		// Build one-on-one suggestions (employees below 50% threshold)
		threshold := float64(totalDays) * 0.5
		suggestions := make([]gin.H, 0)
		for _, r := range rankings {
			if float64(r.Days) < threshold {
				suggestions = append(suggestions, gin.H{
					"id":   r.ID,
					"name": r.Name,
					"days": r.Days,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"period": period,
			"date_range": gin.H{
				"start": startDate.Format("2006-01-02"),
				"end":   now.Format("2006-01-02"),
			},
			"submission_rate":        fmt.Sprintf("%.1f%%", submissionRate),
			"ranking":               rankings,
			"one_on_one_suggestions": suggestions,
		})
	}
}

// openClawCommandRequest holds the request body for the command endpoint.
type openClawCommandRequest struct {
	Command string `json:"command" binding:"required"`
}

// handleOpenClawCommand processes structured commands.
func handleOpenClawCommand(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req openClawCommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "command is required"})
			return
		}

		ctx := c.Request.Context()
		cmd := strings.TrimSpace(req.Command)
		lower := strings.ToLower(cmd)

		switch {
		case strings.HasPrefix(lower, "switch mentor"):
			mentorArg := strings.TrimSpace(strings.TrimPrefix(lower, "switch mentor"))
			mentorArg = strings.TrimSpace(mentorArg)

			// Match by ID or name
			var matchedID string
			for id, info := range mentorDescriptions {
				if id == mentorArg || strings.EqualFold(info.Name, mentorArg) {
					matchedID = id
					break
				}
			}

			if matchedID == "" || !brain.ValidMentors[matchedID] {
				available := make([]string, 0, len(mentorDescriptions))
				for id, info := range mentorDescriptions {
					available = append(available, fmt.Sprintf("%s (%s)", id, info.Name))
				}
				c.JSON(http.StatusBadRequest, gin.H{
					"error":            fmt.Sprintf("unknown mentor: %s", mentorArg),
					"available_mentors": available,
				})
				return
			}

			if err := queries.UpdateTenantMentor(ctx, sqlc.UpdateTenantMentorParams{
				ID:          tenantID,
				MentorID:    matchedID,
				MentorBlend: nil,
			}); err != nil {
				slog.Error("openclaw command: switch mentor", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}

			info := mentorDescriptions[matchedID]
			c.JSON(http.StatusOK, gin.H{
				"result":    "mentor switched",
				"mentor_id": matchedID,
				"name":      info.Name,
			})

		case strings.HasPrefix(lower, "list employees"):
			employees, err := queries.ListActiveEmployees(ctx, tenantID)
			if err != nil {
				slog.Error("openclaw command: list employees", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}

			result := make([]gin.H, 0, len(employees))
			for _, e := range employees {
				result = append(result, gin.H{
					"id":   formatUUID(e.ID),
					"name": e.Name,
					"role": e.Role,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"result":    "employees",
				"employees": result,
			})

		case strings.HasPrefix(lower, "list mentors"):
			mentors := make([]gin.H, 0, len(mentorDescriptions))
			for _, info := range mentorDescriptions {
				mentors = append(mentors, gin.H{
					"id":          info.ID,
					"name":        info.Name,
					"description": info.Description,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"result":  "mentors",
				"mentors": mentors,
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "unknown command",
				"available_commands": []string{
					"switch mentor <id_or_name>",
					"list employees",
					"list mentors",
				},
			})
		}
	}
}

// handleOpenClawAlerts returns active alerts for employees with consecutive missed days.
func handleOpenClawAlerts(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		ctx := c.Request.Context()

		employees, err := queries.ListActiveEmployees(ctx, tenantID)
		if err != nil {
			slog.Error("openclaw alerts: list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		now := time.Now().Truncate(24 * time.Hour)

		type alert struct {
			EmployeeID   string `json:"employee_id"`
			EmployeeName string `json:"employee_name"`
			MissedDays   int    `json:"missed_days"`
			Severity     string `json:"severity"`
		}

		alerts := make([]alert, 0)

		for _, emp := range employees {
			// Check consecutive missed days by iterating last 7 days
			consecutiveMissed := 0
			for d := 1; d <= 7; d++ {
				checkDate := now.AddDate(0, 0, -d)
				pgDate := pgtype.Date{Time: checkDate, Valid: true}
				count, err := queries.CountReportsByEmployeeDate(ctx, sqlc.CountReportsByEmployeeDateParams{
					EmployeeID: emp.ID,
					ReportDate: pgDate,
				})
				if err != nil || count == 0 {
					consecutiveMissed++
				} else {
					break
				}
			}

			var severity string
			if consecutiveMissed >= 5 {
				severity = "critical"
			} else if consecutiveMissed >= 3 {
				severity = "warning"
			}

			if severity != "" {
				alerts = append(alerts, alert{
					EmployeeID:   formatUUID(emp.ID),
					EmployeeName: emp.Name,
					MissedDays:   consecutiveMissed,
					Severity:     severity,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"alerts": alerts,
			"total":  len(alerts),
		})
	}
}
