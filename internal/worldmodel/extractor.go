package worldmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ExtractionResult is the structured output from LLM extraction.
type ExtractionResult struct {
	Skills        []SkillEntry        `json:"skills"`
	Relationships []RelationshipEntry `json:"relationships"`
	Blockers      []BlockerEntry      `json:"blockers"`
	GrowthEvents  []GrowthEventEntry  `json:"growth_events"`
}

type SkillEntry struct {
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
}

type RelationshipEntry struct {
	Colleague string `json:"colleague"`
	Type      string `json:"type"`
	Context   string `json:"context"`
}

type BlockerEntry struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Resolved    bool   `json:"resolved"`
}

type GrowthEventEntry struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// EmployeeWorldModel is a summary of existing World Model for context.
type EmployeeWorldModel struct {
	Skills        []SkillEntry        `json:"skills"`
	Blockers      []BlockerEntry      `json:"blockers"`
	Relationships []RelationshipEntry `json:"relationships"`
}

// Extractor extracts World Model data from check-in reports.
type Extractor struct {
	llm *brain.LLMService
	q   *sqlc.Queries
}

func NewExtractor(llm *brain.LLMService, q *sqlc.Queries) *Extractor {
	return &Extractor{llm: llm, q: q}
}

// ExtractFromReport processes a submitted report and updates the World Model.
func (e *Extractor) ExtractFromReport(ctx context.Context, tenantID, employeeID, answersJSON string) error {
	if e.llm == nil {
		return nil
	}

	existing, err := e.loadEmployeeWorldModel(ctx, tenantID, employeeID)
	if err != nil {
		slog.Warn("load existing world model", "employee_id", employeeID, "error", err)
		existing = &EmployeeWorldModel{}
	}

	userPrompt := BuildExtractionPrompt(answersJSON, existing)

	systemPrompt := `You extract structured knowledge from employee daily check-in reports.
Given the report answers and the employee's existing World Model, extract NEW or UPDATED information.
Do NOT repeat information already in the existing World Model unless it has changed.

Return ONLY valid JSON in this exact format:
{
  "skills": [{"name": "skill name", "proficiency": "low|medium|high|expert"}],
  "relationships": [{"colleague": "name", "type": "collaborates|mentors|blocks|depends_on", "context": "what they worked on"}],
  "blockers": [{"category": "cross_team|tooling|requirements|skills_gap|external", "description": "brief description", "resolved": false}],
  "growth_events": [{"type": "new_skill|skill_upgrade|first_solo|mentoring_others", "description": "what happened"}]
}

Rules:
- Only include items that are clearly mentioned or strongly implied in the report
- Use empty arrays [] if nothing found for a category
- For blockers, set resolved=true if the report mentions solving a previous blocker
- For proficiency, infer from context (teaching others=expert, struggling=low, routine use=high)`

	resp, err := e.llm.ExtractWithFastModel(ctx, systemPrompt, userPrompt)
	if err != nil {
		slog.Error("world model extraction failed", "employee_id", employeeID, "error", err)
		return nil
	}

	result, err := ParseExtractionResponse(resp)
	if err != nil {
		slog.Warn("parse extraction response", "employee_id", employeeID, "error", err)
		return nil
	}

	if err := e.mergeResults(ctx, tenantID, employeeID, result); err != nil {
		slog.Error("merge world model results", "employee_id", employeeID, "error", err)
		return nil
	}

	slog.Info("world model extracted",
		"employee_id", employeeID,
		"skills", len(result.Skills),
		"relationships", len(result.Relationships),
		"blockers", len(result.Blockers),
		"growth_events", len(result.GrowthEvents),
	)
	return nil
}

// BuildExtractionPrompt creates the user prompt for extraction.
func BuildExtractionPrompt(answersText string, existing *EmployeeWorldModel) string {
	var sb strings.Builder
	sb.WriteString("## Employee's Daily Check-in Report\n\n")
	sb.WriteString(answersText)

	if existing != nil && (len(existing.Skills) > 0 || len(existing.Blockers) > 0) {
		sb.WriteString("\n\n## Existing World Model (do NOT repeat unless changed)\n\n")
		existingJSON, _ := json.MarshalIndent(existing, "", "  ")
		sb.WriteString(string(existingJSON))
	}

	return sb.String()
}

// ParseExtractionResponse parses the LLM response into ExtractionResult.
func ParseExtractionResponse(raw string) (*ExtractionResult, error) {
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse extraction JSON: %w", err)
	}

	if result.Skills == nil {
		result.Skills = []SkillEntry{}
	}
	if result.Relationships == nil {
		result.Relationships = []RelationshipEntry{}
	}
	if result.Blockers == nil {
		result.Blockers = []BlockerEntry{}
	}
	if result.GrowthEvents == nil {
		result.GrowthEvents = []GrowthEventEntry{}
	}

	return &result, nil
}

