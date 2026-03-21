package memory

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMemoryEngine_RecallForMentor_NoEmbedder(t *testing.T) {
	engine := NewMemoryEngine(nil, nil, nil, nil, nil, nil)

	result, err := engine.RecallForMentor(context.Background(), "tenant-1", "emp-1", "How are you?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestFormatForPrompt(t *testing.T) {
	now := time.Now()
	result := &RecallResult{
		Profile: &Memory{
			Content: "Diligent worker, prefers written communication.",
		},
		Insights: []Memory{
			{Content: "Reported deadline stress", Importance: 0.8, CreatedAt: now},
			{Content: "Asked about learning opportunities", Importance: 0.6, CreatedAt: now},
		},
		Strategies: []Memory{
			{Content: "Gratitude-style chase improved reply rate from 60% to 90%"},
		},
		TokenCount: 150,
	}

	output := FormatForPrompt(result)

	if !strings.Contains(output, "<memory>") {
		t.Error("expected <memory> tag")
	}
	if !strings.Contains(output, "</memory>") {
		t.Error("expected </memory> tag")
	}
	if !strings.Contains(output, "Employee Profile") {
		t.Error("expected profile section")
	}
	if !strings.Contains(output, "Relevant Memories") {
		t.Error("expected memories section")
	}
	if !strings.Contains(output, "Strategy Insights") {
		t.Error("expected strategy section")
	}
	if !strings.Contains(output, "Gratitude-style") {
		t.Error("expected strategy content")
	}
}

func TestFormatForPrompt_Empty(t *testing.T) {
	output := FormatForPrompt(nil)
	if output != "" {
		t.Errorf("expected empty string for nil result, got %q", output)
	}

	output = FormatForPrompt(&RecallResult{})
	if !strings.Contains(output, "<memory>") {
		t.Error("expected <memory> tag even for empty result")
	}
}
