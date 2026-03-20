package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ConversationState represents the state of a report collection conversation.
type ConversationState string

const (
	StateIdle       ConversationState = "idle"
	StateCollecting ConversationState = "collecting"
	StateConfirming ConversationState = "confirming"
	StateComplete   ConversationState = "complete"
)

const conversationTTL = 4 * time.Hour

// RedisClient defines the Redis operations needed by the collector.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// conversationData stored in Redis for each employee's active conversation.
type conversationData struct {
	State           ConversationState `json:"state"`
	CurrentQuestion int               `json:"current_question"`
	Answers         map[string]string `json:"answers"`
	Questions       []string          `json:"questions"`
}

// Collector manages multi-turn report collection conversations via Redis.
type Collector struct {
	redis     RedisClient
	questions []string
}

// NewCollector creates a new report collector.
func NewCollector(redis RedisClient, questions []string) *Collector {
	return &Collector{redis: redis, questions: questions}
}

func redisKey(employeeID string) string {
	return fmt.Sprintf("conv:%s", employeeID)
}

// Start begins a new conversation for the employee.
func (c *Collector) Start(ctx context.Context, employeeID string) (ConversationState, string, error) {
	data := conversationData{
		State:           StateCollecting,
		CurrentQuestion: 0,
		Answers:         make(map[string]string),
		Questions:       c.questions,
	}

	if err := c.saveConversation(ctx, employeeID, data); err != nil {
		return StateIdle, "", err
	}

	slog.Info("conversation started", "employee_id", employeeID)
	return StateCollecting, c.questions[0], nil
}

// HandleAnswer processes an answer to the current question.
func (c *Collector) HandleAnswer(ctx context.Context, employeeID, answer string) (ConversationState, string, error) {
	data, err := c.loadConversation(ctx, employeeID)
	if err != nil {
		return StateIdle, "", nil // No active conversation
	}

	if data.State != StateCollecting {
		return data.State, "", nil
	}

	// Store the answer with key "q{N}" (1-based)
	qKey := fmt.Sprintf("q%d", data.CurrentQuestion+1)
	data.Answers[qKey] = answer
	data.CurrentQuestion++

	// Check if all questions answered
	if data.CurrentQuestion >= len(data.Questions) {
		data.State = StateConfirming
		if err := c.saveConversation(ctx, employeeID, *data); err != nil {
			return StateIdle, "", err
		}
		summary := c.buildSummary(data)
		return StateConfirming, summary, nil
	}

	// More questions to ask
	if err := c.saveConversation(ctx, employeeID, *data); err != nil {
		return StateIdle, "", err
	}
	return StateCollecting, data.Questions[data.CurrentQuestion], nil
}

// Confirm confirms the report and moves to complete state.
func (c *Collector) Confirm(ctx context.Context, employeeID string) (ConversationState, string, error) {
	data, err := c.loadConversation(ctx, employeeID)
	if err != nil {
		return StateIdle, "", nil
	}

	data.State = StateComplete
	if err := c.saveConversation(ctx, employeeID, *data); err != nil {
		return StateIdle, "", err
	}

	slog.Info("report confirmed", "employee_id", employeeID)
	return StateComplete, "Report saved! Thank you.", nil
}

// GetAnswers returns the answers collected so far.
func (c *Collector) GetAnswers(ctx context.Context, employeeID string) map[string]string {
	data, err := c.loadConversation(ctx, employeeID)
	if err != nil {
		return nil
	}
	return data.Answers
}

// IsCollecting returns true if the employee has an active collecting conversation.
func (c *Collector) IsCollecting(ctx context.Context, employeeID string) bool {
	data, err := c.loadConversation(ctx, employeeID)
	if err != nil {
		return false
	}
	return data.State == StateCollecting
}

// GetState returns the current conversation state, or StateIdle if none.
func (c *Collector) GetState(ctx context.Context, employeeID string) ConversationState {
	data, err := c.loadConversation(ctx, employeeID)
	if err != nil {
		return StateIdle
	}
	return data.State
}

// Cancel removes any active conversation for the employee.
func (c *Collector) Cancel(ctx context.Context, employeeID string) {
	c.redis.Del(ctx, redisKey(employeeID))
}

func (c *Collector) buildSummary(data *conversationData) string {
	var sb strings.Builder
	sb.WriteString("Here's your report summary:\n\n")
	for i, q := range data.Questions {
		qKey := fmt.Sprintf("q%d", i+1)
		sb.WriteString(fmt.Sprintf("**%s**\n%s\n\n", q, data.Answers[qKey]))
	}
	sb.WriteString("Reply 'confirm' to submit or 'edit' to start over.")
	return sb.String()
}

func (c *Collector) saveConversation(ctx context.Context, employeeID string, data conversationData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, redisKey(employeeID), string(b), conversationTTL)
}

func (c *Collector) loadConversation(ctx context.Context, employeeID string) (*conversationData, error) {
	raw, err := c.redis.Get(ctx, redisKey(employeeID))
	if err != nil {
		return nil, err
	}

	var data conversationData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("unmarshal conversation: %w", err)
	}
	return &data, nil
}
