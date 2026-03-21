package channel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSignalAdapter_Type(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{})
	if adapter.Type() != TypeSignal {
		t.Errorf("Type() = %q, want %q", adapter.Type(), TypeSignal)
	}
}

func TestSignalAdapter_Send(t *testing.T) {
	var gotBody signalSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/send" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+639111111111",
	})

	err := adapter.Send(context.Background(), "+639222222222", "Hello Signal")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotBody.Message != "Hello Signal" {
		t.Errorf("message = %q, want %q", gotBody.Message, "Hello Signal")
	}
	if gotBody.Number != "+639111111111" {
		t.Errorf("number = %q, want %q", gotBody.Number, "+639111111111")
	}
	if len(gotBody.Recipients) != 1 || gotBody.Recipients[0] != "+639222222222" {
		t.Errorf("recipients = %v, want [+639222222222]", gotBody.Recipients)
	}
}

func TestSignalAdapter_SendToUser(t *testing.T) {
	var gotBody signalSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+639111111111",
	})

	err := adapter.SendToUser(context.Background(), "+639333333333", "Direct message")
	if err != nil {
		t.Fatalf("SendToUser: %v", err)
	}

	if gotBody.Recipients[0] != "+639333333333" {
		t.Errorf("recipient = %q, want %q", gotBody.Recipients[0], "+639333333333")
	}
}

func TestSignalAdapter_Send_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid number"}`))
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+639111111111",
	})

	err := adapter.Send(context.Background(), "invalid", "test")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Errorf("error should contain status code, got: %v", err)
	}
}

func TestSignalAdapter_Broadcast(t *testing.T) {
	sentTo := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body signalSendRequest
		json.NewDecoder(r.Body).Decode(&body)
		sentTo = append(sentTo, body.Recipients...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+639111111111",
	})

	err := adapter.Broadcast(context.Background(), []string{"+639222222222", "+639333333333"}, "broadcast msg")
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if len(sentTo) != 2 {
		t.Errorf("sent to %d recipients, want 2", len(sentTo))
	}
}

func TestSignalAdapter_HandleWebhook(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+639111111111"})

	var receivedMsg Message
	adapter.OnMessage(func(ctx context.Context, msg Message) error {
		receivedMsg = msg
		return nil
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	payload := `{
		"envelope": {
			"source": "+639222222222",
			"sourceNumber": "+639222222222",
			"sourceName": "Test User",
			"dataMessage": {
				"message": "Today I completed the API integration.",
				"timestamp": 1234567890
			}
		},
		"account": "+639111111111"
	}`

	r.POST("/webhook", adapter.HandleWebhook)
	c.Request = httptest.NewRequest("POST", "/webhook", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if receivedMsg.UserID != "+639222222222" {
		t.Errorf("userID = %q, want +639222222222", receivedMsg.UserID)
	}
	if receivedMsg.Text != "Today I completed the API integration." {
		t.Errorf("text = %q, want report text", receivedMsg.Text)
	}
	if receivedMsg.ChannelType != TypeSignal {
		t.Errorf("channelType = %q, want signal", receivedMsg.ChannelType)
	}
	if receivedMsg.IsCommand {
		t.Error("should not be a command")
	}
}

func TestSignalAdapter_HandleWebhook_Command(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+639111111111"})

	var receivedMsg Message
	adapter.OnMessage(func(ctx context.Context, msg Message) error {
		receivedMsg = msg
		return nil
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	payload := `{
		"envelope": {
			"sourceNumber": "+639222222222",
			"dataMessage": {"message": "/join abc123", "timestamp": 1234567890}
		}
	}`

	r.POST("/webhook", adapter.HandleWebhook)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if !receivedMsg.IsCommand {
		t.Error("should be a command")
	}
	if receivedMsg.Command != "join" {
		t.Errorf("command = %q, want join", receivedMsg.Command)
	}
	if receivedMsg.Args != "abc123" {
		t.Errorf("args = %q, want abc123", receivedMsg.Args)
	}
}

func TestSignalAdapter_HandleWebhook_EmptyMessage(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+639111111111"})

	handlerCalled := false
	adapter.OnMessage(func(ctx context.Context, msg Message) error {
		handlerCalled = true
		return nil
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	payload := `{
		"envelope": {
			"sourceNumber": "+639222222222",
			"dataMessage": {"message": "", "timestamp": 1234567890}
		}
	}`

	r.POST("/webhook", adapter.HandleWebhook)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called for empty message")
	}
}

func TestSignalAdapter_HandleWebhook_NoHandler(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+639111111111"})
	// No handler registered — should not panic

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	payload := `{
		"envelope": {
			"sourceNumber": "+639222222222",
			"dataMessage": {"message": "hello", "timestamp": 1234567890}
		}
	}`

	r.POST("/webhook", adapter.HandleWebhook)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even without handler", w.Code)
	}
}

func TestSignalAdapter_HandleWebhook_InvalidJSON(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+639111111111"})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/webhook", adapter.HandleWebhook)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid JSON", w.Code)
	}
}
