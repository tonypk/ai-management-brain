package report_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/report"
)

func TestCollector_StartConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?", "Q3?"})
	state, msg, err := c.Start(context.Background(), "emp-123")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if state != report.StateCollecting {
		t.Errorf("state = %v, want Collecting", state)
	}
	if msg != "Q1?" {
		t.Errorf("first question = %q", msg)
	}
}

func TestCollector_AnswerAllQuestions_EntersConfirming(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?", "Q3?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")

	// Answer Q1
	state, msg, _ := c.HandleAnswer(ctx, "emp-123", "Answer to Q1")
	if state != report.StateCollecting {
		t.Errorf("after Q1: state = %v", state)
	}
	if msg != "Q2?" {
		t.Errorf("second question = %q", msg)
	}

	// Answer Q2
	state, msg, _ = c.HandleAnswer(ctx, "emp-123", "Answer to Q2")
	if msg != "Q3?" {
		t.Errorf("third question = %q", msg)
	}

	// Answer Q3 → should enter confirming state (not complete)
	state, msg, _ = c.HandleAnswer(ctx, "emp-123", "Answer to Q3")
	if state != report.StateConfirming {
		t.Errorf("after Q3: state = %v, want Confirming", state)
	}
	if msg == "" {
		t.Error("confirming message should show report summary")
	}
}

func TestCollector_ConfirmReport(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	c.HandleAnswer(ctx, "emp-123", "A1") // → confirming

	// Confirm
	state, _, _ := c.Confirm(ctx, "emp-123")
	if state != report.StateComplete {
		t.Errorf("after confirm: state = %v, want Complete", state)
	}
}

func TestCollector_GetAnswers(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	c.HandleAnswer(ctx, "emp-123", "A1")
	c.HandleAnswer(ctx, "emp-123", "A2")

	answers := c.GetAnswers(ctx, "emp-123")
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}
	if answers["q1"] != "A1" || answers["q2"] != "A2" {
		t.Errorf("answers = %v", answers)
	}
}

func TestCollector_NoActiveConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	state, _, _ := c.HandleAnswer(ctx, "emp-123", "random message")
	if state != report.StateIdle {
		t.Errorf("no active conversation should return Idle, got %v", state)
	}
}

func TestCollector_MidConversationRedirect(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-123")
	if !c.IsCollecting(ctx, "emp-123") {
		t.Error("should be collecting")
	}
}

func TestCollector_GetState_Idle(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	state := c.GetState(context.Background(), "emp-unknown")
	if state != report.StateIdle {
		t.Errorf("expected Idle for unknown employee, got %v", state)
	}
}

func TestCollector_GetState_Collecting(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	state := c.GetState(ctx, "emp-1")
	if state != report.StateCollecting {
		t.Errorf("expected Collecting, got %v", state)
	}
}

func TestCollector_GetState_Confirming(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	c.HandleAnswer(ctx, "emp-1", "A1") // → confirming

	state := c.GetState(ctx, "emp-1")
	if state != report.StateConfirming {
		t.Errorf("expected Confirming, got %v", state)
	}
}

func TestCollector_GetState_Complete(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	c.HandleAnswer(ctx, "emp-1", "A1")
	c.Confirm(ctx, "emp-1")

	state := c.GetState(ctx, "emp-1")
	if state != report.StateComplete {
		t.Errorf("expected Complete, got %v", state)
	}
}

func TestCollector_Cancel(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?", "Q2?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	if !c.IsCollecting(ctx, "emp-1") {
		t.Fatal("should be collecting before cancel")
	}

	c.Cancel(ctx, "emp-1")

	if c.IsCollecting(ctx, "emp-1") {
		t.Error("should not be collecting after cancel")
	}
	state := c.GetState(ctx, "emp-1")
	if state != report.StateIdle {
		t.Errorf("expected Idle after cancel, got %v", state)
	}
}

func TestCollector_StartWithQuestions_Custom(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Default Q1?", "Default Q2?"})
	ctx := context.Background()

	customQs := []string{"Custom Q1?", "Custom Q2?", "Custom Q3?"}
	state, msg, err := c.StartWithQuestions(ctx, "emp-1", customQs)
	if err != nil {
		t.Fatalf("StartWithQuestions: %v", err)
	}
	if state != report.StateCollecting {
		t.Errorf("expected Collecting, got %v", state)
	}
	if msg != "Custom Q1?" {
		t.Errorf("first question = %q, want Custom Q1?", msg)
	}

	// Verify it asks all 3 custom questions
	state, msg, _ = c.HandleAnswer(ctx, "emp-1", "A1")
	if msg != "Custom Q2?" {
		t.Errorf("second question = %q, want Custom Q2?", msg)
	}
	state, msg, _ = c.HandleAnswer(ctx, "emp-1", "A2")
	if msg != "Custom Q3?" {
		t.Errorf("third question = %q, want Custom Q3?", msg)
	}
	state, _, _ = c.HandleAnswer(ctx, "emp-1", "A3")
	if state != report.StateConfirming {
		t.Errorf("expected Confirming after all questions, got %v", state)
	}
}

func TestCollector_StartWithQuestions_FallbackToDefaults(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Default Q1?"})
	ctx := context.Background()

	// Empty custom questions should fallback to defaults
	state, msg, err := c.StartWithQuestions(ctx, "emp-1", nil)
	if err != nil {
		t.Fatalf("StartWithQuestions: %v", err)
	}
	if state != report.StateCollecting {
		t.Errorf("expected Collecting, got %v", state)
	}
	if msg != "Default Q1?" {
		t.Errorf("should fallback to default questions, got %q", msg)
	}
}

func TestCollector_GetAnswers_NoConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	answers := c.GetAnswers(context.Background(), "emp-unknown")
	if answers != nil {
		t.Errorf("expected nil answers for unknown employee, got %v", answers)
	}
}

func TestCollector_IsCollecting_NotCollecting(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	if c.IsCollecting(context.Background(), "emp-unknown") {
		t.Error("should not be collecting for unknown employee")
	}
}

func TestCollector_HandleAnswer_InConfirmingState(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	c.HandleAnswer(ctx, "emp-1", "A1") // → confirming

	// Sending another answer while in confirming state
	state, _, _ := c.HandleAnswer(ctx, "emp-1", "extra answer")
	if state != report.StateConfirming {
		t.Errorf("expected Confirming (not accepting new answers), got %v", state)
	}
}

func TestCollector_Confirm_NoConversation(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	state, _, _ := c.Confirm(context.Background(), "emp-unknown")
	if state != report.StateIdle {
		t.Errorf("expected Idle for confirm without conversation, got %v", state)
	}
}

func TestCollector_MultipleEmployees(t *testing.T) {
	c := report.NewCollector(mockRedis(), []string{"Q1?"})
	ctx := context.Background()

	c.Start(ctx, "emp-1")
	c.Start(ctx, "emp-2")

	// Both should be collecting independently
	if !c.IsCollecting(ctx, "emp-1") {
		t.Error("emp-1 should be collecting")
	}
	if !c.IsCollecting(ctx, "emp-2") {
		t.Error("emp-2 should be collecting")
	}

	// Answer only emp-1
	c.HandleAnswer(ctx, "emp-1", "A1")
	if c.IsCollecting(ctx, "emp-1") {
		t.Error("emp-1 should no longer be collecting (in confirming)")
	}
	if !c.IsCollecting(ctx, "emp-2") {
		t.Error("emp-2 should still be collecting")
	}
}
