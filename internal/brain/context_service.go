package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
)

// MemoryReader is the subset of memory.MemoryStore needed by ContextService.
type MemoryReader interface {
	GetProfile(ctx context.Context, tenantID, employeeID string) (*memory.Memory, error)
	List(ctx context.Context, tenantID, memType, memTier, employeeID string, limit, offset int32) ([]memory.Memory, error)
}

// ContextService aggregates company context from multiple data sources
// for use by prompts and MCP tools.
type ContextService struct {
	queries     *sqlc.Queries
	memoryStore MemoryReader
}

// NewContextService creates a new ContextService.
func NewContextService(queries *sqlc.Queries) *ContextService {
	return &ContextService{queries: queries}
}

// SetMemoryReader injects the memory reader dependency after construction.
func (cs *ContextService) SetMemoryReader(mr MemoryReader) {
	cs.memoryStore = mr
}

// CompanyContext holds the aggregated company context.
type CompanyContext struct {
	Organization     *OrgContext               `json:"organization,omitempty"`
	Goals            []GoalContext             `json:"goals,omitempty"`
	Metrics          []MetricContext           `json:"metrics,omitempty"`
	TopRisks         []RiskContext             `json:"top_risks,omitempty"`
	TeamSize         int                       `json:"team_size"`
	HRInsights       *HRInsightsContext        `json:"hr_insights,omitempty"`
	MemoryHighlights []EmployeeMemoryHighlight `json:"memory_highlights,omitempty"`
}

// HRInsightsContext holds aggregated HR signal insights from HalaOS.
type HRInsightsContext struct {
	HighRiskEmployees int     `json:"high_risk_employees,omitempty"`
	HighBurnoutCount  int     `json:"high_burnout_count,omitempty"`
	AvgTeamHealth     float64 `json:"avg_team_health,omitempty"`
	ActiveBlindSpots  int     `json:"active_blindspots,omitempty"`
	RecentAnomalies   int     `json:"recent_anomalies,omitempty"`
}

// EmployeeMemoryHighlight summarises a single employee's memory signals
// for inclusion in the company context sent to the AI.
type EmployeeMemoryHighlight struct {
	Name           string   `json:"name"`
	EmployeeID     string   `json:"employee_id"`
	ProfileSummary string   `json:"profile_summary,omitempty"`
	RecentThemes   []string `json:"recent_themes,omitempty"`
	MemoryCount    int      `json:"memory_count"`
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

	// HalaOS HR Insights (only populated if HalaOS signals exist)
	hrInsights := &HRInsightsContext{}
	thirtyDaysAgo := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true}
	sevenDaysAgo := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -7), Valid: true}

	var minScore70 pgtype.Numeric
	_ = minScore70.Scan("70")

	// Count high-risk employees (flight_risk score >= 70)
	highRisk, err := cs.queries.CountHighRiskSignals(ctx, sqlc.CountHighRiskSignalsParams{
		TenantID:    tenantID,
		SignalType:  "flight_risk",
		Column3:     minScore70,
		GeneratedAt: thirtyDaysAgo,
	})
	if err == nil {
		hrInsights.HighRiskEmployees = int(highRisk)
	}

	// Count high-burnout employees (burnout_risk score >= 70)
	highBurnout, err := cs.queries.CountHighRiskSignals(ctx, sqlc.CountHighRiskSignalsParams{
		TenantID:    tenantID,
		SignalType:  "burnout_risk",
		Column3:     minScore70,
		GeneratedAt: thirtyDaysAgo,
	})
	if err == nil {
		hrInsights.HighBurnoutCount = int(highBurnout)
	}

	// Count active blindspots (last 30 days)
	blindspots, err := cs.queries.CountRecentCommunicationEvents(ctx, sqlc.CountRecentCommunicationEventsParams{
		TenantID:   tenantID,
		EventType:  "blindspot_detected",
		OccurredAt: thirtyDaysAgo,
	})
	if err == nil {
		hrInsights.ActiveBlindSpots = int(blindspots)
	}

	// Count recent anomalies (last 7 days)
	anomalies, err := cs.queries.CountRecentCommunicationEvents(ctx, sqlc.CountRecentCommunicationEventsParams{
		TenantID:   tenantID,
		EventType:  "attendance_anomaly",
		OccurredAt: sevenDaysAgo,
	})
	if err == nil {
		hrInsights.RecentAnomalies = int(anomalies)
	}

	// Only attach HRInsights if at least one field is non-zero
	if hrInsights.HighRiskEmployees > 0 || hrInsights.HighBurnoutCount > 0 ||
		hrInsights.ActiveBlindSpots > 0 || hrInsights.RecentAnomalies > 0 {
		result.HRInsights = hrInsights
	}

	// Enrich with employee memory highlights if memory store is available
	result.MemoryHighlights = cs.GetMemoryHighlights(ctx, tenantID)

	return result, nil
}

