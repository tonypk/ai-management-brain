package seats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxBoardSeats      = 6
	boardRateLimitTTL  = 5 * time.Minute
	seatHistoryTTL     = 24 * time.Hour
	maxHistoryMessages = 10
	maxUserMessageLen  = 4000
)

// RedisClient is the subset of Redis operations SeatService needs.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// BoardResponse represents one seat's contribution to a board discussion.
type BoardResponse struct {
	SeatType  string `json:"seat_type"`
	Title     string `json:"title"`
	PersonaID string `json:"persona_id"`
	Content   string `json:"content"`
}

// SeatServiceConfig holds dependencies for creating a SeatService.
type SeatServiceConfig struct {
	DB            *sqlc.Queries
	EngineFactory *brain.EngineFactory
	Memory        *memory.MemoryEngine
	LLM           brain.ChatLLMClient
	Redis         RedisClient
}

// SeatService manages C-suite seats, chat, and board discussions.
type SeatService struct {
	db            *sqlc.Queries
	engineFactory *brain.EngineFactory
	memory        *memory.MemoryEngine
	llm           brain.ChatLLMClient
	redis         RedisClient
}

// NewSeatService creates a SeatService with all dependencies.
func NewSeatService(cfg SeatServiceConfig) *SeatService {
	return &SeatService{
		db:            cfg.DB,
		engineFactory: cfg.EngineFactory,
		memory:        cfg.Memory,
		llm:           cfg.LLM,
		redis:         cfg.Redis,
	}
}

// Chat handles a message to a specific seat, using independent history and shared memory.
func (s *SeatService) Chat(ctx context.Context, tenantID, seatType, cultureCode, userMessage string) (string, error) {
	if s.llm == nil {
		return "AI features are not enabled.", nil
	}
	if len(userMessage) > maxUserMessageLen {
		return "", fmt.Errorf("message too long (max %d characters)", maxUserMessageLen)
	}

	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	seat, err := s.db.GetSeatByType(ctx, sqlc.GetSeatByTypeParams{
		TenantID: tenantUUID,
		SeatType: seatType,
	})
	if err != nil {
		return "", fmt.Errorf("seat %q not found: %w", seatType, err)
	}

	if !seat.IsActive.Bool {
		return fmt.Sprintf("The %s seat is currently inactive.", seat.Title), nil
	}

	// Load engine for this seat's persona
	if cultureCode == "" {
		cultureCode = "default"
	}
	engine, err := s.engineFactory.ForTenant(seat.PersonaID, cultureCode)
	if err != nil {
		slog.Error("failed to load engine for seat", "persona", seat.PersonaID, "error", err)
		engine, _ = s.engineFactory.ForTenant("inamori", "default")
	}

	// Recall company-level memories (empty employeeID)
	var memoryContext string
	if s.memory != nil && s.memory.Enabled() {
		recall, err := s.memory.RecallForMentor(ctx, tenantID, "", seat.Scope+" "+userMessage)
		if err != nil {
			slog.Warn("memory recall failed for seat chat", "seat", seatType, "error", err)
		} else if recall != nil {
			memoryContext = memory.FormatForPrompt(recall)
		}
	}

	// Build system prompt
	systemPrompt := BuildSeatChatPrompt(engine, seat.Title, seat.Scope, memoryContext)

	// Load chat history
	historyKey := fmt.Sprintf("seat:%s:%s", tenantID, seatType)
	history := s.loadHistory(ctx, historyKey)

	// Call LLM with history
	response, err := s.llm.ChatWithHistory(ctx, systemPrompt, history, userMessage)
	if err != nil {
		slog.Error("LLM call failed for seat chat", "seat", seatType, "error", err)
		return "Sorry, I'm unable to respond right now. Please try again later.", nil
	}

	// Save updated history
	history = append(history,
		brain.ChatMessage{Role: "user", Content: userMessage},
		brain.ChatMessage{Role: "assistant", Content: response},
	)
	if len(history) > maxHistoryMessages*2 {
		history = history[len(history)-maxHistoryMessages*2:]
	}
	s.saveHistory(ctx, historyKey, history)

	return response, nil
}

