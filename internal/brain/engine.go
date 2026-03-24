package brain

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

// ValidMentors lists all available mentor IDs.
var ValidMentors = map[string]bool{
	"inamori":     true,
	"dalio":       true,
	"grove":       true,
	"ren":         true,
	"son":         true,
	"jobs":        true,
	"bezos":       true,
	"ma":          true,
	"musk":        true,
	"buffett":     true,
	"zhangyiming": true,
	"leijun":      true,
	"caodewang":   true,
	"chushijian":  true,
	"trout":       true,
	"meyer":       true,
}

// ValidCultures lists all available culture codes.
var ValidCultures = map[string]bool{
	"default":     true,
	"philippines": true,
	"singapore":   true,
	"indonesia":   true,
	"srilanka":    true,
	"malaysia":    true,
	"china":       true,
}

// Engine assembles mentor strategy + culture pack into executable decisions.
type Engine struct {
	mentor       *MentorConfig
	culture      *CulturePack
	memoryEngine *memory.MemoryEngine // optional, nil = memory disabled
}

// SetMemoryEngine injects the memory engine for prompt enrichment.
func (e *Engine) SetMemoryEngine(me *memory.MemoryEngine) {
	e.memoryEngine = me
}

// MemoryEngine returns the associated memory engine (may be nil).
func (e *Engine) MemoryEngine() *memory.MemoryEngine {
	return e.memoryEngine
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

// MentorID returns the loaded mentor's ID.
func (e *Engine) MentorID() string {
	return e.mentor.ID
}

// MentorName returns the loaded mentor's display name.
func (e *Engine) MentorName() string {
	return e.mentor.Name
}

// EngineFactory creates Engine instances with caching.
type EngineFactory struct {
	mu           sync.RWMutex
	cache        map[string]*Engine
	memoryEngine *memory.MemoryEngine // shared across all engines
}

// SetMemoryEngine injects the memory engine into the factory and all cached engines.
func (f *EngineFactory) SetMemoryEngine(me *memory.MemoryEngine) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.memoryEngine = me
	for _, e := range f.cache {
		e.memoryEngine = me
	}
}

// NewEngineFactory creates a new factory.
func NewEngineFactory() *EngineFactory {
	return &EngineFactory{cache: make(map[string]*Engine)}
}

// ForTenant returns a cached or newly created Engine for the given mentor + culture pair.
func (f *EngineFactory) ForTenant(mentorID, cultureCode string) (*Engine, error) {
	key := mentorID + ":" + cultureCode
	f.mu.RLock()
	if e, ok := f.cache[key]; ok {
		f.mu.RUnlock()
		return e, nil
	}
	f.mu.RUnlock()

	e, err := NewEngine(mentorID, cultureCode)
	if err != nil {
		return nil, err
	}
	e.memoryEngine = f.memoryEngine

	f.mu.Lock()
	f.cache[key] = e
	f.mu.Unlock()
	return e, nil
}

// Invalidate removes a cached engine (e.g. after mentor switch).
func (f *EngineFactory) Invalidate(mentorID, cultureCode string) {
	key := mentorID + ":" + cultureCode
	f.mu.Lock()
	delete(f.cache, key)
	f.mu.Unlock()
}

// ForBlend creates a blended engine from two mentors with a weight.
// weight is for the primary mentor (e.g. 0.7 = 70% primary, 30% secondary).
func (f *EngineFactory) ForBlend(primaryID, secondaryID string, weight float64, cultureCode string) (*Engine, error) {
	key := fmt.Sprintf("blend:%s+%s@%.0f:%s", primaryID, secondaryID, weight*100, cultureCode)
	f.mu.RLock()
	if e, ok := f.cache[key]; ok {
		f.mu.RUnlock()
		return e, nil
	}
	f.mu.RUnlock()

	e, err := NewBlendedEngine(primaryID, secondaryID, weight, cultureCode)
	if err != nil {
		return nil, err
	}
	e.memoryEngine = f.memoryEngine

	f.mu.Lock()
	f.cache[key] = e
	f.mu.Unlock()
	return e, nil
}

// BlendConfig holds mentor blending weights.
type BlendConfig struct {
	PrimaryID   string  `json:"primary_id"`
	SecondaryID string  `json:"secondary_id"`
	Weight      float64 `json:"weight"` // 0.0–1.0 for primary
}

