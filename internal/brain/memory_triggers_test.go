package brain

import (
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

func makeMemory(content string, daysAgo int) memory.Memory {
	return memory.Memory{
		Content:   content,
		Summary:   content,
		CreatedAt: time.Now().AddDate(0, 0, -daysAgo),
		Importance: 0.7,
	}
}

func TestCheckStressPattern_BelowThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("feeling stressed today", 1),
		makeMemory("had a good meeting", 2),
		makeMemory("normal day", 3),
	}
	result := eval.checkStressPattern(memories, "emp-1", "Alice")
	if result != nil {
		t.Fatal("expected nil for <3 stress mentions")
	}
}

func TestCheckStressPattern_TriggersAtThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("I'm so stressed with deadlines", 1),
		makeMemory("feeling overwhelmed with work", 2),
		makeMemory("too much pressure from management", 3),
	}
	result := eval.checkStressPattern(memories, "emp-1", "Alice")
	if result == nil {
		t.Fatal("expected recommendation for 3 stress mentions")
	}
	if result.Priority != "high" {
		t.Errorf("expected priority high, got %s", result.Priority)
	}
	if result.Category != "people" {
		t.Errorf("expected category people, got %s", result.Category)
	}
}

func TestCheckStressPattern_ChineseKeywords(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("最近工作压力很大", 1),
		makeMemory("又加班到很晚", 2),
		makeMemory("感觉很疲惫", 3),
	}
	result := eval.checkStressPattern(memories, "emp-1", "张三")
	if result == nil {
		t.Fatal("expected recommendation for Chinese stress keywords")
	}
}

func TestCheckRepeatedBlocker_BelowThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("blocked by API team", 1),
		makeMemory("making progress on feature", 2),
		makeMemory("completed task", 3),
	}
	result := eval.checkRepeatedBlocker(memories, "emp-1", "Bob")
	if result != nil {
		t.Fatal("expected nil for <3 blocker mentions")
	}
}

func TestCheckRepeatedBlocker_TriggersAtThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("still blocked by infrastructure", 1),
		makeMemory("waiting on dependency from team B", 2),
		makeMemory("stuck on the same issue as yesterday", 3),
	}
	result := eval.checkRepeatedBlocker(memories, "emp-1", "Bob")
	if result == nil {
		t.Fatal("expected recommendation for 3 blocker mentions")
	}
	if result.Category != "project" {
		t.Errorf("expected category project, got %s", result.Category)
	}
}

func TestCheckGrowthSignal_BelowThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("completed the feature on time", 1),
		makeMemory("normal day at work", 2),
		makeMemory("had a meeting", 3),
	}
	result := eval.checkGrowthSignal(memories, "emp-1", "Carol")
	if result != nil {
		t.Fatal("expected nil for <3 growth mentions")
	}
}

func TestCheckGrowthSignal_TriggersAtThreshold(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("completed the migration project", 1),
		makeMemory("shipped the new dashboard feature", 2),
		makeMemory("achieved 100% test coverage on the module", 3),
		makeMemory("led the team standup and improved the process", 4),
	}
	result := eval.checkGrowthSignal(memories, "emp-1", "Carol")
	if result == nil {
		t.Fatal("expected recommendation for 4 growth mentions")
	}
	if result.Priority != "medium" {
		t.Errorf("expected priority medium, got %s", result.Priority)
	}
}

func TestCheckStressPattern_NoFalsePositives(t *testing.T) {
	eval := &MemoryPatternEvaluator{}
	memories := []memory.Memory{
		makeMemory("had a great day today", 1),
		makeMemory("everything went smoothly", 2),
		makeMemory("productive meeting with the team", 3),
		makeMemory("on track with all deliverables", 4),
	}
	result := eval.checkStressPattern(memories, "emp-1", "Dave")
	if result != nil {
		t.Fatal("expected nil for positive memories")
	}
}

func TestExtractThemes(t *testing.T) {
	mems := []memory.Memory{
		{Content: "Working on deployment pipeline", Summary: "pipeline work"},
		{Content: "Deployment issues with pipeline", Summary: "deployment problems"},
		{Content: "Fixed deployment bug in pipeline", Summary: "bug fix"},
	}
	themes := extractThemes(mems)
	if len(themes) == 0 {
		t.Fatal("expected at least one theme")
	}
	// "deployment" and "pipeline" should be top themes
	found := false
	for _, theme := range themes {
		if theme == "deployment" || theme == "pipeline" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected deployment or pipeline in themes, got %v", themes)
	}
}

func TestExtractThemes_Empty(t *testing.T) {
	themes := extractThemes(nil)
	if len(themes) != 0 {
		t.Errorf("expected empty themes for nil input, got %v", themes)
	}
}
