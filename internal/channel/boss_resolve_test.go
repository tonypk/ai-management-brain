package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// mockBossResolver implements BossResolver for testing.
type mockBossResolver struct {
	telegramTenant *sqlc.Tenant
	slackTenant    *sqlc.Tenant
	larkTenant     *sqlc.Tenant
}

var errNotFound = errors.New("no rows in result set")

func (m *mockBossResolver) GetTenantByBossChatID(_ context.Context, bossChatID int64) (sqlc.Tenant, error) {
	if m.telegramTenant != nil && m.telegramTenant.BossChatID == bossChatID {
		return *m.telegramTenant, nil
	}
	return sqlc.Tenant{}, errNotFound
}

func (m *mockBossResolver) GetTenantByBossSlackID(_ context.Context, bossSlackID pgtype.Text) (sqlc.Tenant, error) {
	if m.slackTenant != nil && bossSlackID.Valid && m.slackTenant.BossSlackID.String == bossSlackID.String {
		return *m.slackTenant, nil
	}
	return sqlc.Tenant{}, errNotFound
}

func (m *mockBossResolver) GetTenantByBossLarkID(_ context.Context, bossLarkID pgtype.Text) (sqlc.Tenant, error) {
	if m.larkTenant != nil && bossLarkID.Valid && m.larkTenant.BossLarkID.String == bossLarkID.String {
		return *m.larkTenant, nil
	}
	return sqlc.Tenant{}, errNotFound
}

func newTestTenant(name string, bossChatID int64, slackID, larkID string) sqlc.Tenant {
	t := sqlc.Tenant{
		Name:       name,
		BossChatID: bossChatID,
	}
	if slackID != "" {
		t.BossSlackID = pgtype.Text{String: slackID, Valid: true}
	}
	if larkID != "" {
		t.BossLarkID = pgtype.Text{String: larkID, Valid: true}
	}
	return t
}

func TestResolveBoss_Telegram(t *testing.T) {
	tenant := newTestTenant("TelegramCorp", 12345, "", "")
	db := &mockBossResolver{telegramTenant: &tenant}

	got, err := ResolveBoss(context.Background(), db, "telegram", "12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "TelegramCorp" {
		t.Errorf("expected tenant name TelegramCorp, got %s", got.Name)
	}
	if got.BossChatID != 12345 {
		t.Errorf("expected boss_chat_id 12345, got %d", got.BossChatID)
	}
}

func TestResolveBoss_Slack(t *testing.T) {
	tenant := newTestTenant("SlackCorp", 0, "U01ABC", "")
	db := &mockBossResolver{slackTenant: &tenant}

	got, err := ResolveBoss(context.Background(), db, "slack", "U01ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "SlackCorp" {
		t.Errorf("expected tenant name SlackCorp, got %s", got.Name)
	}
	if got.BossSlackID.String != "U01ABC" {
		t.Errorf("expected boss_slack_id U01ABC, got %s", got.BossSlackID.String)
	}
}

func TestResolveBoss_Lark(t *testing.T) {
	tenant := newTestTenant("LarkCorp", 0, "", "ou_abc123")
	db := &mockBossResolver{larkTenant: &tenant}

	got, err := ResolveBoss(context.Background(), db, "lark", "ou_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "LarkCorp" {
		t.Errorf("expected tenant name LarkCorp, got %s", got.Name)
	}
	if got.BossLarkID.String != "ou_abc123" {
		t.Errorf("expected boss_lark_id ou_abc123, got %s", got.BossLarkID.String)
	}
}

func TestResolveBoss_UnsupportedChannel(t *testing.T) {
	db := &mockBossResolver{}

	_, err := ResolveBoss(context.Background(), db, "whatsapp", "123")
	if err == nil {
		t.Fatal("expected error for unsupported channel, got nil")
	}
	expected := "unsupported channel type for boss resolution: whatsapp"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestResolveBoss_UnknownUser(t *testing.T) {
	// Empty mock — no tenants configured
	db := &mockBossResolver{}

	_, err := ResolveBoss(context.Background(), db, "telegram", "99999")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
	}
}

func TestResolveBoss_InvalidTelegramID(t *testing.T) {
	db := &mockBossResolver{}

	_, err := ResolveBoss(context.Background(), db, "telegram", "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid telegram ID, got nil")
	}
	if got := err.Error(); len(got) == 0 {
		t.Error("expected non-empty error message")
	}
}

func TestResolveBoss_SlackUnknownUser(t *testing.T) {
	db := &mockBossResolver{}

	_, err := ResolveBoss(context.Background(), db, "slack", "U_UNKNOWN")
	if err == nil {
		t.Fatal("expected error for unknown slack user, got nil")
	}
}

func TestResolveBoss_LarkUnknownUser(t *testing.T) {
	db := &mockBossResolver{}

	_, err := ResolveBoss(context.Background(), db, "lark", "ou_unknown")
	if err == nil {
		t.Fatal("expected error for unknown lark user, got nil")
	}
}
