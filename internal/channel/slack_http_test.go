package channel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

func TestSlackAdapter_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer xoxb-test" {
			t.Errorf("wrong auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content type: %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["channel"] != "C123" {
			t.Errorf("channel = %q, want C123", body["channel"])
		}
		if body["text"] != "hello" {
			t.Errorf("text = %q, want hello", body["text"])
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.Send(context.Background(), "C123", "hello")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestSlackAdapter_Send_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.Send(context.Background(), "C999", "hello")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !containsString(err.Error(), "channel_not_found") {
		t.Errorf("error should contain 'channel_not_found', got: %v", err)
	}
}

func TestSlackAdapter_Send_HTTPError(t *testing.T) {
	// Use a closed server to trigger HTTP error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.Send(context.Background(), "C123", "hello")
	if err == nil {
		t.Fatal("expected error for HTTP failure")
	}
}

func TestSlackAdapter_SendToUser_Success(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// conversations.open call
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":      true,
				"channel": map[string]string{"id": "D_DM_123"},
			})
		} else {
			// chat.postMessage call
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["channel"] != "D_DM_123" {
				t.Errorf("should send to DM channel, got %q", body["channel"])
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		}
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.SendToUser(context.Background(), "U_USER_1", "hello from DM")
	if err != nil {
		t.Fatalf("SendToUser: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (openDM + postMessage), got %d", callCount)
	}
}

func TestSlackAdapter_SendToUser_OpenDMFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "user_not_found",
		})
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.SendToUser(context.Background(), "U_UNKNOWN", "hello")
	if err == nil {
		t.Fatal("expected error when openDM fails")
	}
}

func TestSlackAdapter_Broadcast(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// All calls succeed (alternate between openDM and postMessage)
		if calls%2 == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":      true,
				"channel": map[string]string{"id": "D_DM"},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		}
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.Broadcast(context.Background(), []string{"U1", "U2", "U3"}, "broadcast msg")
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	// 3 users × 2 calls each (openDM + postMessage) = 6 calls
	if calls != 6 {
		t.Errorf("expected 6 API calls, got %d", calls)
	}
}

func TestSlackAdapter_StartStop(t *testing.T) {
	adapter, err := channel.NewSlackAdapter(channel.SlackConfig{BotToken: "xoxb-test"})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- adapter.Start(ctx)
	}()

	// Stop should unblock Start
	cancel()
	err = <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestSlackAdapter_StopUnblocksStart(t *testing.T) {
	adapter, err := channel.NewSlackAdapter(channel.SlackConfig{BotToken: "xoxb-test"})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- adapter.Start(context.Background())
	}()

	// Stop via Stop() method
	adapter.Stop()
	err = <-done
	if err != nil {
		t.Errorf("expected nil error from Stop, got %v", err)
	}
}

func TestSlackAdapter_SendMessage_BackwardCompat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.SendMessage("C123", "legacy message")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
}

func TestSlackAdapter_Send_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	adapter := newSlackAdapterWithURL(t, "xoxb-test", srv.URL)
	err := adapter.Send(context.Background(), "C123", "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

// newSlackAdapterWithURL creates a Slack adapter pointing to a test server.
// We inject the URL by replacing the HTTP client's transport.
func newSlackAdapterWithURL(t *testing.T, token, baseURL string) *channel.SlackAdapter {
	t.Helper()
	adapter, err := channel.NewSlackAdapterWithHTTPClient(channel.SlackConfig{
		BotToken: token,
	}, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && searchStr(s, sub))
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
