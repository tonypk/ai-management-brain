package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/bot"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/onboarding"
	"github.com/tonypk/ai-management-brain/internal/seats"
)

// redactHandler wraps slog.Handler to mask sensitive fields.
type redactHandler struct {
	slog.Handler
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) error {
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		key := strings.ToLower(a.Key)
		if key == "api_key" || key == "bot_token" || key == "password" ||
			key == "encryption_key" || key == "token" || key == "secret" {
			attrs = append(attrs, slog.String(a.Key, "***REDACTED***"))
		} else {
			attrs = append(attrs, a)
		}
		return true
	})
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	for _, a := range attrs {
		newRecord.AddAttrs(a)
	}
	return h.Handler.Handle(ctx, newRecord)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &redactHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{Handler: h.Handler.WithGroup(name)}
}

// schedulerCallbacks wires scheduler to report/chase/summary.
type schedulerCallbacks struct {
	remindFn  func(ctx context.Context) error
	chaseFn   func(ctx context.Context) error
	summaryFn func(ctx context.Context) error
}

func (s *schedulerCallbacks) Remind(ctx context.Context) error  { return s.remindFn(ctx) }
func (s *schedulerCallbacks) Chase(ctx context.Context) error   { return s.chaseFn(ctx) }
func (s *schedulerCallbacks) Summary(ctx context.Context) error { return s.summaryFn(ctx) }

// redisWrapper adapts go-redis to our RedisClient interface.
type redisWrapper struct {
	client *redis.Client
}

func (r *redisWrapper) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisWrapper) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisWrapper) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *redisWrapper) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *redisWrapper) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// groupDBAdapter adapts sqlc.Queries to bot.GroupQuerier.
type groupDBAdapter struct {
	q *sqlc.Queries
}

func (a *groupDBAdapter) CreateGroupChat(ctx context.Context, tenantID, platform, platformChatID, name, groupType string) (bot.GroupChat, error) {
	tid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return bot.GroupChat{}, fmt.Errorf("parse tenant ID: %w", err)
	}
	gc, err := a.q.CreateGroupChat(ctx, sqlc.CreateGroupChatParams{
		TenantID:       tid,
		Platform:       platform,
		PlatformChatID: platformChatID,
		Name:           name,
		GroupType:      groupType,
	})
	if err != nil {
		return bot.GroupChat{}, err
	}
	return bot.GroupChat{
		ID:       formatPgUUID(gc.ID),
		TenantID: formatPgUUID(gc.TenantID),
		Name:     gc.Name,
	}, nil
}

func (a *groupDBAdapter) GetGroupChatByPlatformID(ctx context.Context, platform, platformChatID string) (bot.GroupChat, error) {
	gc, err := a.q.GetGroupChatByPlatformID(ctx, sqlc.GetGroupChatByPlatformIDParams{
		Platform:       platform,
		PlatformChatID: platformChatID,
	})
	if err != nil {
		return bot.GroupChat{}, err
	}
	return bot.GroupChat{
		ID:       formatPgUUID(gc.ID),
		TenantID: formatPgUUID(gc.TenantID),
		Name:     gc.Name,
	}, nil
}

// seatServiceAdapter bridges seats.SeatService to bot.SeatServicer.
type seatServiceAdapter struct {
	svc *seats.SeatService
}

func (a *seatServiceAdapter) SetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64, seatType string) error {
	return a.svc.SetActiveSeat(ctx, tenantID, telegramUserID, seatType)
}

func (a *seatServiceAdapter) GetActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) string {
	return a.svc.GetActiveSeat(ctx, tenantID, telegramUserID)
}

func (a *seatServiceAdapter) ClearActiveSeat(ctx context.Context, tenantID string, telegramUserID int64) error {
	return a.svc.ClearActiveSeat(ctx, tenantID, telegramUserID)
}

func (a *seatServiceAdapter) Chat(ctx context.Context, tenantID, seatType, cultureCode, userMessage string) (string, error) {
	return a.svc.Chat(ctx, tenantID, seatType, cultureCode, userMessage)
}

