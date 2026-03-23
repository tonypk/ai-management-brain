package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

const (
	maxHistoryMessages = 10
	historyTTL         = 24 * time.Hour
	rateLimitWindow    = 60 * time.Second
	rateLimitMax       = 5
	gapThreshold       = 6 * time.Hour
	rateLimitMessage   = "请稍等一下再继续对话"
	aiDisabledMessage  = "AI功能未启用，请联系管理员"
	aiErrorMessage     = "系统繁忙，请稍后再试"
)

// ChatRedisClient defines the Redis operations needed by ChatService.
type ChatRedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// chatHistoryMessage represents a single message stored in Redis history.
type chatHistoryMessage struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	TS      time.Time `json:"ts"`
}

// RosterEntry holds basic employee info for boss context.
type RosterEntry struct {
	ID       string
	Name     string
	JobTitle string
	Role     string
	IsActive bool
}

// BossContext holds team data fetched from DB for boss chat.
type BossContext struct {
	LatestSummary  string
	SubmittedCount int
	TotalEmployees int
	EmployeeRoster []RosterEntry
}

// EmployeeChatRequest holds all data needed for an employee chat message.
type EmployeeChatRequest struct {
	EmployeeID       string
	TenantID         string
	Name             string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
	MentorID         string
	CultureCode      string
	Text             string
}

// ChatServiceConfig holds dependencies for creating a ChatService.
type ChatServiceConfig struct {
	LLM           ChatLLMClient
	Redis         ChatRedisClient
	EngineFactory *EngineFactory
	BossTgID      int64
	MemoryEngine  *memory.MemoryEngine // optional, nil = no chat extraction
}

// ChatService orchestrates mentor chat for employees and boss.
type ChatService struct {
	llm           ChatLLMClient
	redis         ChatRedisClient
	engineFactory *EngineFactory
	bossTgID      int64
	memoryEngine  *memory.MemoryEngine
}

// NewChatService creates a new ChatService.
func NewChatService(cfg ChatServiceConfig) *ChatService {
	return &ChatService{
		llm:           cfg.LLM,
		redis:         cfg.Redis,
		engineFactory: cfg.EngineFactory,
		bossTgID:      cfg.BossTgID,
		memoryEngine:  cfg.MemoryEngine,
	}
}

// SetMemoryEngine injects the memory engine for chat extraction.
func (s *ChatService) SetMemoryEngine(me *memory.MemoryEngine) {
	s.memoryEngine = me
}

// AIDisabledMessage returns the user-facing message when AI is not configured.
func AIDisabledMessage() string { return aiDisabledMessage }

// AIErrorMessage returns the user-facing message when AI encounters an error.
func AIErrorMessage() string { return aiErrorMessage }

// HandleEmployee processes a chat message from an employee.
func (s *ChatService) HandleEmployee(ctx context.Context, req EmployeeChatRequest) (string, error) {
	if s.llm == nil {
		return aiDisabledMessage, nil
	}

	// Rate limiting
	if limited, err := s.checkRateLimit(ctx, req.EmployeeID); err != nil {
		slog.Error("rate limit check failed", "employee_id", req.EmployeeID, "error", err)
	} else if limited {
		return rateLimitMessage, nil
	}

	// Load engine for this tenant's mentor+culture
	engine, err := s.engineFactory.ForTenant(req.MentorID, req.CultureCode)
	if err != nil {
		engine, _ = s.engineFactory.ForTenant("inamori", "default")
	}

	// Load history and check for gap-based extraction
	history, err := s.loadHistory(ctx, chatKey(req.EmployeeID))
	if err != nil {
		slog.Warn("load chat history failed", "employee_id", req.EmployeeID, "error", err)
	}
	history = s.checkGapAndTrim(ctx, chatKey(req.EmployeeID), history, req.EmployeeID, req.TenantID)

	// Build system prompt with memory recall
	systemPrompt := engine.BuildEmployeeChatPrompt(ctx, req.TenantID, req.EmployeeID, EmployeeContext{
		Name:             req.Name,
		JobTitle:         req.JobTitle,
		Responsibilities: req.Responsibilities,
		Country:          req.Country,
		Language:         req.Language,
	}, req.Text)

	// Convert history to ChatMessage format
	chatHistory := historyToChatMessages(history)

	// Call LLM
	response, err := s.llm.ChatWithHistory(ctx, systemPrompt, chatHistory, req.Text)
	if err != nil {
		slog.Error("chat LLM call failed", "employee_id", req.EmployeeID, "error", err)
		if IsAuthError(err) {
			return aiDisabledMessage, nil
		}
		return aiErrorMessage, nil
	}

	// Append user message and assistant response to history
	now := time.Now()
	history = append(history,
		chatHistoryMessage{Role: "user", Content: req.Text, TS: now},
		chatHistoryMessage{Role: "assistant", Content: response, TS: now},
	)

	// Trim to max and save
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}
	if err := s.saveHistory(ctx, chatKey(req.EmployeeID), history); err != nil {
		slog.Error("save chat history failed", "employee_id", req.EmployeeID, "error", err)
	}

	return response, nil
}

