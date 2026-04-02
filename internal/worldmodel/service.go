package worldmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

type Overview struct {
	SkillCount         int64           `json:"skill_count"`
	RelationshipCount  int64           `json:"relationship_count"`
	ActiveBlockerCount int64           `json:"active_blocker_count"`
	GrowthEventsMonth  int64           `json:"growth_events_month"`
	BlockerBreakdown   []CategoryCount `json:"blocker_breakdown"`
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type Service struct {
	q *sqlc.Queries
}

func NewService(q *sqlc.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) GetOverview(ctx context.Context, tenantID string) (*Overview, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}

	skillCount, _ := s.q.CountSkillsByTenant(ctx, tid)
	relCount, _ := s.q.CountRelationshipsByTenant(ctx, tid)
	blockerCount, _ := s.q.CountActiveBlockersByTenant(ctx, tid)
	growthCount, _ := s.q.CountGrowthEventsByTenant(ctx, tid)

	breakdown, _ := s.q.GetBlockerCategoryBreakdown(ctx, tid)
	cats := make([]CategoryCount, len(breakdown))
	for i, b := range breakdown {
		cats[i] = CategoryCount{Category: b.Category, Count: b.Count}
	}

	return &Overview{
		SkillCount:         skillCount,
		RelationshipCount:  relCount,
		ActiveBlockerCount: blockerCount,
		GrowthEventsMonth:  growthCount,
		BlockerBreakdown:   cats,
	}, nil
}

type SkillRow struct {
	EmployeeName string  `json:"employee_name"`
	SkillName    string  `json:"skill_name"`
	Proficiency  string  `json:"proficiency"`
	Confidence   float64 `json:"confidence"`
	MentionCount int     `json:"mention_count"`
}

func (s *Service) GetTeamSkills(ctx context.Context, tenantID string) ([]SkillRow, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListSkillsByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]SkillRow, len(rows))
	for i, r := range rows {
		result[i] = SkillRow{
			EmployeeName: r.EmployeeName,
			SkillName:    r.SkillName,
			Proficiency:  r.Proficiency,
			Confidence:   numericToFloat(r.Confidence),
			MentionCount: int(r.MentionCount),
		}
	}
	return result, nil
}

type RelationshipRow struct {
	EmployeeAName    string  `json:"employee_a_name"`
	EmployeeBName    string  `json:"employee_b_name"`
	RelationType     string  `json:"relation_type"`
	Context          string  `json:"context"`
	Strength         float64 `json:"strength"`
	InteractionCount int     `json:"interaction_count"`
}

func (s *Service) GetTeamRelationships(ctx context.Context, tenantID string) ([]RelationshipRow, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListRelationshipsByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]RelationshipRow, len(rows))
	for i, r := range rows {
		ctx := ""
		if r.Context.Valid {
			ctx = r.Context.String
		}
		result[i] = RelationshipRow{
			EmployeeAName:    r.EmployeeAName,
			EmployeeBName:    r.EmployeeBName,
			RelationType:     r.RelationType,
			Context:          ctx,
			Strength:         numericToFloat(r.Strength),
			InteractionCount: int(r.InteractionCount),
		}
	}
	return result, nil
}

