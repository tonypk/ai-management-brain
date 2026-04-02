package worldmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

type InsightsGenerator struct {
	q   *sqlc.Queries
	llm *brain.LLMService
}

func NewInsightsGenerator(q *sqlc.Queries, llm *brain.LLMService) *InsightsGenerator {
	return &InsightsGenerator{q: q, llm: llm}
}

func (g *InsightsGenerator) GenerateForAllTenants(ctx context.Context) error {
	tenants, err := g.q.ListActiveTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range tenants {
		tenantID := formatUUID(t.ID)
		if err := g.GenerateForTenant(ctx, tenantID); err != nil {
			slog.Error("generate insights", "tenant_id", tenantID, "error", err)
		}
	}

	return nil
}

func (g *InsightsGenerator) GenerateForTenant(ctx context.Context, tenantID string) error {
	if g.llm == nil {
		return nil
	}

	tid, err := parseUUID(tenantID)
	if err != nil {
		return err
	}

	skills, _ := g.q.ListSkillsByTenant(ctx, tid)
	blockers, _ := g.q.ListActiveBlockersByTenant(ctx, tid)
	rels, _ := g.q.ListRelationshipsByTenant(ctx, tid)
	growth, _ := g.q.ListGrowthEventsByTenant(ctx, tid)

	if len(skills) == 0 && len(blockers) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("## Team World Model Data\n\n")
	sb.WriteString(fmt.Sprintf("### Skills (%d entries)\n", len(skills)))
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s (%s, confidence=%.2f)\n", s.EmployeeName, s.SkillName, s.Proficiency, numericToFloat(s.Confidence)))
	}
	sb.WriteString(fmt.Sprintf("\n### Active Blockers (%d)\n", len(blockers)))
	for _, b := range blockers {
		sb.WriteString(fmt.Sprintf("- %s [%s]: %s (recurring x%d)\n", b.EmployeeName, b.Category, b.Description, b.RecurrenceCount))
	}
	sb.WriteString(fmt.Sprintf("\n### Relationships (%d)\n", len(rels)))
	for _, r := range rels {
		sb.WriteString(fmt.Sprintf("- %s <-> %s: %s (strength=%.2f)\n", r.EmployeeAName, r.EmployeeBName, r.RelationType, numericToFloat(r.Strength)))
	}
	sb.WriteString(fmt.Sprintf("\n### Recent Growth (%d)\n", len(growth)))
	limit := 20
	if len(growth) < limit {
		limit = len(growth)
	}
	for _, ge := range growth[:limit] {
		sb.WriteString(fmt.Sprintf("- %s [%s]: %s\n", ge.EmployeeName, ge.EventType, ge.Description))
	}

	systemPrompt := `You are a team intelligence analyst. Given the team's World Model data, generate 3-5 actionable insights.

Each insight must have:
- dimension: one of "rhythm", "context", "risk", "opportunity"
- insight_text: 1-2 sentence actionable insight
- confidence: 0.0-1.0 how confident you are

Focus on patterns that a busy founder would miss:
- Knowledge silos (only 1 person knows something critical)
- Collaboration gaps (people working on related things but not talking)
- Recurring blockers that need structural fixes
- Growth opportunities for team members
- Risk signals (declining engagement, increasing blockers)

Return ONLY valid JSON array:
[{"dimension": "...", "insight_text": "...", "confidence": 0.8}]`

	resp, err := g.llm.Client().Chat(ctx, systemPrompt, sb.String())
	if err != nil {
		return fmt.Errorf("generate insights: %w", err)
	}

	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var insights []struct {
		Dimension   string  `json:"dimension"`
		InsightText string  `json:"insight_text"`
		Confidence  float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp), &insights); err != nil {
		slog.Warn("parse insights JSON", "error", err, "response", resp)
		return nil
	}

	g.q.ExpireOldInsights(ctx) //nolint:errcheck

	expiresAt := time.Now().Add(48 * time.Hour)
	for _, ins := range insights {
		_, err := g.q.CreateWorldModelInsight(ctx, sqlc.CreateWorldModelInsightParams{
			TenantID:    tid,
			Dimension:   ins.Dimension,
			InsightText: ins.InsightText,
			Evidence:    []byte("{}"),
			Confidence:  numericFromFloat(ins.Confidence),
			ExpiresAt:   pgTimestamptz(expiresAt),
		})
		if err != nil {
			slog.Warn("create insight", "error", err)
		}
	}

	slog.Info("insights generated", "tenant_id", tenantID, "count", len(insights))
	return nil
}
