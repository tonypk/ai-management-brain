package brain

import "fmt"

// Engine assembles mentor strategy + culture pack into executable decisions.
type Engine struct {
	mentor  *MentorConfig
	culture *CulturePack
}

// NewEngine creates an Engine by loading the given mentor and culture configs.
func NewEngine(mentorID, cultureCode string) (*Engine, error) {
	m, err := LoadMentor(mentorID)
	if err != nil {
		return nil, fmt.Errorf("load mentor %q: %w", mentorID, err)
	}
	c, err := LoadCulture(cultureCode)
	if err != nil {
		return nil, fmt.Errorf("load culture %q: %w", cultureCode, err)
	}
	return &Engine{mentor: m, culture: c}, nil
}

// BuildSystemPrompt assembles the mentor's system prompt augmented with
// cultural context, forbidden phrases, and preferred phrases.
func (e *Engine) BuildSystemPrompt() string {
	prompt := e.mentor.Strategy.SystemPrompt
	prompt += "\n\n--- Cultural Context ---\n"
	prompt += fmt.Sprintf("Employee culture: %s\n", e.culture.Market)
	prompt += fmt.Sprintf("Communication directness: %s\n", e.culture.CommunicationStyle.Directness)
	if len(e.culture.ForbiddenPatterns) > 0 {
		prompt += "FORBIDDEN phrases (never use these):\n"
		for _, p := range e.culture.ForbiddenPatterns {
			prompt += fmt.Sprintf("- %s\n", p)
		}
	}
	if len(e.culture.PreferredPatterns) > 0 {
		prompt += "Preferred phrases:\n"
		for _, p := range e.culture.PreferredPatterns {
			prompt += fmt.Sprintf("- %s\n", p)
		}
	}
	return prompt
}

// GetCheckinQuestions returns the mentor's check-in question list.
func (e *Engine) GetCheckinQuestions() []string {
	return e.mentor.GetCheckinQuestions()
}

// GetSummaryConfig returns the mentor's summary configuration.
func (e *Engine) GetSummaryConfig() SummaryConfig {
	return e.mentor.GetSummaryConfig()
}

// GetEffectiveChaseStep returns the chase step with cultural overrides applied.
// It returns a copy of the EscalationStep so the original config is not mutated.
func (e *Engine) GetEffectiveChaseStep(step int) EscalationStep {
	s := e.mentor.GetChaseStep(step)
	// Culture override: if culture says never public, downgrade to private.
	if e.culture.ShouldOverride(s.Action) {
		s.Action = "private_message"
	}
	return s
}
