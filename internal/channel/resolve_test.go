package channel

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func mockResolveEmp(telegramID int64, signalPhone, slackID, larkID, preferred string) ResolveEmployee {
	e := ResolveEmployee{PreferredChannel: preferred}
	if telegramID != 0 {
		e.TelegramID = pgtype.Int8{Int64: telegramID, Valid: true}
	}
	if signalPhone != "" {
		e.SignalPhone = pgtype.Text{String: signalPhone, Valid: true}
	}
	if slackID != "" {
		e.SlackID = pgtype.Text{String: slackID, Valid: true}
	}
	if larkID != "" {
		e.LarkID = pgtype.Text{String: larkID, Valid: true}
	}
	return e
}

func TestResolveChannel_PreferredTelegram(t *testing.T) {
	emp := mockResolveEmp(12345, "+639177918392", "", "", "telegram")
	chType, chID := ResolveChannel(emp)
	if chType != TypeTelegram || chID != "12345" {
		t.Errorf("expected telegram/12345, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_PreferredSignal(t *testing.T) {
	emp := mockResolveEmp(12345, "+639177918392", "", "", "signal")
	chType, chID := ResolveChannel(emp)
	if chType != TypeSignal || chID != "+639177918392" {
		t.Errorf("expected signal/+639177918392, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_PreferredSlack(t *testing.T) {
	emp := mockResolveEmp(0, "", "U01ABC", "", "slack")
	chType, chID := ResolveChannel(emp)
	if chType != TypeSlack || chID != "U01ABC" {
		t.Errorf("expected slack/U01ABC, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_PreferredLark(t *testing.T) {
	emp := mockResolveEmp(0, "", "", "ou_abc123", "lark")
	chType, chID := ResolveChannel(emp)
	if chType != TypeLark || chID != "ou_abc123" {
		t.Errorf("expected lark/ou_abc123, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_FallbackToTelegram(t *testing.T) {
	// preferred signal but no phone set — should fall back to telegram
	emp := mockResolveEmp(12345, "", "", "", "signal")
	chType, chID := ResolveChannel(emp)
	if chType != TypeTelegram || chID != "12345" {
		t.Errorf("expected fallback to telegram/12345, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_FallbackToSignal(t *testing.T) {
	// preferred telegram but no telegram_id — should fall back to signal
	emp := mockResolveEmp(0, "+639177918392", "", "", "telegram")
	chType, chID := ResolveChannel(emp)
	if chType != TypeSignal || chID != "+639177918392" {
		t.Errorf("expected fallback to signal, got %s/%s", chType, chID)
	}
}

func TestResolveChannel_NoChannels(t *testing.T) {
	emp := mockResolveEmp(0, "", "", "", "telegram")
	chType, chID := ResolveChannel(emp)
	if chType != "" || chID != "" {
		t.Errorf("expected empty, got %s/%s", chType, chID)
	}
}