// HandleBoss processes a chat message from the boss (chairman).
// Boss has no rate limit (per spec).
func (s *ChatService) HandleBoss(ctx context.Context, tenantID, mentorID, cultureCode, text string, bctx BossContext) (string, error) {
	if s.llm == nil {
		return aiDisabledMessage, nil
	}

	engine, err := s.engineFactory.ForTenant(mentorID, cultureCode)
	if err != nil {
		engine, _ = s.engineFactory.ForTenant("inamori", "default")
	}

	// Load boss history
	bossKey := bossHistoryKey(tenantID)
	history, err := s.loadHistory(ctx, bossKey)
	if err != nil {
		slog.Warn("load boss chat history failed", "tenant_id", tenantID, "error", err)
	}
	history = s.checkGapAndTrim(ctx, bossKey, history, "", tenantID)

	// Build employee roster text
	var rosterSB strings.Builder
	for i, emp := range bctx.EmployeeRoster {
		status := "active"
		if !emp.IsActive {
			status = "inactive"
		}
		if emp.JobTitle != "" {
			fmt.Fprintf(&rosterSB, "%d. %s - %s (%s, %s)\n", i+1, emp.Name, emp.JobTitle, emp.Role, status)
		} else {
			fmt.Fprintf(&rosterSB, "%d. %s (%s, %s)\n", i+1, emp.Name, emp.Role, status)
		}
	}

	// Check if boss mentions an employee by name — recall their memories
	memorySection := ""
	if me := engine.MemoryEngine(); me != nil && me.Enabled() {
		const maxRecall = 3
		matched := 0
		var memorySB strings.Builder
		for _, emp := range bctx.EmployeeRoster {
			if matched >= maxRecall {
				break
			}
			if matchEmployeeName(text, emp.Name) {
				result, err := me.RecallForMentor(ctx, tenantID, emp.ID, text)
				if err != nil {
					slog.Warn("boss memory recall failed", "employee", emp.Name, "error", err)
					continue
				}
				formatted := memory.FormatForPrompt(result)
				if formatted != "" {
					if matched > 0 {
						memorySB.WriteString("\n")
					}
					fmt.Fprintf(&memorySB, "<!-- Memories for %s -->\n%s", emp.Name, formatted)
					matched++
				}
			}
		}
		memorySection = memorySB.String()
	}

	// Build rate string
	rate := "0% (0/0)"
	if bctx.TotalEmployees > 0 {
		pct := float64(bctx.SubmittedCount) / float64(bctx.TotalEmployees) * 100
		rate = fmt.Sprintf("%.0f%% (%d/%d)", pct, bctx.SubmittedCount, bctx.TotalEmployees)
	}

	systemPrompt := engine.BuildBossPrompt(ctx, tenantID, BuildBossContext{
		LatestSummary:  bctx.LatestSummary,
		SubmissionRate: rate,
		EmployeeList:   rosterSB.String(),
		MemorySection:  memorySection,
	})

	chatHistory := historyToChatMessages(history)

	response, err := s.llm.ChatWithHistory(ctx, systemPrompt, chatHistory, text)
	if err != nil {
		slog.Error("boss chat LLM call failed", "tenant_id", tenantID, "error", err)
		if IsAuthError(err) {
			return aiDisabledMessage, nil
		}
		return aiErrorMessage, nil
	}

	now := time.Now()
	history = append(history,
		chatHistoryMessage{Role: "user", Content: text, TS: now},
		chatHistoryMessage{Role: "assistant", Content: response, TS: now},
	)
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}
	if err := s.saveHistory(ctx, bossKey, history); err != nil {
		slog.Error("save boss chat history failed", "tenant_id", tenantID, "error", err)
	}

	return response, nil
}

