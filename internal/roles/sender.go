package roles

// BossSender sends branded messages to the boss via Telegram.
type BossSender struct {
	sender     MessageSender
	bossChatID int64
}

// MessageSender sends messages to a chat ID.
type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

// NewBossSender creates a new boss sender.
func NewBossSender(sender MessageSender, bossChatID int64) *BossSender {
	return &BossSender{sender: sender, bossChatID: bossChatID}
}

// SendToBoss sends a text message to the boss.
func (s *BossSender) SendToBoss(text string) error {
	return s.sender.SendMessage(s.bossChatID, text)
}
