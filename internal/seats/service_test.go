package seats

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/brain"
)

// mockRedis implements RedisClient for testing
type mockRedis struct {
	data map[string]string
}

func newMockRedis() *mockRedis {
	return &mockRedis{data: make(map[string]string)}
}

func (m *mockRedis) Get(_ context.Context, key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return val, nil
}

func (m *mockRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	m.data[key] = fmt.Sprintf("%v", value)
	return nil
}

func (m *mockRedis) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

// mockLLM implements ChatLLMClient for testing
type mockLLM struct {
	historyResponse string
	callCount       int
	lastSystem      string
}

func (m *mockLLM) ChatWithHistory(_ context.Context, systemPrompt string, history []brain.ChatMessage, userMessage string) (string, error) {
	m.callCount++
	m.lastSystem = systemPrompt
	return m.historyResponse, nil
}

func TestSetAndGetActiveSeat(t *testing.T) {
	redis := newMockRedis()
	svc := &SeatService{redis: redis}
	ctx := context.Background()

	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	userID := int64(12345)

	// No active seat initially
	seat := svc.GetActiveSeat(ctx, tenantID, userID)
	if seat != "" {
		t.Errorf("expected empty, got %q", seat)
	}

	// Set active seat
	err := svc.SetActiveSeat(ctx, tenantID, userID, "cmo")
	if err != nil {
		t.Fatalf("SetActiveSeat: %v", err)
	}

	// Get active seat
	seat = svc.GetActiveSeat(ctx, tenantID, userID)
	if seat != "cmo" {
		t.Errorf("expected cmo, got %q", seat)
	}

	// Clear active seat
	err = svc.ClearActiveSeat(ctx, tenantID, userID)
	if err != nil {
		t.Fatalf("ClearActiveSeat: %v", err)
	}
	seat = svc.GetActiveSeat(ctx, tenantID, userID)
	if seat != "" {
		t.Errorf("expected empty after clear, got %q", seat)
	}
}

func TestChatNilLLM(t *testing.T) {
	svc := &SeatService{llm: nil}
	reply, err := svc.Chat(context.Background(), "tenant", "ceo", "default", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "AI features are not enabled." {
		t.Errorf("expected disabled message, got %q", reply)
	}
}

func TestBoardDiscussNilLLM(t *testing.T) {
	svc := &SeatService{llm: nil}
	_, _, err := svc.BoardDiscuss(context.Background(), "tenant", "default", "topic")
	if err == nil || err.Error() != "AI features are not enabled" {
		t.Errorf("expected AI not enabled error, got %v", err)
	}
}

func TestBoardDiscussRateLimit(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	redis := newMockRedis()
	redis.data["board_rate:"+tenantID] = "1" // already rate limited

	svc := &SeatService{
		llm:   &mockLLM{},
		redis: redis,
	}

	_, _, err := svc.BoardDiscuss(context.Background(), tenantID, "default", "topic")
	if err == nil || !strings.Contains(err.Error(), "limited") {
		t.Errorf("expected rate limit error, got %v", err)
	}
}

func TestParseUUID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	u, err := parseUUID(valid)
	if err != nil {
		t.Fatalf("parseUUID(%q): %v", valid, err)
	}
	if !u.Valid {
		t.Error("expected valid UUID")
	}

	_, err = parseUUID("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestFormatUUID(t *testing.T) {
	u := pgtype.UUID{Valid: false}
	if got := formatUUID(u); got != "" {
		t.Errorf("expected empty for invalid UUID, got %q", got)
	}
}

func TestBuildSynthesisPrompt(t *testing.T) {
	responses := []BoardResponse{
		{SeatType: "ceo", Title: "CEO", PersonaID: "inamori", Content: "We should expand."},
		{SeatType: "cfo", Title: "CFO", PersonaID: "buffett", Content: "We need to check finances."},
		{SeatType: "cmo", Title: "CMO", PersonaID: "trout", Content: "[unavailable \u2014 AI response failed]"},
	}

	prompt := BuildSynthesisPrompt("Enter SEA market?", responses)

	if !strings.Contains(prompt, "CEO") {
		t.Error("expected CEO in synthesis prompt")
	}
	if !strings.Contains(prompt, "CFO") {
		t.Error("expected CFO in synthesis prompt")
	}
	// Unavailable responses should be skipped
	if strings.Contains(prompt, "CMO") {
		t.Error("unavailable CMO should be excluded from synthesis")
	}
	if !strings.Contains(prompt, "Enter SEA market?") {
		t.Error("expected topic in prompt")
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	redis := newMockRedis()
	svc := &SeatService{redis: redis}
	ctx := context.Background()
	key := "seat:test:ceo"

	// Empty history
	history := svc.loadHistory(ctx, key)
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d", len(history))
	}

	// Save history
	msgs := []brain.ChatMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	svc.saveHistory(ctx, key, msgs)

	// Load it back
	loaded := svc.loadHistory(ctx, key)
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Role != "user" || loaded[0].Content != "hello" {
		t.Errorf("unexpected first message: %+v", loaded[0])
	}
	if loaded[1].Role != "assistant" || loaded[1].Content != "hi there" {
		t.Errorf("unexpected second message: %+v", loaded[1])
	}
}
