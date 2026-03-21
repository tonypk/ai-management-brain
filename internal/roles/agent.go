package roles

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// RoleAgent is a single AI role agent instance bound to a tenant.
type RoleAgent struct {
	RoleID   string
	Title    string
	MentorID string
	TenantID string

	definition *RoleDefinition
	engine     *brain.Engine
	caps       *capabilityRunner
}

// AgentDeps holds the shared dependencies injected into a RoleAgent.
type AgentDeps struct {
	EngineFactory *brain.EngineFactory
	LLM           brain.LLMClient
	Chaser        ChaserIface
	Summarizer    SummarizerIface
	AlertChecker  AlertCheckerIface
	ActionExec    ActionExecIface
	ReportDB      ReportDBIface
	Queries       QueriesIface
	Sender        *BossSender
}

// ChaserIface abstracts the report.Chaser for use by role agents.
type ChaserIface interface {
	ChaseAll(ctx context.Context, tenantID, date, mentorID string) error
}

// SummarizerIface abstracts the report.Summarizer for use by role agents.
type SummarizerIface interface {
	Generate(ctx context.Context, tenantID, date string, engine *brain.Engine) (*SummaryResult, error)
}

// SummaryResult mirrors report.SummaryResult to avoid import cycle.
type SummaryResult struct {
	Content        string
	SubmissionRate float64
	BlockersCount  int
}

// AlertCheckerIface abstracts the report.AlertChecker.
type AlertCheckerIface interface {
	CheckAll(ctx context.Context, tenantID string, bossChatID int64) ([]AlertResult, error)
}

// AlertResult mirrors report.Alert to avoid import cycle.
type AlertResult struct {
	EmployeeID   string
	EmployeeName string
	AlertType    string
	Message      string
	Severity     string
}

// ActionExecIface abstracts the report.ActionExecutor.
type ActionExecIface interface {
	RunWeekly(ctx context.Context, tenantID, mentorID string, bossChatID int64) error
}

// ReportDBIface defines database queries used by capabilities.
type ReportDBIface interface {
	GetTenantIDByBossChatID(ctx context.Context, bossChatID int64) (string, error)
}

// QueriesIface defines sqlc queries used by role agents for suggestions.
type QueriesIface interface {
	CreateAISuggestion(ctx context.Context, params CreateSuggestionParams) error
}

// CreateSuggestionParams mirrors sqlc.CreateAISuggestionParams to avoid import cycle.
type CreateSuggestionParams struct {
	TenantID    string
	RoleID      string
	RoleTitle   string
	Capability  string
	Title       string
	Content     string
	ContextData []byte
}

// NewRoleAgent creates a new agent from a role definition.
func NewRoleAgent(def *RoleDefinition, tenantID, mentorID string, deps *AgentDeps) (*RoleAgent, error) {
	engine, err := deps.EngineFactory.ForTenant(mentorID, "default")
	if err != nil {
		return nil, fmt.Errorf("load engine for role %s: %w", def.RoleID, err)
	}

	agent := &RoleAgent{
		RoleID:     def.RoleID,
		Title:      def.DefaultTitle,
		MentorID:   mentorID,
		TenantID:   tenantID,
		definition: def,
		engine:     engine,
	}
	agent.caps = newCapabilityRunner(agent, deps)
	return agent, nil
}

// SystemPrompt builds the role-specific system prompt.
func (a *RoleAgent) SystemPrompt() string {
	base := a.engine.BuildSystemPrompt()
	return fmt.Sprintf("%s\n\n--- Role Identity ---\nYou are the AI %s for this organization.\nYour role: operational oversight, team performance tracking, and process improvement.\nSpeak from the perspective of a %s. Be concise and actionable.", base, a.Title, a.Title)
}

// Brand prefixes a message with the role title.
func (a *RoleAgent) Brand(msg string) string {
	return fmt.Sprintf("[%s]\n\n%s", a.Title, msg)
}

// RunCapability dispatches execution of a named capability.
func (a *RoleAgent) RunCapability(ctx context.Context, name string) error {
	slog.Info("role agent running capability", "role", a.RoleID, "capability", name, "tenant", a.TenantID)
	return a.caps.Run(ctx, name)
}
