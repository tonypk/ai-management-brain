package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// Intent represents a detected boss intent.
type Intent struct {
	Type       IntentType
	Target     string // employee name or "team"
	Content    string // the actual message/question
	OriginalNL string // original natural language input
}

// IntentType classifies the boss's intent.
type IntentType string

const (
	IntentAskEmployee  IntentType = "ask_employee"   // "Ask John about X"
	IntentAnnounce     IntentType = "announce"        // "Tell the team X"
	IntentCheckStatus  IntentType = "check_status"    // "How is John doing?"
	IntentSetReminder  IntentType = "set_reminder"    // "Remind me to X"
	IntentSwitchMentor IntentType = "switch_mentor"   // "Switch to Dalio"
	IntentGetSummary   IntentType = "get_summary"     // "Give me today's summary"
	IntentUnknown      IntentType = "unknown"
)

// Task is a structured action dispatched by the orchestrator.
type Task struct {
	Intent  Intent
	Action  string            // the concrete action to take
	Params  map[string]string // action parameters
	Status  string            // "pending", "in_progress", "completed", "failed"
	Result  string            // result message after execution
}

// Orchestrator detects intent from natural language and dispatches sub-tasks.
type Orchestrator struct {
	llm LLMClient
}

// NewOrchestrator creates a new agent orchestrator.
func NewOrchestrator(llm LLMClient) *Orchestrator {
	return &Orchestrator{llm: llm}
}

// DetectIntent parses a natural language boss command and returns a structured intent.
// Uses pattern matching first, then falls back to LLM for ambiguous inputs.
func (o *Orchestrator) DetectIntent(ctx context.Context, text string) (Intent, error) {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Pattern matching for common intents
	if intent, ok := matchPattern(lower, text); ok {
		return intent, nil
	}

	// Fall back to LLM for complex NL
	if o.llm == nil {
		return Intent{Type: IntentUnknown, OriginalNL: text}, nil
	}

	return o.detectIntentLLM(ctx, text)
}

// Dispatch converts an intent into an executable task.
func (o *Orchestrator) Dispatch(intent Intent) Task {
	task := Task{
		Intent: intent,
		Status: "pending",
		Params: make(map[string]string),
	}

	switch intent.Type {
	case IntentAskEmployee:
		task.Action = "send_question"
		task.Params["target"] = intent.Target
		task.Params["question"] = intent.Content
	case IntentAnnounce:
		task.Action = "broadcast"
		task.Params["message"] = intent.Content
	case IntentCheckStatus:
		task.Action = "check_employee_status"
		task.Params["employee"] = intent.Target
	case IntentSetReminder:
		task.Action = "create_reminder"
		task.Params["content"] = intent.Content
	case IntentSwitchMentor:
		task.Action = "switch_mentor"
		task.Params["mentor_id"] = intent.Target
	case IntentGetSummary:
		task.Action = "generate_summary"
	default:
		task.Action = "unknown"
		task.Status = "failed"
		task.Result = "I didn't understand that command. Try: 'Ask [name] about [topic]' or 'Tell the team [message]'"
	}

	return task
}

