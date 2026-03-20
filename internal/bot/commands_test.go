package bot_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

// mockBotContext implements BotContext for testing.
type mockBotContext struct {
	senderID int64
	text     string
	sent     []string
}

func (m *mockBotContext) SenderID() int64 { return m.senderID }
func (m *mockBotContext) Text() string    { return m.text }
func (m *mockBotContext) Send(msg string) error {
	m.sent = append(m.sent, msg)
	return nil
}

// mockCommandDB implements CommandQuerier for testing.
type mockCommandDB struct {
	tenantByBoss    *bot.Tenant
	tenantErr       error
	createdTenant   *bot.Tenant
	createTenantErr error

	employees    []bot.Employee
	listEmpErr   error
	createdEmp   *bot.Employee
	createEmpErr error

	employeeByCode    *bot.Employee
	empByCodeErr      error
	updateTelegramErr error
	updateMentorErr   error
}

func (m *mockCommandDB) GetTenantByBossChatID(_ context.Context, _ int64) (*bot.Tenant, error) {
	return m.tenantByBoss, m.tenantErr
}

func (m *mockCommandDB) CreateTenant(_ context.Context, params bot.CreateTenantParams) (*bot.Tenant, error) {
	if m.createTenantErr != nil {
		return nil, m.createTenantErr
	}
	if m.createdTenant != nil {
		return m.createdTenant, nil
	}
	return &bot.Tenant{ID: "new-tenant-1", Name: params.Name, BossChatID: params.BossChatID}, nil
}

func (m *mockCommandDB) ListEmployeesByTenant(_ context.Context, _ string) ([]bot.Employee, error) {
	return m.employees, m.listEmpErr
}

func (m *mockCommandDB) CreateEmployee(_ context.Context, params bot.CreateEmployeeParams) (*bot.Employee, error) {
	if m.createEmpErr != nil {
		return nil, m.createEmpErr
	}
	if m.createdEmp != nil {
		return m.createdEmp, nil
	}
	return &bot.Employee{
		ID:          "emp-1",
		Name:        params.Name,
		TenantID:    params.TenantID,
		CultureCode: params.CultureCode,
		InviteCode:  params.InviteCode,
	}, nil
}

func (m *mockCommandDB) GetEmployeeByInviteCode(_ context.Context, _ string) (*bot.Employee, error) {
	return m.employeeByCode, m.empByCodeErr
}

func (m *mockCommandDB) UpdateEmployeeTelegramID(_ context.Context, _ string, _ int64) error {
	return m.updateTelegramErr
}

func (m *mockCommandDB) UpdateTenantMentor(_ context.Context, _, _ string) error {
	return m.updateMentorErr
}

func (m *mockCommandDB) UpdateEmployeeCulture(_ context.Context, _, _ string) error {
	return nil
}

// --- Tests ---

const bossChatID int64 = 999

func TestCommand_Start_AutoCreateTenant(t *testing.T) {
	db := &mockCommandDB{
		tenantByBoss: nil,
		tenantErr:    bot.ErrNotFound,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/start"}

	if err := h.HandleStart(ctx); err != nil {
		t.Fatalf("HandleStart: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	// Should mention team setup
	if !strings.Contains(ctx.sent[0], "Welcome") {
		t.Errorf("expected welcome message, got: %s", ctx.sent[0])
	}
}

func TestCommand_Start_ExistingTenant(t *testing.T) {
	existingTenant := &bot.Tenant{ID: "t-existing", Name: "Existing Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: existingTenant,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/start"}

	if err := h.HandleStart(ctx); err != nil {
		t.Fatalf("HandleStart: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	// Should greet without creating a new tenant — verify tenant name appears
	if !strings.Contains(ctx.sent[0], "Existing Team") {
		t.Errorf("expected existing team name in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Status_BossOnly(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/status"} // not boss

	if err := h.HandleStatus(ctx); err != nil {
		t.Fatalf("HandleStatus: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "permission denied") {
		t.Errorf("expected permission denied, got: %s", ctx.sent[0])
	}
}

func TestCommand_AddEmployee(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "My Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/addemployee Alice ph"}

	if err := h.HandleAddEmployee(ctx); err != nil {
		t.Fatalf("HandleAddEmployee: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(ctx.sent[0], "Alice") {
		t.Errorf("expected employee name in reply, got: %s", ctx.sent[0])
	}
	// Invite code should appear in the reply
	if !strings.Contains(ctx.sent[0], "Invite code") {
		t.Errorf("expected invite code in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Join(t *testing.T) {
	emp := &bot.Employee{ID: "emp-1", Name: "Alice", TenantID: "t1", InviteCode: "ABC123"}
	db := &mockCommandDB{
		employeeByCode: emp,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 55555, text: "/join ABC123"}

	if err := h.HandleJoin(ctx); err != nil {
		t.Fatalf("HandleJoin: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(ctx.sent[0], "Alice") {
		t.Errorf("expected employee name in welcome message, got: %s", ctx.sent[0])
	}
}

func TestCommand_Join_InvalidCode(t *testing.T) {
	db := &mockCommandDB{
		empByCodeErr: bot.ErrNotFound,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 55555, text: "/join BADCODE"}

	if err := h.HandleJoin(ctx); err != nil {
		t.Fatalf("HandleJoin: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "invalid") {
		t.Errorf("expected invalid code message, got: %s", ctx.sent[0])
	}
}

func TestCommand_Help(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/help"}

	if err := h.HandleHelp(ctx); err != nil {
		t.Fatalf("HandleHelp: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	// Help text should mention at least one command
	if !strings.Contains(ctx.sent[0], "/start") {
		t.Errorf("expected /start in help text, got: %s", ctx.sent[0])
	}
	if !strings.Contains(ctx.sent[0], "/join") {
		t.Errorf("expected /join in help text, got: %s", ctx.sent[0])
	}
}

func TestCommand_Help_AnyUser(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	// Non-boss user should also get help
	ctx := &mockBotContext{senderID: 99999, text: "/help"}

	if err := h.HandleHelp(ctx); err != nil {
		t.Fatalf("HandleHelp for non-boss: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message for non-boss user")
	}
}

func TestCommand_Mentor_BossOnly(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/mentor inamori"}

	if err := h.HandleMentor(ctx); err != nil {
		t.Fatalf("HandleMentor: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "permission denied") {
		t.Errorf("expected permission denied, got: %s", ctx.sent[0])
	}
}

func TestCommand_Mentor_ValidSwitch(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "My Team", BossChatID: bossChatID}
	db := &mockCommandDB{tenantByBoss: tenant}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/mentor dalio"}

	if err := h.HandleMentor(ctx); err != nil {
		t.Fatalf("HandleMentor: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(ctx.sent[0], "dalio") {
		t.Errorf("expected dalio in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_AddEmployee_NonBoss(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/addemployee Alice ph"}

	if err := h.HandleAddEmployee(ctx); err != nil {
		t.Fatalf("HandleAddEmployee: %v", err)
	}

	if len(ctx.sent) == 0 {
		t.Fatal("expected a reply message")
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "permission denied") {
		t.Errorf("expected permission denied, got: %s", ctx.sent[0])
	}
}

// Ensure bot.ErrNotFound is a usable sentinel error.
func TestErrNotFound_Sentinel(t *testing.T) {
	wrapped := errors.Join(bot.ErrNotFound, errors.New("extra"))
	if !errors.Is(wrapped, bot.ErrNotFound) {
		t.Error("ErrNotFound should be detectable via errors.Is")
	}
}
