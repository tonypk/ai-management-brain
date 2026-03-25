package channel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// BossResolver provides DB queries needed for boss identity resolution.
type BossResolver interface {
	GetTenantByBossChatID(ctx context.Context, bossChatID int64) (sqlc.Tenant, error)
	GetTenantByBossSlackID(ctx context.Context, bossSlackID pgtype.Text) (sqlc.Tenant, error)
	GetTenantByBossLarkID(ctx context.Context, bossLarkID pgtype.Text) (sqlc.Tenant, error)
}

// ResolveBoss checks if the sender is a boss on any channel.
// Returns the tenant if found, or an error if not a boss.
func ResolveBoss(ctx context.Context, db BossResolver, channelType string, userID string) (*sqlc.Tenant, error) {
	switch Type(channelType) {
	case TypeTelegram:
		id, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid telegram ID %q: %w", userID, err)
		}
		t, err := db.GetTenantByBossChatID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &t, nil

	case TypeSlack:
		t, err := db.GetTenantByBossSlackID(ctx, pgtype.Text{String: userID, Valid: true})
		if err != nil {
			return nil, err
		}
		return &t, nil

	case TypeLark:
		t, err := db.GetTenantByBossLarkID(ctx, pgtype.Text{String: userID, Valid: true})
		if err != nil {
			return nil, err
		}
		return &t, nil

	default:
		return nil, fmt.Errorf("unsupported channel type for boss resolution: %s", channelType)
	}
}
