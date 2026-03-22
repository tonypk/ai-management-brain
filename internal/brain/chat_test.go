package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// --- Mocks ---

type mockChatLLM struct {
	response string
	err      error
	calls    []mockChatCall
	mu       sync.Mutex
}

type mockChatCall struct {
	SystemPrompt string
	History      []ChatMessage
	UserMessage  string
}

func (m *mockChatLLM) ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockChatCall{systemPrompt, history, userMessage})
	return m.response, m.err
}

type mockChatRedis struct {
	data map[string]string
	ttls map[string]time.Duration
	incr map[string]int64
	mu   sync.Mutex
}

func newMockChatRedis() *mockChatRedis {
	return &mockChatRedis{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
		incr: make(map[string]int64),
	}
}

func (m *mockChatRedis) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("redis: nil")
	}
	return v, nil
}

func (m *mockChatRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = fmt.Sprint(value)
	m.ttls[key] = ttl
	return nil
}

func (m *mockChatRedis) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *mockChatRedis) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incr[key]++
	return m.incr[key], nil
}

func (m *mockChatRedis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttls[key] = ttl
	return nil
}

// --- Tests ---

func TestNewChatService(t *testing.T) {
	llm := &mockChatLLM{response: "hello"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})
	if svc == nil {
		t.Fatal("ChatService should not be nil")
	}
}

func TestChatService_HandleEmployee_Basic(t *testing.T) {
	llm := &mockChatLLM{response: "I recommend focusing on priorities."}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "How do I manage my team?")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "I recommend focusing on priorities." {
		t.Fatalf("unexpected response: %s", resp)
	}

	// Verify history was stored
	histKey := "chat:emp-1"
	stored, err := rdb.Get(context.Background(), histKey)
	if err != nil {
		t.Fatal("history should be stored in Redis")
	}
	var msgs []chatHistoryMessage
	if err := json.Unmarshal([]byte(stored), &msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user+assistant), got %d", len(msgs))
	}
}

func TestChatService_HandleEmployee_RateLimit(t *testing.T) {
	llm := &mockChatLLM{response: "ok"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	// Send 5 messages (should all succeed)
	for i := 0; i < 5; i++ {
		_, err := svc.HandleEmployee(context.Background(), "emp-rate", "tenant-1", "Test", "inamori", "default", "msg")
		if err != nil {
			t.Fatalf("message %d should succeed: %v", i, err)
		}
	}

	// 6th message should be rate-limited
	resp, err := svc.HandleEmployee(context.Background(), "emp-rate", "tenant-1", "Test", "inamori", "default", "msg")
	if err != nil {
		t.Fatal(err)
	}
	if resp != rateLimitMessage {
		t.Fatalf("expected rate limit message, got: %s", resp)
	}
}

func TestChatService_HandleEmployee_AIDisabled(t *testing.T) {
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           nil, // AI not configured
		Redis:         rdb,
		EngineFactory: factory,
	})

	resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if resp != aiDisabledMessage {
		t.Fatalf("expected AI disabled message, got: %s", resp)
	}
}

func TestChatService_HandleEmployee_LLMError(t *testing.T) {
	llm := &mockChatLLM{err: fmt.Errorf("api timeout")}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
	})

	resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "hello")
	if err != nil {
		t.Fatal(err) // HandleEmployee should not return error to caller
	}
	if resp != aiErrorMessage {
		t.Fatalf("expected error message, got: %s", resp)
	}
}

func TestChatService_HandleEmployee_AuthError(t *testing.T) {
	llm := &mockChatLLM{err: &AuthError{Msg: "401 unauthorized"}}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
	})

	resp, err := svc.HandleEmployee(context.Background(), "emp-1", "tenant-1", "Alice", "inamori", "default", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if resp != aiDisabledMessage {
		t.Fatalf("expected AI disabled message for auth error, got: %s", resp)
	}
}