// BoardDiscuss runs a multi-role board discussion on a topic.
func (s *SeatService) BoardDiscuss(ctx context.Context, tenantID, cultureCode, topic string) ([]BoardResponse, string, error) {
	if s.llm == nil {
		return nil, "", fmt.Errorf("AI features are not enabled")
	}
	if len(topic) > maxUserMessageLen {
		return nil, "", fmt.Errorf("topic too long (max %d characters)", maxUserMessageLen)
	}

	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Rate limit check
	rateLimitKey := fmt.Sprintf("board_rate:%s", tenantID)
	if val, err := s.redis.Get(ctx, rateLimitKey); err == nil && val != "" {
		return nil, "", fmt.Errorf("board discussions are limited to once per 5 minutes")
	} else if err != nil {
		slog.Warn("board rate limit check failed, allowing request", "error", err)
	}

	// Get all active seats
	seats, err := s.db.ListActiveSeatsByTenant(ctx, tenantUUID)
	if err != nil {
		return nil, "", fmt.Errorf("list seats: %w", err)
	}
	if len(seats) == 0 {
		return nil, "", fmt.Errorf("no active seats found — assign seats first using /assign")
	}
	if len(seats) > maxBoardSeats {
		seats = seats[:maxBoardSeats]
	}

	// Recall company-level memories
	if cultureCode == "" {
		cultureCode = "default"
	}
	var memoryContext string
	if s.memory != nil && s.memory.Enabled() {
		recall, err := s.memory.RecallForMentor(ctx, tenantID, "", topic)
		if err != nil {
			slog.Warn("memory recall failed for board discussion", "error", err)
		} else if recall != nil {
			memoryContext = memory.FormatForPrompt(recall)
		}
	}

	// Sequential calls — each seat sees prior responses
	var responses []BoardResponse
	var priorContext strings.Builder

	for _, seat := range seats {
		engine, err := s.engineFactory.ForTenant(seat.PersonaID, cultureCode)
		if err != nil {
			slog.Error("failed to load engine for board seat", "persona", seat.PersonaID, "error", err)
			responses = append(responses, BoardResponse{
				SeatType: seat.SeatType, Title: seat.Title,
				PersonaID: seat.PersonaID, Content: "[unavailable — persona load failed]",
			})
			continue
		}

		prompt := BuildBoardPrompt(engine, seat.Title, seat.Scope, topic, memoryContext, priorContext.String())
		reply, err := s.llm.ChatWithHistory(ctx, prompt, nil, topic)
		if err != nil {
			slog.Error("LLM call failed for board seat", "seat", seat.SeatType, "error", err)
			responses = append(responses, BoardResponse{
				SeatType: seat.SeatType, Title: seat.Title,
				PersonaID: seat.PersonaID, Content: "[unavailable — AI response failed]",
			})
			continue
		}

		responses = append(responses, BoardResponse{
			SeatType:  seat.SeatType,
			Title:     seat.Title,
			PersonaID: seat.PersonaID,
			Content:   reply,
		})
		priorContext.WriteString(fmt.Sprintf("[%s]: %s\n\n", seat.Title, reply))
	}

	// Synthesis
	synthesisPrompt := BuildSynthesisPrompt(topic, responses)
	synthesis, err := s.llm.ChatWithHistory(ctx, synthesisPrompt, nil, "Synthesize the board discussion above.")
	if err != nil {
		slog.Error("synthesis LLM call failed", "error", err)
		synthesis = "Synthesis unavailable due to AI error."
	}

	// Set rate limit
	if err := s.redis.Set(ctx, rateLimitKey, "1", boardRateLimitTTL); err != nil {
		slog.Warn("failed to set board rate limit", "error", err)
	}

	return responses, synthesis, nil
}

// SetActiveSeat sets the user's current active seat for Telegram chat routing.
func (s *SeatService) SetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64, seatType string) error {
	key := fmt.Sprintf("active_seat:%s:%d", tenantID, telegramUserID)
	return s.redis.Set(ctx, key, seatType, 0) // no expiry
}

// GetActiveSeat returns the user's current active seat type, or "" if none.
func (s *SeatService) GetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) string {
	key := fmt.Sprintf("active_seat:%s:%d", tenantID, telegramUserID)
	val, err := s.redis.Get(ctx, key)
	if err != nil {
		return ""
	}
	return val
}

// ClearActiveSeat removes the user's active seat.
func (s *SeatService) ClearActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) error {
	key := fmt.Sprintf("active_seat:%s:%d", tenantID, telegramUserID)
	return s.redis.Del(ctx, key)
}

// helpers

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return u, nil
}

func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *SeatService) loadHistory(ctx context.Context, key string) []brain.ChatMessage {
	val, err := s.redis.Get(ctx, key)
	if err != nil || val == "" {
		return nil
	}
	var history []brain.ChatMessage
	if err := json.Unmarshal([]byte(val), &history); err != nil {
		slog.Warn("failed to parse seat chat history", "key", key, "error", err)
		return nil
	}
	return history
}

func (s *SeatService) saveHistory(ctx context.Context, key string, history []brain.ChatMessage) {
	data, err := json.Marshal(history)
	if err != nil {
		slog.Error("failed to marshal seat chat history", "key", key, "error", err)
		return
	}
	if err := s.redis.Set(ctx, key, string(data), seatHistoryTTL); err != nil {
		slog.Error("failed to save seat chat history", "key", key, "error", err)
	}
}
