package brain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// mockLLM implements brain.LLMClient for testing
type mockLLM struct {
	response string
	err      error
	calls    int
}

func (m *mockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	m.calls++
	return m.response, m.err
}

func TestLLM_GenerateChaseMessage(t *testing.T) {
	mock := &mockLLM{response: "Hi! Just a friendly reminder to submit your daily report."}
	svc := brain.NewLLMService(mock)

	msg, err := svc.GenerateChaseMessage(context.Background(), "system prompt", "John", "warm_reminder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestLLM_GenerateSummary(t *testing.T) {
	mock := &mockLLM{response: "## Daily Summary\n3/5 employees submitted..."}
	svc := brain.NewLLMService(mock)

	summary, err := svc.GenerateSummary(context.Background(), "system prompt", []brain.ReportData{
		{EmployeeName: "Alice", Answers: map[string]string{"q1": "did X"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestLLM_ErrorReturned(t *testing.T) {
	mock := &mockLLM{err: errors.New("api error")}
	svc := brain.NewLLMService(mock)

	_, err := svc.GenerateChaseMessage(context.Background(), "prompt", "John", "warm")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLLM_AuthError_Classification(t *testing.T) {
	authErr := &brain.AuthError{Msg: "invalid api key"}
	if !brain.IsAuthError(authErr) {
		t.Error("should detect auth error")
	}
	if brain.IsAuthError(errors.New("timeout")) {
		t.Error("timeout should not be auth error")
	}
}

func TestLLM_AnalyzeReport_ValidJSON(t *testing.T) {
	mock := &mockLLM{response: `{"blockers": "waiting on API", "sentiment": "stressed"}`}
	svc := brain.NewLLMService(mock)

	result, err := svc.AnalyzeReport(context.Background(), "I'm blocked on the API integration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blockers != "waiting on API" {
		t.Errorf("expected blockers 'waiting on API', got %q", result.Blockers)
	}
	if result.Sentiment != "stressed" {
		t.Errorf("expected sentiment 'stressed', got %q", result.Sentiment)
	}
}

func TestLLM_AnalyzeReport_MarkdownWrapped(t *testing.T) {
	mock := &mockLLM{response: "```json\n{\"blockers\": \"\", \"sentiment\": \"positive\"}\n```"}
	svc := brain.NewLLMService(mock)

	result, err := svc.AnalyzeReport(context.Background(), "All good today!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sentiment != "positive" {
		t.Errorf("expected sentiment 'positive', got %q", result.Sentiment)
	}
}

func TestLLM_AnalyzeReport_InvalidJSON_Fallback(t *testing.T) {
	mock := &mockLLM{response: "not json at all"}
	svc := brain.NewLLMService(mock)

	result, err := svc.AnalyzeReport(context.Background(), "some report")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sentiment != "neutral" {
		t.Errorf("expected fallback sentiment 'neutral', got %q", result.Sentiment)
	}
}
