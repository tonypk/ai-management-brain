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

		// Map support role title to a registered role definition
		def := matchRole(role.Title)
		if def == nil {
			slog.Info("no registry match for AI role", "title", role.Title)
			continue
		}

		// Upsert DB record
		configJSON, _ := json.Marshal(map[string]string{"scope": role.Scope})
		_, err := m.cfg.Queries.CreateAIRoleInstance(ctx, sqlc.CreateAIRoleInstanceParams{
			TenantID: tid,
			RoleID:   def.RoleID,
			Title:    role.Title,
			MentorID: mentorID,
			Config:   configJSON,
		})
		if err != nil {
			slog.Error("create ai role instance", "role_id", def.RoleID, "error", err)
			continue
		}

		// Create and register agent
		if err := m.registerAgent(ctx, def, tenantID, mentorID); err != nil {
			slog.Error("register agent", "role_id", def.RoleID, "error", err)
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
		def := LookupDefinition(role.RoleID)
		if def == nil {
			slog.Warn("unknown role in DB", "role_id", role.RoleID)
			continue
		}

		if err := m.registerAgent(ctx, def, tenantID, role.MentorID); err != nil {
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
func (m *Manager) registerAgent(ctx context.Context, def *RoleDefinition, tenantID, mentorID string) error {
	deps := m.buildDeps()
	agent, err := NewRoleAgent(def, tenantID, mentorID, deps)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.agents[def.RoleID] = agent
	m.mu.Unlock()

	// Register capabilities
	for _, cap := range def.Capabilities {
		if cap.Mode != AutoExecute {
			continue
		}

		capName := cap.Name

		// Cron-based capabilities
		if cap.CronExpr != "" {
			jobName := fmt.Sprintf("role_%s_%s", def.RoleID, capName)
			if err := m.cfg.Scheduler.AddJob(jobName, cap.CronExpr, func(ctx context.Context) error {
				return agent.RunCapability(ctx, capName)
			}); err != nil {
				slog.Error("register role cron", "job", jobName, "error", err)
			} else {
				slog.Info("registered role cron", "job", jobName, "cron", cap.CronExpr)
			}
		}

		// Event-based capabilities
		for _, trigger := range cap.EventTriggers {
			eventType := events.EventType(trigger)
			capNameCopy := capName
			m.cfg.EventBus.Subscribe(eventType, func(ctx context.Context, event events.Event) error {
				// Only process events for our tenant
				if event.TenantID != tenantID {
					return nil
				}
				return agent.RunCapability(ctx, capNameCopy)
			})
			slog.Info("subscribed role to event", "role", def.RoleID, "event", trigger)
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

// matchRole maps a plan support role title to a registry definition.
// It checks if the title contains known keywords.
func matchRole(title string) *RoleDefinition {
	// Direct lookup first
	for _, def := range Registry {
		if def.DefaultTitle == title {
			return &def
		}
	}

	// Keyword matching for common variations
	keywords := map[string]string{
		"COO":             "ai-coo",
		"Operations":      "ai-coo",
		"运营":              "ai-coo",
		"Chief Operating": "ai-coo",
	}
	for keyword, roleID := range keywords {
		if containsCI(title, keyword) {
			def := LookupDefinition(roleID)
			return def
		}
	}

	return nil
}

// containsCI checks if s contains substr (case-insensitive).
func containsCI(s, substr string) bool {
	sLower := make([]byte, len(s))
	subLower := make([]byte, len(substr))
	for i, c := range []byte(s) {
		if c >= 'A' && c <= 'Z' {
			sLower[i] = c + 32
		} else {
			sLower[i] = c
		}
	}
	for i, c := range []byte(substr) {
		if c >= 'A' && c <= 'Z' {
			subLower[i] = c + 32
		} else {
			subLower[i] = c
		}
	}
	return len(subLower) > 0 && contains(string(sLower), string(subLower))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
