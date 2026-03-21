package roles

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestNewManager_ListAgents_Empty(t *testing.T) {
	m := NewManager(ManagerConfig{})
	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("new manager should have 0 agents, got %d", len(agents))
	}
}

func TestActivateForTenant_SkipsHumanRoles(t *testing.T) {
	factory := brain.NewEngineFactory()
	m := NewManager(ManagerConfig{
		EngineFactory: factory,
	})

	plan := &brain.ManagementPlan{
		OrgDesign: brain.OrgDesign{
			SupportRoles: []brain.SupportRole{
				{Title: "HR Director", Type: "human", Scope: "People management"},
				{Title: "Finance Lead", Type: "human", Scope: "Financial oversight"},
			},
		},
	}

	err := m.ActivateForTenant(nil, "00000000-0000-0000-0000-000000000001", plan, "inamori")
	if err != nil {
		t.Fatalf("ActivateForTenant with only human roles: %v", err)
	}

	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("should have 0 agents for human-only roles, got %d", len(agents))
	}
}

func TestActivateForTenant_SkipsInvalidActions(t *testing.T) {
	factory := brain.NewEngineFactory()
	m := NewManager(ManagerConfig{
		EngineFactory: factory,
	})

	plan := &brain.ManagementPlan{
		OrgDesign: brain.OrgDesign{
			SupportRoles: []brain.SupportRole{
				{
					Title:   "AI 测试员",
					TitleEn: "AI Tester",
					RoleID:  "ai-tester",
					Type:    "ai",
					Scope:   "Testing",
					Capabilities: []brain.RoleCapability{
						{Action: "nonexistent_action", Schedule: "0 8 * * *"},
					},
				},
			},
		},
	}

	// Should not error — just skip roles with no valid capabilities
	err := m.ActivateForTenant(nil, "00000000-0000-0000-0000-000000000001", plan, "inamori")
	if err != nil {
		t.Fatalf("ActivateForTenant: %v", err)
	}

	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("should have 0 agents when all capabilities are invalid, got %d", len(agents))
	}
}

func TestActivateForTenant_NilPlan(t *testing.T) {
	m := NewManager(ManagerConfig{})
	err := m.ActivateForTenant(nil, "00000000-0000-0000-0000-000000000001", nil, "inamori")
	if err != nil {
		t.Fatalf("nil plan should return nil error, got %v", err)
	}
}

func TestBuildDynamicConfig(t *testing.T) {
	role := brain.SupportRole{
		Title:       "首席运营官",
		TitleEn:     "Chief Operating Officer",
		RoleID:      "ai-coo",
		Type:        "ai",
		Scope:       "Daily operations",
		Personality: "Pragmatic",
		Capabilities: []brain.RoleCapability{
			{Action: "daily_summary", Schedule: "0 8 * * *"},
			{Action: "check_alerts", Trigger: "alert.fired"},
		},
	}

	cfg := buildDynamicConfig(role)

	if cfg.RoleID != "ai-coo" {
		t.Errorf("role_id = %q, want ai-coo", cfg.RoleID)
	}
	if cfg.TitleEn != "Chief Operating Officer" {
		t.Errorf("title_en = %q, want Chief Operating Officer", cfg.TitleEn)
	}
	if cfg.Scope != "Daily operations" {
		t.Errorf("scope = %q, want Daily operations", cfg.Scope)
	}
	if cfg.Personality != "Pragmatic" {
		t.Errorf("personality = %q, want Pragmatic", cfg.Personality)
	}
	if len(cfg.Capabilities) != 2 {
		t.Fatalf("capabilities len = %d, want 2", len(cfg.Capabilities))
	}
	if cfg.Capabilities[0].Action != "daily_summary" {
		t.Errorf("cap[0].action = %q, want daily_summary", cfg.Capabilities[0].Action)
	}
	if cfg.Capabilities[1].Trigger != "alert.fired" {
		t.Errorf("cap[1].trigger = %q, want alert.fired", cfg.Capabilities[1].Trigger)
	}
}

func TestBuildDynamicConfig_GeneratesRoleID(t *testing.T) {
	role := brain.SupportRole{
		Title:   "运营总监",
		TitleEn: "Operations Director",
		Type:    "ai",
		Scope:   "Ops",
	}

	cfg := buildDynamicConfig(role)

	if cfg.RoleID != "ai-operations-director" {
		t.Errorf("generated role_id = %q, want ai-operations-director", cfg.RoleID)
	}
}

func TestBuildDynamicConfig_FallbackTitle(t *testing.T) {
	role := brain.SupportRole{
		Title: "运营总监",
		Type:  "ai",
		Scope: "Ops",
	}

	cfg := buildDynamicConfig(role)

	if cfg.TitleEn != "运营总监" {
		t.Errorf("title_en should fallback to title, got %q", cfg.TitleEn)
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Chief Operating Officer", "chief-operating-officer"},
		{"AI COO", "ai-coo"},
		{"simple", "simple"},
		{"With  Spaces", "with-spaces"},
		{"CamelCase", "camelcase"},
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizeID(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDynamicRoleConfig_JSON(t *testing.T) {
	cfg := testCOOConfig()
	if cfg.RoleID != "ai-coo" {
		t.Errorf("role_id = %q, want ai-coo", cfg.RoleID)
	}
	if len(cfg.Capabilities) != 3 {
		t.Errorf("capabilities len = %d, want 3", len(cfg.Capabilities))
	}
}
