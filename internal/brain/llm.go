package brain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// LLMClient abstracts the LLM chat interface for testing.
type LLMClient interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// ChatMessage represents a single message in a multi-turn conversation.
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ChatLLMClient extends LLM capabilities with multi-turn conversation.
type ChatLLMClient interface {
	ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error)
}

// ReportData holds one employee's report for summary generation.
type ReportData struct {
	EmployeeName string
	Answers      map[string]string
}

// AuthError indicates an authentication failure (401/403) that should NOT be retried.
type AuthError struct {
	Msg string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error: %s", e.Msg)
}

// IsAuthError checks if the given error is an AuthError.
func IsAuthError(err error) bool {
	var authErr *AuthError
	return errors.As(err, &authErr)
}

// AnthropicClient is the real Claude API client.
type AnthropicClient struct {
	client *anthropic.Client
	model  anthropic.Model
}

// NewAnthropicClient creates a real Anthropic client.
func NewAnthropicClient(apiKey string) (*AnthropicClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicClient{
		client: &client,
		model:  anthropic.ModelClaudeSonnet4_20250514,
	}, nil
}

// chatWithTokens sends a message to Claude with a configurable max token limit.
func (a *AnthropicClient) chatWithTokens(ctx context.Context, systemPrompt, userPrompt string, maxTokens int64) (string, error) {
	start := time.Now()
	var lastErr error

	// Retry up to 3 times with exponential backoff for transient errors.
	backoffs := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for attempt := 0; attempt <= 2; attempt++ {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: maxTokens,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
			},
		})
		if err != nil {
			// Check for auth errors — do NOT retry.
			errMsg := err.Error()
			if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") ||
				strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "unauthorized") {
				return "", &AuthError{Msg: errMsg}
			}

			lastErr = err
			slog.Warn("LLM API call failed",
				"attempt", attempt+1,
				"error", err,
				"duration", time.Since(start),
			)

			if attempt < 2 {
				time.Sleep(backoffs[attempt])
			}
			continue
		}

		// Extract text from response content blocks.
		var result string
		for _, block := range resp.Content {
			if block.Type == "text" {
				result += block.Text
			}
		}

		slog.Info("LLM API call succeeded",
			"duration", time.Since(start),
			"input_tokens", resp.Usage.InputTokens,
			"output_tokens", resp.Usage.OutputTokens,
		)
		return result, nil
	}

	return "", fmt.Errorf("LLM API failed after 3 attempts: %w", lastErr)
}

// Chat sends a message to Claude and returns the response.
func (a *AnthropicClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.chatWithTokens(ctx, systemPrompt, userPrompt, 1024)
}

// ChatLong sends a message to Claude with higher token limit for complex generation.
func (a *AnthropicClient) ChatLong(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.chatWithTokens(ctx, systemPrompt, userPrompt, 4096)
}

// ChatWithHistory sends a multi-turn conversation to Claude and returns the response.
func (a *AnthropicClient) ChatWithHistory(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	start := time.Now()
	var lastErr error

	// Build messages array from history + new user message
	messages := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	backoffs := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for attempt := 0; attempt <= 2; attempt++ {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 1024,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: messages,
		})
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") ||
				strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "unauthorized") {
				return "", &AuthError{Msg: errMsg}
			}
			lastErr = err
			slog.Warn("LLM ChatWithHistory API call failed",
				"attempt", attempt+1,
				"error", err,
				"duration", time.Since(start),
			)
			if attempt < 2 {
				time.Sleep(backoffs[attempt])
			}
			continue
		}

		var result string
		for _, block := range resp.Content {
			if block.Type == "text" {
				result += block.Text
			}
		}
		slog.Info("LLM ChatWithHistory succeeded",
			"duration", time.Since(start),
			"input_tokens", resp.Usage.InputTokens,
			"output_tokens", resp.Usage.OutputTokens,
			"history_len", len(history),
		)
		return result, nil
	}

	return "", fmt.Errorf("LLM ChatWithHistory failed after 3 attempts: %w", lastErr)
}

// LLMService wraps an LLMClient with domain-specific methods.
type LLMService struct {
	client LLMClient
}

// NewLLMService creates a new LLM service wrapping the given client.
func NewLLMService(client LLMClient) *LLMService {
	return &LLMService{client: client}
}

// GenerateChaseMessage generates a personalized chase message.
func (s *LLMService) GenerateChaseMessage(ctx context.Context, systemPrompt, employeeName, tone string) (string, error) {
	userPrompt := fmt.Sprintf(
		"Generate a chase message for employee %q with tone %q. "+
			"This is a reminder for them to submit their daily report. "+
			"Keep it brief (1-2 sentences).",
		employeeName, tone,
	)
	return s.client.Chat(ctx, systemPrompt, userPrompt)
}

// ReportAnalysis holds extracted blockers and sentiment from a report.
type ReportAnalysis struct {
	Blockers  string `json:"blockers"`
	Sentiment string `json:"sentiment"`
}

// AnalyzeReport extracts blockers and sentiment from report answers.
func (s *LLMService) AnalyzeReport(ctx context.Context, answerText string) (*ReportAnalysis, error) {
	systemPrompt := `You are an AI assistant that analyzes employee daily reports.
Extract two things:
1. BLOCKERS: Any obstacles, problems, or blockers mentioned. If none, return empty string.
2. SENTIMENT: Classify the overall tone as one of: positive, neutral, negative, stressed.

Respond in this exact JSON format only, no extra text:
{"blockers": "...", "sentiment": "..."}`

	userPrompt := fmt.Sprintf("Analyze this employee's daily report answers:\n\n%s", answerText)

	resp, err := s.client.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analyze: %w", err)
	}

	// Parse JSON response
	resp = strings.TrimSpace(resp)
	// Handle markdown code blocks
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result ReportAnalysis
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Fallback: treat entire response as blockers with neutral sentiment
		slog.Warn("failed to parse LLM analysis JSON", "response", resp, "error", err)
		return &ReportAnalysis{Blockers: "", Sentiment: "neutral"}, nil
	}

	return &result, nil
}

// GenerateSummary generates an AI summary of all submitted reports.
func (s *LLMService) GenerateSummary(ctx context.Context, systemPrompt string, reports []ReportData) (string, error) {
	var sb strings.Builder
	sb.WriteString("Generate a daily team summary based on these reports:\n\n")
	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("**%s:**\n", r.EmployeeName))
		for q, a := range r.Answers {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", q, a))
		}
		sb.WriteString("\n")
	}
	return s.client.Chat(ctx, systemPrompt, sb.String())
}
