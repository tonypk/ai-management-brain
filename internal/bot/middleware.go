package bot

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// Employee represents a resolved employee identity.
type Employee struct {
	ID               string
	Name             string
	TenantID         string
	TelegramID       int64
	CultureCode      string
	InviteCode       string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
}

// Tenant represents a resolved tenant.
type Tenant struct {
	ID                    string
	BotToken              string
	Name                  string
	BossChatID            int64
	MentorID              string
	MentorBlend           []byte     // JSON: {"primary_id":"...","secondary_id":"...","weight":0.7}
	Timezone              string
	OnboardingCompletedAt *time.Time // nil = not completed
}

// IdentityResult holds the result of identity resolution.
type IdentityResult struct {
	Employee *Employee
	Tenant   *Tenant
	IsBoss   bool
}

// IdentityQuerier defines the DB queries needed for identity resolution.
type IdentityQuerier interface {
	GetEmployeeByTelegramID(ctx context.Context, telegramID int64) (*Employee, error)
	GetTenantByBossChatID(ctx context.Context, bossChatID int64) (*Tenant, error)
}

// IdentityResolver resolves Telegram user IDs to employee/boss identities.
type IdentityResolver struct {
	querier    IdentityQuerier
	bossChatID int64
}

// NewIdentityResolver creates a new identity resolver.
func NewIdentityResolver(querier IdentityQuerier, bossChatID int64) *IdentityResolver {
	return &IdentityResolver{
		querier:    querier,
		bossChatID: bossChatID,
	}
}

// Resolve resolves a Telegram user ID to an identity.
func (r *IdentityResolver) Resolve(ctx context.Context, telegramID int64) (*IdentityResult, error) {
	result := &IdentityResult{}

	// Check if sender is the boss
	if telegramID == r.bossChatID {
		result.IsBoss = true
		slog.Info("identity resolved: boss", "telegram_id", telegramID)
		return result, nil
	}

	// Look up employee by telegram ID
	emp, err := r.querier.GetEmployeeByTelegramID(ctx, telegramID)
	if err != nil {
		slog.Debug("employee not found", "telegram_id", telegramID, "error", err)
		return result, nil // not found = unknown user, no error
	}

	if emp != nil {
		result.Employee = emp
		slog.Info("identity resolved: employee", "telegram_id", telegramID, "name", emp.Name)
	}

	return result, nil
}

// AllowWithoutIdentity checks if a message text should bypass identity check.
func (r *IdentityResolver) AllowWithoutIdentity(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "/join")
}
