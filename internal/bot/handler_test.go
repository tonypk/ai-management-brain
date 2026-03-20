package bot_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

func TestMessageHandler_HandleText_JoinBypass(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)
	handler := bot.NewMessageHandler(resolver)

	reply := handler.HandleText(12345, "/join ABC123")
	if reply != "" {
		t.Errorf("expected empty reply for bypass command, got %q", reply)
	}
}

func TestMessageHandler_HandleText_NormalText(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)
	handler := bot.NewMessageHandler(resolver)

	reply := handler.HandleText(12345, "hello world")
	if reply != "" {
		t.Errorf("expected empty reply for normal text, got %q", reply)
	}
}

func TestNewMessageHandler(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)
	handler := bot.NewMessageHandler(resolver)
	if handler == nil {
		t.Fatal("NewMessageHandler returned nil")
	}
}

func TestAllowWithoutIdentity_EdgeCases(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)

	tests := []struct {
		text     string
		expected bool
	}{
		{"/join", true},
		{"/join ABC123", true},
		{"  /join ABC", true}, // leading space
		{"/start", false},
		{"/status", false},
		{"join", false},          // no slash
		{"/JOIN", false},         // case sensitive
		{"", false},
		{"/join_team", true},     // starts with /join prefix
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := resolver.AllowWithoutIdentity(tt.text)
			if got != tt.expected {
				t.Errorf("AllowWithoutIdentity(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestIdentityResult_Fields(t *testing.T) {
	result := &bot.IdentityResult{
		Employee: &bot.Employee{
			ID:          "emp-1",
			Name:        "Alice",
			TenantID:    "t-1",
			TelegramID:  12345,
			CultureCode: "philippines",
			InviteCode:  "ABC123",
		},
		Tenant: &bot.Tenant{
			ID:         "t-1",
			Name:       "Test Team",
			BossChatID: 999,
			MentorID:   "inamori",
			Timezone:   "Asia/Manila",
		},
		IsBoss: false,
	}

	if result.Employee.Name != "Alice" {
		t.Errorf("Employee.Name = %q", result.Employee.Name)
	}
	if result.Tenant.MentorID != "inamori" {
		t.Errorf("Tenant.MentorID = %q", result.Tenant.MentorID)
	}
}
