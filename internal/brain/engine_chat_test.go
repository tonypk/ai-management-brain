package brain

import (
	"context"
	"strings"
	"testing"
)

func TestEngine_MentorName(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	name := e.MentorName()
	if name == "" {
		t.Fatal("MentorName should not be empty")
	}
}

func TestEngine_BuildBossPrompt(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildBossPrompt(context.Background(), "tenant-1", BuildBossContext{
		LatestSummary:  "Team performed well today.",
		SubmissionRate: "80% (4/5)",
		EmployeeList:   "1. Alice (member, active)\n2. Bob (manager, active)",
		MemorySection:  "",
	})
	if !strings.Contains(prompt, "chairman") || !strings.Contains(prompt, "CEO") {
		t.Fatal("boss prompt should reference chairman/CEO roles")
	}
	if !strings.Contains(prompt, "Team performed well") {
		t.Fatal("boss prompt should contain the latest summary")
	}
	if !strings.Contains(prompt, "80%") {
		t.Fatal("boss prompt should contain submission rate")
	}
}

func TestEngine_BuildBossPrompt_NoSummary(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildBossPrompt(context.Background(), "tenant-1", BuildBossContext{
		LatestSummary:  "",
		SubmissionRate: "0% (0/5)",
		EmployeeList:   "1. Alice (member, active)",
	})
	if !strings.Contains(prompt, "No summary available") {
		t.Fatal("boss prompt should show fallback when no summary")
	}
}

func TestEngine_BuildEmployeeChatPrompt(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", EmployeeContext{Name: "Alice"}, "I have a problem")
	if !strings.Contains(prompt, "Alice") {
		t.Fatal("employee chat prompt should contain employee name")
	}
	if !strings.Contains(prompt, "coach") {
		t.Fatal("employee chat prompt should reference coaching role")
	}
}

func TestEngine_BuildEmployeeChatPrompt_WithProfile(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", EmployeeContext{
		Name:             "Alice",
		JobTitle:         "Frontend Developer",
		Responsibilities: "Handles UI/UX",
		Country:          "Philippines",
		Language:         "Chinese",
	}, "I have a problem")
	if !strings.Contains(prompt, "<employee_context>") {
		t.Fatal("prompt should contain employee_context block")
	}
	if !strings.Contains(prompt, "Frontend Developer") {
		t.Fatal("prompt should contain job title")
	}
	if !strings.Contains(prompt, "Reply in Chinese") {
		t.Fatal("prompt should contain language instruction")
	}
}

func TestEngine_BuildEmployeeChatPrompt_EmptyProfile(t *testing.T) {
	e, err := NewEngine("inamori", "default")
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.BuildEmployeeChatPrompt(context.Background(), "tenant-1", "emp-1", EmployeeContext{
		Name: "Alice",
	}, "I have a problem")
	if strings.Contains(prompt, "<employee_context>") {
		t.Fatal("prompt should NOT contain employee_context block when all fields empty")
	}
	if strings.Contains(prompt, "Reply in") {
		t.Fatal("prompt should NOT contain language instruction when language empty")
	}
}
