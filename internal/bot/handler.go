package bot

import (
	"log/slog"
)

// MessageHandler routes incoming messages.
type MessageHandler struct {
	resolver *IdentityResolver
}

// NewMessageHandler creates a new message handler.
func NewMessageHandler(resolver *IdentityResolver) *MessageHandler {
	return &MessageHandler{resolver: resolver}
}

// HandleText processes a text message.
// It resolves identity and routes to the appropriate handler.
func (h *MessageHandler) HandleText(telegramID int64, text string) string {
	// Check bypass commands first
	if h.resolver.AllowWithoutIdentity(text) {
		slog.Info("bypass command received", "telegram_id", telegramID, "text", text)
		return "" // handled by command handler
	}

	slog.Info("text message received",
		"telegram_id", telegramID,
		"length", len(text),
	)

	// Route to report collector or command handler (will be wired in later tasks)
	return ""
}
