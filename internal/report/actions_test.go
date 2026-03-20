package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// mockActionDB implements report.ActionDB for testing.
type mockActionDB struct {
	employees     []report.EmployeeInfo
	submittedDays map[string]int
}

func (m *mockActionDB) ListActiveEmployeesWithTelegram(_ context.Context, _ string) ([]report.EmployeeInfo, error) {
	return m.employees, nil
}

func (m *mockActionDB) GetSubmittedDaysLast7(_ context.Context, empID string) (int, error) {
	return m.submittedDays[empID], nil
}

// ---------- helpers ----------

func newActionExecutor(db *mockActionDB, sender *mockSender) *report.ActionExecutor {
	factory := brain.NewEngineFactory()
	return report.NewActionExecutor(db, sender, nil, factory)
}

// ---------- 1. RunWeekly with valid mentor (inamori) ----------

func TestRunWeekly_Inamori_SendsMessages(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
		},
		submittedDays: map[string]int{
			"e1": 5,
			"e2": 3,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	// inamori has 2 weekly actions: recognition + team_pulse
	if len(sender.sentMessages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(sender.sentMessages))
		for i, m := range sender.sentMessages {
			t.Logf("  message[%d]: %s", i, m.Message)
		}
	}

	for _, m := range sender.sentMessages {
		if m.ChatID != 999 {
			t.Errorf("expected chatID 999, got %d", m.ChatID)
		}
	}
}

// ---------- 2. RunWeekly with no employees ----------

func TestRunWeekly_NoEmployees_SkipsRecognition(t *testing.T) {
	db := &mockActionDB{
		employees:     nil,
		submittedDays: map[string]int{},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Recognition") {
			t.Error("should not send recognition when no employees")
		}
	}
}

// ---------- 3. RunWeekly with unknown mentor ----------

func TestRunWeekly_UnknownMentor_ReturnsError(t *testing.T) {
	db := &mockActionDB{}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "nonexistent_mentor", 999)
	if err == nil {
		t.Fatal("expected error for unknown mentor, got nil")
	}
	if !strings.Contains(err.Error(), "load engine") {
		t.Errorf("expected 'load engine' in error, got: %v", err)
	}
	if len(sender.sentMessages) != 0 {
		t.Errorf("expected 0 messages on error, got %d", len(sender.sentMessages))
	}
}

// ---------- 4. RunMonthly with valid mentor ----------

func TestRunMonthly_Inamori_SendsMessages(t *testing.T) {
	db := &mockActionDB{}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunMonthly(context.Background(), "tenant-1", "inamori", 888)
	if err != nil {
		t.Fatalf("RunMonthly: %v", err)
	}

	if len(sender.sentMessages) != 1 {
		t.Fatalf("expected 1 monthly message, got %d", len(sender.sentMessages))
	}
	msg := sender.sentMessages[0]
	if msg.ChatID != 888 {
		t.Errorf("expected chatID 888, got %d", msg.ChatID)
	}
	if !strings.Contains(msg.Message, "Monthly Action") {
		t.Errorf("expected 'Monthly Action' in message, got: %s", msg.Message)
	}
}

// ---------- 5. RunMonthly with unknown mentor ----------

func TestRunMonthly_UnknownMentor_ReturnsError(t *testing.T) {
	db := &mockActionDB{}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunMonthly(context.Background(), "tenant-1", "no_such_mentor", 888)
	if err == nil {
		t.Fatal("expected error for unknown mentor, got nil")
	}
	if !strings.Contains(err.Error(), "load engine") {
		t.Errorf("expected 'load engine' in error, got: %v", err)
	}
}

// ---------- 6. Recognition shows top contributor ----------

func TestRunWeekly_Recognition_ShowsTopContributor(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
			{ID: "e3", Name: "Charlie", TelegramID: 333},
		},
		submittedDays: map[string]int{
			"e1": 3,
			"e2": 7,
			"e3": 5,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var recognitionMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Recognition") {
			recognitionMsg = m.Message
			break
		}
	}
	if recognitionMsg == "" {
		t.Fatal("no recognition message found")
	}
	if !strings.Contains(recognitionMsg, "Bob") {
		t.Errorf("expected Bob as top contributor, got: %s", recognitionMsg)
	}
	if !strings.Contains(recognitionMsg, "7/7") {
		t.Errorf("expected 7/7 days in recognition, got: %s", recognitionMsg)
	}
}

// ---------- 7. Ranking shows all employees sorted ----------

