package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// StateEngine handles:
// 1. Extracting communication events from check-in reports (Communication Parser)
// 2. Generating execution signals from events + metrics + tasks (Signal Generator)
// 3. Generating working memory snapshots

type StateEngine struct {
	llm     LLMClient
	queries *sqlc.Queries
}

// NewStateEngine creates a new StateEngine.
func NewStateEngine(llm LLMClient, queries *sqlc.Queries) *StateEngine {
	return &StateEngine{llm: llm, queries: queries}
}

// ParsedEvent is a structured event extracted by the Communication Parser prompt.
type ParsedEvent struct {
	EventType    string                 `json:"event_type"`
	ActorHint    string                 `json:"actor_hint"`
	TargetHint   string                 `json:"target_hint"`
	TaskHint     string                 `json:"task_hint"`
	DeadlineHint string                 `json:"deadline_hint"`
	Payload      map[string]interface{} `json:"payload"`
	Confidence   float64                `json:"confidence"`
}

// ParsedEventsResponse is the LLM response from Communication Parser.
type ParsedEventsResponse struct {
	Events []ParsedEvent `json:"events"`
}

const communicationParserPrompt = `You are Communication Parser for Boss AI Agent.

Convert daily check-in reports and chat messages into structured management events.

INPUT: A report or message with sender info and context.

EXTRACT management-relevant events. Supported event types:
- report_submitted: Employee submitted their daily check-in
- blocker_reported: Employee mentioned a blocker or obstacle
- task_completed: Something was finished or delivered
- commitment_made: Employee committed to a deadline or deliverable
- delay_reported: Something is delayed or behind schedule
- escalation_needed: Situation requires manager attention
- proactive_update: Employee proactively shared progress
- acknowledgment: Employee acknowledged a request or task

For each event, return JSON:
{
  "events": [
    {
      "event_type": "blocker_reported",
      "actor_hint": "employee name or ID",
      "target_hint": "who needs to know",
      "task_hint": "related task or project",
      "deadline_hint": "any mentioned deadline",
      "payload": { "summary": "...", "severity": "low|medium|high" },
      "confidence": 0.85
    }
  ]
}

RULES:
- Only extract management-relevant content
- Ignore greetings, casual chat, emojis unless they signal mood
- High confidence (>0.8) for explicit statements: "I'm blocked on X"
- Lower confidence (0.5-0.8) for implied signals: "still working on it" might mean delay
- Never fabricate events. If nothing management-relevant, return empty events array.
- Return ONLY valid JSON, no markdown code blocks.`