// GetMemoryHighlights builds per-employee memory highlights from the memory store.
// Returns nil if no memory store has been configured.
func (cs *ContextService) GetMemoryHighlights(ctx context.Context, tenantID pgtype.UUID) []EmployeeMemoryHighlight {
	if cs.memoryStore == nil {
		return nil
	}

	tenantStr := uuidToString(tenantID)

	employees, err := cs.queries.ListActiveEmployees(ctx, tenantID)
	if err != nil || len(employees) == 0 {
		return nil
	}

	highlights := make([]EmployeeMemoryHighlight, 0, len(employees))

	for _, emp := range employees {
		empID := uuidToString(emp.ID)

		h := EmployeeMemoryHighlight{
			Name:       emp.Name,
			EmployeeID: empID,
		}

		// Fetch profile memory for summary
		profile, err := cs.memoryStore.GetProfile(ctx, tenantStr, empID)
		if err == nil && profile != nil {
			h.ProfileSummary = profile.Summary
		}

		// Fetch recent short-term employee_insight memories (up to 20)
		recentMems, err := cs.memoryStore.List(
			ctx,
			tenantStr,
			memory.TypeEmployeeInsight,
			memory.TierShortTerm,
			empID,
			20,
			0,
		)
		if err == nil {
			h.MemoryCount = len(recentMems)
			h.RecentThemes = extractThemes(recentMems)
		}

		// Only include employees that have some memory data
		if h.ProfileSummary != "" || h.MemoryCount > 0 {
			highlights = append(highlights, h)
		}
	}

	// Sort by memory count descending, cap at 10
	sort.Slice(highlights, func(i, j int) bool {
		return highlights[i].MemoryCount > highlights[j].MemoryCount
	})
	if len(highlights) > 10 {
		highlights = highlights[:10]
	}

	if len(highlights) == 0 {
		return nil
	}
	return highlights
}

// extractThemes performs simple keyword frequency analysis over memory content
// and returns the top themes (up to 5).
func extractThemes(mems []memory.Memory) []string {
	// Stopwords to ignore
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "is": true, "was": true,
		"are": true, "has": true, "had": true, "have": true, "be": true,
		"been": true, "it": true, "its": true, "this": true, "that": true,
		"they": true, "he": true, "she": true, "we": true, "you": true,
		"i": true, "my": true, "his": true, "her": true, "our": true,
		"by": true, "as": true, "from": true, "not": true, "no": true,
		"s": true, "also": true, "very": true, "more": true, "about": true,
	}

	freq := make(map[string]int)
	for _, m := range mems {
		text := strings.ToLower(m.Content + " " + m.Summary)
		words := strings.FieldsFunc(text, func(r rune) bool {
			return !('a' <= r && r <= 'z')
		})
		for _, w := range words {
			if len(w) < 4 || stopwords[w] {
				continue
			}
			freq[w]++
		}
	}

	type kv struct {
		word  string
		count int
	}
	ranked := make([]kv, 0, len(freq))
	for w, c := range freq {
		ranked = append(ranked, kv{w, c})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].word < ranked[j].word
	})

	const maxThemes = 5
	themes := make([]string, 0, maxThemes)
	for _, kv := range ranked {
		if len(themes) >= maxThemes {
			break
		}
		themes = append(themes, kv.word)
	}
	return themes
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