func TestRunWeekly_Ranking_ShowsAllEmployeesSorted(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
			{ID: "e3", Name: "Charlie", TelegramID: 333},
		},
		submittedDays: map[string]int{
			"e1": 3,
			"e2": 7,
			"e3": 5,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var rankingMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Ranking") {
			rankingMsg = m.Message
			break
		}
	}
	if rankingMsg == "" {
		t.Fatal("no ranking message found")
	}

	bobIdx := strings.Index(rankingMsg, "Bob")
	charlieIdx := strings.Index(rankingMsg, "Charlie")
	aliceIdx := strings.Index(rankingMsg, "Alice")

	if bobIdx < 0 || charlieIdx < 0 || aliceIdx < 0 {
		t.Fatalf("not all names found in ranking: %s", rankingMsg)
	}
	if bobIdx > charlieIdx {
		t.Errorf("Bob should appear before Charlie in ranking")
	}
	if charlieIdx > aliceIdx {
		t.Errorf("Charlie should appear before Alice in ranking")
	}
	if !strings.Contains(rankingMsg, "7/7") {
		t.Errorf("expected 7/7 for Bob in ranking, got: %s", rankingMsg)
	}
}

// ---------- 8. OneOnOne flags employees with <=3 days ----------

func TestRunWeekly_OneOnOne_FlagsLowSubmitters(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
			{ID: "e3", Name: "Charlie", TelegramID: 333},
		},
		submittedDays: map[string]int{
			"e1": 6,
			"e2": 2,
			"e3": 1,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var oneOnOneMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "1:1") && strings.Contains(m.Message, "scheduling") {
			oneOnOneMsg = m.Message
			break
		}
	}
	if oneOnOneMsg == "" {
		t.Fatal("no 1:1 suggestion message found")
	}
	if !strings.Contains(oneOnOneMsg, "Bob") {
		t.Errorf("expected Bob flagged for 1:1, got: %s", oneOnOneMsg)
	}
	if !strings.Contains(oneOnOneMsg, "Charlie") {
		t.Errorf("expected Charlie flagged for 1:1, got: %s", oneOnOneMsg)
	}
	if strings.Contains(oneOnOneMsg, "Alice") {
		t.Errorf("Alice (6 days) should NOT be flagged, got: %s", oneOnOneMsg)
	}
}

// ---------- 9. OneOnOne all performing well ----------

func TestRunWeekly_OneOnOne_AllPerformingWell(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
		},
		submittedDays: map[string]int{
			"e1": 5,
			"e2": 7,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var oneOnOneMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "1:1") {
			oneOnOneMsg = m.Message
			break
		}
	}
	if oneOnOneMsg == "" {
		t.Fatal("no 1:1 suggestion message found")
	}
	if !strings.Contains(oneOnOneMsg, "performing well") {
		t.Errorf("expected 'performing well' message, got: %s", oneOnOneMsg)
	}
	if strings.Contains(oneOnOneMsg, "scheduling 1:1s with") {
		t.Errorf("should not suggest scheduling when all performing well, got: %s", oneOnOneMsg)
	}
}

// ---------- 10. Recognition no employees -> skipped ----------

func TestRunWeekly_Recognition_NoEmployees_Skipped(t *testing.T) {
	db := &mockActionDB{
		employees:     nil,
		submittedDays: map[string]int{},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Recognition") {
			t.Errorf("should not send recognition when no employees, got: %s", m.Message)
		}
	}
}

// ---------- 11. Recognition all zero submissions -> skipped ----------

func TestRunWeekly_Recognition_AllZeroSubmissions_Skipped(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
			{ID: "e2", Name: "Bob", TelegramID: 222},
		},
		submittedDays: map[string]int{
			"e1": 0,
			"e2": 0,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Recognition") {
			t.Errorf("should not send recognition with all zero, got: %s", m.Message)
		}
	}
}

// ---------- Additional coverage ----------

func TestRunWeekly_Recognition_SingleEmployee(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Solo", TelegramID: 111},
		},
		submittedDays: map[string]int{"e1": 4},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "inamori", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var recognitionMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Recognition") {
			recognitionMsg = m.Message
			break
		}
	}
	if recognitionMsg == "" {
		t.Fatal("expected recognition message for single employee")
	}
	if !strings.Contains(recognitionMsg, "Solo") {
		t.Errorf("expected Solo in recognition, got: %s", recognitionMsg)
	}
	if !strings.Contains(recognitionMsg, "4/7") {
		t.Errorf("expected 4/7 in recognition, got: %s", recognitionMsg)
	}
}