// NewBlendedEngine creates an engine that blends two mentors.
// Questions: primary's questions + last question from secondary (if different).
// Chase: primary's chase strategy.
// Summary: merged focus areas from both.
// System prompt: primary's prompt + secondary's key principles appended.
// Triggers/Actions: merged from both.
func NewBlendedEngine(primaryID, secondaryID string, weight float64, cultureCode string) (*Engine, error) {
	primary, err := LoadMentor(primaryID)
	if err != nil {
		return nil, fmt.Errorf("load primary mentor %q: %w", primaryID, err)
	}
	secondary, err := LoadMentor(secondaryID)
	if err != nil {
		return nil, fmt.Errorf("load secondary mentor %q: %w", secondaryID, err)
	}
	culture, err := LoadCulture(cultureCode)
	if err != nil {
		return nil, fmt.Errorf("load culture %q: %w", cultureCode, err)
	}

	blended := blendMentors(primary, secondary, weight)
	return &Engine{mentor: blended, culture: culture}, nil
}

// blendMentors creates a new MentorConfig that blends two mentors.
func blendMentors(primary, secondary *MentorConfig, weight float64) *MentorConfig {
	blended := *primary // copy primary as base

	blended.ID = primary.ID + "+" + secondary.ID
	blended.Name = primary.Name + " × " + secondary.Name
	blended.NameEn = primary.NameEn + " × " + secondary.NameEn
	blended.Philosophy = primary.Philosophy + " + " + secondary.Philosophy

	// Questions: primary's questions + optionally append one from secondary
	questions := make([]string, len(primary.Strategy.CheckinQuestions))
	copy(questions, primary.Strategy.CheckinQuestions)
	if len(secondary.Strategy.CheckinQuestions) > 0 {
		// Add last question from secondary (most distinctive)
		lastQ := secondary.Strategy.CheckinQuestions[len(secondary.Strategy.CheckinQuestions)-1]
		questions = append(questions, lastQ)
	}
	blended.Strategy.CheckinQuestions = questions

	// Chase: use primary's strategy (culture will override if needed)

	// Summary: merge focus areas (deduplicate)
	focusMap := make(map[string]bool)
	var mergedFocus []string
	for _, f := range primary.Strategy.Summary.Focus {
		if !focusMap[f] {
			focusMap[f] = true
			mergedFocus = append(mergedFocus, f)
		}
	}
	for _, f := range secondary.Strategy.Summary.Focus {
		if !focusMap[f] {
			focusMap[f] = true
			mergedFocus = append(mergedFocus, f)
		}
	}
	blended.Strategy.Summary.Focus = mergedFocus

	// Merge metrics from both
	metricsMap := make(map[string]bool)
	var mergedMetrics []MetricConfig
	for _, m := range primary.Strategy.Summary.Metrics {
		if !metricsMap[m.Name] {
			metricsMap[m.Name] = true
			mergedMetrics = append(mergedMetrics, m)
		}
	}
	for _, m := range secondary.Strategy.Summary.Metrics {
		if !metricsMap[m.Name] {
			metricsMap[m.Name] = true
			mergedMetrics = append(mergedMetrics, m)
		}
	}
	blended.Strategy.Summary.Metrics = mergedMetrics

	// System prompt: primary + secondary key principles
	blended.Strategy.SystemPrompt = fmt.Sprintf(
		"%s\n\n--- Secondary Mentor Influence (%s, %.0f%%) ---\n%s",
		primary.Strategy.SystemPrompt,
		secondary.NameEn,
		(1-weight)*100,
		secondary.Strategy.SystemPrompt,
	)

	// Triggers: merge from both (deduplicate by event)
	triggerMap := make(map[string]bool)
	var mergedTriggers []TriggerRule
	for _, tr := range primary.Strategy.Actions.Triggers {
		triggerMap[tr.Event] = true
		mergedTriggers = append(mergedTriggers, tr)
	}
	for _, tr := range secondary.Strategy.Actions.Triggers {
		if !triggerMap[tr.Event] {
			mergedTriggers = append(mergedTriggers, tr)
		}
	}
	blended.Strategy.Actions.Triggers = mergedTriggers

	return &blended
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

// BuildSystemPromptWithMemory assembles the system prompt and injects relevant memories.
// If the memory engine is nil/disabled or recall fails, it falls back to the base prompt.
func (e *Engine) BuildSystemPromptWithMemory(ctx context.Context, tenantID, employeeID, queryText string) string {
	prompt := e.BuildSystemPrompt()

	if e.memoryEngine == nil || !e.memoryEngine.Enabled() {
		return prompt
	}

	result, err := e.memoryEngine.RecallForMentor(ctx, tenantID, employeeID, queryText)
	if err != nil {
		slog.Warn("memory recall failed, using base prompt", "error", err)
		return prompt
	}

	memorySection := memory.FormatForPrompt(result)
	if memorySection != "" {
		prompt += "\n\n" + memorySection + "\n"
	}

	return prompt
}

// BuildBossContext holds the team data to inject into the boss system prompt.
type BuildBossContext struct {
	LatestSummary  string
	SubmissionRate string
	EmployeeList   string
	MemorySection  string // pre-formatted memory recall (empty if no employee mentioned)
}

// BuildBossPrompt assembles the system prompt for boss (chairman) conversations.
func (e *Engine) BuildBossPrompt(ctx context.Context, tenantID string, bctx BuildBossContext) string {
	prompt := e.BuildSystemPrompt()

	prompt += "\n\n<team_context>\n"
	prompt += "## Latest Team Summary\n"
	if bctx.LatestSummary != "" {
		prompt += bctx.LatestSummary
	} else {
		prompt += "(No summary available yet)"
	}
	prompt += "\n\n## Today's Status\n"
	prompt += "Submission rate: " + bctx.SubmissionRate + "\n"
	prompt += "\n## Team Roster\n"
	prompt += bctx.EmployeeList + "\n"
	prompt += "</team_context>\n"

	if bctx.MemorySection != "" {
		prompt += "\n" + bctx.MemorySection + "\n"
	}

	prompt += fmt.Sprintf("\nYou are %s, acting as CEO reporting to the chairman. "+
		"The chairman is consulting you about management decisions. "+
		"Provide data-driven insights based on team performance. Be candid and strategic.",
		e.MentorName())

	return prompt
}

// EmployeeContext holds employee profile data for prompt building.
type EmployeeContext struct {
	Name             string
	JobTitle         string
	Responsibilities string
	Country          string
	Language         string
}

// BuildEmployeeChatPrompt assembles the system prompt for employee mentor conversations.
func (e *Engine) BuildEmployeeChatPrompt(ctx context.Context, tenantID, employeeID string, profile EmployeeContext, queryText string) string {
	prompt := e.BuildSystemPromptWithMemory(ctx, tenantID, employeeID, queryText)

	// Inject employee context if any profile fields are set
	var ctxParts []string
	if profile.JobTitle != "" {
		ctxParts = append(ctxParts, "Job Title: "+profile.JobTitle)
	}
	if profile.Responsibilities != "" {
		ctxParts = append(ctxParts, "Responsibilities: "+profile.Responsibilities)
	}
	if profile.Country != "" {
		ctxParts = append(ctxParts, "Country: "+profile.Country)
	}
	if len(ctxParts) > 0 {
		prompt += "\n\n<employee_context>\n" + strings.Join(ctxParts, "\n") + "\n</employee_context>"
	}

	prompt += fmt.Sprintf("\nYou are %s, acting as CEO and management coach. "+
		"The employee %q is asking you for guidance. "+
		"Respond based on your management philosophy. Keep responses concise and actionable.",
		e.MentorName(), profile.Name)

	if profile.Language != "" {
		prompt += fmt.Sprintf("\nReply in %s.", profile.Language)
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

// GetTriggerRules returns the mentor's trigger rules.
func (e *Engine) GetTriggerRules() []TriggerRule {
	return e.mentor.Strategy.Actions.Triggers
}

// GetWeeklyActions returns the mentor's weekly proactive actions.
func (e *Engine) GetWeeklyActions() []ActionItem {
	return e.mentor.Strategy.Actions.Weekly
}

// GetMonthlyActions returns the mentor's monthly proactive actions.
func (e *Engine) GetMonthlyActions() []ActionItem {
	return e.mentor.Strategy.Actions.Monthly
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