// ExtractEventsFromReport parses a check-in report and extracts communication events.
func (se *StateEngine) ExtractEventsFromReport(
	ctx context.Context,
	tenantID pgtype.UUID,
	reportID pgtype.UUID,
	employeeName string,
	answers map[string]string,
) ([]sqlc.CommunicationEvent, error) {
	// Build the input text from report answers
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Employee: %s\n", employeeName))
	sb.WriteString("Report answers:\n")
	for q, a := range answers {
		sb.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", q, a))
	}

	// Call LLM to parse
	resp, err := se.llm.Chat(ctx, communicationParserPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("LLM communication parser: %w", err)
	}

	// Parse JSON response
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var parsed ParsedEventsResponse
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		slog.Warn("failed to parse communication events JSON",
			"response", resp, "error", err)
		return nil, nil // Don't fail the report submission
	}

	// Convert parsed events to database records
	var events []sqlc.CommunicationEvent
	now := time.Now()

	for _, pe := range parsed.Events {
		payload, _ := json.Marshal(pe.Payload)
		var confidence pgtype.Numeric
		_ = confidence.Scan(fmt.Sprintf("%.2f", pe.Confidence))

		event, err := se.queries.CreateCommunicationEvent(ctx, sqlc.CreateCommunicationEventParams{
			TenantID:   tenantID,
			SourceType: "report",
			SourceID:   reportID,
			Platform:   "checkin",
			EventType:  pe.EventType,
			Payload:    payload,
			Confidence: confidence,
			OccurredAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil {
			slog.Error("failed to create communication event",
				"event_type", pe.EventType, "error", err)
			continue
		}
		events = append(events, event)
	}

	slog.Info("extracted communication events from report",
		"employee", employeeName,
		"events_count", len(events),
	)

	return events, nil
}

// GeneratedSignal is a signal produced by the Signal Generator prompt.
type GeneratedSignal struct {
	SignalType string   `json:"signal_type"`
	Score      float64  `json:"score"`
	Reasons    []string `json:"reasons"`
	TimeWindow string   `json:"time_window"`
}

// GeneratedSignalsResponse is the LLM response from Signal Generator.
type GeneratedSignalsResponse struct {
	Signals []GeneratedSignal `json:"signals"`
}

const signalGeneratorPrompt = `You are Execution Signal Generator for Boss AI Agent.

Analyze communication events, task status, and metrics to generate execution risk signals.

Signal types:
- overload_risk: Person has too many tasks or commitments
- delivery_risk: Tasks are overdue or commitments missed
- engagement_drop: Person stopped reporting or communicating
- blocker_cascade: A blocker is affecting multiple people/projects
- performance_spike: Someone is delivering exceptionally well
- metric_anomaly: A KPI is trending significantly off-target

For each signal, return JSON:
{
  "signals": [
    {
      "signal_type": "delivery_risk",
      "score": 0.75,
      "reasons": ["3 tasks overdue", "no update for 2 days"],
      "time_window": "7d"
    }
  ]
}

RULES:
- Score 0.0-1.0 (higher = more urgent)
- Only generate signals with score >= 0.3
- Include specific evidence in reasons
- Return ONLY valid JSON, no markdown code blocks.`

// GenerateSignals analyzes data for a subject and generates execution signals.
func (se *StateEngine) GenerateSignals(
	ctx context.Context,
	tenantID pgtype.UUID,
	subjectType string,
	subjectID pgtype.UUID,
	subjectName string,
) ([]sqlc.ExecutionSignal, error) {
	// Gather context for the subject
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Subject: %s (%s)\n\n", subjectName, subjectType))

	// Get recent events for this person
	events, err := se.queries.ListCommunicationEvents(ctx, sqlc.ListCommunicationEventsParams{
		TenantID: tenantID,
		Limit:    20,
	})
	if err == nil && len(events) > 0 {
		sb.WriteString("Recent communication events:\n")
		for _, e := range events {
			sb.WriteString(fmt.Sprintf("- [%s] %s (confidence: %v)\n",
				e.EventType, string(e.Payload), e.Confidence))
		}
		sb.WriteString("\n")
	}

	// Get overdue tasks
	overdue, err := se.queries.ListOverdueTasks(ctx, tenantID)
	if err == nil && len(overdue) > 0 {
		sb.WriteString("Overdue tasks:\n")
		for _, t := range overdue {
			dueStr := "unknown"
			if t.DueAt.Valid {
				dueStr = t.DueAt.Time.Format("2006-01-02")
			}
			sb.WriteString(fmt.Sprintf("- %s (priority: %s, due: %s)\n",
				t.Title, t.Priority, dueStr))
		}
		sb.WriteString("\n")
	}

	// Get task stats
	stats, err := se.queries.CountTasksByStatus(ctx, tenantID)
	if err == nil && len(stats) > 0 {
		sb.WriteString("Task status breakdown:\n")
		for _, s := range stats {
			sb.WriteString(fmt.Sprintf("- %s: %d\n", s.Status, s.Count))
		}
		sb.WriteString("\n")
	}

	// Call LLM to generate signals
	resp, err := se.llm.Chat(ctx, signalGeneratorPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("LLM signal generator: %w", err)
	}

	// Parse response
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var parsed GeneratedSignalsResponse
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		slog.Warn("failed to parse execution signals JSON",
			"response", resp, "error", err)
		return nil, nil
	}

	// Store signals
	var signals []sqlc.ExecutionSignal
	for _, gs := range parsed.Signals {
		reasons, _ := json.Marshal(gs.Reasons)
		var score pgtype.Numeric
		_ = score.Scan(fmt.Sprintf("%.2f", gs.Score))

		signal, err := se.queries.CreateExecutionSignal(ctx, sqlc.CreateExecutionSignalParams{
			TenantID:    tenantID,
			SubjectType: subjectType,
			SubjectID:   subjectID,
			SignalType:  gs.SignalType,
			Score:       score,
			Reasons:     reasons,
			TimeWindow:  pgtype.Text{String: gs.TimeWindow, Valid: gs.TimeWindow != ""},
		})
		if err != nil {
			slog.Error("failed to create execution signal",
				"signal_type", gs.SignalType, "error", err)
			continue
		}
		signals = append(signals, signal)
	}

	slog.Info("generated execution signals",
		"subject", subjectName,
		"signals_count", len(signals),
	)

	return signals, nil
}

// GenerateWorkingMemory creates a working memory snapshot summarizing current state.
func (se *StateEngine) GenerateWorkingMemory(
	ctx context.Context,
	tenantID pgtype.UUID,
	contextJSON string,
) (*sqlc.WorkingMemorySnapshot, error) {
	systemPrompt := `You are Working Memory Generator for Boss AI Agent.

Given the current company context (goals, metrics, risks, events), produce a concise working memory snapshot.

This snapshot will be used by the AI manager to maintain situational awareness across conversations.

OUTPUT JSON:
{
  "focus_areas": ["Top 3 things needing attention"],
  "risk_summary": "One sentence about top risks",
  "momentum": "positive|neutral|negative",
  "key_decisions_pending": ["Decisions that need to be made"],
  "recent_wins": ["Good things that happened"],
  "action_items": ["What should happen next"]
}

Be concise. This is a memory aid, not a report. Return ONLY valid JSON.`

	resp, err := se.llm.Chat(ctx, systemPrompt, contextJSON)
	if err != nil {
		return nil, fmt.Errorf("LLM working memory: %w", err)
	}

	// Parse and validate JSON
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Validate it's valid JSON
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &content); err != nil {
		slog.Warn("failed to parse working memory JSON", "response", resp, "error", err)
		// Store raw response as content
		content = map[string]interface{}{"raw": resp}
	}

	contentBytes, _ := json.Marshal(content)

	snapshot, err := se.queries.CreateWorkingMemorySnapshot(ctx, sqlc.CreateWorkingMemorySnapshotParams{
		TenantID:     tenantID,
		SnapshotType: "daily",
		Content:      contentBytes,
		GeneratedBy:  pgtype.Text{String: "state_engine", Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create working memory: %w", err)
	}

	slog.Info("generated working memory snapshot", "snapshot_id", snapshot.ID)

	return &snapshot, nil
}
