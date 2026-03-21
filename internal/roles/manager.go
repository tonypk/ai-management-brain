package roles

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/events"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

// ManagerConfig holds all dependencies for the role manager.
type ManagerConfig struct {
	Scheduler     *scheduler.Scheduler
	EventBus      *events.Bus
	EngineFactory *brain.EngineFactory
	LLM           brain.LLMClient
	Chaser        ChaserIface
	Summarizer    SummarizerIface
	AlertChecker  AlertCheckerIface
	ActionExec    ActionExecIface
	ReportDB      ReportDBIface
	Queries       *sqlc.Queries
	Sender        *BossSender
}

// Manager manages AI role agent lifecycle.
type Manager struct {
	mu     sync.RWMutex
	agents map[string]*RoleAgent // roleID → agent
	cfg    ManagerConfig
}

// NewManager creates a new role manager.
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		agents: make(map[string]*RoleAgent),
		cfg:    cfg,
	}
}

// ActivateForTenant scans a management plan's support roles, creates AI role instances,
// and registers their scheduled capabilities.
func (m *Manager) ActivateForTenant(ctx context.Context, tenantID string, plan *brain.ManagementPlan, mentorID string) error {
	if plan == nil {
		return nil
	}

	tid, err := parseUUID(tenantID)
	if err != nil {
		return fmt.Errorf("parse tenant ID: %w", err)
	}

	for _, role := range plan.OrgDesign.SupportRoles {
		if role.Type != "ai" {
			continue
		}

		// Build DynamicRoleConfig from plan's support role
		cfg := buildDynamicConfig(role)

		// Validate capabilities — skip unknown actions
		validCaps := make([]DynamicCapability, 0, len(cfg.Capabilities))
		for _, cap := range cfg.Capabilities {
			if ValidAction(cap.Action) {
				validCaps = append(validCaps, cap)
			} else {
				slog.Warn("skipping unknown action primitive", "role", cfg.RoleID, "action", cap.Action)
			}
		}
		cfg.Capabilities = validCaps

		if len(cfg.Capabilities) == 0 {
			slog.Info("AI role has no valid capabilities, skipping", "role_id", cfg.RoleID)
			continue
		}

		// Upsert DB record
		configJSON, _ := json.Marshal(cfg)
		_, err := m.cfg.Queries.CreateAIRoleInstance(ctx, sqlc.CreateAIRoleInstanceParams{
			TenantID: tid,
			RoleID:   cfg.RoleID,
			Title:    cfg.TitleEn,
			MentorID: mentorID,
			Config:   configJSON,
		})
		if err != nil {
			slog.Error("create ai role instance", "role_id", cfg.RoleID, "error", err)
			continue
		}

		// Create and register agent
		if err := m.registerAgent(ctx, cfg, tenantID, mentorID); err != nil {
			slog.Error("register agent", "role_id", cfg.RoleID, "error", err)
		}
	}

	return nil
}

// LoadExistingForTenant loads already-activated roles from DB and registers their agents.
func (m *Manager) LoadExistingForTenant(ctx context.Context, tenantID string) error {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return fmt.Errorf("parse tenant ID: %w", err)
	}

	roles, err := m.cfg.Queries.ListActiveAIRoles(ctx, tid)
	if err != nil {
		return fmt.Errorf("list active roles: %w", err)
	}

	for _, role := range roles {
		var cfg DynamicRoleConfig
		if err := json.Unmarshal(role.Config, &cfg); err != nil {
			slog.Error("unmarshal role config", "role_id", role.RoleID, "error", err)
			continue
		}

		// Fill in role_id from DB if not in config
		if cfg.RoleID == "" {
			cfg.RoleID = role.RoleID
		}
		if cfg.TitleEn == "" {
			cfg.TitleEn = role.Title
		}

		if err := m.registerAgent(ctx, &cfg, tenantID, role.MentorID); err != nil {
			slog.Error("load existing agent", "role_id", role.RoleID, "error", err)
		}
	}

	if len(roles) > 0 {
		slog.Info("loaded existing AI roles", "count", len(roles), "tenant_id", tenantID)
	}

	return nil
}

// ListAgents returns a snapshot of active agent role IDs.
func (m *Manager) ListAgents() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids
}