func (e *Extractor) loadEmployeeWorldModel(ctx context.Context, tenantID, employeeID string) (*EmployeeWorldModel, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return nil, err
	}

	skills, err := e.q.ListSkillsByEmployee(ctx, sqlc.ListSkillsByEmployeeParams{TenantID: tid, EmployeeID: eid})
	if err != nil {
		return nil, err
	}

	result := &EmployeeWorldModel{}
	for _, s := range skills {
		prof := "medium"
		if s.Proficiency != "" {
			prof = s.Proficiency
		}
		result.Skills = append(result.Skills, SkillEntry{Name: s.SkillName, Proficiency: prof})
	}

	blockers, err := e.q.ListBlockersByEmployee(ctx, sqlc.ListBlockersByEmployeeParams{TenantID: tid, EmployeeID: eid})
	if err != nil {
		return nil, err
	}
	for _, b := range blockers {
		if b.Status == "active" || b.Status == "recurring" {
			result.Blockers = append(result.Blockers, BlockerEntry{
				Category:    b.Category,
				Description: b.Description,
			})
		}
	}

	return result, nil
}

func (e *Extractor) mergeResults(ctx context.Context, tenantID, employeeID string, result *ExtractionResult) error {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return err
	}
	eid, err := parseUUID(employeeID)
	if err != nil {
		return err
	}

	for _, s := range result.Skills {
		conf := proficiencyToConfidence(s.Proficiency)
		_, err := e.q.UpsertWorldModelSkill(ctx, sqlc.UpsertWorldModelSkillParams{
			TenantID:    tid,
			EmployeeID:  eid,
			SkillName:   s.Name,
			Proficiency: s.Proficiency,
			Source:      "inferred",
			Confidence:  numericFromFloat(conf),
		})
		if err != nil {
			slog.Warn("upsert skill", "skill", s.Name, "error", err)
		}
	}

	for _, r := range result.Relationships {
		colleagueID, err := e.resolveEmployeeByName(ctx, tenantID, r.Colleague)
		if err != nil {
			slog.Debug("skip relationship, colleague not found", "colleague", r.Colleague)
			continue
		}
		cid, _ := parseUUID(colleagueID)

		aID, bID := eid, cid
		if r.Type == "collaborates" && formatUUID(eid) > formatUUID(cid) {
			aID, bID = cid, eid
		}

		_, err = e.q.UpsertWorldModelRelationship(ctx, sqlc.UpsertWorldModelRelationshipParams{
			TenantID:     tid,
			EmployeeAID:  aID,
			EmployeeBID:  bID,
			RelationType: r.Type,
			Context:      textFromString(r.Context),
			Strength:     numericFromFloat(0.5),
		})
		if err != nil {
			slog.Warn("upsert relationship", "colleague", r.Colleague, "error", err)
		}
	}

	for _, b := range result.Blockers {
		if b.Resolved {
			existing, err := e.q.FindSimilarBlocker(ctx, sqlc.FindSimilarBlockerParams{
				TenantID:   tid,
				EmployeeID: eid,
				Category:   b.Category,
			})
			if err == nil {
				e.q.ResolveBlocker(ctx, existing.ID)
			}
		} else {
			existing, err := e.q.FindSimilarBlocker(ctx, sqlc.FindSimilarBlockerParams{
				TenantID:   tid,
				EmployeeID: eid,
				Category:   b.Category,
			})
			if err == nil {
				e.q.IncrementBlockerRecurrence(ctx, existing.ID)
			} else {
				_, err := e.q.CreateWorldModelBlocker(ctx, sqlc.CreateWorldModelBlockerParams{
					TenantID:    tid,
					EmployeeID:  eid,
					Category:    b.Category,
					Description: b.Description,
				})
				if err != nil {
					slog.Warn("create blocker", "category", b.Category, "error", err)
				}
			}
		}
	}

	for _, g := range result.GrowthEvents {
		_, err := e.q.CreateGrowthEvent(ctx, sqlc.CreateGrowthEventParams{
			TenantID:    tid,
			EmployeeID:  eid,
			EventType:   g.Type,
			Description: g.Description,
		})
		if err != nil {
			slog.Warn("create growth event", "type", g.Type, "error", err)
		}
	}

	return nil
}

func (e *Extractor) resolveEmployeeByName(ctx context.Context, tenantID, name string) (string, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return "", err
	}
	emps, err := e.q.ListActiveEmployees(ctx, tid)
	if err != nil {
		return "", err
	}
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for _, emp := range emps {
		if strings.ToLower(emp.Name) == nameLower || strings.Contains(strings.ToLower(emp.Name), nameLower) {
			return formatUUID(emp.ID), nil
		}
	}
	return "", fmt.Errorf("employee %q not found", name)
}

func proficiencyToConfidence(prof string) float64 {
	switch prof {
	case "expert":
		return 0.95
	case "high":
		return 0.80
	case "medium":
		return 0.60
	case "low":
		return 0.40
	default:
		return 0.50
	}
}
