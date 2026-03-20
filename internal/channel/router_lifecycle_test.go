package channel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

// errorChannel is a mock channel that returns an error from Start.
type errorChannel struct {
	channelType channel.Type
	startErr    error
	startDelay  time.Duration
	stopCalled  bool
}

func (e *errorChannel) Type() channel.Type { return e.channelType }
func (e *errorChannel) Send(ctx context.Context, channelID, text string) error {
	return nil
}
func (e *errorChannel) SendToUser(ctx context.Context, userID, text string) error {
	return nil
}
func (e *errorChannel) Broadcast(ctx context.Context, userIDs []string, text string) error {
	return nil
}
func (e *errorChannel) Start(ctx context.Context) error {
	if e.startDelay > 0 {
		select {
		case <-time.After(e.startDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if e.startErr != nil {
		return e.startErr
	}
	<-ctx.Done()
	return ctx.Err()
}
func (e *errorChannel) Stop() {
	e.stopCalled = true
}

func TestRouter_StartAll_ContextCancel(t *testing.T) {
	r := channel.NewRouter()
	r.Register(&mockChannel{channelType: channel.TypeTelegram})
	r.Register(&mockChannel{channelType: channel.TypeSlack})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- r.StartAll(ctx)
	}()

	// Cancel context to unblock
	cancel()
	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRouter_StartAll_AdapterError(t *testing.T) {
	r := channel.NewRouter()
	expectedErr := errors.New("adapter crashed")
	r.Register(&errorChannel{
		channelType: channel.TypeTelegram,
		startErr:    expectedErr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := r.StartAll(ctx)
	if err == nil {
		t.Fatal("expected error from failed adapter")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected %q, got %q", expectedErr, err)
	}
}

func TestRouter_StartAll_NoChannels(t *testing.T) {
	r := channel.NewRouter()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := r.StartAll(ctx)
	// With no channels, select blocks until ctx.Done
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestRouter_StopAll(t *testing.T) {
	r := channel.NewRouter()
	ch1 := &errorChannel{channelType: channel.TypeTelegram}
	ch2 := &errorChannel{channelType: channel.TypeSlack}
	r.Register(ch1)
	r.Register(ch2)

	r.StopAll()

	if !ch1.stopCalled {
		t.Error("telegram Stop() not called")
	}
	if !ch2.stopCalled {
		t.Error("slack Stop() not called")
	}
}

func TestRouter_BroadcastUnregistered(t *testing.T) {
	r := channel.NewRouter()
	err := r.Broadcast(context.Background(), channel.TypeLark, []string{"u1"}, "msg")
	if err == nil {
		t.Error("expected error for unregistered channel")
	}
}

func TestRouter_Send_ErrorFromAdapter(t *testing.T) {
	r := channel.NewRouter()
	ch := &mockChannelWithError{
		channelType: channel.TypeTelegram,
		sendErr:     errors.New("send failed"),
	}
	r.Register(ch)

	err := r.Send(context.Background(), channel.TypeTelegram, "u1", "hello")
	if err == nil {
		t.Error("expected error from adapter")
	}
	if err.Error() != "send failed" {
		t.Errorf("expected 'send failed', got %q", err.Error())
	}
}

func TestRouter_RegisterOverwrite(t *testing.T) {
	r := channel.NewRouter()
	ch1 := &mockChannel{channelType: channel.TypeTelegram}
	ch2 := &mockChannel{channelType: channel.TypeTelegram}

	r.Register(ch1)
	r.Register(ch2) // overwrite

	got, ok := r.Get(channel.TypeTelegram)
	if !ok {
		t.Fatal("expected to find channel")
	}
	// Should be the second one
	if got != ch2 {
		t.Error("expected second registered channel to win")
	}

	types := r.Types()
	if len(types) != 1 {
		t.Errorf("expected 1 type after overwrite, got %d", len(types))
	}
}

// mockChannelWithError is a mock channel that returns errors.
type mockChannelWithError struct {
	channelType channel.Type
	sendErr     error
}

func (m *mockChannelWithError) Type() channel.Type { return m.channelType }
func (m *mockChannelWithError) Send(ctx context.Context, channelID, text string) error {
	return m.sendErr
}
func (m *mockChannelWithError) SendToUser(ctx context.Context, userID, text string) error {
	return m.sendErr
}
func (m *mockChannelWithError) Broadcast(ctx context.Context, userIDs []string, text string) error {
	return m.sendErr
}
func (m *mockChannelWithError) Start(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
func (m *mockChannelWithError) Stop()                           {}