func (a *seatServiceAdapter) BoardDiscuss(ctx context.Context, tenantID, cultureCode, topic string) ([]bot.SeatBoardResponse, string, error) {
	responses, synthesis, err := a.svc.BoardDiscuss(ctx, tenantID, cultureCode, topic)
	if err != nil {
		return nil, "", err
	}
	result := make([]bot.SeatBoardResponse, len(responses))
	for i, r := range responses {
		result[i] = bot.SeatBoardResponse{
			SeatType:  r.SeatType,
			Title:     r.Title,
			PersonaID: r.PersonaID,
			Content:   r.Content,
		}
	}
	return result, synthesis, nil
}

// onboardingAdapter bridges onboarding.Service (pgtype.UUID) to bot.OnboardingHandler (string IDs).
type onboardingAdapter struct {
	svc *onboarding.Service
}

func (a *onboardingAdapter) HandleMessage(ctx context.Context, tenantID string, channelType, userID, text string) (string, error) {
	uid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return "", fmt.Errorf("parse tenant ID: %w", err)
	}
	return a.svc.HandleMessage(ctx, uid, channelType, userID, text)
}

// consultingBotAdapter bridges brain.ConsultingEngine (pgtype.UUID) to bot.ConsultingServicer (string IDs).
type consultingBotAdapter struct {
	engine  *brain.ConsultingEngine
	queries *sqlc.Queries
}

func (a *consultingBotAdapter) StartEngagement(ctx context.Context, tenantID, problem, mentorID, cultureCode string) (string, string, error) {
	tid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("parse tenant ID: %w", err)
	}
	eng, firstQuestion, err := a.engine.StartEngagement(ctx, tid, problem, mentorID, cultureCode)
	if err != nil {
		return "", "", err
	}
	return formatPgUUID(eng.ID), firstQuestion, nil
}

func (a *consultingBotAdapter) AnswerQuestion(ctx context.Context, engagementID, answer string) (string, string, bool, error) {
	eid, err := parseUUIDForChat(engagementID)
	if err != nil {
		return "", "", false, fmt.Errorf("parse engagement ID: %w", err)
	}
	return a.engine.AnswerQuestion(ctx, eid, answer)
}

func (a *consultingBotAdapter) ReviewActions(ctx context.Context, engagementID string, approved bool) (string, error) {
	eid, err := parseUUIDForChat(engagementID)
	if err != nil {
		return "", fmt.Errorf("parse engagement ID: %w", err)
	}
	actions, err := a.queries.ListEngagementActions(ctx, eid)
	if err != nil {
		return "", fmt.Errorf("list engagement actions: %w", err)
	}
	for _, action := range actions {
		if action.Status == "pending" {
			if reviewErr := a.engine.ReviewAction(ctx, action.ID, approved); reviewErr != nil {
				slog.Warn("consulting bot: review action failed",
					"action_id", formatPgUUID(action.ID), "error", reviewErr)
			}
		}
	}
	status := "rejected"
	if approved {
		status = "approved"
	}
	return fmt.Sprintf("All pending actions marked as %s.", status), nil
}

func (a *consultingBotAdapter) ExecuteApproved(ctx context.Context, engagementID string) (string, error) {
	eid, err := parseUUIDForChat(engagementID)
	if err != nil {
		return "", fmt.Errorf("parse engagement ID: %w", err)
	}
	results, err := a.engine.ExecuteApproved(ctx, eid)
	if err != nil {
		return "", err
	}
	succeeded := 0
	for _, r := range results {
		if r.Success {
			succeeded++
		}
	}
	return fmt.Sprintf("Executed %d/%d actions successfully.", succeeded, len(results)), nil
}

func (a *consultingBotAdapter) ListActiveEngagements(ctx context.Context, tenantID string) (string, error) {
	tid, err := parseUUIDForChat(tenantID)
	if err != nil {
		return "", fmt.Errorf("parse tenant ID: %w", err)
	}
	engagements, err := a.queries.ListActiveEngagements(ctx, tid)
	if err != nil {
		return "", fmt.Errorf("list active engagements: %w", err)
	}
	if len(engagements) == 0 {
		return "No active consulting engagements.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active Engagements (%d):\n\n", len(engagements)))
	for i, e := range engagements {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n   ID: %s\n",
			i+1, e.Phase, e.Title, formatPgUUID(e.ID)))
	}
	return sb.String(), nil
}
