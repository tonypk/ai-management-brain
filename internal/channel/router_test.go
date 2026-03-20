package channel_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

func TestRouter_RegisterAndGet(t *testing.T) {
	r := channel.NewRouter()
	ch := &mockChannel{channelType: channel.TypeTelegram}
	r.Register(ch)

	got, ok := r.Get(channel.TypeTelegram)
	if !ok {
		t.Fatal("expected to find telegram channel")
	}
	if got.Type() != channel.TypeTelegram {
		t.Errorf("type = %q, want %q", got.Type(), channel.TypeTelegram)
	}
}

func TestRouter_GetUnregistered(t *testing.T) {
	r := channel.NewRouter()
	_, ok := r.Get(channel.TypeSlack)
	if ok {
		t.Error("expected not found for unregistered channel")
	}
}

func TestRouter_Send(t *testing.T) {
	r := channel.NewRouter()
	ch := &mockChannel{channelType: channel.TypeTelegram}
	r.Register(ch)

	err := r.Send(context.Background(), channel.TypeTelegram, "user-1", "hello")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(ch.sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ch.sent))
	}
	if ch.sent[0].userID != "user-1" {
		t.Errorf("userID = %q, want %q", ch.sent[0].userID, "user-1")
	}
	if ch.sent[0].text != "hello" {
		t.Errorf("text = %q, want %q", ch.sent[0].text, "hello")
	}
}

func TestRouter_SendUnregistered(t *testing.T) {
	r := channel.NewRouter()
	err := r.Send(context.Background(), channel.TypeSlack, "user-1", "hello")
	if err == nil {
		t.Error("expected error for unregistered channel")
	}
}

func TestRouter_Broadcast(t *testing.T) {
	r := channel.NewRouter()
	ch := &mockChannel{channelType: channel.TypeSlack}
	r.Register(ch)

	err := r.Broadcast(context.Background(), channel.TypeSlack, []string{"u1", "u2"}, "news")
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if len(ch.sent) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(ch.sent))
	}
}

func TestRouter_Types(t *testing.T) {
	r := channel.NewRouter()
	r.Register(&mockChannel{channelType: channel.TypeTelegram})
	r.Register(&mockChannel{channelType: channel.TypeSlack})

	types := r.Types()
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
	found := map[channel.Type]bool{}
	for _, tt := range types {
		found[tt] = true
	}
	if !found[channel.TypeTelegram] || !found[channel.TypeSlack] {
		t.Errorf("missing expected types, got %v", types)
	}
}

func TestRouter_MultipleChannels(t *testing.T) {
	r := channel.NewRouter()
	tg := &mockChannel{channelType: channel.TypeTelegram}
	sl := &mockChannel{channelType: channel.TypeSlack}
	r.Register(tg)
	r.Register(sl)

	r.Send(context.Background(), channel.TypeTelegram, "tg-user", "tg msg")
	r.Send(context.Background(), channel.TypeSlack, "sl-user", "sl msg")

	if len(tg.sent) != 1 || tg.sent[0].text != "tg msg" {
		t.Errorf("telegram got wrong message: %v", tg.sent)
	}
	if len(sl.sent) != 1 || sl.sent[0].text != "sl msg" {
		t.Errorf("slack got wrong message: %v", sl.sent)
	}
}