// --- Rate Limiting ---

func (s *ChatService) checkRateLimit(ctx context.Context, employeeID string) (bool, error) {
	key := "chat_rate:" + employeeID
	count, err := s.redis.Incr(ctx, key)
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, key, rateLimitWindow); err != nil {
			slog.Warn("set rate limit expire failed", "error", err)
		}
	}
	return count > rateLimitMax, nil
}

// --- History Management ---

func chatKey(employeeID string) string {
	return "chat:" + employeeID
}

func bossHistoryKey(tenantID string) string {
	return "chat:boss:" + tenantID
}

func (s *ChatService) loadHistory(ctx context.Context, key string) ([]chatHistoryMessage, error) {
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Key not found = empty history
	}
	var history []chatHistoryMessage
	if err := json.Unmarshal([]byte(data), &history); err != nil {
		return nil, err
	}
	return history, nil
}

func (s *ChatService) saveHistory(ctx context.Context, key string, history []chatHistoryMessage) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, key, string(data), historyTTL)
}

// checkGapAndTrim checks if there's a >6h gap since last message.
// If so, extracts memories from the conversation and clears history.
func (s *ChatService) checkGapAndTrim(ctx context.Context, key string, history []chatHistoryMessage, employeeID, tenantID string) []chatHistoryMessage {
	if len(history) == 0 {
		return history
	}

	lastMsg := history[len(history)-1]
	if time.Since(lastMsg.TS) > gapThreshold {
		slog.Info("chat gap detected, extracting memories and clearing history",
			"key", key,
			"last_message", lastMsg.TS,
			"gap", time.Since(lastMsg.TS),
		)

		// Extract memories from the completed conversation (fire-and-forget)
		if s.memoryEngine != nil && s.memoryEngine.Enabled() && len(history) >= 2 {
			transcript := formatTranscript(history)
			go func() {
				extractCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := s.memoryEngine.ExtractFromChat(extractCtx, memory.ChatInput{
					TenantID:   tenantID,
					EmployeeID: employeeID,
					Transcript: transcript,
				}); err != nil {
					slog.Error("chat memory extraction failed",
						"employee_id", employeeID,
						"tenant_id", tenantID,
						"error", err,
					)
				} else {
					slog.Info("chat memories extracted",
						"employee_id", employeeID,
						"messages", len(history),
					)
				}
			}()
		}

		_ = s.redis.Del(ctx, key)
		return nil
	}
	return history
}

// --- Helpers ---

func historyToChatMessages(history []chatHistoryMessage) []ChatMessage {
	msgs := make([]ChatMessage, len(history))
	for i, h := range history {
		msgs[i] = ChatMessage{Role: h.Role, Content: h.Content}
	}
	return msgs
}

// formatTranscript formats chat history into a readable transcript for memory extraction.
func formatTranscript(history []chatHistoryMessage) string {
	var sb strings.Builder
	for _, msg := range history {
		label := "Employee"
		if msg.Role == "assistant" {
			label = "Mentor"
		}
		fmt.Fprintf(&sb, "[%s] %s: %s\n", msg.TS.Format("15:04"), label, msg.Content)
	}
	return sb.String()
}

// matchEmployeeName checks if the text mentions an employee by name.
func matchEmployeeName(text, employeeName string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, strings.ToLower(employeeName)) {
		return true
	}
	parts := strings.Fields(employeeName)
	for _, part := range parts {
		if strings.Contains(lower, strings.ToLower(part)) {
			return true
		}
	}
	return false
}
