package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/bot"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// engineForTenant returns the appropriate engine for a tenant (blended or single mentor).
func engineForTenant(factory *brain.EngineFactory, tenant *bot.Tenant, cultureCode string) (*brain.Engine, error) {
	if len(tenant.MentorBlend) > 0 {
		var blend brain.BlendConfig
		if err := json.Unmarshal(tenant.MentorBlend, &blend); err == nil && blend.PrimaryID != "" && blend.SecondaryID != "" {
			return factory.ForBlend(blend.PrimaryID, blend.SecondaryID, blend.Weight, cultureCode)
		}
	}
	return factory.ForTenant(tenant.MentorID, cultureCode)
}

// fetchBossContext gathers team data for boss chat from the database.
func fetchBossContext(ctx context.Context, queries *sqlc.Queries, tenantID string, loc *time.Location) brain.BossContext {
	uid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return brain.BossContext{}
	}

	latestSummary := ""
	if summary, err := queries.GetLatestSummary(ctx, uid); err == nil {
		latestSummary = summary.Content
	}

	today := time.Now().In(loc).Format("2006-01-02")
	todayDate, _ := time.Parse("2006-01-02", today)
	pgDate := pgtype.Date{Time: todayDate, Valid: true}
	submitted, _ := queries.CountReportsByTenantDate(ctx, sqlc.CountReportsByTenantDateParams{
		TenantID:   uid,
		ReportDate: pgDate,
	})

	emps, _ := queries.ListActiveEmployees(ctx, uid)
	roster := make([]brain.RosterEntry, 0, len(emps))
	for _, e := range emps {
		roster = append(roster, brain.RosterEntry{
			ID:       formatPgUUID(e.ID),
			Name:     e.Name,
			JobTitle: e.JobTitle,
			Role:     e.Role,
			IsActive: e.IsActive,
		})
	}

	return brain.BossContext{
		LatestSummary:  latestSummary,
		SubmittedCount: int(submitted),
		TotalEmployees: len(emps),
		EmployeeRoster: roster,
	}
}

// parseUUIDForChat parses a UUID string into pgtype.UUID.
func parseUUIDForChat(s string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(s); err != nil {
		return uid, err
	}
	return uid, nil
}

// numericFromFloat converts a float64 to pgtype.Numeric with 2 decimal places.
func numericFromFloat(f float64) pgtype.Numeric {
	bf := new(big.Float).SetFloat64(f)
	scaled := new(big.Float).Mul(bf, big.NewFloat(100))
	intVal, _ := scaled.Int(nil)
	return pgtype.Numeric{Int: intVal, Exp: -2, Valid: true}
}

// formatPgUUID formats a pgtype.UUID as a hex string.
func formatPgUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// numericToFloat64 converts a pgtype.Numeric to float64.
func numericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}
