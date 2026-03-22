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

	config *DynamicRoleConfig
	engine *brain.Engine
	deps   *AgentDeps
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
	CheckAll(ctx context.Context, tenantID string, bossChannelType string, bossChannelID string) ([]AlertResult, error)
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
	RunWeekly(ctx context.Context, tenantID, mentorID string, bossChannelType string, bossChannelID string) error
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

// NewRoleAgent creates a new agent from a dynamic role config.
func NewRoleAgent(cfg *DynamicRoleConfig, tenantID, mentorID string, deps *AgentDeps) (*RoleAgent, error) {
	engine, err := deps.EngineFactory.ForTenant(mentorID, "default")
	if err != nil {
		return nil, fmt.Errorf("load engine for role %s: %w", cfg.RoleID, err)
	}

	return &RoleAgent{
		RoleID:   cfg.RoleID,
		Title:    cfg.TitleEn,
		MentorID: mentorID,
		TenantID: tenantID,
		config:   cfg,
		engine:   engine,
		deps:     deps,
	}, nil
}

// SystemPrompt builds the role-specific system prompt.
func (a *RoleAgent) SystemPrompt() string {
	base := a.engine.BuildSystemPrompt()

	title := a.config.TitleEn
	if title == "" {
		title = a.config.Title
	}

	scope := a.config.Scope
	personality := a.config.Personality

	prompt := fmt.Sprintf("%s\n\n--- Role Identity ---\nYou are the AI %s for this organization.", base, title)
	if scope != "" {
		prompt += fmt.Sprintf("\nResponsibilities: %s", scope)
	}
	if personality != "" {
		prompt += fmt.Sprintf("\nCommunication style: %s", personality)
	}
	prompt += "\nBe concise and actionable."
	return prompt
}

// Brand prefixes a message with the role title.
func (a *RoleAgent) Brand(msg string) string {
	title := a.config.TitleEn
	if title == "" {
		title = a.config.Title
	}
	return fmt.Sprintf("[%s]\n\n%s", title, msg)
}

// RunCapability dispatches execution of a named action primitive.
func (a *RoleAgent) RunCapability(ctx context.Context, actionName string) error {
	slog.Info("role agent running capability", "role", a.RoleID, "action", actionName, "tenant", a.TenantID)

	fn, ok := ActionRegistry[actionName]
	if !ok {
		return fmt.Errorf("unknown action primitive: %s", actionName)
	}
	return fn(ctx, a, a.deps)
}
