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
