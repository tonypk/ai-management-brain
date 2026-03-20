package channel_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

// mockChannel implements Channel for testing.
type mockChannel struct {
	channelType channel.Type
	sent        []sentMsg
}

type sentMsg struct {
	channelID string
	userID    string
	text      string
}

func (m *mockChannel) Type() channel.Type { return m.channelType }
func (m *mockChannel) Send(ctx context.Context, channelID, text string) error {
	m.sent = append(m.sent, sentMsg{channelID: channelID, text: text})
	return nil
}
func (m *mockChannel) SendToUser(ctx context.Context, userID, text string) error {
	m.sent = append(m.sent, sentMsg{userID: userID, text: text})
	return nil
}
func (m *mockChannel) Broadcast(ctx context.Context, userIDs []string, text string) error {
	for _, uid := range userIDs {
		m.sent = append(m.sent, sentMsg{userID: uid, text: text})
	}
	return nil
}
func (m *mockChannel) Start(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
func (m *mockChannel) Stop()                           {}

func TestMockChannel_Send(t *testing.T) {
	ch := &mockChannel{channelType: channel.TypeTelegram}
	if err := ch.Send(context.Background(), "123", "hello"); err != nil {
		t.Fatal(err)
	}
	if len(ch.sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ch.sent))
	}
	if ch.sent[0].text != "hello" {
		t.Errorf("text = %q, want %q", ch.sent[0].text, "hello")
	}
}

func TestMockChannel_Broadcast(t *testing.T) {
	ch := &mockChannel{channelType: channel.TypeSlack}
	err := ch.Broadcast(context.Background(), []string{"u1", "u2", "u3"}, "announcement")
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.sent) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(ch.sent))
	}
	for _, m := range ch.sent {
		if m.text != "announcement" {
			t.Errorf("text = %q, want %q", m.text, "announcement")
		}
	}
}

func TestChannelTypes(t *testing.T) {
	if channel.TypeTelegram != "telegram" {
		t.Errorf("TypeTelegram = %q", channel.TypeTelegram)
	}
	if channel.TypeSlack != "slack" {
		t.Errorf("TypeSlack = %q", channel.TypeSlack)
	}
	if channel.TypeLark != "lark" {
		t.Errorf("TypeLark = %q", channel.TypeLark)
	}
}

func TestMessage_CommandParsing(t *testing.T) {
	msg := channel.Message{
		ChannelType: channel.TypeTelegram,
		UserID:      "123",
		Text:        "/start hello world",
		IsCommand:   true,
		Command:     "start",
		Args:        "hello world",
	}

	if msg.Command != "start" {
		t.Errorf("Command = %q, want %q", msg.Command, "start")
	}
	if msg.Args != "hello world" {
		t.Errorf("Args = %q, want %q", msg.Args, "hello world")
	}
}
