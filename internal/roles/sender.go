package roles

import (
	"context"

	"github.com/tonypk/ai-management-brain/internal/channel"
)

// BossSender sends branded messages to the boss via their preferred channel.
type BossSender struct {
	sender          channel.Sender
	bossChannelType channel.Type
	bossChannelID   string
}

// NewBossSender creates a new boss sender.
func NewBossSender(sender channel.Sender, bossChannelType channel.Type, bossChannelID string) *BossSender {
	return &BossSender{sender: sender, bossChannelType: bossChannelType, bossChannelID: bossChannelID}
}

// SendToBoss sends a text message to the boss.
func (s *BossSender) SendToBoss(ctx context.Context, text string) error {
	return s.sender.Send(ctx, s.bossChannelType, s.bossChannelID, text)
}
