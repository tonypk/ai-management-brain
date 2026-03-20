package bot

import (
	"fmt"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"
)

// Bot wraps the Telegram bot with project-specific dependencies.
type Bot struct {
	bot      *tele.Bot
	resolver *IdentityResolver
}

// NewBot creates and configures a new Telegram bot.
func NewBot(token string, bossChatID int64, querier IdentityQuerier) (*Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	resolver := NewIdentityResolver(querier, bossChatID)

	return &Bot{
		bot:      b,
		resolver: resolver,
	}, nil
}

// teleBotContext adapts telebot.Context to BotContext for testability.
type teleBotContext struct {
	c tele.Context
}

func (t *teleBotContext) SenderID() int64 { return t.c.Sender().ID }
func (t *teleBotContext) Text() string    { return t.c.Text() }
func (t *teleBotContext) Send(msg string) error {
	return t.c.Send(msg)
}

// RegisterCommands registers command handlers with the telebot.
func (b *Bot) RegisterCommands(h *CommandHandler) {
	wrap := func(fn func(BotContext) error) tele.HandlerFunc {
		return func(c tele.Context) error {
			return fn(&teleBotContext{c: c})
		}
	}

	b.bot.Handle("/start", wrap(h.HandleStart))
	b.bot.Handle("/help", wrap(h.HandleHelp))
	b.bot.Handle("/status", wrap(h.HandleStatus))
	b.bot.Handle("/addemployee", wrap(h.HandleAddEmployee))
	b.bot.Handle("/join", wrap(h.HandleJoin))
	b.bot.Handle("/mentor", wrap(h.HandleMentor))

	slog.Info("bot commands registered",
		"commands", []string{"/start", "/help", "/status", "/addemployee", "/join", "/mentor"},
	)
}

// Start begins polling for messages. This blocks until Stop is called.
func (b *Bot) Start() {
	slog.Info("telegram bot starting")
	b.bot.Start()
}

// Stop gracefully shuts down the bot.
func (b *Bot) Stop() {
	slog.Info("telegram bot stopping")
	b.bot.Stop()
}

// SendMessage sends a text message to the specified chat ID.
func (b *Bot) SendMessage(chatID int64, text string) error {
	_, err := b.bot.Send(tele.ChatID(chatID), text)
	if err != nil {
		slog.Error("failed to send message",
			"chat_id", chatID,
			"error", err,
		)
		return fmt.Errorf("send message to %d: %w", chatID, err)
	}
	slog.Info("message sent", "chat_id", chatID)
	return nil
}
