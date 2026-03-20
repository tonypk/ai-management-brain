package bot_test

import (
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/bot"
)

func TestCommand_Diagnostics_Boss(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/diagnostics"}

	if err := h.HandleDiagnostics(ctx); err != nil {
		t.Fatalf("HandleDiagnostics: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "Running") {
		t.Errorf("expected 'Running' in diagnostics, got: %s", ctx.sent[0])
	}
}

func TestCommand_Diagnostics_WithFn(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	h.DiagnosticsFn = func() string {
		return "Custom diagnostics output"
	}
	ctx := &mockBotContext{senderID: bossChatID, text: "/diagnostics"}

	if err := h.HandleDiagnostics(ctx); err != nil {
		t.Fatalf("HandleDiagnostics: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if ctx.sent[0] != "Custom diagnostics output" {
		t.Errorf("expected custom output, got: %s", ctx.sent[0])
	}
}

func TestCommand_Diagnostics_NonBoss(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/diagnostics"}

	if err := h.HandleDiagnostics(ctx); err != nil {
		t.Fatalf("HandleDiagnostics: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "permission denied") {
		t.Errorf("expected permission denied, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_ValidBlend(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "My Team", BossChatID: bossChatID, MentorID: "inamori"}
	db := &mockCommandDB{tenantByBoss: tenant}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend inamori 70 dalio"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "70%") {
		t.Errorf("expected 70%% in reply, got: %s", ctx.sent[0])
	}
	if !strings.Contains(ctx.sent[0], "inamori") {
		t.Errorf("expected inamori in reply, got: %s", ctx.sent[0])
	}
	if !strings.Contains(ctx.sent[0], "dalio") {
		t.Errorf("expected dalio in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_Off(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "My Team", BossChatID: bossChatID, MentorID: "grove"}
	db := &mockCommandDB{tenantByBoss: tenant}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend off"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend off: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "disabled") {
		t.Errorf("expected 'disabled' in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_SameMentor(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend inamori 70 inamori"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "different") {
		t.Errorf("expected 'different' in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_InvalidWeight(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)

	tests := []struct {
		name string
		text string
	}{
		{"too low", "/blend inamori 30 dalio"},
		{"too high", "/blend inamori 95 dalio"},
		{"not a number", "/blend inamori abc dalio"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &mockBotContext{senderID: bossChatID, text: tt.text}
			if err := h.HandleBlend(ctx); err != nil {
				t.Fatalf("HandleBlend: %v", err)
			}
			if len(ctx.sent) == 0 {
				t.Fatal("expected reply")
			}
			if !strings.Contains(ctx.sent[0], "50-90") {
				t.Errorf("expected weight range error, got: %s", ctx.sent[0])
			}
		})
	}
}

func TestCommand_Blend_UnknownMentor(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend unknown 70 dalio"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "Unknown") {
		t.Errorf("expected unknown mentor msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "Usage") {
		t.Errorf("expected usage message, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_NonBoss(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/blend inamori 70 dalio"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if !strings.Contains(strings.ToLower(ctx.sent[0]), "permission denied") {
		t.Errorf("expected permission denied, got: %s", ctx.sent[0])
	}
}

func TestCommand_Mentor_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/mentor"}

	if err := h.HandleMentor(ctx); err != nil {
		t.Fatalf("HandleMentor: %v", err)
	}
	if len(ctx.sent) == 0 {
		t.Fatal("expected reply")
	}
	if !strings.Contains(ctx.sent[0], "Available mentors") {
		t.Errorf("expected mentor list, got: %s", ctx.sent[0])
	}
}

func TestCommand_Mentor_UnknownMentor(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/mentor unknown"}

	if err := h.HandleMentor(ctx); err != nil {
		t.Fatalf("HandleMentor: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Unknown") {
		t.Errorf("expected unknown mentor msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Culture_ValidSet(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
		employees: []bot.Employee{
			{ID: "e1", Name: "Alice", CultureCode: "default"},
		},
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/culture Alice philippines"}

	if err := h.HandleCulture(ctx); err != nil {
		t.Fatalf("HandleCulture: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "philippines") {
		t.Errorf("expected philippines in reply, got: %s", ctx.sent[0])
	}
}

func TestCommand_Culture_InvalidCode(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/culture Alice badcode"}

	if err := h.HandleCulture(ctx); err != nil {
		t.Fatalf("HandleCulture: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Unknown culture") {
		t.Errorf("expected unknown culture msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Culture_EmployeeNotFound(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
		employees:    []bot.Employee{}, // no employees
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/culture Alice philippines"}

	if err := h.HandleCulture(ctx); err != nil {
		t.Fatalf("HandleCulture: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "not found") {
		t.Errorf("expected not found msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Culture_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/culture"}

	if err := h.HandleCulture(ctx); err != nil {
		t.Fatalf("HandleCulture: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Usage") {
		t.Errorf("expected usage msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_AddEmployee_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/addemployee"}

	if err := h.HandleAddEmployee(ctx); err != nil {
		t.Fatalf("HandleAddEmployee: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Usage") {
		t.Errorf("expected usage msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Join_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/join"}

	if err := h.HandleJoin(ctx); err != nil {
		t.Fatalf("HandleJoin: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Usage") {
		t.Errorf("expected usage msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Profile_NoArgs(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/profile"}

	if err := h.HandleProfile(ctx); err != nil {
		t.Fatalf("HandleProfile: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Usage") {
		t.Errorf("expected usage msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Profile_EmployeeNotFound(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
		employees:    []bot.Employee{},
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/profile Unknown"}

	if err := h.HandleProfile(ctx); err != nil {
		t.Fatalf("HandleProfile: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "not found") {
		t.Errorf("expected not found msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Start_NonBoss(t *testing.T) {
	db := &mockCommandDB{}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: 12345, text: "/start"}

	if err := h.HandleStart(ctx); err != nil {
		t.Fatalf("HandleStart: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "contact your manager") {
		t.Errorf("expected contact manager message, got: %s", ctx.sent[0])
	}
}

func TestCommand_Status_NoTeam(t *testing.T) {
	db := &mockCommandDB{
		tenantErr: bot.ErrNotFound,
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/status"}

	if err := h.HandleStatus(ctx); err != nil {
		t.Fatalf("HandleStatus: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "/start") {
		t.Errorf("expected prompt to use /start, got: %s", ctx.sent[0])
	}
}

func TestCommand_Status_NoEmployees(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
		employees:    []bot.Employee{},
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/status"}

	if err := h.HandleStatus(ctx); err != nil {
		t.Fatalf("HandleStatus: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "No employees") {
		t.Errorf("expected no employees msg, got: %s", ctx.sent[0])
	}
}

func TestCommand_Status_WithEmployees(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID}
	db := &mockCommandDB{
		tenantByBoss: tenant,
		employees: []bot.Employee{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 0},
		},
	}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/status"}

	if err := h.HandleStatus(ctx); err != nil {
		t.Fatalf("HandleStatus: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "Alice") {
		t.Errorf("expected Alice in status, got: %s", ctx.sent[0])
	}
	if !strings.Contains(ctx.sent[0], "active") {
		t.Errorf("expected 'active' for linked employee, got: %s", ctx.sent[0])
	}
	if !strings.Contains(ctx.sent[0], "not linked") {
		t.Errorf("expected 'not linked' for unlinked employee, got: %s", ctx.sent[0])
	}
}

func TestCommand_Blend_WeightWithPercent(t *testing.T) {
	tenant := &bot.Tenant{ID: "t1", Name: "Team", BossChatID: bossChatID, MentorID: "inamori"}
	db := &mockCommandDB{tenantByBoss: tenant}
	h := bot.NewCommandHandler(db, nil, nil, bossChatID)
	ctx := &mockBotContext{senderID: bossChatID, text: "/blend inamori 70% dalio"}

	if err := h.HandleBlend(ctx); err != nil {
		t.Fatalf("HandleBlend: %v", err)
	}
	if !strings.Contains(ctx.sent[0], "70%") {
		t.Errorf("expected 70%% with trailing %% stripped, got: %s", ctx.sent[0])
	}
}