// matchPattern tries rule-based intent detection for common patterns.
func matchPattern(lower, original string) (Intent, bool) {
	// "ask [name] about [topic]" / "ask [name] [topic]"
	if strings.HasPrefix(lower, "ask ") {
		rest := strings.TrimPrefix(lower, "ask ")
		if name, content, ok := splitNameContent(rest, "about"); ok {
			return Intent{Type: IntentAskEmployee, Target: name, Content: content, OriginalNL: original}, true
		}
	}

	// "tell the team [message]" / "announce [message]"
	if strings.HasPrefix(lower, "tell the team ") || strings.HasPrefix(lower, "tell everyone ") {
		content := strings.TrimPrefix(lower, "tell the team ")
		content = strings.TrimPrefix(content, "tell everyone ")
		return Intent{Type: IntentAnnounce, Target: "team", Content: content, OriginalNL: original}, true
	}
	if strings.HasPrefix(lower, "announce ") {
		content := strings.TrimPrefix(lower, "announce ")
		return Intent{Type: IntentAnnounce, Target: "team", Content: content, OriginalNL: original}, true
	}

	// "how is [name]" / "status of [name]"
	if strings.HasPrefix(lower, "how is ") {
		name := strings.TrimPrefix(lower, "how is ")
		name = strings.TrimSuffix(name, "?")
		name = strings.TrimSuffix(name, " doing")
		return Intent{Type: IntentCheckStatus, Target: strings.TrimSpace(name), OriginalNL: original}, true
	}
	if strings.HasPrefix(lower, "status of ") {
		name := strings.TrimPrefix(lower, "status of ")
		name = strings.TrimSuffix(name, "?")
		return Intent{Type: IntentCheckStatus, Target: strings.TrimSpace(name), OriginalNL: original}, true
	}

	// "switch to [mentor]" / "use [mentor]"
	if strings.HasPrefix(lower, "switch to ") {
		mentor := strings.TrimPrefix(lower, "switch to ")
		return Intent{Type: IntentSwitchMentor, Target: strings.TrimSpace(mentor), OriginalNL: original}, true
	}
	if strings.HasPrefix(lower, "use ") && ValidMentors[strings.TrimSpace(strings.TrimPrefix(lower, "use "))] {
		mentor := strings.TrimPrefix(lower, "use ")
		return Intent{Type: IntentSwitchMentor, Target: strings.TrimSpace(mentor), OriginalNL: original}, true
	}

	// "summary" / "give me today's summary"
	if lower == "summary" || strings.Contains(lower, "today's summary") || strings.Contains(lower, "daily summary") {
		return Intent{Type: IntentGetSummary, OriginalNL: original}, true
	}

	// "remind me to [content]"
	if strings.HasPrefix(lower, "remind me ") {
		content := strings.TrimPrefix(lower, "remind me ")
		content = strings.TrimPrefix(content, "to ")
		return Intent{Type: IntentSetReminder, Content: content, OriginalNL: original}, true
	}

	return Intent{}, false
}

// splitNameContent splits "name about content" into (name, content).
func splitNameContent(s, separator string) (string, string, bool) {
	parts := strings.SplitN(s, " "+separator+" ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	// Try splitting on first space (for "ask John how are you")
	parts = strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return "", "", false
}

// detectIntentLLM uses Claude to classify ambiguous natural language.
func (o *Orchestrator) detectIntentLLM(ctx context.Context, text string) (Intent, error) {
	systemPrompt := `You are an intent classifier for a management bot. Classify the boss's message into one of these intents:
- ask_employee: Boss wants to ask a specific employee something. Extract: target (employee name), content (the question)
- announce: Boss wants to tell the whole team something. Extract: content (the message)
- check_status: Boss wants to check on an employee. Extract: target (employee name)
- switch_mentor: Boss wants to change management mentor. Extract: target (mentor id: inamori/dalio/grove/ren)
- get_summary: Boss wants today's report summary
- set_reminder: Boss wants a reminder. Extract: content (reminder text)
- unknown: Can't determine intent

Respond in this exact JSON format only:
{"type": "ask_employee", "target": "John", "content": "how is the project going?"}`

	resp, err := o.llm.Chat(ctx, systemPrompt, text)
	if err != nil {
		slog.Warn("LLM intent detection failed, using unknown", "error", err)
		return Intent{Type: IntentUnknown, OriginalNL: text}, nil
	}

	resp = strings.TrimSpace(resp)
	// Handle markdown code blocks
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var parsed struct {
		Type    string `json:"type"`
		Target  string `json:"target"`
		Content string `json:"content"`
	}
	if err := parseJSON([]byte(resp), &parsed); err != nil {
		slog.Warn("failed to parse LLM intent response", "response", resp, "error", err)
		return Intent{Type: IntentUnknown, OriginalNL: text}, nil
	}

	return Intent{
		Type:       IntentType(parsed.Type),
		Target:     parsed.Target,
		Content:    parsed.Content,
		OriginalNL: text,
	}, nil
}

// parseJSON is a helper that wraps json.Unmarshal.
func parseJSON(data []byte, v interface{}) error {
	return fmt.Errorf("%w", func() error {
		// Try to find JSON in the response
		start := strings.Index(string(data), "{")
		end := strings.LastIndex(string(data), "}")
		if start >= 0 && end > start {
			data = data[start : end+1]
		}

		if err := json.Unmarshal(data, v); err != nil {
			return err
		}
		return nil
	}())
}
