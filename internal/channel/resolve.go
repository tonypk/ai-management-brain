package channel

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// ResolveEmployee is a subset of fields needed for channel resolution.
// Using a separate type avoids importing the sqlc package from the channel package.
type ResolveEmployee struct {
	TelegramID       pgtype.Int8
	SignalPhone      pgtype.Text
	SlackID          pgtype.Text
	LarkID           pgtype.Text
	PreferredChannel string
}

// ResolveChannel returns the preferred channel type and user ID for an employee.
// Falls back through available channels if the preferred one is not configured.
func ResolveChannel(emp ResolveEmployee) (Type, string) {
	// Try preferred channel first
	switch Type(emp.PreferredChannel) {
	case TypeTelegram:
		if emp.TelegramID.Valid && emp.TelegramID.Int64 != 0 {
			return TypeTelegram, strconv.FormatInt(emp.TelegramID.Int64, 10)
		}
	case TypeSignal:
		if emp.SignalPhone.Valid && emp.SignalPhone.String != "" {
			return TypeSignal, emp.SignalPhone.String
		}
	case TypeSlack:
		if emp.SlackID.Valid && emp.SlackID.String != "" {
			return TypeSlack, emp.SlackID.String
		}
	case TypeLark:
		if emp.LarkID.Valid && emp.LarkID.String != "" {
			return TypeLark, emp.LarkID.String
		}
	}

	// Fallback: try all channels in priority order
	if emp.TelegramID.Valid && emp.TelegramID.Int64 != 0 {
		return TypeTelegram, strconv.FormatInt(emp.TelegramID.Int64, 10)
	}
	if emp.SignalPhone.Valid && emp.SignalPhone.String != "" {
		return TypeSignal, emp.SignalPhone.String
	}
	if emp.SlackID.Valid && emp.SlackID.String != "" {
		return TypeSlack, emp.SlackID.String
	}
	if emp.LarkID.Valid && emp.LarkID.String != "" {
		return TypeLark, emp.LarkID.String
	}

	return "", ""
}

// ToResolveEmployee converts individual pgtype fields to ResolveEmployee.
// Helper for callers that have individual fields (like from sqlc Employee model).
func ToResolveEmployee(telegramID pgtype.Int8, signalPhone, slackID, larkID pgtype.Text, preferred string) ResolveEmployee {
	return ResolveEmployee{
		TelegramID:       telegramID,
		SignalPhone:      signalPhone,
		SlackID:          slackID,
		LarkID:           larkID,
		PreferredChannel: preferred,
	}
}
