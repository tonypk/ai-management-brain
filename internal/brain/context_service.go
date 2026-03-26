package brain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ContextService aggregates company context from multiple data sources
// for use by prompts and MCP tools.
type ContextService struct {
	queries *sqlc.Queries
}

// NewContextService creates a new ContextService.
func NewContextService(queries *sqlc.Queries) *ContextService {
	return &ContextService{queries: queries}
}

// CompanyContext holds the aggregated company context.
type CompanyContext struct {
	Organization *OrgContext     `json:"organization,omitempty"`
	Goals        []GoalContext   `json:"goals,omitempty"`
	Metrics      []MetricContext `json:"metrics,omitempty"`
	TopRisks     []RiskContext   `json:"top_risks,omitempty"`
	TeamSize     int             `json:"team_size"`
}

// OrgContext holds organization-level context.
type OrgContext struct {
	Industry            string `json:"industry,omitempty"`
	Size                int    `json:"size"`
	Stage               string `json:"stage,omitempty"`
	MentorID            string `json:"mentor_id,omitempty"`
	StrategicPriorities string `json:"strategic_priorities,omitempty"`
}

// GoalContext holds a goal summary for context.
type GoalContext struct {
	Title  string `json:"title"`
	Level  string `json:"level,omitempty"`
	Status string `json:"status"`
	Cycle  string `json:"cycle,omitempty"`
}

// MetricContext holds a metric summary for context.
type MetricContext struct {
	Name        string `json:"name"`
	LatestValue string `json:"latest_value,omitempty"`
	Target      string `json:"target,omitempty"`
	Unit        string `json:"unit,omitempty"`
}

// RiskContext holds a risk signal summary.
type RiskContext struct {
	SignalType string   `json:"signal_type"`
	Score      string   `json:"score"`
	Reasons    []string `json:"reasons"`
}

// GetCompanyContext aggregates company context for a tenant.
func (cs *ContextService) GetCompanyContext(ctx context.Context, tenantID pgtype.UUID) (*CompanyContext, error) {
	result := &CompanyContext{}

	// Get organization info
	org, err := cs.queries.GetOrganizationByTenant(ctx, tenantID)
	if err == nil {
		oc := &OrgContext{
			MentorID: org.MentorID,
		}
		if org.Industry.Valid {
			oc.Industry = org.Industry.String
		}
		if org.Size.Valid {
			oc.Size = int(org.Size.Int32)
		}
		if org.Stage.Valid {
			oc.Stage = org.Stage.String
		}
		if org.StrategicPriorities != nil {
			oc.StrategicPriorities = string(org.StrategicPriorities)
		}
		result.Organization = oc
	}

	// Get team size
	employees, err := cs.queries.ListActiveEmployees(ctx, tenantID)
	if err == nil {
		result.TeamSize = len(employees)
	}

	// Get active goals
	goals, err := cs.queries.ListActiveGoalsByTenant(ctx, tenantID)
	if err == nil {
		for _, g := range goals {
			result.Goals = append(result.Goals, GoalContext{
				Title: g.Title,
			})
		}
	}

	// Get metrics with latest values
	metrics, err := cs.queries.GetMetricsWithLatestValues(ctx, tenantID)
	if err == nil {
		for _, m := range metrics {
			mc := MetricContext{
				Name: m.Name,
			}
			if m.Unit.Valid {
				mc.Unit = m.Unit.String
			}
			if m.TargetValue.Valid {
				tBytes, _ := m.TargetValue.MarshalJSON()
				mc.Target = string(tBytes)
			}
			if m.LatestValue.Valid {
				lvBytes, _ := m.LatestValue.MarshalJSON()
				mc.LatestValue = string(lvBytes)
			}
			result.Metrics = append(result.Metrics, mc)
		}
	}

	// Get top risks
	risks, err := cs.queries.GetTopRisks(ctx, sqlc.GetTopRisksParams{
		TenantID: tenantID,
		Limit:    5,
	})
	if err == nil {
		for _, r := range risks {
			var reasons []string
			if r.Reasons != nil {
				_ = json.Unmarshal(r.Reasons, &reasons)
			}
			scoreStr := "0"
			if r.Score.Valid {
				sBytes, _ := r.Score.MarshalJSON()
				scoreStr = string(sBytes)
			}
			result.TopRisks = append(result.TopRisks, RiskContext{
				SignalType: r.SignalType,
				Score:      scoreStr,
				Reasons:    reasons,
			})
		}
	}

	return result, nil
}

// FormatContextForPrompt formats company context as a text block for prompt injection.
func (cs *ContextService) FormatContextForPrompt(ctx context.Context, tenantID pgtype.UUID) (string, error) {
	cc, err := cs.GetCompanyContext(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("get company context: %w", err)
	}

	data, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal context: %w", err)
	}

	return string(data), nil
}
