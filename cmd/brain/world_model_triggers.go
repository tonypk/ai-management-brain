package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/worldmodel"
)

// evaluateWorldModelTriggers runs World Model triggers after extraction completes.
// Runs in main.go because worldmodel imports brain (import cycle prevents brain from importing worldmodel).
func evaluateWorldModelTriggers(ctx context.Context, tenantIDStr, employeeIDStr, employeeName string, queries *sqlc.Queries, recommender *brain.Recommender) {
	if recommender == nil {
		return
	}

	tenantID, err := parseUUIDForChat(tenantIDStr)
	if err != nil {
		return
	}
	employeeID, err := parseUUIDForChat(employeeIDStr)
	if err != nil {
		return
	}

	// Trigger 1: Blocker escalation
	escalating, err := queries.GetEscalatingBlockers(ctx, tenantID)
	if err == nil {
		for _, bl := range escalating {
			if bl.EmployeeID != employeeID {
				continue
			}
			trigRec := worldmodel.EvalBlockerEscalation(worldmodel.BlockerEscalationInput{
				EmployeeID:      formatPgUUID(bl.EmployeeID),
				EmployeeName:    bl.EmployeeName,
				Category:        bl.Category,
				Description:     bl.Description,
				RecurrenceCount: int(bl.RecurrenceCount),
				FirstSeenAt:     bl.FirstSeenAt.Time.Format("2006-01-02"),
			})
			if trigRec != nil {
				if err := recommender.StoreRecommendationIfNew(ctx, tenantID, *trigRec, "realtime_trigger"); err != nil {
					slog.Error("recommendation: blocker_escalation store failed", "error", err)
				}
			}
		}
	}

	// Trigger 2: Skill match
	blockers, err := queries.GetActiveBlockersByEmployee(ctx, sqlc.GetActiveBlockersByEmployeeParams{
		TenantID: tenantID, EmployeeID: employeeID,
	})
	if err == nil {
		for _, bl := range blockers {
			matches, matchErr := queries.FindSkillMatchForBlocker(ctx, sqlc.FindSkillMatchForBlockerParams{
				TenantID:   tenantID,
				Column2:    pgtype.Text{String: bl.Category, Valid: true},
				EmployeeID: employeeID,
			})
			if matchErr != nil || len(matches) == 0 {
				continue
			}
			best := matches[0]
			trigRec := worldmodel.EvalSkillMatch(worldmodel.SkillMatchInput{
				BlockedEmployeeID:   employeeIDStr,
				BlockedEmployeeName: employeeName,
				BlockerCategory:     bl.Category,
				HelperEmployeeID:    formatPgUUID(best.EmployeeID),
				HelperEmployeeName:  best.EmployeeName,
				HelperSkillName:     best.SkillName,
				HelperConfidence:    numericToFloat64(best.Confidence),
			})
			if trigRec != nil {
				if err := recommender.StoreRecommendationIfNew(ctx, tenantID, *trigRec, "realtime_trigger"); err != nil {
					slog.Error("recommendation: skill_match store failed", "error", err)
				}
			}
			break // one match per extraction
		}
	}

	// Trigger 3: Compound risk (sentiment decline + active blockers)
	sentiments, sentErr := queries.GetRecentSentiments(ctx, sqlc.GetRecentSentimentsParams{
		EmployeeID: employeeID, Limit: 3,
	})
	if sentErr == nil && len(sentiments) >= 3 && len(blockers) > 0 {
		trend := make([]string, len(sentiments))
		for i, s := range sentiments {
			if s.Valid {
				trend[i] = s.String
			} else {
				trend[i] = "neutral"
			}
		}
		blockerCats := make([]string, len(blockers))
		for i, b := range blockers {
			blockerCats[i] = b.Category
		}
		trigRec := worldmodel.EvalCompoundRisk(worldmodel.CompoundRiskInput{
			EmployeeID:     employeeIDStr,
			EmployeeName:   employeeName,
			SentimentTrend: trend,
			ActiveBlockers: blockerCats,
		})
		if trigRec != nil {
			if err := recommender.StoreRecommendationIfNew(ctx, tenantID, *trigRec, "realtime_trigger"); err != nil {
				slog.Error("recommendation: compound_risk store failed", "error", err)
			}
		}
	}
}