func TestRunWeekly_Ranking_MedalsAssigned(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Gold", TelegramID: 111},
			{ID: "e2", Name: "Silver", TelegramID: 222},
			{ID: "e3", Name: "Bronze", TelegramID: 333},
			{ID: "e4", Name: "Fourth", TelegramID: 444},
		},
		submittedDays: map[string]int{
			"e1": 7,
			"e2": 5,
			"e3": 3,
			"e4": 1,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var rankingMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "Ranking") {
			rankingMsg = m.Message
			break
		}
	}
	if rankingMsg == "" {
		t.Fatal("no ranking message found")
	}

	lines := strings.Split(rankingMsg, "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, "/7 days") {
			dataLines = append(dataLines, line)
		}
	}

	if len(dataLines) != 4 {
		t.Fatalf("expected 4 ranking lines, got %d: %v", len(dataLines), dataLines)
	}

	if !strings.Contains(dataLines[0], "Gold") {
		t.Errorf("first line should be Gold: %s", dataLines[0])
	}
	if !strings.Contains(dataLines[1], "Silver") {
		t.Errorf("second line should be Silver: %s", dataLines[1])
	}
	if !strings.Contains(dataLines[2], "Bronze") {
		t.Errorf("third line should be Bronze: %s", dataLines[2])
	}
	if !strings.Contains(dataLines[3], "Fourth") {
		t.Errorf("fourth line should be Fourth: %s", dataLines[3])
	}
}

func TestRunWeekly_SelfCriticism_GenericMessage(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
		},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "ren", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var selfCriticismMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "self_criticism") || strings.Contains(m.Message, "self-reflection") {
			selfCriticismMsg = m.Message
			break
		}
	}
	if selfCriticismMsg == "" {
		t.Fatal("no self_criticism message found")
	}
	if !strings.Contains(selfCriticismMsg, "self-reflection") {
		t.Errorf("expected self-reflection text, got: %s", selfCriticismMsg)
	}
}

func TestRunWeekly_OKRReview_GenericMessage(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "Alice", TelegramID: 111},
		},
		submittedDays: map[string]int{"e1": 5},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var okrMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "okr_review") || strings.Contains(m.Message, "OKR") {
			okrMsg = m.Message
			break
		}
	}
	if okrMsg == "" {
		t.Fatal("no okr_review message found")
	}
	if !strings.Contains(okrMsg, "OKR progress") {
		t.Errorf("expected OKR progress text, got: %s", okrMsg)
	}
}

func TestRunMonthly_MessageContainsActionType(t *testing.T) {
	db := &mockActionDB{}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunMonthly(context.Background(), "tenant-1", "grove", 777)
	if err != nil {
		t.Fatalf("RunMonthly: %v", err)
	}

	if len(sender.sentMessages) != 1 {
		t.Fatalf("expected 1 monthly message, got %d", len(sender.sentMessages))
	}
	msg := sender.sentMessages[0]
	if msg.ChatID != 777 {
		t.Errorf("expected chatID 777, got %d", msg.ChatID)
	}
	if !strings.Contains(msg.Message, "report") {
		t.Errorf("expected 'report' type in message, got: %s", msg.Message)
	}
}

func TestRunWeekly_OneOnOne_BoundaryThreeDays(t *testing.T) {
	db := &mockActionDB{
		employees: []report.EmployeeInfo{
			{ID: "e1", Name: "OnBoundary", TelegramID: 111},
			{ID: "e2", Name: "JustAbove", TelegramID: 222},
		},
		submittedDays: map[string]int{
			"e1": 3,
			"e2": 4,
		},
	}
	sender := &mockSender{}
	exec := newActionExecutor(db, sender)

	err := exec.RunWeekly(context.Background(), "tenant-1", "grove", 999)
	if err != nil {
		t.Fatalf("RunWeekly: %v", err)
	}

	var oneOnOneMsg string
	for _, m := range sender.sentMessages {
		if strings.Contains(m.Message, "1:1") && strings.Contains(m.Message, "scheduling") {
			oneOnOneMsg = m.Message
			break
		}
	}
	if oneOnOneMsg == "" {
		t.Fatal("no 1:1 suggestion message found")
	}
	if !strings.Contains(oneOnOneMsg, "OnBoundary") {
		t.Errorf("employee with exactly 3 days should be flagged, got: %s", oneOnOneMsg)
	}
	if strings.Contains(oneOnOneMsg, "JustAbove") {
		t.Errorf("employee with 4 days should NOT be flagged, got: %s", oneOnOneMsg)
	}
}
