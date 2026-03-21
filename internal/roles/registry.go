package roles

// CapabilityMode determines how a capability is executed.
type CapabilityMode int

const (
	// AutoExecute capabilities run automatically via cron or event triggers.
	AutoExecute CapabilityMode = iota
	// SuggestOnly capabilities create suggestions for human approval.
	SuggestOnly
)

// Capability defines one thing a role agent can do.
type Capability struct {
	Name          string
	Mode          CapabilityMode
	CronExpr      string   // for scheduled capabilities
	EventTriggers []string // for event-driven capabilities
}

// RoleDefinition describes a registered AI role template.
type RoleDefinition struct {
	RoleID       string
	DefaultTitle string
	Capabilities []Capability
}

// Registry maps role IDs to their definitions.
var Registry = map[string]RoleDefinition{
	"ai-coo": {
		RoleID:       "ai-coo",
		DefaultTitle: "Chief Operating Officer",
		Capabilities: []Capability{
			{Name: "daily_status_check", Mode: AutoExecute, CronExpr: "0 8 * * *"},
			{Name: "chase_missing_reports", Mode: AutoExecute, CronExpr: "30 17 * * *"},
			{Name: "weekly_summary", Mode: AutoExecute, CronExpr: "0 18 * * 5"},
			{Name: "detect_anomalies", Mode: AutoExecute, EventTriggers: []string{"alert.fired"}},
			{Name: "org_structure_change", Mode: SuggestOnly},
		},
	},
}

// LookupDefinition finds a role definition by ID, returns nil if not found.
func LookupDefinition(roleID string) *RoleDefinition {
	def, ok := Registry[roleID]
	if !ok {
		return nil
	}
	return &def
}
