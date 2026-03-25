package onboarding

import "fmt"

// ProposedPlan is the AI-generated management plan, validated before storage.
type ProposedPlan struct {
	Mentor    MentorPlan    `json:"mentor"`
	Board     []SeatPlan    `json:"board"`
	OrgDesign OrgDesignPlan `json:"org_design"`
	Policies  PolicyPlan    `json:"policies"`
	Schedule  SchedulePlan  `json:"schedule"`
	Reasoning string        `json:"reasoning"`
}

type MentorPlan struct {
	PrimaryID   string  `json:"primary_id"`
	SecondaryID string  `json:"secondary_id,omitempty"`
	BlendWeight float64 `json:"blend_weight,omitempty"`
	Reasoning   string  `json:"reasoning"`
}

type SeatPlan struct {
	SeatType  string `json:"seat_type"`
	PersonaID string `json:"persona_id"`
	Reasoning string `json:"reasoning"`
}

type OrgDesignPlan struct {
	Units     []OrgUnitPlan `json:"units"`
	Reasoning string        `json:"reasoning"`
}

type OrgUnitPlan struct {
	RefID            string `json:"ref_id"`
	ParentRefID      string `json:"parent_ref_id"`
	Name             string `json:"name"`
	UnitType         string `json:"unit_type"`
	HeadRole         string `json:"head_role"`
	Responsibilities string `json:"responsibilities"`
}

type PolicyPlan struct {
	Framework        string    `json:"framework"`
	CheckinQuestions []string  `json:"checkin_questions"`
	TrackingFocus    []string  `json:"tracking_focus"`
	RiskRules        RiskRules `json:"risk_rules"`
	Cadence          Cadence   `json:"cadence"`
	Reasoning        string    `json:"reasoning"`
}

type RiskRules struct {
	ConsecutiveMisses      int      `json:"consecutive_misses"`
	SentimentDropThreshold float64  `json:"sentiment_drop_threshold"`
	UrgentKeywords         []string `json:"urgent_keywords"`
}

type Cadence struct {
	DailyActions   []string `json:"daily_actions"`
	WeeklyActions  []string `json:"weekly_actions"`
	WeeklyDay      string   `json:"weekly_day"`
	MonthlyActions []string `json:"monthly_actions"`
	MonthlyDay     int      `json:"monthly_day"`
}

type SchedulePlan struct {
	Checkin    string `json:"checkin"`
	Chase      string `json:"chase"`
	Summary    string `json:"summary"`
	Briefing   string `json:"briefing"`
	SignalScan string `json:"signal_scan"`
	Timezone   string `json:"timezone"`
}

// CollectedData tracks what info has been extracted from the onboarding dialogue.
type CollectedData struct {
	Industry        string   `json:"industry,omitempty"`
	CompanyStage    string   `json:"company_stage,omitempty"`
	BusinessModel   string   `json:"business_model,omitempty"`
	TeamSize        int      `json:"team_size,omitempty"`
	OrgStructure    string   `json:"org_structure,omitempty"`
	CurrentProjects string   `json:"current_projects,omitempty"`
	PainPoints      []string `json:"pain_points,omitempty"`
	CommTools       []string `json:"comm_tools,omitempty"`
	CulturePrefs    string   `json:"culture_prefs,omitempty"`
	GoalFramework   string   `json:"goal_framework,omitempty"`
}

// RequiredFieldsCovered returns true when all required onboarding info has been collected.
func (c *CollectedData) RequiredFieldsCovered() bool {
	return c.Industry != "" &&
		c.CompanyStage != "" &&
		c.BusinessModel != "" &&
		c.TeamSize > 0 &&
		c.OrgStructure != "" &&
		c.CurrentProjects != "" &&
		len(c.PainPoints) > 0 &&
		len(c.CommTools) > 0
}

// Validate checks that a ProposedPlan has all required fields populated.
func (p *ProposedPlan) Validate() error {
	if p.Mentor.PrimaryID == "" {
		return fmt.Errorf("mentor primary_id is required")
	}
	if len(p.Board) == 0 {
		return fmt.Errorf("at least one board seat is required")
	}
	if len(p.OrgDesign.Units) == 0 {
		return fmt.Errorf("at least one org unit is required")
	}
	if p.Policies.Framework == "" {
		return fmt.Errorf("policy framework is required")
	}
	if p.Schedule.Timezone == "" {
		return fmt.Errorf("schedule timezone is required")
	}
	return nil
}
