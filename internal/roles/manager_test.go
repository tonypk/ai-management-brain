package roles

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestMatchRole_DirectTitle(t *testing.T) {
	def := matchRole("Chief Operating Officer")
	if def == nil {
		t.Fatal("matchRole should match COO by exact title")
	}
	if def.RoleID != "ai-coo" {
		t.Errorf("got roleID %q, want ai-coo", def.RoleID)
	}
}

func TestMatchRole_KeywordCOO(t *testing.T) {
	tests := []struct {
		title string
		match bool
	}{
		{"AI COO", true},
		{"Chief Operating Officer", true},
		{"Operations Manager", true},
		{"运营总监", true},
		{"Chief Financial Officer", false},
		{"Random Title", false},
	}

	for _, tt := range tests {
		def := matchRole(tt.title)
		if tt.match && def == nil {
			t.Errorf("matchRole(%q) = nil, expected match", tt.title)
		}
		if !tt.match && def != nil {
			t.Errorf("matchRole(%q) = %v, expected nil", tt.title, def)
		}
	}
}

func TestMatchRole_SkipsHumanRoles(t *testing.T) {
	// matchRole only matches titles, but the actual filtering by type happens
	// in ActivateForTenant. This just tests the title matching.
	def := matchRole("HR Director")
	if def != nil {
		t.Errorf("matchRole should not match HR Director, got %v", def)
	}
}

func TestContainsCI(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"COO Assistant", "coo", true},
		{"", "test", false},
		{"test", "", false},
	}

	for _, tt := range tests {
		got := containsCI(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsCI(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

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

	// This should not panic even with nil scheduler/eventbus since no AI roles exist
	err := m.ActivateForTenant(nil, "00000000-0000-0000-0000-000000000001", plan, "inamori")
	if err != nil {
		t.Fatalf("ActivateForTenant with only human roles: %v", err)
	}

	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("should have 0 agents for human-only roles, got %d", len(agents))
	}
}