// registerAgent creates an agent, adds cron jobs, and subscribes to events.
func (m *Manager) registerAgent(ctx context.Context, cfg *DynamicRoleConfig, tenantID, mentorID string) error {
	deps := m.buildDeps()
	agent, err := NewRoleAgent(cfg, tenantID, mentorID, deps)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.agents[cfg.RoleID] = agent
	m.mu.Unlock()

	// Register capabilities
	for _, cap := range cfg.Capabilities {
		actionName := cap.Action

		// Cron-based capabilities
		if cap.Schedule != "" {
			jobName := fmt.Sprintf("role_%s_%s", cfg.RoleID, actionName)
			if err := m.cfg.Scheduler.AddJob(jobName, cap.Schedule, func(ctx context.Context) error {
				return agent.RunCapability(ctx, actionName)
			}); err != nil {
				slog.Error("register role cron", "job", jobName, "error", err)
			} else {
				slog.Info("registered role cron", "job", jobName, "cron", cap.Schedule)
			}
		}

		// Event-based capabilities
		if cap.Trigger != "" {
			eventType := events.EventType(cap.Trigger)
			actionCopy := actionName
			m.cfg.EventBus.Subscribe(eventType, func(ctx context.Context, event events.Event) error {
				if event.TenantID != tenantID {
					return nil
				}
				return agent.RunCapability(ctx, actionCopy)
			})
			slog.Info("subscribed role to event", "role", cfg.RoleID, "event", cap.Trigger)
		}
	}

	return nil
}

// buildDeps constructs the AgentDeps from the manager's config.
func (m *Manager) buildDeps() *AgentDeps {
	return &AgentDeps{
		EngineFactory: m.cfg.EngineFactory,
		LLM:           m.cfg.LLM,
		Chaser:        m.cfg.Chaser,
		Summarizer:    m.cfg.Summarizer,
		AlertChecker:  m.cfg.AlertChecker,
		ActionExec:    m.cfg.ActionExec,
		ReportDB:      m.cfg.ReportDB,
		Queries:       &sqlcQueriesAdapter{q: m.cfg.Queries},
		Sender:        m.cfg.Sender,
	}
}

// buildDynamicConfig creates a DynamicRoleConfig from a plan's SupportRole.
func buildDynamicConfig(role brain.SupportRole) *DynamicRoleConfig {
	roleID := role.RoleID
	if roleID == "" {
		// Generate a role_id from title if LLM didn't provide one
		roleID = "ai-" + sanitizeID(role.TitleEn)
		if roleID == "ai-" {
			roleID = "ai-" + sanitizeID(role.Title)
		}
	}

	titleEn := role.TitleEn
	if titleEn == "" {
		titleEn = role.Title
	}

	caps := make([]DynamicCapability, 0, len(role.Capabilities))
	for _, c := range role.Capabilities {
		caps = append(caps, DynamicCapability{
			Action:   c.Action,
			Schedule: c.Schedule,
			Trigger:  c.Trigger,
		})
	}

	return &DynamicRoleConfig{
		Title:        role.Title,
		TitleEn:      titleEn,
		RoleID:       roleID,
		Scope:        role.Scope,
		Personality:  role.Personality,
		Capabilities: caps,
	}
}

// sanitizeID converts a string to a simple kebab-case ID.
func sanitizeID(s string) string {
	result := make([]byte, 0, len(s))
	prevDash := false
	for _, c := range []byte(s) {
		if c >= 'A' && c <= 'Z' {
			result = append(result, c+32)
			prevDash = false
		} else if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			result = append(result, c)
			prevDash = false
		} else if !prevDash && len(result) > 0 {
			result = append(result, '-')
			prevDash = true
		}
	}
	// Trim trailing dash
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}

// sqlcQueriesAdapter adapts *sqlc.Queries to QueriesIface.
type sqlcQueriesAdapter struct {
	q *sqlc.Queries
}

func (a *sqlcQueriesAdapter) CreateAISuggestion(ctx context.Context, params CreateSuggestionParams) error {
	tid, err := parseUUID(params.TenantID)
	if err != nil {
		return err
	}
	_, err = a.q.CreateAISuggestion(ctx, sqlc.CreateAISuggestionParams{
		TenantID:    tid,
		RoleID:      params.RoleID,
		RoleTitle:   params.RoleTitle,
		Capability:  params.Capability,
		Title:       params.Title,
		Content:     params.Content,
		ContextData: params.ContextData,
	})
	return err
}

// parseUUID parses a string UUID into pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}
