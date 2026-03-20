package channel_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

func TestNewLarkAdapter_RequiresCredentials(t *testing.T) {
	_, err := channel.NewLarkAdapter(channel.LarkConfig{})
	if err == nil {
		t.Error("expected error for empty credentials")
	}
}

func TestNewLarkAdapter_RequiresBoth(t *testing.T) {
	_, err := channel.NewLarkAdapter(channel.LarkConfig{AppID: "app123"})
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestNewLarkAdapter_Success(t *testing.T) {
	adapter, err := channel.NewLarkAdapter(channel.LarkConfig{
		AppID:     "cli_test123",
		AppSecret: "secret123",
	})
	if err != nil {
		t.Fatalf("NewLarkAdapter: %v", err)
	}
	if adapter.Type() != channel.TypeLark {
		t.Errorf("type = %q, want %q", adapter.Type(), channel.TypeLark)
	}
}

func TestLarkAdapter_ImplementsChannel(t *testing.T) {
	adapter, _ := channel.NewLarkAdapter(channel.LarkConfig{
		AppID:     "cli_test123",
		AppSecret: "secret123",
	})
	var _ channel.Channel = adapter
}
