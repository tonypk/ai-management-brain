package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CustomMentorRequest holds the input for creating a custom mentor via AI.
type CustomMentorRequest struct {
	Name        string `json:"name"`         // e.g., "Steve Jobs"
	Description string `json:"description"`  // optional: additional context about the person
}

// CustomMentorResult holds the generated mentor configuration.
type CustomMentorResult struct {
	Config   *MentorConfig `json:"config"`
	FilePath string        `json:"file_path"`
}

// MentorGenerator creates custom mentors using AI.
type MentorGenerator struct {
	llm LLMClient
}

// NewMentorGenerator creates a new custom mentor generator.
func NewMentorGenerator(llm LLMClient) *MentorGenerator {
	return &MentorGenerator{llm: llm}
}

// Generate creates a custom mentor YAML from a person's name using AI.
func (g *MentorGenerator) Generate(ctx context.Context, req CustomMentorRequest) (*CustomMentorResult, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("mentor name is required")
	}
	if g.llm == nil {
		return nil, fmt.Errorf("LLM client is required for custom mentor generation")
	}

	// Generate the mentor config via Claude
	config, err := g.generateConfig(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	// Save to YAML file
	filePath, err := g.saveConfig(config)
	if err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	// Register as a valid mentor
	ValidMentors[config.ID] = true

	slog.Info("custom mentor created",
		"id", config.ID,
		"name", config.Name,
		"name_en", config.NameEn,
		"file", filePath,
	)

	return &CustomMentorResult{
		Config:   config,
		FilePath: filePath,
	}, nil
}

// generateConfig uses Claude to extract management philosophy and generate a MentorConfig.
func (g *MentorGenerator) generateConfig(ctx context.Context, req CustomMentorRequest) (*MentorConfig, error) {
	systemPrompt := `You are an expert on management philosophies and leadership styles. Given a person's name (and optional description), extract their management philosophy and generate a complete management mentor configuration.

Respond in this exact JSON format:
{
  "id": "lowercase_no_spaces",
  "name": "Name in their native language or English",
  "name_en": "Name in English",
  "company": "Their most famous company/organization",
  "philosophy": "Core philosophy in one sentence (Chinese preferred if applicable, otherwise English)",
  "checkin_questions": ["3 daily check-in questions that reflect their philosophy"],
  "chase_method": "private_first or group_first",
  "chase_escalation": [
    {"action": "private_message", "delay": "0", "tone": "warm_reminder"},
    {"action": "manager_notify", "delay": "2h", "tone": "caring_concern"},
    {"action": "skip_today", "delay": "4h"}
  ],
  "chase_forbidden": ["list of forbidden chase approaches"],
  "chase_encouraged": ["list of encouraged chase approaches"],
  "summary_focus": ["3-4 areas to focus on in daily summaries"],
  "summary_highlight": "what to highlight",
  "summary_flag": "what to flag as warning",
  "summary_metrics": [{"name": "Metric name", "source": "data_source"}],
  "weekly_actions": [{"type": "action_type", "desc": "description"}],
  "monthly_actions": [{"type": "action_type", "desc": "description"}],
  "triggers": [
    {"event": "consecutive_miss_3days", "action": "action_type", "message": "message with {name} placeholder"},
    {"event": "sentiment_drop", "action": "action_type", "message": "message"}
  ],
  "system_prompt": "A detailed system prompt (5-10 lines) capturing this person's management philosophy and principles"
}

Make the configuration authentic to the person's known management style. The check-in questions should reflect what this person would actually ask their team.`

	userPrompt := fmt.Sprintf("Create a management mentor configuration for: %s", req.Name)
	if req.Description != "" {
		userPrompt += fmt.Sprintf("\n\nAdditional context: %s", req.Description)
	}

	resp, err := g.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generate mentor: %w", err)
	}

	// Parse JSON response
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Find JSON in response
	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start >= 0 && end > start {
		resp = resp[start : end+1]
	}

	var raw struct {
		ID               string         `json:"id"`
		Name             string         `json:"name"`
		NameEn           string         `json:"name_en"`
		Company          string         `json:"company"`
		Philosophy       string         `json:"philosophy"`
		CheckinQuestions []string       `json:"checkin_questions"`
		ChaseMethod      string         `json:"chase_method"`
		ChaseEscalation  []struct {
			Action string `json:"action"`
			Delay  string `json:"delay"`
			Tone   string `json:"tone"`
		} `json:"chase_escalation"`
		ChaseForbidden   []string       `json:"chase_forbidden"`
		ChaseEncouraged  []string       `json:"chase_encouraged"`
		SummaryFocus     []string       `json:"summary_focus"`
		SummaryHighlight string         `json:"summary_highlight"`
		SummaryFlag      string         `json:"summary_flag"`
		SummaryMetrics   []MetricConfig `json:"summary_metrics"`
		WeeklyActions    []ActionItem   `json:"weekly_actions"`
		MonthlyActions   []ActionItem   `json:"monthly_actions"`
		Triggers         []TriggerRule  `json:"triggers"`
		SystemPrompt     string         `json:"system_prompt"`
	}

	if err := json.Unmarshal([]byte(resp), &raw); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w (response: %.200s)", err, resp)
	}

	// Build escalation steps
	escalation := make([]EscalationStep, len(raw.ChaseEscalation))
	for i, e := range raw.ChaseEscalation {
		escalation[i] = EscalationStep{
			Action: e.Action,
			Delay:  e.Delay,
			Tone:   e.Tone,
		}
	}

	config := &MentorConfig{
		ID:         raw.ID,
		Name:       raw.Name,
		NameEn:     raw.NameEn,
		Company:    raw.Company,
		Philosophy: raw.Philosophy,
		Version:    1,
		Strategy: Strategy{
			CheckinQuestions: raw.CheckinQuestions,
			Chase: ChaseConfig{
				Method:     raw.ChaseMethod,
				Escalation: escalation,
				Forbidden:  raw.ChaseForbidden,
				Encouraged: raw.ChaseEncouraged,
			},
			Summary: SummaryConfig{
				Focus:     raw.SummaryFocus,
				Highlight: raw.SummaryHighlight,
				Flag:      raw.SummaryFlag,
				Metrics:   raw.SummaryMetrics,
			},
			Actions: ActionsConfig{
				Weekly:   raw.WeeklyActions,
				Monthly:  raw.MonthlyActions,
				Triggers: raw.Triggers,
			},
			SystemPrompt: raw.SystemPrompt,
		},
	}

	return config, nil
}

// saveConfig writes the mentor config as a YAML file in configs/mentors/.
func (g *MentorGenerator) saveConfig(config *MentorConfig) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal YAML: %w", err)
	}

	// Find configs/mentors/ directory
	dir, err := findMentorsDir()
	if err != nil {
		return "", fmt.Errorf("find mentors dir: %w", err)
	}

	filePath := filepath.Join(dir, config.ID+".yaml")

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return filePath, nil
}

// findMentorsDir locates the configs/mentors/ directory by searching upward.
func findMentorsDir() (string, error) {
	rel := filepath.Join("configs", "mentors")

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, rel)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("configs/mentors/ not found (searched from %s)", cwd)
}
