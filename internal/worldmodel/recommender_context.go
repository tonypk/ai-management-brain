package worldmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// RecommenderContext holds structured World Model data for DailyScan prompt injection.
type RecommenderContext struct {
	KnowledgeSilos     []SiloEntry            `json:"knowledge_silos"`
	EscalatingBlockers []EscalatingBlockerInfo `json:"escalating_blockers"`
	GrowthSignals      []GrowthSignalInfo      `json:"growth_signals"`
	RiskInsights       []RiskInsightInfo       `json:"risk_insights"`
}

// SiloEntry represents a skill held by only one person.
type SiloEntry struct {
	SkillName    string  `json:"skill_name"`
	EmployeeName string  `json:"employee_name"`
	Confidence   float64 `json:"confidence"`
}

func (e SiloEntry) String() string {
	return fmt.Sprintf("- %s: only %s knows this (confidence %d%%)", e.SkillName, e.EmployeeName, int(e.Confidence*100))
}

// EscalatingBlockerInfo represents a recurring blocker.
type EscalatingBlockerInfo struct {
	EmployeeID      string `json:"employee_id"`
	EmployeeName    string `json:"employee_name"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	RecurrenceCount int    `json:"recurrence_count"`
	FirstSeenAt     string `json:"first_seen_at"`
}

func (b EscalatingBlockerInfo) String() string {
	return fmt.Sprintf("- %s [%s]: %s (recurring x%d, since %s)", b.EmployeeName, b.Category, b.Description, b.RecurrenceCount, b.FirstSeenAt)
}

// GrowthSignalInfo represents a recent growth event.
type GrowthSignalInfo struct {
	EmployeeName string `json:"employee_name"`
	EventType    string `json:"event_type"`
	Description  string `json:"description"`
}

func (g GrowthSignalInfo) String() string {
	return fmt.Sprintf("- %s [%s]: %s", g.EmployeeName, g.EventType, g.Description)
}

// RiskInsightInfo represents a risk/opportunity insight.
type RiskInsightInfo struct {
	Dimension   string  `json:"dimension"`
	InsightText string  `json:"insight_text"`
	Confidence  float64 `json:"confidence"`
}

func (r RiskInsightInfo) String() string {
	return fmt.Sprintf("- [%s] %s (confidence %d%%)", r.Dimension, r.InsightText, int(r.Confidence*100))
}

// FormatForPrompt formats the context as a text block for prompt injection.
// Returns empty string if no data.
func (rc *RecommenderContext) FormatForPrompt() string {
	var b strings.Builder

	if len(rc.KnowledgeSilos) > 0 {
		b.WriteString("### Knowledge Silos (bus factor = 1)\n")
		for _, s := range rc.KnowledgeSilos {
			b.WriteString(s.String() + "\n")
		}
		b.WriteString("\n")
	}

	if len(rc.EscalatingBlockers) > 0 {
		b.WriteString("### Escalating Blockers (recurring 3+ times)\n")
		for _, bl := range rc.EscalatingBlockers {
			b.WriteString(bl.String() + "\n")
		}
		b.WriteString("\n")
	}

	if len(rc.GrowthSignals) > 0 {
		b.WriteString("### Recent Growth Events (7 days)\n")
		for _, g := range rc.GrowthSignals {
			b.WriteString(g.String() + "\n")
		}
		b.WriteString("\n")
	}

	if len(rc.RiskInsights) > 0 {
		b.WriteString("### Risk & Opportunity Insights\n")
		for _, r := range rc.RiskInsights {
			b.WriteString(r.String() + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ForRecommenderContext gathers structured World Model data for the DailyScan prompt.
func (s *Service) ForRecommenderContext(ctx context.Context, tenantID pgtype.UUID) (*RecommenderContext, error) {
	rc := &RecommenderContext{}

	// Knowledge silos
	silos, err := s.q.FindKnowledgeSilos(ctx, tenantID)
	if err == nil {
		for _, row := range silos {
			rc.KnowledgeSilos = append(rc.KnowledgeSilos, SiloEntry{
				SkillName:    row.SkillName,
				EmployeeName: row.EmployeeName,
				Confidence:   numericToFloat(row.Confidence),
			})
		}
	}

	// Escalating blockers
	escalating, err := s.q.GetEscalatingBlockers(ctx, tenantID)
	if err == nil {
		for _, row := range escalating {
			rc.EscalatingBlockers = append(rc.EscalatingBlockers, EscalatingBlockerInfo{
				EmployeeID:      formatUUID(row.EmployeeID),
				EmployeeName:    row.EmployeeName,
				Category:        row.Category,
				Description:     row.Description,
				RecurrenceCount: int(row.RecurrenceCount),
				FirstSeenAt:     row.FirstSeenAt.Time.Format("2006-01-02"),
			})
		}
	}

	// Growth signals (last 7 days)
	growth, err := s.q.GetRecentGrowthEventsForTenant(ctx, tenantID)
	if err == nil {
		for _, row := range growth {
			rc.GrowthSignals = append(rc.GrowthSignals, GrowthSignalInfo{
				EmployeeName: row.EmployeeName,
				EventType:    row.EventType,
				Description:  row.Description,
			})
		}
	}

	// Risk insights (filter: confidence > 0.6, dimension = risk or opportunity)
	insights, err := s.q.ListActiveInsightsByTenant(ctx, tenantID)
	if err == nil {
		for _, row := range insights {
			conf := numericToFloat(row.Confidence)
			if conf > 0.6 && (row.Dimension == "risk" || row.Dimension == "opportunity") {
				rc.RiskInsights = append(rc.RiskInsights, RiskInsightInfo{
					Dimension:   row.Dimension,
					InsightText: row.InsightText,
					Confidence:  conf,
				})
			}
		}
	}

	return rc, nil
}

// ForRecommenderPrompt returns formatted World Model text for DailyScan prompt injection.
// Satisfies brain.WorldModelContextProvider interface.
func (s *Service) ForRecommenderPrompt(ctx context.Context, tenantID pgtype.UUID) (string, error) {
	rc, err := s.ForRecommenderContext(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if rc == nil {
		return "", nil
	}
	return rc.FormatForPrompt(), nil
}