type BlockerRow struct {
	EmployeeName    string `json:"employee_name"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	RecurrenceCount int    `json:"recurrence_count"`
}

func (s *Service) GetTeamBlockers(ctx context.Context, tenantID string) ([]BlockerRow, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListActiveBlockersByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]BlockerRow, len(rows))
	for i, r := range rows {
		result[i] = BlockerRow{
			EmployeeName:    r.EmployeeName,
			Category:        r.Category,
			Description:     r.Description,
			Status:          r.Status,
			RecurrenceCount: int(r.RecurrenceCount),
		}
	}
	return result, nil
}

type InsightRow struct {
	Dimension   string  `json:"dimension"`
	InsightText string  `json:"insight_text"`
	Confidence  float64 `json:"confidence"`
	GeneratedAt string  `json:"generated_at"`
}

func (s *Service) GetActiveInsights(ctx context.Context, tenantID string) ([]InsightRow, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListActiveInsightsByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	result := make([]InsightRow, len(rows))
	for i, r := range rows {
		result[i] = InsightRow{
			Dimension:   r.Dimension,
			InsightText: r.InsightText,
			Confidence:  numericToFloat(r.Confidence),
			GeneratedAt: r.GeneratedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result, nil
}

type GrowthEventRow struct {
	EventType   string `json:"event_type"`
	Description string `json:"description"`
	DetectedAt  string `json:"detected_at"`
}

type EmployeeFullWorldModel struct {
	Skills        []SkillRow        `json:"skills"`
	Relationships []RelationshipRow `json:"relationships"`
	Blockers      []BlockerRow      `json:"blockers"`
	GrowthEvents  []GrowthEventRow  `json:"growth_events"`
}

func (s *Service) GetEmployeeWorldModel(ctx context.Context, tenantID, employeeID string) (*EmployeeFullWorldModel, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return nil, err
	}

	skills, _ := s.q.ListSkillsByEmployee(ctx, sqlc.ListSkillsByEmployeeParams{TenantID: tid, EmployeeID: eid})
	rels, _ := s.q.ListRelationshipsByEmployee(ctx, sqlc.ListRelationshipsByEmployeeParams{TenantID: tid, EmployeeAID: eid})
	blockers, _ := s.q.ListBlockersByEmployee(ctx, sqlc.ListBlockersByEmployeeParams{TenantID: tid, EmployeeID: eid})
	growth, _ := s.q.ListGrowthEventsByEmployee(ctx, sqlc.ListGrowthEventsByEmployeeParams{TenantID: tid, EmployeeID: eid})

	result := &EmployeeFullWorldModel{
		Skills:        make([]SkillRow, len(skills)),
		Relationships: make([]RelationshipRow, 0, len(rels)),
		Blockers:      make([]BlockerRow, len(blockers)),
		GrowthEvents:  make([]GrowthEventRow, len(growth)),
	}

	for i, sk := range skills {
		result.Skills[i] = SkillRow{
			SkillName:    sk.SkillName,
			Proficiency:  sk.Proficiency,
			Confidence:   numericToFloat(sk.Confidence),
			MentionCount: int(sk.MentionCount),
		}
	}
	for _, r := range rels {
		relCtx := ""
		if r.Context.Valid {
			relCtx = r.Context.String
		}
		result.Relationships = append(result.Relationships, RelationshipRow{
			EmployeeAName: r.EmployeeAName,
			EmployeeBName: r.EmployeeBName,
			RelationType:  r.RelationType,
			Context:       relCtx,
			Strength:      numericToFloat(r.Strength),
		})
	}
	for i, b := range blockers {
		result.Blockers[i] = BlockerRow{
			Category:        b.Category,
			Description:     b.Description,
			Status:          b.Status,
			RecurrenceCount: int(b.RecurrenceCount),
		}
	}
	for i, g := range growth {
		result.GrowthEvents[i] = GrowthEventRow{
			EventType:   g.EventType,
			Description: g.Description,
			DetectedAt:  g.DetectedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
	}

	return result, nil
}

// ForSummaryContext returns a text summary of the World Model for injection into summary prompts.
func (s *Service) ForSummaryContext(ctx context.Context, tenantID string) (string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return "", err
	}

	skills, _ := s.q.ListSkillsByTenant(ctx, tid)
	blockers, _ := s.q.ListActiveBlockersByTenant(ctx, tid)
	insights, _ := s.q.ListActiveInsightsByTenant(ctx, tid)
	growth, _ := s.q.ListGrowthEventsByTenant(ctx, tid)

	var b strings.Builder
	b.WriteString("## Team World Model Context\n\n")

	if len(skills) > 0 {
		b.WriteString("### Team Skills\n")
		seen := map[string][]string{}
		for _, sk := range skills {
			seen[sk.EmployeeName] = append(seen[sk.EmployeeName], fmt.Sprintf("%s(%s)", sk.SkillName, sk.Proficiency))
		}
		for name, skillList := range seen {
			b.WriteString(fmt.Sprintf("- %s: %s\n", name, strings.Join(skillList, ", ")))
		}
		b.WriteString("\n")
	}

	if len(blockers) > 0 {
		b.WriteString("### Active Blockers\n")
		for _, bl := range blockers {
			recur := ""
			if bl.RecurrenceCount > 1 {
				recur = fmt.Sprintf(" (recurring x%d)", bl.RecurrenceCount)
			}
			b.WriteString(fmt.Sprintf("- %s [%s]: %s%s\n", bl.EmployeeName, bl.Category, bl.Description, recur))
		}
		b.WriteString("\n")
	}

	if len(growth) > 0 {
		b.WriteString("### Recent Growth Events\n")
		limit := 10
		if len(growth) < limit {
			limit = len(growth)
		}
		for _, g := range growth[:limit] {
			b.WriteString(fmt.Sprintf("- %s [%s]: %s\n", g.EmployeeName, g.EventType, g.Description))
		}
		b.WriteString("\n")
	}

	if len(insights) > 0 {
		b.WriteString("### Active Insights\n")
		for _, ins := range insights {
			b.WriteString(fmt.Sprintf("- [%s] %s\n", ins.Dimension, ins.InsightText))
		}
	}

	return b.String(), nil
}
