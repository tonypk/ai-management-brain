package roles

import (
	"context"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func testCOOConfig() *DynamicRoleConfig {
	return &DynamicRoleConfig{
		Title:       "首席运营官",
		TitleEn:     "Chief Operating Officer",
		RoleID:      "ai-coo",
		Scope:       "Daily operations monitoring, team follow-up",
		Personality: "Data-driven, pragmatic",
		Capabilities: []DynamicCapability{
			{Action: "daily_summary", Schedule: "0 8 * * *"},
			{Action: "chase_missing", Schedule: "30 17 * * *"},
			{Action: "check_alerts", Trigger: "alert.fired"},
		},
	}
}

func TestRoleAgent_SystemPrompt(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	cfg := testCOOConfig()
	agent, err := NewRoleAgent(cfg, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	prompt := agent.SystemPrompt()

	if !strings.Contains(prompt, "Chief Operating Officer") {
		t.Errorf("system prompt should contain role title")
	}
	if !strings.Contains(prompt, "Role Identity") {
		t.Errorf("system prompt should contain Role Identity section")
	}
	if !strings.Contains(prompt, "Daily operations monitoring") {
		t.Errorf("system prompt should contain scope")
	}
	if !strings.Contains(prompt, "Data-driven") {
		t.Errorf("system prompt should contain personality")
	}
}

func TestRoleAgent_SystemPrompt_FallbackTitle(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	cfg := &DynamicRoleConfig{
		Title:  "运营总监",
		RoleID: "ai-ops",
		Scope:  "运营管理",
	}
	agent, err := NewRoleAgent(cfg, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	prompt := agent.SystemPrompt()
	if !strings.Contains(prompt, "运营总监") {
		t.Errorf("system prompt should fallback to Chinese title when title_en is empty")
	}
}

func TestRoleAgent_Brand(t *testing.T) {
	factory := brain.NewEngineFactory()
	deps := &AgentDeps{EngineFactory: factory}

	cfg := testCOOConfig()
	agent, err := NewRoleAgent(cfg, "test-tenant", "inamori", deps)
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

	cfg := testCOOConfig()
	agent, err := NewRoleAgent(cfg, "test-tenant", "inamori", deps)
	if err != nil {
		t.Fatalf("NewRoleAgent: %v", err)
	}

	err = agent.RunCapability(context.Background(), "nonexistent_action")
	if err == nil {
		t.Error("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action primitive") {
		t.Errorf("expected 'unknown action primitive' error, got: %v", err)
	}
}

func TestActionRegistry_AllPrimitivesRegistered(t *testing.T) {
	expected := []string{
		"chase_missing",
		"daily_summary",
		"weekly_summary",
		"check_alerts",
		"create_suggestion",
		"send_branded_msg",
	}

	for _, name := range expected {
		if !ValidAction(name) {
			t.Errorf("action %q not registered in ActionRegistry", name)
		}
		if ActionRegistry[name] == nil {
			t.Errorf("action %q has nil executor", name)
		}
	}
}

func TestValidAction(t *testing.T) {
	if !ValidAction("chase_missing") {
		t.Error("chase_missing should be valid")
	}
	if ValidAction("nonexistent") {
		t.Error("nonexistent should not be valid")
	}
}

func TestAvailableActions(t *testing.T) {
	actions := AvailableActions()
	if len(actions) != 6 {
		t.Errorf("expected 6 available actions, got %d", len(actions))
	}

	names := make(map[string]bool)
	for _, a := range actions {
		names[a.Name] = true
		if a.Description == "" {
			t.Errorf("action %q has empty description", a.Name)
		}
	}

	if !names["chase_missing"] {
		t.Error("missing chase_missing in available actions")
	}
	if !names["daily_summary"] {
		t.Error("missing daily_summary in available actions")
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
