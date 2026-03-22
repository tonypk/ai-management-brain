package brain

import (
	"testing"
)

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{Role: "user", Content: "hello"}
	if msg.Role != "user" || msg.Content != "hello" {
		t.Fatalf("unexpected fields: %+v", msg)
	}
}

func TestChatLLMClient_InterfaceSatisfaction(t *testing.T) {
	// AnthropicClient must satisfy ChatLLMClient at compile time.
	var _ ChatLLMClient = (*AnthropicClient)(nil)
}