func TestChatService_HandleBoss_Basic(t *testing.T) {
	llm := &mockChatLLM{response: "Based on today's data, the team is performing well."}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	resp, err := svc.HandleBoss(context.Background(), "tenant-1", "inamori", "default", "How is the team doing?",
		BossContext{
			LatestSummary:  "Good progress today.",
			SubmittedCount: 4,
			TotalEmployees: 5,
			EmployeeRoster: []RosterEntry{{Name: "Alice", Role: "member", IsActive: true}},
		})
	if err != nil {
		t.Fatal(err)
	}
	if resp == "" {
		t.Fatal("response should not be empty")
	}
}

func TestChatService_HandleBoss_NoRateLimit(t *testing.T) {
	llm := &mockChatLLM{response: "ok"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	// Boss should be able to send 10 messages without rate limiting
	for i := 0; i < 10; i++ {
		resp, err := svc.HandleBoss(context.Background(), "tenant-1", "inamori", "default", "msg",
			BossContext{TotalEmployees: 1, EmployeeRoster: []RosterEntry{{Name: "Alice", Role: "member", IsActive: true}}})
		if err != nil {
			t.Fatalf("boss message %d should succeed: %v", i, err)
		}
		if resp == rateLimitMessage {
			t.Fatalf("boss should not be rate limited at message %d", i)
		}
	}
}

func TestMatchEmployeeName(t *testing.T) {
	tests := []struct {
		text string
		name string
		want bool
	}{
		{"How is Alice doing?", "Alice Smith", true},
		{"how is alice doing?", "Alice Smith", true},
		{"Tell me about Bob", "Alice Smith", false},
		{"What about Smith?", "Alice Smith", true},
		{"", "Alice", false},
	}
	for _, tt := range tests {
		got := matchEmployeeName(tt.text, tt.name)
		if got != tt.want {
			t.Errorf("matchEmployeeName(%q, %q) = %v, want %v", tt.text, tt.name, got, tt.want)
		}
	}
}

func TestChatService_GapDetection(t *testing.T) {
	rdb := newMockChatRedis()
	svc := &ChatService{redis: rdb}

	old := []chatHistoryMessage{
		{Role: "user", Content: "hi", TS: time.Now().Add(-7 * time.Hour)},
		{Role: "assistant", Content: "hello", TS: time.Now().Add(-7 * time.Hour)},
	}
	data, _ := json.Marshal(old)
	rdb.Set(context.Background(), "chat:emp-gap", string(data), historyTTL)

	result := svc.checkGapAndTrim(context.Background(), "chat:emp-gap", old, "emp-gap", "tenant-1")
	if len(result) != 0 {
		t.Fatalf("expected empty history after gap, got %d messages", len(result))
	}
}

func TestChatService_HistoryTrimming(t *testing.T) {
	llm := &mockChatLLM{response: "ok"}
	rdb := newMockChatRedis()
	factory := NewEngineFactory()
	svc := NewChatService(ChatServiceConfig{
		LLM:           llm,
		Redis:         rdb,
		EngineFactory: factory,
		BossTgID:      12345,
	})

	// Send 6 messages (each creates 2 entries: user + assistant = 12, trimmed to 10)
	for i := 0; i < 6; i++ {
		_, _ = svc.HandleEmployee(context.Background(), "emp-trim", "tenant-1", "Test", "inamori", "default", fmt.Sprintf("msg %d", i))
	}

	history, _ := svc.loadHistory(context.Background(), "chat:emp-trim")
	if len(history) > maxHistoryMessages {
		t.Fatalf("expected max %d messages, got %d", maxHistoryMessages, len(history))
	}
}

func TestAIDisabledMessage(t *testing.T) {
	if AIDisabledMessage() != aiDisabledMessage {
		t.Fatal("AIDisabledMessage should return the constant")
	}
}

func TestAIErrorMessage(t *testing.T) {
	if AIErrorMessage() != aiErrorMessage {
		t.Fatal("AIErrorMessage should return the constant")
	}
}
