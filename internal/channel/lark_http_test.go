package channel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

func newLarkAdapterWithURL(t *testing.T, baseURL string) *channel.LarkAdapter {
	t.Helper()
	adapter, err := channel.NewLarkAdapterWithBaseURL(channel.LarkConfig{
		AppID:     "cli_test",
		AppSecret: "secret_test",
	}, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func TestLarkAdapter_Send_Success(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// Token request
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"msg":                 "ok",
				"tenant_access_token": "t-test-token",
				"expire":              7200,
			})
		} else {
			// Message request
			if r.Header.Get("Authorization") != "Bearer t-test-token" {
				t.Errorf("wrong auth: %s", r.Header.Get("Authorization"))
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "ok"})
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Send(context.Background(), "oc_CHAT123", "hello lark")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls (token + message), got %d", callCount)
	}
}

func TestLarkAdapter_Send_TokenCached(t *testing.T) {
	var tokenCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/v3/tenant_access_token/internal" {
			atomic.AddInt32(&tokenCalls, 1)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"msg":                 "ok",
				"tenant_access_token": "t-cached",
				"expire":              7200,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "ok"})
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)

	// First send → token fetch
	adapter.Send(context.Background(), "oc_1", "msg1")
	// Second send → should reuse cached token
	adapter.Send(context.Background(), "oc_2", "msg2")

	if atomic.LoadInt32(&tokenCalls) != 1 {
		t.Errorf("expected 1 token call (cached), got %d", tokenCalls)
	}
}

func TestLarkAdapter_Send_TokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 10003,
			"msg":  "invalid app_id",
		})
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Send(context.Background(), "oc_1", "hello")
	if err == nil {
		t.Fatal("expected error for token failure")
	}
	if !containsString(err.Error(), "invalid app_id") {
		t.Errorf("error should contain 'invalid app_id', got: %v", err)
	}
}

func TestLarkAdapter_Send_MessageError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			// Token OK
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"tenant_access_token": "t-ok",
				"expire":              7200,
			})
		} else {
			// Message error
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 230001,
				"msg":  "invalid receive_id",
			})
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Send(context.Background(), "invalid", "hello")
	if err == nil {
		t.Fatal("expected error for message send failure")
	}
}

func TestLarkAdapter_SendToUser(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"tenant_access_token": "t-ok",
				"expire":              7200,
			})
		} else {
			// Verify receive_id_type=open_id
			if r.URL.Query().Get("receive_id_type") != "open_id" {
				t.Errorf("expected receive_id_type=open_id, got %s", r.URL.Query().Get("receive_id_type"))
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0})
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.SendToUser(context.Background(), "ou_USER123", "hello user")
	if err != nil {
		t.Fatalf("SendToUser: %v", err)
	}
}

func TestLarkAdapter_Broadcast(t *testing.T) {
	var msgCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/v3/tenant_access_token/internal" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"tenant_access_token": "t-ok",
				"expire":              7200,
			})
		} else {
			atomic.AddInt32(&msgCalls, 1)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0})
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Broadcast(context.Background(), []string{"ou_1", "ou_2", "ou_3"}, "announcement")
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if atomic.LoadInt32(&msgCalls) != 3 {
		t.Errorf("expected 3 message calls, got %d", msgCalls)
	}
}

func TestLarkAdapter_StartStop(t *testing.T) {
	adapter, err := channel.NewLarkAdapter(channel.LarkConfig{
		AppID:     "cli_test",
		AppSecret: "secret_test",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- adapter.Start(ctx)
	}()

	cancel()
	err = <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestLarkAdapter_StopUnblocksStart(t *testing.T) {
	adapter, err := channel.NewLarkAdapter(channel.LarkConfig{
		AppID:     "cli_test",
		AppSecret: "secret_test",
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- adapter.Start(context.Background())
	}()

	adapter.Stop()
	err = <-done
	if err != nil {
		t.Errorf("expected nil error from Stop, got %v", err)
	}
}

func TestLarkAdapter_Send_HTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // Close immediately to cause connection errors

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Send(context.Background(), "oc_1", "hello")
	if err == nil {
		t.Fatal("expected error for HTTP failure")
	}
}

func TestLarkAdapter_Send_InvalidJSONResponse(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":                0,
				"tenant_access_token": "t-ok",
				"expire":              7200,
			})
		} else {
			w.Write([]byte("not json"))
		}
	}))
	defer srv.Close()

	adapter := newLarkAdapterWithURL(t, srv.URL)
	err := adapter.Send(context.Background(), "oc_1", "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
