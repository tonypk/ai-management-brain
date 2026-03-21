package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SignalConfig holds configuration for the Signal adapter.
type SignalConfig struct {
	APIURL      string // signal-cli-rest-api base URL, e.g. "http://signal-cli:8080"
	PhoneNumber string // registered phone number, e.g. "+639123456789"
	WebhookURL  string // brain's webhook URL, e.g. "http://brain:8080/api/v1/signal/webhook"
}

// SignalAdapter implements Channel for Signal via signal-cli-rest-api.
type SignalAdapter struct {
	cfg     SignalConfig
	client  *http.Client
	handler MessageHandler
}

// NewSignalAdapter creates a new Signal channel adapter.
func NewSignalAdapter(cfg SignalConfig) *SignalAdapter {
	return &SignalAdapter{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the channel type.
func (s *SignalAdapter) Type() Type {
	return TypeSignal
}

// OnMessage registers a handler for incoming messages.
func (s *SignalAdapter) OnMessage(h MessageHandler) {
	s.handler = h
}

// Send sends a message to a channel (phone number or group).
func (s *SignalAdapter) Send(ctx context.Context, channelID string, text string) error {
	return s.sendMessage(ctx, channelID, text)
}

// SendToUser sends a message to a user by phone number.
func (s *SignalAdapter) SendToUser(ctx context.Context, userID string, text string) error {
	return s.sendMessage(ctx, userID, text)
}

// Broadcast sends a message to multiple users.
func (s *SignalAdapter) Broadcast(ctx context.Context, userIDs []string, text string) error {
	var lastErr error
	for _, uid := range userIDs {
		if err := s.sendMessage(ctx, uid, text); err != nil {
			slog.Error("signal broadcast failed for user", "user", uid, "error", err)
			lastErr = err
		}
	}
	return lastErr
}

// Start registers the webhook and blocks until context is cancelled.
func (s *SignalAdapter) Start(ctx context.Context) error {
	if s.cfg.WebhookURL != "" {
		if err := s.registerWebhook(ctx); err != nil {
			slog.Warn("failed to register signal webhook, will retry", "error", err)
		}
	}

	slog.Info("signal adapter started", "phone", s.cfg.PhoneNumber)
	<-ctx.Done()
	return ctx.Err()
}

// Stop gracefully shuts down the adapter.
func (s *SignalAdapter) Stop() {
	slog.Info("signal adapter stopped")
}

// HandleWebhook is a Gin handler for incoming Signal messages from signal-cli-rest-api.
func (s *SignalAdapter) HandleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var payload signalWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Only process data messages with text
	if payload.Envelope.DataMessage.Message == "" {
		c.JSON(http.StatusOK, gin.H{"status": "skipped"})
		return
	}

	source := payload.Envelope.SourceNumber
	if source == "" {
		source = payload.Envelope.Source
	}

	text := payload.Envelope.DataMessage.Message
	msg := Message{
		ChannelType: TypeSignal,
		ChannelID:   source,
		UserID:      source,
		Text:        text,
	}

	// Parse commands (messages starting with /)
	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		msg.IsCommand = true
		msg.Command = strings.TrimPrefix(parts[0], "/")
		if len(parts) > 1 {
			msg.Args = parts[1]
		}
	}

	if s.handler != nil {
		if err := s.handler(c.Request.Context(), msg); err != nil {
			slog.Error("signal message handler error", "source", source, "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// sendMessage sends a text message to a recipient via signal-cli-rest-api.
func (s *SignalAdapter) sendMessage(ctx context.Context, recipient string, text string) error {
	payload := signalSendRequest{
		Message:    text,
		Number:     s.cfg.PhoneNumber,
		Recipients: []string{recipient},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal send request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/send", s.cfg.APIURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("signal send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signal send failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// registerWebhook registers the webhook URL with signal-cli-rest-api.
func (s *SignalAdapter) registerWebhook(ctx context.Context) error {
	payload := map[string]string{
		"url": s.cfg.WebhookURL,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/receive/%s", s.cfg.APIURL, s.cfg.PhoneNumber)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("register webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook registration failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	slog.Info("signal webhook registered", "url", s.cfg.WebhookURL)
	return nil
}

// --- Request/Response types ---

type signalSendRequest struct {
	Message    string   `json:"message"`
	Number     string   `json:"number"`
	Recipients []string `json:"recipients"`
}

type signalWebhookPayload struct {
	Envelope signalEnvelope `json:"envelope"`
	Account  string         `json:"account"`
}

type signalEnvelope struct {
	Source       string            `json:"source"`
	SourceNumber string           `json:"sourceNumber"`
	SourceName   string           `json:"sourceName"`
	DataMessage  signalDataMessage `json:"dataMessage"`
}

type signalDataMessage struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
