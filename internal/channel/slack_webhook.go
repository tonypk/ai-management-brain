package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// HandleSlackEvent processes Slack Events API callbacks.
func (s *SlackAdapter) HandleSlackEvent(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	// Verify Slack signature if signing secret is set
	if s.signingSecret != "" {
		if !s.verifySlackSignature(c.Request.Header, body) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	}

	var payload struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Event     struct {
			Type    string `json:"type"`
			User    string `json:"user"`
			Text    string `json:"text"`
			Channel string `json:"channel"`
			BotID   string `json:"bot_id"` // present if message from bot
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// URL verification challenge
	if payload.Type == "url_verification" {
		c.JSON(http.StatusOK, gin.H{"challenge": payload.Challenge})
		return
	}

	// Process message events (ignore bot messages)
	if payload.Type == "event_callback" && payload.Event.Type == "message" && payload.Event.BotID == "" {
		if s.msgHandler != nil {
			text := payload.Event.Text
			isCmd := strings.HasPrefix(text, "/")
			var cmd, args string
			if isCmd {
				parts := strings.SplitN(text, " ", 2)
				cmd = strings.TrimPrefix(parts[0], "/")
				if len(parts) > 1 {
					args = parts[1]
				}
			}
			msg := Message{
				ChannelType: TypeSlack,
				ChannelID:   payload.Event.Channel,
				UserID:      payload.Event.User,
				Text:        text,
				IsCommand:   isCmd,
				Command:     cmd,
				Args:        args,
			}
			if err := s.msgHandler(c.Request.Context(), msg); err != nil {
				slog.Error("slack message handler error", "error", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// verifySlackSignature verifies the request signature from Slack.
func (s *SlackAdapter) verifySlackSignature(headers http.Header, body []byte) bool {
	timestamp := headers.Get("X-Slack-Request-Timestamp")
	sig := headers.Get("X-Slack-Signature")
	if timestamp == "" || sig == "" {
		return false
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Now().Unix()-ts > 300 {
		return false
	}
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(s.signingSecret))
	mac.Write([]byte(baseString))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// SetMessageHandler sets the handler function for incoming Slack messages.
func (s *SlackAdapter) SetMessageHandler(h func(ctx context.Context, msg Message) error) {
	s.msgHandler = h
}
