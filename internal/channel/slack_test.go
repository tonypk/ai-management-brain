package channel_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

func TestNewSlackAdapter_RequiresToken(t *testing.T) {
	_, err := channel.NewSlackAdapter(channel.SlackConfig{})
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestNewSlackAdapter_Success(t *testing.T) {
	adapter, err := channel.NewSlackAdapter(channel.SlackConfig{
		BotToken: "xoxb-test-token",
	})
	if err != nil {
		t.Fatalf("NewSlackAdapter: %v", err)
	}
	if adapter.Type() != channel.TypeSlack {
		t.Errorf("type = %q, want %q", adapter.Type(), channel.TypeSlack)
	}
}

func TestSlackAdapter_ImplementsChannel(t *testing.T) {
	adapter, _ := channel.NewSlackAdapter(channel.SlackConfig{
		BotToken: "xoxb-test-token",
	})
	// Verify interface compliance at compile time
	var _ channel.Channel = adapter
}
