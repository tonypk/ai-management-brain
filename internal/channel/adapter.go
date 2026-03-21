// Package channel provides a unified messaging abstraction for multi-channel support.
// Each channel adapter (Telegram, Slack, Lark) implements the Channel interface,
// allowing the core business logic to be channel-agnostic.
package channel

import "context"

// Type identifies a messaging platform.
type Type string

const (
	TypeTelegram Type = "telegram"
	TypeSlack    Type = "slack"
	TypeLark     Type = "lark"
	TypeSignal   Type = "signal"
)

// Message represents a platform-agnostic incoming or outgoing message.
type Message struct {
	ChannelType Type   // Which platform this message is from/to
	ChannelID   string // Platform-specific channel/chat identifier
	UserID      string // Platform-specific user identifier
	Text        string // Message content
	IsCommand   bool   // Whether this is a command (e.g., /start)
	Command     string // Command name without slash (e.g., "start")
	Args        string // Command arguments
}

// Channel defines the interface for a messaging platform adapter.
type Channel interface {
	// Type returns the channel type identifier.
	Type() Type

	// Send sends a text message to a specific user/chat.
	Send(ctx context.Context, channelID string, text string) error

	// SendToUser sends a message to a user by their platform-specific user ID.
	SendToUser(ctx context.Context, userID string, text string) error

	// Broadcast sends a message to multiple users.
	Broadcast(ctx context.Context, userIDs []string, text string) error

	// Start begins listening for incoming messages. Blocks until ctx is cancelled.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the channel adapter.
	Stop()
}

// MessageHandler is called when an incoming message is received from any channel.
type MessageHandler func(ctx context.Context, msg Message) error

// CommandHandler handles a specific command from any channel.
type CommandHandler func(ctx context.Context, msg Message) (reply string, err error)

// Sender is a minimal interface for sending messages — used by business logic
// that doesn't need the full Channel interface.
type Sender interface {
	// Send sends a message to a user identified by channel type + user ID.
	Send(ctx context.Context, channelType Type, userID string, text string) error
}
