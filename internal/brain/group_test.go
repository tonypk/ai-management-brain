package brain

import (
	"strings"
	"testing"
)

func TestBuildGroupReplyPrompt(t *testing.T) {
	prompt := BuildGroupReplyPrompt("稻盛和夫", "engineering", "本周提交率85%", "如何提升代码质量？")
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "engineering") {
		t.Error("expected group type in prompt")
	}
	if !strings.Contains(prompt, "稻盛和夫") {
		t.Error("expected mentor name in prompt")
	}
	if !strings.Contains(prompt, "如何提升代码质量") {
		t.Error("expected user question in prompt")
	}
	if !strings.Contains(prompt, "NEVER mention individual") {
		t.Error("expected privacy rule in prompt")
	}
}

func TestBuildGroupReplyPrompt_NoSummary(t *testing.T) {
	prompt := BuildGroupReplyPrompt("Dalio", "sales", "", "how to close deals?")
	if strings.Contains(prompt, "Latest team summary") {
		t.Error("should not include summary header when empty")
	}
	if !strings.Contains(prompt, "sales") {
		t.Error("expected group type")
	}
}

func TestBuildGroupDecisionPrompt(t *testing.T) {
	prompt := BuildGroupDecisionPrompt("马斯克", "engineering", GroupTeamData{
		SubmissionRate: "80%",
		SentimentDist:  "positive: 5, neutral: 2",
		LatestSummary:  "团队状态良好",
		Weekday:        "Friday",
	})
	if !strings.Contains(prompt, "马斯克") {
		t.Error("expected mentor name")
	}
	if !strings.Contains(prompt, "engineering") {
		t.Error("expected group type")
	}
	if !strings.Contains(prompt, "SKIP") {
		t.Error("expected SKIP instruction")
	}
	if !strings.Contains(prompt, "Friday") {
		t.Error("expected weekday")
	}
	if !strings.Contains(prompt, "80%") {
		t.Error("expected submission rate")
	}
}

func TestIsSkipDecision(t *testing.T) {
	tests := []struct {
		input    string
		wantSkip bool
	}{
		{"SKIP", true},
		{"  SKIP  ", true},
		{"skip", true},
		{"SKIP\n", true},
		{"Skip", true},
		{"大家早上好！今天继续加油！", false},
		{"", true},
		{"   ", true},
		{"SKIPPING today", false},
		{"Not SKIP", false},
	}
	for _, tt := range tests {
		got := IsSkipDecision(tt.input)
		if got != tt.wantSkip {
			t.Errorf("IsSkipDecision(%q) = %v, want %v", tt.input, got, tt.wantSkip)
		}
	}
}
