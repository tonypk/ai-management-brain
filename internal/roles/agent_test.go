package roles

import (
	"context"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestRoleAgent_SystemPrompt(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	def := LookupDefinition("ai-coo")
	if def == nil {
		t.Fatal("ai-coo not found in registry")
	}

	agent, err := NewRoleAgent(def, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	prompt := agent.SystemPrompt()

	if !strings.Contains(prompt, "Chief Operating Officer") {
		t.Errorf("system prompt should contain role title, got: %s", prompt[:200])
	}
	if !strings.Contains(prompt, "Role Identity") {
		t.Errorf("system prompt should contain Role Identity section")
	}
}

func TestRoleAgent_Brand(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	def := LookupDefinition("ai-coo")
	agent, err := NewRoleAgent(def, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	msg := agent.Brand("Hello world")

	if !strings.HasPrefix(msg, "[Chief Operating Officer]") {
		t.Errorf("brand should prefix with title, got: %s", msg)
	}
	if !strings.Contains(msg, "Hello world") {
		t.Errorf("brand should contain original message")
	}
}

func TestRoleAgent_RunCapability_Unknown(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	def := LookupDefinition("ai-coo")
	agent, err := NewRoleAgent(def, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	err = agent.RunCapability(context.Background(), "nonexistent_capability")
	if err == nil {
		t.Error("expected error for unknown capability")
	}
	if !strings.Contains(err.Error(), "unknown capability") {
		t.Errorf("expected 'unknown capability' error, got: %v", err)
	}
}

func TestRegistry_LookupDefinition(t *testing.T) {
	tests := []struct {
		roleID string
		exists bool
	}{
		{"ai-coo", true},
		{"ai-cfo", false},
		{"", false},
	}

	for _, tt := range tests {
		def := LookupDefinition(tt.roleID)
		if tt.exists && def == nil {
			t.Errorf("LookupDefinition(%q) = nil, want non-nil", tt.roleID)
		}
		if !tt.exists && def != nil {
			t.Errorf("LookupDefinition(%q) = %v, want nil", tt.roleID, def)
		}
	}
}

func TestRegistry_COO_HasCapabilities(t *testing.T) {
	def := LookupDefinition("ai-coo")
	if def == nil {
		t.Fatal("ai-coo not in registry")
	}

	if len(def.Capabilities) != 5 {
		t.Errorf("COO should have 5 capabilities, got %d", len(def.Capabilities))
	}

	// Check capability names
	names := make(map[string]bool)
	for _, cap := range def.Capabilities {
		names[cap.Name] = true
	}

	expected := []string{
		"daily_status_check",
		"chase_missing_reports",
		"weekly_summary",
		"detect_anomalies",
		"org_structure_change",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("COO missing capability: %s", name)
		}
	}
}

func TestBossSender_SendToBoss(t *testing.T) {
	var sentTo int64
	var sentText string
	sender := &mockSender{fn: func(chatID int64, text string) error {
		sentTo = chatID
		sentText = text
		return nil
	}}

	bs := NewBossSender(sender, 12345)
	err := bs.SendToBoss("test message")
	if err != nil {
		t.Fatalf("SendToBoss: %v", err)
	}
	if sentTo != 12345 {
		t.Errorf("sent to %d, want 12345", sentTo)
	}
	if sentText != "test message" {
		t.Errorf("sent text %q, want 'test message'", sentText)
	}
}

type mockSender struct {
	fn func(chatID int64, text string) error
}

func (m *mockSender) SendMessage(chatID int64, text string) error {
	return m.fn(chatID, text)
}
