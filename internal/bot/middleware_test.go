package bot_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

// MockQuerier implements the subset of sqlc.Querier used by middleware
type MockQuerier struct {
	EmployeeByTelegramID *bot.Employee // nil = not found
	TenantByBossChatID   *bot.Tenant   // nil = not found
}

func (m *MockQuerier) GetEmployeeByTelegramID(ctx context.Context, telegramID int64) (*bot.Employee, error) {
	return m.EmployeeByTelegramID, nil
}

func (m *MockQuerier) GetTenantByBossChatID(ctx context.Context, bossChatID int64) (*bot.Tenant, error) {
	return m.TenantByBossChatID, nil
}

func TestIdentityResolve_KnownEmployee(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{
		EmployeeByTelegramID: &bot.Employee{Name: "Alice", TenantID: "t1"},
	}, 999)

	result, err := resolver.Resolve(context.Background(), 12345)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if result.Employee == nil || result.Employee.Name != "Alice" {
		t.Error("should resolve to Alice")
	}
	if result.IsBoss {
		t.Error("should not be boss")
	}
}

func TestIdentityResolve_Boss(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)

	result, err := resolver.Resolve(context.Background(), 999)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !result.IsBoss {
		t.Error("should be boss")
	}
}

func TestIdentityResolve_Unknown(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)

	result, err := resolver.Resolve(context.Background(), 55555)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if result.Employee != nil || result.IsBoss {
		t.Error("unknown user should have no identity")
	}
}

func TestIdentityResolve_JoinBypass(t *testing.T) {
	resolver := bot.NewIdentityResolver(&MockQuerier{}, 999)
	// /join command should be allowed even for unknown users
	if !resolver.AllowWithoutIdentity("/join ABC123") {
		t.Error("/join should bypass identity check")
	}
	if resolver.AllowWithoutIdentity("/status") {
		t.Error("/status should NOT bypass identity check")
	}
}
