package brain

import (
	"context"
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

// Chat sends a message to Claude and returns the response.
func (a *AnthropicClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	start := time.Now()
	var lastErr error

	// Retry up to 3 times with exponential backoff for transient errors.
	backoffs := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for attempt := 0; attempt <= 2; attempt++ {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 1024,
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
