package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// LarkConfig holds configuration for the Lark adapter.
type LarkConfig struct {
	AppID             string // Lark app ID
	AppSecret         string // Lark app secret
	VerificationToken string // optional, from Lark developer console
}

// LarkAdapter implements Channel for Lark (Feishu) using the Open API.
type LarkAdapter struct {
	appID             string
	appSecret         string
	verificationToken string

	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time

	httpClient *http.Client
	stopCh     chan struct{}
	baseURL    string
	msgHandler func(ctx context.Context, msg Message) error
}

const (
	larkBaseURL     = "https://open.feishu.cn/open-apis"
	larkTokenURL    = larkBaseURL + "/auth/v3/tenant_access_token/internal"
	larkMessageURL  = larkBaseURL + "/im/v1/messages"
	larkBatchMsgURL = larkBaseURL + "/message/v4/batch_send/"
)

// NewLarkAdapter creates a new Lark channel adapter.
func NewLarkAdapter(cfg LarkConfig) (*LarkAdapter, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("lark app ID and secret are required")
	}
	return &LarkAdapter{
		appID:             cfg.AppID,
		appSecret:         cfg.AppSecret,
		verificationToken: cfg.VerificationToken,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
		stopCh:            make(chan struct{}),
		baseURL:           larkBaseURL,
	}, nil
}

// NewLarkAdapterWithBaseURL creates a Lark adapter with a custom base URL (for testing).
func NewLarkAdapterWithBaseURL(cfg LarkConfig, baseURL string) (*LarkAdapter, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("lark app ID and secret are required")
	}
	return &LarkAdapter{
		appID:             cfg.AppID,
		appSecret:         cfg.AppSecret,
		verificationToken: cfg.VerificationToken,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
		stopCh:            make(chan struct{}),
		baseURL:           baseURL,
	}, nil
}

func (l *LarkAdapter) Type() Type { return TypeLark }

// Send sends a message to a Lark chat by chat ID.
func (l *LarkAdapter) Send(ctx context.Context, chatID, text string) error {
	return l.sendMessage(ctx, "chat_id", chatID, text)
}

// SendToUser sends a message to a Lark user by their open_id.
func (l *LarkAdapter) SendToUser(ctx context.Context, userID, text string) error {
	return l.sendMessage(ctx, "open_id", userID, text)
}

// Broadcast sends a message to multiple users.
func (l *LarkAdapter) Broadcast(ctx context.Context, userIDs []string, text string) error {
	for _, uid := range userIDs {
		if err := l.SendToUser(ctx, uid, text); err != nil {
			slog.Error("lark broadcast failed for user", "user_id", uid, "error", err)
		}
	}
	return nil
}

// Start begins listening for Lark events. Placeholder that blocks until ctx cancelled.
func (l *LarkAdapter) Start(ctx context.Context) error {
	slog.Info("lark adapter started (event listener placeholder)")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.stopCh:
		return nil
	}
}

// Stop gracefully shuts down the Lark adapter.
func (l *LarkAdapter) Stop() {
	close(l.stopCh)
}

// getAccessToken returns a valid tenant access token, refreshing if needed.
func (l *LarkAdapter) getAccessToken(ctx context.Context) (string, error) {
	l.mu.RLock()
	if l.accessToken != "" && time.Now().Before(l.tokenExpiry) {
		token := l.accessToken
		l.mu.RUnlock()
		return token, nil
	}
	l.mu.RUnlock()

	return l.refreshAccessToken(ctx)
}

// refreshAccessToken fetches a new tenant access token from Lark.
func (l *LarkAdapter) refreshAccessToken(ctx context.Context) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if l.accessToken != "" && time.Now().Before(l.tokenExpiry) {
		return l.accessToken, nil
	}

	payload := map[string]string{
		"app_id":     l.appID,
		"app_secret": l.appSecret,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("lark token API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"` // seconds
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse lark token response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("lark token error: %s (code %d)", result.Msg, result.Code)
	}

	l.accessToken = result.TenantAccessToken
	// Refresh 60 seconds before expiry
	l.tokenExpiry = time.Now().Add(time.Duration(result.Expire-60) * time.Second)

	return l.accessToken, nil
}

// sendMessage sends a text message via Lark's messaging API.
func (l *LarkAdapter) sendMessage(ctx context.Context, receiveIDType, receiveID, text string) error {
	token, err := l.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	content, _ := json.Marshal(map[string]string{"text": text})
	payload := map[string]string{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(content),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal lark message: %w", err)
	}

	url := l.baseURL + "/im/v1/messages?receive_id_type=" + receiveIDType
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("lark API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse lark response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("lark API error: %s (code %d)", result.Msg, result.Code)
	}

	return nil
}
