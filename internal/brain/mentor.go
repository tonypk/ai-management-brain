package brain

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EscalationStep represents one step in the chase escalation sequence.
type EscalationStep struct {
	Action string `yaml:"action"`
	Delay  string `yaml:"delay"`
	Tone   string `yaml:"tone"`
}

// ChaseConfig defines how to chase missing check-ins.
type ChaseConfig struct {
	Method     string           `yaml:"method"`
	Escalation []EscalationStep `yaml:"escalation"`
	Forbidden  []string         `yaml:"forbidden"`
	Encouraged []string         `yaml:"encouraged"`
}

// MetricConfig defines a trackable metric derived from check-in data.
type MetricConfig struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
}

// SummaryConfig defines how to summarise the team's check-ins.
type SummaryConfig struct {
	Focus     []string       `yaml:"focus"`
	Highlight string         `yaml:"highlight"`
	Flag      string         `yaml:"flag"`
	Metrics   []MetricConfig `yaml:"metrics"`
}

// ActionItem represents a recurring action (weekly / monthly).
type ActionItem struct {
	Type string `yaml:"type"`
	Desc string `yaml:"desc"`
}

// TriggerRule defines an automated response to a detected event.
type TriggerRule struct {
	Event   string `yaml:"event"`
	Action  string `yaml:"action"`
	Message string `yaml:"message"`
}

// ActionsConfig groups recurring and event-driven actions.
type ActionsConfig struct {
	Weekly   []ActionItem  `yaml:"weekly"`
	Monthly  []ActionItem  `yaml:"monthly"`
	Triggers []TriggerRule `yaml:"triggers"`
}

// Strategy holds the full behavioural playbook for a mentor.
type Strategy struct {
	CheckinQuestions []string      `yaml:"checkin_questions"`
	Chase            ChaseConfig   `yaml:"chase"`
	Summary          SummaryConfig `yaml:"summary"`
	Actions          ActionsConfig `yaml:"actions"`
	SystemPrompt     string        `yaml:"system_prompt"`
}

// MentorConfig is the top-level structure parsed from a mentor YAML file.
type MentorConfig struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	NameEn     string   `yaml:"name_en"`
	Company    string   `yaml:"company"`
	Philosophy string   `yaml:"philosophy"`
	Version    int      `yaml:"version"`
	Strategy   Strategy `yaml:"strategy"`
}

// GetCheckinQuestions returns the mentor's check-in question list.
func (m *MentorConfig) GetCheckinQuestions() []string {
	return m.Strategy.CheckinQuestions
}

// GetChaseStep returns the 1-indexed escalation step.
// If n exceeds the number of defined steps, a skip_today step is returned.
func (m *MentorConfig) GetChaseStep(n int) EscalationStep {
	steps := m.Strategy.Chase.Escalation
	if n < 1 || n > len(steps) {
		return EscalationStep{Action: "skip_today"}
	}
	return steps[n-1]
}

// GetSummaryConfig returns the mentor's summary configuration.
func (m *MentorConfig) GetSummaryConfig() SummaryConfig {
	return m.Strategy.Summary
}

// BuildSystemPrompt returns the mentor's system prompt text.
func (m *MentorConfig) BuildSystemPrompt() string {
	return m.Strategy.SystemPrompt
}

// LoadMentor reads and parses the YAML config for the given mentor id.
// It searches for configs/mentors/{id}.yaml starting from the working
// directory and walking up the directory tree until found.
func LoadMentor(id string) (*MentorConfig, error) {
	path, err := findMentorFile(id)
	if err != nil {
		return nil, fmt.Errorf("mentor %q not found: %w", id, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mentor file %q: %w", path, err)
	}

	var cfg MentorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mentor file %q: %w", path, err)
	}

	return &cfg, nil
}

// findMentorFile locates configs/mentors/{id}.yaml by searching from the
// current working directory upward.
func findMentorFile(id string) (string, error) {
	rel := filepath.Join("configs", "mentors", id+".yaml")

	// Start from cwd and walk up.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, rel)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding the file.
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("configs/mentors/%s.yaml not found (searched from %s)", id, cwd)
}
