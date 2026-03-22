package channel

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleLarkEvent processes Lark event subscription callbacks.
func (l *LarkAdapter) HandleLarkEvent(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	var envelope struct {
		Encrypt   string `json:"encrypt"`
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
		Header    struct {
			EventType string `json:"event_type"`
			Token     string `json:"token"`
		} `json:"header"`
		Event json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// URL verification challenge (Event Subscription v1)
	if envelope.Type == "url_verification" {
		c.JSON(http.StatusOK, gin.H{"challenge": envelope.Challenge})
		return
	}

	// Verify token if set
	token := envelope.Token
	if token == "" {
		token = envelope.Header.Token
	}
	if l.verificationToken != "" && token != l.verificationToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Process message events (Event Subscription v2)
	eventType := envelope.Header.EventType
	if eventType == "im.message.receive_v1" && l.msgHandler != nil {
		var msgEvent struct {
			Sender struct {
				SenderID struct {
					OpenID string `json:"open_id"`
				} `json:"sender_id"`
			} `json:"sender"`
			Message struct {
				ChatID      string `json:"chat_id"`
				MessageType string `json:"message_type"`
				Content     string `json:"content"` // JSON string
			} `json:"message"`
		}
		if err := json.Unmarshal(envelope.Event, &msgEvent); err != nil {
			slog.Error("parse lark message event", "error", err)
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		// Extract text content
		var text string
		if msgEvent.Message.MessageType == "text" {
			var content struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(msgEvent.Message.Content), &content); err == nil {
				text = content.Text
			}
		}

		if text != "" {
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
				ChannelType: TypeLark,
				ChannelID:   msgEvent.Message.ChatID,
				UserID:      msgEvent.Sender.SenderID.OpenID,
				Text:        text,
				IsCommand:   isCmd,
				Command:     cmd,
				Args:        args,
			}
			if err := l.msgHandler(c.Request.Context(), msg); err != nil {
				slog.Error("lark message handler error", "error", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SetMessageHandler sets the handler function for incoming Lark messages.
func (l *LarkAdapter) SetMessageHandler(h func(ctx context.Context, msg Message) error) {
	l.msgHandler = h
}
