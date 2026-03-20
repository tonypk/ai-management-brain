package channel

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Router manages multiple channel adapters and routes messages to the correct one.
type Router struct {
	mu       sync.RWMutex
	channels map[Type]Channel
}

// NewRouter creates a new multi-channel router.
func NewRouter() *Router {
	return &Router{
		channels: make(map[Type]Channel),
	}
}

// Register adds a channel adapter to the router.
func (r *Router) Register(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[ch.Type()] = ch
	slog.Info("channel registered", "type", ch.Type())
}

// Get returns the channel adapter for a given type.
func (r *Router) Get(channelType Type) (Channel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.channels[channelType]
	return ch, ok
}

// Send routes a message to the appropriate channel adapter.
func (r *Router) Send(ctx context.Context, channelType Type, userID string, text string) error {
	r.mu.RLock()
	ch, ok := r.channels[channelType]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no adapter registered for channel type %q", channelType)
	}

	return ch.SendToUser(ctx, userID, text)
}

// Broadcast sends a message to all users across the specified channel.
func (r *Router) Broadcast(ctx context.Context, channelType Type, userIDs []string, text string) error {
	r.mu.RLock()
	ch, ok := r.channels[channelType]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no adapter registered for channel type %q", channelType)
	}

	return ch.Broadcast(ctx, userIDs, text)
}

// Types returns all registered channel types.
func (r *Router) Types() []Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]Type, 0, len(r.channels))
	for t := range r.channels {
		types = append(types, t)
	}
	return types
}

// StartAll starts all registered channel adapters.
// Blocks until ctx is cancelled or an adapter returns an error.
func (r *Router) StartAll(ctx context.Context) error {
	r.mu.RLock()
	channels := make([]Channel, 0, len(r.channels))
	for _, ch := range r.channels {
		channels = append(channels, ch)
	}
	r.mu.RUnlock()

	errCh := make(chan error, len(channels))
	for _, ch := range channels {
		go func(c Channel) {
			errCh <- c.Start(ctx)
		}(ch)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// StopAll gracefully stops all registered channel adapters.
func (r *Router) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ch := range r.channels {
		ch.Stop()
	}
}
