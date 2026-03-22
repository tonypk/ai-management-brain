package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// UnifiedHandler processes incoming messages from any channel.
type UnifiedHandler struct {
	queries   *sqlc.Queries
	sender    Sender
	onText    func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (response string, err error)
	onCommand func(ctx context.Context, employeeID, tenantID, command, args, channelType string) (response string, err error)
}

// UnifiedHandlerConfig holds the dependencies for creating a UnifiedHandler.
type UnifiedHandlerConfig struct {
	Queries   *sqlc.Queries
	Sender    Sender
	OnText    func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (response string, err error)
	OnCommand func(ctx context.Context, employeeID, tenantID, command, args, channelType string) (response string, err error)
}

// NewUnifiedHandler creates a new channel-agnostic message handler.
func NewUnifiedHandler(cfg UnifiedHandlerConfig) *UnifiedHandler {
	return &UnifiedHandler{
		queries:   cfg.Queries,
		sender:    cfg.Sender,
		onText:    cfg.OnText,
		onCommand: cfg.OnCommand,
	}
}

// HandleMessage processes an incoming message from any channel.
func (h *UnifiedHandler) HandleMessage(ctx context.Context, msg Message) error {
	emp, err := h.resolveEmployee(ctx, msg.ChannelType, msg.UserID)
	if err != nil {
		slog.Warn("unknown sender", "channel", msg.ChannelType, "user_id", msg.UserID)
		return nil // don't error on unknown senders
	}

	empID := formatPgUUID(emp.ID)
	tenantID := formatPgUUID(emp.TenantID)

	var response string
	if msg.IsCommand && h.onCommand != nil {
		response, err = h.onCommand(ctx, empID, tenantID, msg.Command, msg.Args, string(msg.ChannelType))
	} else if h.onText != nil {
		response, err = h.onText(ctx, empID, tenantID, msg.Text, string(msg.ChannelType), emp.Name, emp.JobTitle, emp.Responsibilities, emp.Country, emp.Language, emp.CultureCode)
	}

	if err != nil {
		return fmt.Errorf("handle message: %w", err)
	}

	if response != "" {
		// Reply on the originating channel (not preferred channel)
		return h.sender.Send(ctx, msg.ChannelType, msg.UserID, response)
	}
	return nil
}

// resolveEmployee finds the employee by their channel-specific user ID.
func (h *UnifiedHandler) resolveEmployee(ctx context.Context, ct Type, userID string) (sqlc.Employee, error) {
	switch ct {
	case TypeTelegram:
		id, _ := strconv.ParseInt(userID, 10, 64)
		return h.queries.GetEmployeeByTelegramID(ctx, pgtype.Int8{Int64: id, Valid: true})
	case TypeSignal:
		return h.queries.GetEmployeeBySignalPhone(ctx, pgtype.Text{String: userID, Valid: true})
	case TypeSlack:
		return h.queries.GetEmployeeBySlackID(ctx, pgtype.Text{String: userID, Valid: true})
	case TypeLark:
		return h.queries.GetEmployeeByLarkID(ctx, pgtype.Text{String: userID, Valid: true})
	}
	return sqlc.Employee{}, fmt.Errorf("unknown channel type: %s", ct)
}

// formatPgUUID formats a pgtype.UUID as a string.
func formatPgUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
