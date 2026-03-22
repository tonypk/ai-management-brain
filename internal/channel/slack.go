package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SlackConfig holds configuration for the Slack adapter.
type SlackConfig struct {
	BotToken      string // xoxb-...
	AppToken      string // xapp-... (for Socket Mode, optional)
	WebhookURL    string // Incoming webhook URL (optional, for simple posting)
	SigningSecret string // For verifying Slack Events API request signatures
}

// SlackAdapter implements Channel for Slack using the Web API.
type SlackAdapter struct {
	token      string
	appToken   string
	webhookURL string
	httpClient *http.Client
	stopCh     chan struct{}
	baseURL    string
}

// NewSlackAdapter creates a new Slack channel adapter.
func NewSlackAdapter(cfg SlackConfig) (*SlackAdapter, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("slack bot token is required")
	}
	return &SlackAdapter{
		token:      cfg.BotToken,
		appToken:   cfg.AppToken,
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
		baseURL:    "https://slack.com/api",
	}, nil
}

// NewSlackAdapterWithHTTPClient creates a Slack adapter with a custom base URL (for testing).
func NewSlackAdapterWithHTTPClient(cfg SlackConfig, baseURL string) (*SlackAdapter, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("slack bot token is required")
	}
	return &SlackAdapter{
		token:      cfg.BotToken,
		appToken:   cfg.AppToken,
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
		baseURL:    baseURL,
	}, nil
}

func (s *SlackAdapter) Type() Type { return TypeSlack }

// Send sends a message to a Slack channel by channel ID.
func (s *SlackAdapter) Send(ctx context.Context, channelID, text string) error {
	return s.postMessage(ctx, channelID, text)
}

// SendToUser sends a DM to a Slack user by their user ID.
func (s *SlackAdapter) SendToUser(ctx context.Context, userID, text string) error {
	// Open a DM channel first, then send to it.
	dmChannelID, err := s.openDM(ctx, userID)
	if err != nil {
		return fmt.Errorf("open DM with %s: %w", userID, err)
	}
	return s.postMessage(ctx, dmChannelID, text)
}

// Broadcast sends a message to multiple users.
func (s *SlackAdapter) Broadcast(ctx context.Context, userIDs []string, text string) error {
	for _, uid := range userIDs {
		if err := s.SendToUser(ctx, uid, text); err != nil {
			slog.Error("slack broadcast failed for user", "user_id", uid, "error", err)
		}
	}
	return nil
}

// Start begins listening for Slack events. For now, this is a no-op placeholder
// that blocks until context is cancelled. Real implementation would use Socket Mode
// or Events API webhook.
func (s *SlackAdapter) Start(ctx context.Context) error {
	slog.Info("slack adapter started (event listener placeholder)")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stopCh:
		return nil
	}
}

// Stop gracefully shuts down the Slack adapter.
func (s *SlackAdapter) Stop() {
	close(s.stopCh)
}

// SendMessage implements backward-compatible message sending (used by report system).
func (s *SlackAdapter) SendMessage(channelID string, text string) error {
	return s.postMessage(context.Background(), channelID, text)
}

// postMessage sends a message via Slack's chat.postMessage API.
func (s *SlackAdapter) postMessage(ctx context.Context, channel, text string) error {
	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse slack response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// openDM opens a direct message channel with a user.
func (s *SlackAdapter) openDM(ctx context.Context, userID string) (string, error) {
	payload := map[string]interface{}{
		"users": userID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal DM request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/conversations.open", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("slack API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse slack response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.Channel.ID, nil
}
