package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

// TelegramAdapter implements Channel for Telegram.
type TelegramAdapter struct {
	bot        *tele.Bot
	msgHandler MessageHandler
	cmdHandler map[string]CommandHandler
}

// TelegramConfig holds Telegram adapter configuration.
type TelegramConfig struct {
	Token      string
	PollTimeout time.Duration
}

// NewTelegramAdapter creates a new Telegram channel adapter.
func NewTelegramAdapter(cfg TelegramConfig) (*TelegramAdapter, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	if cfg.PollTimeout == 0 {
		cfg.PollTimeout = 10 * time.Second
	}

	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: cfg.PollTimeout},
	})
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return &TelegramAdapter{
		bot:        bot,
		cmdHandler: make(map[string]CommandHandler),
	}, nil
}

func (t *TelegramAdapter) Type() Type { return TypeTelegram }

// OnMessage registers the handler for all incoming text messages.
func (t *TelegramAdapter) OnMessage(h MessageHandler) {
	t.msgHandler = h
}

// OnCommand registers a handler for a specific command.
func (t *TelegramAdapter) OnCommand(cmd string, h CommandHandler) {
	t.cmdHandler[cmd] = h
}

func (t *TelegramAdapter) Send(ctx context.Context, channelID string, text string) error {
	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat ID %q: %w", channelID, err)
	}
	_, err = t.bot.Send(tele.ChatID(chatID), text)
	if err != nil {
		return fmt.Errorf("telegram send to %s: %w", channelID, err)
	}
	return nil
}

func (t *TelegramAdapter) SendToUser(ctx context.Context, userID string, text string) error {
	return t.Send(ctx, userID, text)
}

func (t *TelegramAdapter) Broadcast(ctx context.Context, userIDs []string, text string) error {
	var errs []error
	for _, uid := range userIDs {
		if err := t.SendToUser(ctx, uid, text); err != nil {
			errs = append(errs, err)
			slog.Error("telegram broadcast failed", "user_id", uid, "error", err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("broadcast failed for %d/%d users", len(errs), len(userIDs))
	}
	return nil
}

func (t *TelegramAdapter) Start(ctx context.Context) error {
	// Register command handlers
	for cmd, handler := range t.cmdHandler {
		c := cmd
		h := handler
		t.bot.Handle("/"+c, func(tc tele.Context) error {
			msg := teleToMessage(tc)
			msg.IsCommand = true
			msg.Command = c
			reply, err := h(ctx, msg)
			if err != nil {
				slog.Error("command handler error", "command", c, "error", err)
				return tc.Send("An error occurred. Please try again.")
			}
			if reply != "" {
				return tc.Send(reply)
			}
			return nil
		})
	}

	// Register text handler
	if t.msgHandler != nil {
		t.bot.Handle(tele.OnText, func(tc tele.Context) error {
			msg := teleToMessage(tc)
			return t.msgHandler(ctx, msg)
		})
	}

	slog.Info("telegram adapter starting")
	go t.bot.Start()

	<-ctx.Done()
	t.bot.Stop()
	return ctx.Err()
}

func (t *TelegramAdapter) Stop() {
	t.bot.Stop()
}

// SendMessage provides backward compatibility with the old MessageSender interface.
// chatID is the Telegram numeric chat ID.
func (t *TelegramAdapter) SendMessage(chatID int64, text string) error {
	_, err := t.bot.Send(tele.ChatID(chatID), text)
	if err != nil {
		slog.Error("failed to send message", "chat_id", chatID, "error", err)
		return fmt.Errorf("send message to %d: %w", chatID, err)
	}
	return nil
}

// Bot returns the underlying telebot.Bot for advanced usage (e.g., registering
// custom handlers during migration).
func (t *TelegramAdapter) Bot() *tele.Bot {
	return t.bot
}

func teleToMessage(c tele.Context) Message {
	text := c.Text()
	msg := Message{
		ChannelType: TypeTelegram,
		ChannelID:   strconv.FormatInt(c.Chat().ID, 10),
		UserID:      strconv.FormatInt(c.Sender().ID, 10),
		Text:        text,
	}

	// Parse command
	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		msg.IsCommand = true
		msg.Command = strings.TrimPrefix(parts[0], "/")
		if len(parts) > 1 {
			msg.Args = parts[1]
		}
	}

	return msg
}
