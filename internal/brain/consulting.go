package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
)

// ConsultingEngine runs McKinsey-style consulting engagements with a structured
// lifecycle: diagnosis -> analysis -> plan -> execute -> tracking -> close.
//
// It uses *AnthropicClient directly (not LLMClient interface) in order to access
// ChatLong() for plan generation, following the same pattern as Recommender.
type ConsultingEngine struct {
	llm            *AnthropicClient
	contextService *ContextService
	dispatcher     *Dispatcher
	queries        *sqlc.Queries
	memStore       *memory.MemoryStore
}

// NewConsultingEngine creates a new ConsultingEngine.
func NewConsultingEngine(
	llm *AnthropicClient,
	cs *ContextService,
	dispatcher *Dispatcher,
	queries *sqlc.Queries,
	memStore *memory.MemoryStore,
) *ConsultingEngine {
	return &ConsultingEngine{
		llm:            llm,
		contextService: cs,
		dispatcher:     dispatcher,
		queries:        queries,
		memStore:       memStore,
	}
}

// ---------------------------------------------------------------------------
// Internal types for LLM response parsing
// ---------------------------------------------------------------------------

type classifyEngagementResponse struct {
	Tier          string `json:"tier"`
	Category      string `json:"category"`
	Title         string `json:"title"`
	Reasoning     string `json:"reasoning"`
	FirstQuestion string `json:"first_question"`
}

type diagnosisQuestionResponse struct {
	Question  string `json:"question"`
	Reasoning string `json:"reasoning"`
	Sufficient bool  `json:"sufficient"`
}

type rootCause struct {
	Cause      string  `json:"cause"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

type analysisResponse struct {
	RootCauses        []rootCause `json:"root_causes"`
	FrameworksApplied []string    `json:"frameworks_applied"`
	KeyInsights       []string    `json:"key_insights"`
	RiskFactors       []string    `json:"risk_factors"`
}

type planAction struct {
	ActionType  string         `json:"action_type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Params      map[string]any `json:"params"`
	OwnerName   string         `json:"owner_name"`
	Priority    string         `json:"priority"`
	Reason      string         `json:"reason"`
}

type planGenerationResponse struct {
	Summary          string       `json:"summary"`
	ExpectedOutcomes []string     `json:"expected_outcomes"`
	Timeline         string       `json:"timeline"`
	Actions          []planAction `json:"actions"`
}

type closeSummaryResponse struct {
	Summary            string   `json:"summary"`
	Lessons            []string `json:"lessons"`
	EffectivenessScore int      `json:"effectiveness_score"`
	WhatWorked         []string `json:"what_worked"`
	WhatDidnt          []string `json:"what_didnt"`
}

// ---------------------------------------------------------------------------
// Public methods
// ---------------------------------------------------------------------------

// StartEngagement classifies the problem, creates an engagement record, and
// returns the first diagnostic question.
func (ce *ConsultingEngine) StartEngagement(
	ctx context.Context,
	tenantID pgtype.UUID,
	problem, mentorID, cultureCode string,
) (*sqlc.Engagement, string, error) {
	// Gather company context for classification
	contextData, err := ce.contextService.FormatContextForPrompt(ctx, tenantID)
	if err != nil {
		slog.Warn("consulting: failed to get company context, proceeding without it",
			"error", err)
		contextData = "{}"
	}

	// Classify engagement via LLM
	prompt := fmt.Sprintf(classifyEngagementPrompt, problem, contextData)
	raw, err := ce.llm.Chat(ctx, "You are an expert management consultant.", prompt)
	if err != nil {
		return nil, "", fmt.Errorf("consulting: classify LLM: %w", err)
	}

	raw = stripMarkdownJSON(raw)
	var classified classifyEngagementResponse
	if err := json.Unmarshal([]byte(raw), &classified); err != nil {
		slog.Warn("consulting: failed to parse classify response",
			"response", raw, "error", err)
		// Fallback defaults
		classified = classifyEngagementResponse{
			Tier:          "standard",
			Category:      "performance",
			Title:         "Management Consulting Engagement",
			FirstQuestion: "Can you describe the main symptoms you're observing and when they first appeared?",
		}
	}

	// Store initial questions array as JSON
	firstQs := []string{classified.FirstQuestion}
	questionsJSON, _ := json.Marshal(firstQs)
	answersJSON, _ := json.Marshal([]string{})
	diagData, _ := json.Marshal(map[string]string{
		"context_snapshot": contextData,
	})

	// Create engagement record
	eng, err := ce.queries.CreateEngagement(ctx, sqlc.CreateEngagementParams{
		TenantID:         tenantID,
		Title:            classified.Title,
		ProblemStatement: problem,
		Tier:             classified.Tier,
		Category:         pgtype.Text{String: classified.Category, Valid: true},
		Phase:            "diagnosis",
		DiagnosisData:    diagData,
		MentorID:         pgtype.Text{String: mentorID, Valid: mentorID != ""},
		CultureCode:      pgtype.Text{String: cultureCode, Valid: cultureCode != ""},
		NextCheckAt:      pgtype.Timestamptz{},
	})
	if err != nil {
		return nil, "", fmt.Errorf("consulting: create engagement: %w", err)
	}

	// Persist initial questions
	if err := ce.queries.UpdateEngagementDiagnosis(ctx, sqlc.UpdateEngagementDiagnosisParams{
		ID:                 eng.ID,
		DiagnosisQuestions: questionsJSON,
		DiagnosisAnswers:   answersJSON,
	}); err != nil {
		slog.Warn("consulting: failed to persist initial questions", "error", err)
	}

	slog.Info("consulting: engagement started",
		"id", uuidToString(eng.ID),
		"tier", classified.Tier,
		"category", classified.Category,
	)

	return &eng, classified.FirstQuestion, nil
}

// AnswerQuestion appends the user's answer to the diagnosis, then either asks
// the next question or (when sufficient info has been gathered) runs analysis
// and plan generation.
//
// Returns: nextQuestion (empty when done), planText (non-empty when plan is
// ready), done (true when plan is generated), err.
func (ce *ConsultingEngine) AnswerQuestion(
	ctx context.Context,
	engagementID pgtype.UUID,
	answer string,
) (nextQuestion string, planText string, done bool, err error) {
	eng, err := ce.queries.GetEngagement(ctx, engagementID)
	if err != nil {
		return "", "", false, fmt.Errorf("consulting: get engagement: %w", err)
	}

	// Deserialise existing questions and answers
	var questions []string
	var answers []string
	if eng.DiagnosisQuestions != nil {
		_ = json.Unmarshal(eng.DiagnosisQuestions, &questions)
	}
	if eng.DiagnosisAnswers != nil {
		_ = json.Unmarshal(eng.DiagnosisAnswers, &answers)
	}

	// Append the new answer
	answers = append(answers, answer)

	// Build conversation transcript for the prompt
	var convBuf strings.Builder
	for i, q := range questions {
		convBuf.WriteString(fmt.Sprintf("Q%d: %s\n", i+1, q))
		if i < len(answers) {
			convBuf.WriteString(fmt.Sprintf("A%d: %s\n\n", i+1, answers[i]))
		}
	}
	conversation := convBuf.String()

	// Get company data for context
	contextData, _ := ce.contextService.FormatContextForPrompt(ctx, eng.TenantID)

	category := ""
	if eng.Category.Valid {
		category = eng.Category.String
	}

	// Check if we have enough info or should ask another question
	maxQ := tierMaxQuestions(eng.Tier)
	shouldFinish := len(questions) >= maxQ

	if !shouldFinish {
		// Ask LLM for next question
		prompt := fmt.Sprintf(diagnosisQuestionPrompt,
			eng.ProblemStatement, eng.Tier, category, contextData, conversation)
		raw, llmErr := ce.llm.Chat(ctx, "You are an expert management consultant.", prompt)
		if llmErr != nil {
			slog.Warn("consulting: diagnosis question LLM failed, finishing early",
				"error", llmErr)
			shouldFinish = true
		} else {
			raw = stripMarkdownJSON(raw)
			var dqResp diagnosisQuestionResponse
			if jsonErr := json.Unmarshal([]byte(raw), &dqResp); jsonErr != nil {
				slog.Warn("consulting: parse diagnosis question response failed",
					"response", raw, "error", jsonErr)
				shouldFinish = true
			} else {
				shouldFinish = dqResp.Sufficient || dqResp.Question == ""
				if !shouldFinish {
					// Persist updated questions + answers and return next question
					questions = append(questions, dqResp.Question)
					questionsJSON, _ := json.Marshal(questions)
					answersJSON, _ := json.Marshal(answers)
					if updateErr := ce.queries.UpdateEngagementDiagnosis(ctx, sqlc.UpdateEngagementDiagnosisParams{
						ID:                 engagementID,
						DiagnosisQuestions: questionsJSON,
						DiagnosisAnswers:   answersJSON,
					}); updateErr != nil {
						slog.Warn("consulting: failed to persist diagnosis update", "error", updateErr)
					}
					return dqResp.Question, "", false, nil
				}
			}
		}
	}

	// --- Sufficient information gathered: run analysis then plan generation ---

	// Persist final diagnosis state
	questionsJSON, _ := json.Marshal(questions)
	answersJSON, _ := json.Marshal(answers)
	if updateErr := ce.queries.UpdateEngagementDiagnosis(ctx, sqlc.UpdateEngagementDiagnosisParams{
		ID:                 engagementID,
		DiagnosisQuestions: questionsJSON,
		DiagnosisAnswers:   answersJSON,
	}); updateErr != nil {
		slog.Warn("consulting: failed to persist final diagnosis", "error", updateErr)
	}

	// Retrieve past strategy memories to enrich analysis
	tenantStr := uuidToString(eng.TenantID)
	pastMemories := ""
	if ce.memStore != nil {
		mems, memErr := ce.memStore.List(ctx, tenantStr,
			memory.TypeStrategyResult, memory.TierShortTerm, "", 5, 0)
		if memErr == nil && len(mems) > 0 {
			var sb strings.Builder
			for _, m := range mems {
				sb.WriteString("- ")
				sb.WriteString(m.Content)
				sb.WriteString("\n")
			}
			pastMemories = sb.String()
		}
	}
	if pastMemories == "" {
		pastMemories = "No past strategy memories available."
	}

	// Phase 1: Root cause analysis
	analysisPromptStr := fmt.Sprintf(analysisPrompt,
		eng.ProblemStatement, category, conversation, contextData, pastMemories)
	analysisRaw, err := ce.llm.Chat(ctx, "You are a senior McKinsey partner.", analysisPromptStr)
	if err != nil {
		return "", "", false, fmt.Errorf("consulting: analysis LLM: %w", err)
	}
	analysisRaw = stripMarkdownJSON(analysisRaw)

	var analysis analysisResponse
	if jsonErr := json.Unmarshal([]byte(analysisRaw), &analysis); jsonErr != nil {
		slog.Warn("consulting: parse analysis failed", "response", analysisRaw, "error", jsonErr)
		analysis = analysisResponse{
			RootCauses:  []rootCause{{Cause: "Could not parse analysis", Confidence: 0.5}},
			KeyInsights: []string{"Analysis generation encountered an error"},
		}
	}

	analysisJSON, _ := json.Marshal(analysis)
	if updateErr := ce.queries.UpdateEngagementAnalysis(ctx, sqlc.UpdateEngagementAnalysisParams{
		ID:       engagementID,
		Analysis: analysisJSON,
	}); updateErr != nil {
		slog.Warn("consulting: failed to persist analysis", "error", updateErr)
	}

	// Phase 2: Plan generation — use ChatLong for 4096 output tokens
	teamList := ce.buildTeamList(ctx, eng.TenantID)
	planPromptStr := fmt.Sprintf(planGenerationPrompt,
		eng.ProblemStatement, analysisRaw, teamList)
	planRaw, err := ce.llm.ChatLong(ctx, "You are a McKinsey engagement manager.", planPromptStr)
	if err != nil {
		return "", "", false, fmt.Errorf("consulting: plan LLM: %w", err)
	}
	planRaw = stripMarkdownJSON(planRaw)

	var plan planGenerationResponse
	if jsonErr := json.Unmarshal([]byte(planRaw), &plan); jsonErr != nil {
		slog.Warn("consulting: parse plan failed", "response", planRaw, "error", jsonErr)
		plan = planGenerationResponse{
			Summary:  "Plan generation encountered a parsing error. Raw response: " + planRaw,
			Timeline: "unknown",
		}
	}

	planJSON, _ := json.Marshal(plan)
	if updateErr := ce.queries.UpdateEngagementPlan(ctx, sqlc.UpdateEngagementPlanParams{
		ID:   engagementID,
		Plan: planJSON,
	}); updateErr != nil {
		slog.Warn("consulting: failed to persist plan", "error", updateErr)
	}

	// Create engagement actions from plan
	for _, a := range plan.Actions {
		paramsJSON, _ := json.Marshal(a.Params)
		_, actionErr := ce.queries.CreateEngagementAction(ctx, sqlc.CreateEngagementActionParams{
			EngagementID: engagementID,
			ActionType:   a.ActionType,
			Title:        a.Title,
			Description:  pgtype.Text{String: a.Description, Valid: a.Description != ""},
			Params:       paramsJSON,
			OwnerName:    pgtype.Text{String: a.OwnerName, Valid: a.OwnerName != ""},
			Priority:     pgtype.Text{String: a.Priority, Valid: a.Priority != ""},
			DueAt:        pgtype.Timestamptz{},
		})
		if actionErr != nil {
			slog.Warn("consulting: failed to create action", "title", a.Title, "error", actionErr)
		}
	}

	// Advance to plan phase
	if phaseErr := ce.queries.UpdateEngagementPhase(ctx, sqlc.UpdateEngagementPhaseParams{
		ID:    engagementID,
		Phase: "plan",
	}); phaseErr != nil {
		slog.Warn("consulting: failed to update phase to plan", "error", phaseErr)
	}

	slog.Info("consulting: plan generated",
		"engagement", uuidToString(engagementID),
		"actions", len(plan.Actions),
	)

	// Format plan as human-readable text
	planText = ce.formatPlanText(plan)
	return "", planText, true, nil
}

// ReviewAction approves or rejects a single engagement action.
func (ce *ConsultingEngine) ReviewAction(ctx context.Context, actionID pgtype.UUID, approved bool) error {
	if approved {
		if err := ce.queries.ApproveEngagementAction(ctx, actionID); err != nil {
			return fmt.Errorf("consulting: approve action: %w", err)
		}
		slog.Info("consulting: action approved", "action_id", uuidToString(actionID))
	} else {
		if err := ce.queries.RejectEngagementAction(ctx, actionID); err != nil {
			return fmt.Errorf("consulting: reject action: %w", err)
		}
		slog.Info("consulting: action rejected", "action_id", uuidToString(actionID))
	}
	return nil
}

// ExecuteApproved runs all approved actions for the engagement via the
// Dispatcher, links back any created task/meeting IDs, and advances the
// engagement to the tracking phase.
func (ce *ConsultingEngine) ExecuteApproved(ctx context.Context, engagementID pgtype.UUID) ([]ActionResult, error) {
	eng, err := ce.queries.GetEngagement(ctx, engagementID)
	if err != nil {
		return nil, fmt.Errorf("consulting: get engagement for execute: %w", err)
	}

	approved, err := ce.queries.ListApprovedEngagementActions(ctx, engagementID)
	if err != nil {
		return nil, fmt.Errorf("consulting: list approved actions: %w", err)
	}

	results := make([]ActionResult, 0, len(approved))

	for _, dbAction := range approved {
		// Build SuggestedAction from stored params
		var params map[string]any
		if dbAction.Params != nil {
			_ = json.Unmarshal(dbAction.Params, &params)
		}
		if params == nil {
			params = map[string]any{}
		}

		sa := SuggestedAction{
			Type:   dbAction.ActionType,
			Params: params,
			Label:  dbAction.Title,
		}

		result := ce.dispatcher.Execute(ctx, eng.TenantID, sa)
		results = append(results, result)

		// Encode result as JSON for persistence
		resultJSON, _ := json.Marshal(result)

		if result.Success {
			if markErr := ce.queries.MarkEngagementActionDone(ctx, sqlc.MarkEngagementActionDoneParams{
				ID:     dbAction.ID,
				Result: resultJSON,
			}); markErr != nil {
				slog.Warn("consulting: mark action done failed",
					"action_id", uuidToString(dbAction.ID), "error", markErr)
			}

			// Link back task or meeting IDs if the dispatcher provided them
			// (Dispatcher returns IDs in the Link field as a UUID string when available)
			if result.Link != "" {
				switch dbAction.ActionType {
				case "create_task":
					taskID, parseErr := consultParseUUID(result.Link)
					if parseErr == nil {
						_ = ce.queries.LinkEngagementActionTask(ctx, sqlc.LinkEngagementActionTaskParams{
							ID:           dbAction.ID,
							LinkedTaskID: taskID,
						})
					}
				case "schedule_meeting":
					meetingID, parseErr := consultParseUUID(result.Link)
					if parseErr == nil {
						_ = ce.queries.LinkEngagementActionMeeting(ctx, sqlc.LinkEngagementActionMeetingParams{
							ID:              dbAction.ID,
							LinkedMeetingID: meetingID,
						})
					}
				}
			}
		} else {
			if markErr := ce.queries.MarkEngagementActionFailed(ctx, sqlc.MarkEngagementActionFailedParams{
				ID:     dbAction.ID,
				Result: resultJSON,
			}); markErr != nil {
				slog.Warn("consulting: mark action failed",
					"action_id", uuidToString(dbAction.ID), "error", markErr)
			}
		}
	}

	// Advance to tracking phase
	tomorrow := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 1), Valid: true}
	var zero pgtype.Numeric
	_ = zero.Scan("0")

	if phaseErr := ce.queries.UpdateEngagementPhase(ctx, sqlc.UpdateEngagementPhaseParams{
		ID:    engagementID,
		Phase: "tracking",
	}); phaseErr != nil {
		slog.Warn("consulting: failed to update phase to tracking", "error", phaseErr)
	}

	if progressErr := ce.queries.UpdateEngagementProgress(ctx, sqlc.UpdateEngagementProgressParams{
		ID:          engagementID,
		ProgressPct: zero,
		NextCheckAt: tomorrow,
	}); progressErr != nil {
		slog.Warn("consulting: failed to set next_check_at", "error", progressErr)
	}

	slog.Info("consulting: executed approved actions",
		"engagement", uuidToString(engagementID),
		"count", len(results),
	)

	return results, nil
}

// CheckProgress loads the current action statuses, calculates completion
// percentage, generates a progress report via LLM, and schedules the next
// check-in.
func (ce *ConsultingEngine) CheckProgress(ctx context.Context, engagementID pgtype.UUID) (string, error) {
	eng, err := ce.queries.GetEngagement(ctx, engagementID)
	if err != nil {
		return "", fmt.Errorf("consulting: get engagement for progress: %w", err)
	}

	actions, err := ce.queries.ListEngagementActionsWithLinks(ctx, engagementID)
	if err != nil {
		return "", fmt.Errorf("consulting: list actions with links: %w", err)
	}

	// Count statuses
	total := len(actions)
	done := 0
	var statusLines []string
	for _, a := range actions {
		if a.Status == "done" {
			done++
		}
		priority := ""
		if a.Priority.Valid {
			priority = a.Priority.String
		}
		statusLines = append(statusLines, fmt.Sprintf("- [%s] %s (%s)", a.Status, a.Title, priority))
	}

	var progressPct float64
	if total > 0 {
		progressPct = float64(done) / float64(total) * 100
	}

	// Extract plan summary
	planSummary := ""
	if eng.Plan != nil {
		var p planGenerationResponse
		if jsonErr := json.Unmarshal(eng.Plan, &p); jsonErr == nil {
			planSummary = p.Summary
		}
	}

	statusSummary := strings.Join(statusLines, "\n")
	if statusSummary == "" {
		statusSummary = "No actions recorded yet."
	}

	// Generate progress report
	prompt := fmt.Sprintf(progressReportPrompt,
		eng.ProblemStatement, planSummary, statusSummary)
	report, err := ce.llm.Chat(ctx, "You are a management consultant providing a status update.", prompt)
	if err != nil {
		slog.Warn("consulting: progress report LLM failed", "error", err)
		report = fmt.Sprintf("Progress: %d/%d actions complete (%.0f%%).\n%s",
			done, total, progressPct, statusSummary)
	}

	// Update progress and schedule next check-in (tomorrow by default)
	var pgPct pgtype.Numeric
	_ = pgPct.Scan(fmt.Sprintf("%.2f", progressPct))

	nextCheck := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 1), Valid: true}
	if updateErr := ce.queries.UpdateEngagementProgress(ctx, sqlc.UpdateEngagementProgressParams{
		ID:          engagementID,
		ProgressPct: pgPct,
		NextCheckAt: nextCheck,
	}); updateErr != nil {
		slog.Warn("consulting: failed to update progress", "error", updateErr)
	}

	slog.Info("consulting: progress checked",
		"engagement", uuidToString(engagementID),
		"done", done,
		"total", total,
		"pct", progressPct,
	)

	return report, nil
}

// CloseEngagement finalises the engagement, stores lessons as strategy_result
// memories, and returns the close summary.
func (ce *ConsultingEngine) CloseEngagement(ctx context.Context, engagementID pgtype.UUID) (string, error) {
	eng, err := ce.queries.GetEngagement(ctx, engagementID)
	if err != nil {
		return "", fmt.Errorf("consulting: get engagement for close: %w", err)
	}

	// Build outcomes summary from action statuses
	actions, _ := ce.queries.ListEngagementActions(ctx, engagementID)
	done, total := 0, len(actions)
	for _, a := range actions {
		if a.Status == "done" {
			done++
		}
	}
	outcomes := fmt.Sprintf("%d of %d actions completed (%.0f%%).",
		done, total, safePercent(done, total))

	planSummary := ""
	if eng.Plan != nil {
		var p planGenerationResponse
		if jsonErr := json.Unmarshal(eng.Plan, &p); jsonErr == nil {
			planSummary = p.Summary
		}
	}

	duration := "unknown"
	if eng.CreatedAt.Valid {
		d := time.Since(eng.CreatedAt.Time)
		duration = fmt.Sprintf("%.1f days", d.Hours()/24)
	}

	// Generate close summary
	prompt := fmt.Sprintf(closeSummaryPrompt,
		eng.ProblemStatement, planSummary, outcomes, duration)
	raw, err := ce.llm.Chat(ctx, "You are a McKinsey partner conducting a retrospective.", prompt)
	if err != nil {
		slog.Warn("consulting: close summary LLM failed", "error", err)
		raw = fmt.Sprintf(`{"summary":"%s","lessons":[],"effectiveness_score":5,"what_worked":[],"what_didnt":[]}`, outcomes)
	}
	raw = stripMarkdownJSON(raw)

	var closeResp closeSummaryResponse
	if jsonErr := json.Unmarshal([]byte(raw), &closeResp); jsonErr != nil {
		slog.Warn("consulting: parse close summary failed", "response", raw, "error", jsonErr)
		closeResp = closeSummaryResponse{
			Summary:            outcomes,
			EffectivenessScore: 5,
		}
	}

	// Persist lessons as strategy_result memories
	if ce.memStore != nil {
		tenantStr := uuidToString(eng.TenantID)
		for _, lesson := range closeResp.Lessons {
			if lesson == "" {
				continue
			}
			_, memErr := ce.memStore.Create(ctx, memory.Memory{
				TenantID:   tenantStr,
				MemoryType: memory.TypeStrategyResult,
				MemoryTier: memory.TierShortTerm,
				SourceType: "consulting_engagement",
				Content:    lesson,
				Summary:    fmt.Sprintf("Lesson from engagement: %s", eng.Title),
				Importance: float64(closeResp.EffectivenessScore) / 10.0,
				Metadata: map[string]any{
					"engagement_id": uuidToString(engagementID),
					"problem":       eng.ProblemStatement,
				},
			})
			if memErr != nil {
				slog.Warn("consulting: failed to store lesson memory", "error", memErr)
			}
		}
	}

	// Mark engagement as closed
	if closeErr := ce.queries.CloseEngagement(ctx, engagementID); closeErr != nil {
		return "", fmt.Errorf("consulting: close engagement DB: %w", closeErr)
	}

	slog.Info("consulting: engagement closed",
		"engagement", uuidToString(engagementID),
		"score", closeResp.EffectivenessScore,
		"lessons", len(closeResp.Lessons),
	)

	return closeResp.Summary, nil
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// tierMaxQuestions returns the maximum number of diagnostic questions allowed
// for a given engagement tier.
func tierMaxQuestions(tier string) int {
	switch tier {
	case "quick":
		return 2
	case "deep":
		return 10
	default: // standard
		return 5
	}
}

// consultParseUUID parses a UUID string into pgtype.UUID.
// Named with "consult" prefix to avoid collision with other parseUUID helpers.
func consultParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %w", err)
	}
	return u, nil
}

// stripMarkdownJSON removes surrounding markdown code fences from an LLM
// response so that the content can be passed directly to json.Unmarshal.
// Follows the same pattern as execution_planner.go lines 121-127.
func stripMarkdownJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return strings.TrimSpace(s)
}

// buildTeamList fetches active employees and returns a formatted string listing
// names and roles for the plan generation prompt.
func (ce *ConsultingEngine) buildTeamList(ctx context.Context, tenantID pgtype.UUID) string {
	emps, err := ce.queries.ListActiveEmployees(ctx, tenantID)
	if err != nil || len(emps) == 0 {
		return "No team data available."
	}
	var sb strings.Builder
	for _, e := range emps {
		role := e.Role
		if role == "" {
			role = "team member"
		}
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", e.Name, role))
	}
	return sb.String()
}

// formatPlanText renders a planGenerationResponse as human-readable text for
// display in chat or bot messages.
func (ce *ConsultingEngine) formatPlanText(plan planGenerationResponse) string {
	var sb strings.Builder
	sb.WriteString("CONSULTING PLAN\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")
	sb.WriteString(plan.Summary)
	sb.WriteString("\n\n")

	if len(plan.ExpectedOutcomes) > 0 {
		sb.WriteString("Expected Outcomes:\n")
		for _, o := range plan.ExpectedOutcomes {
			sb.WriteString("  - " + o + "\n")
		}
		sb.WriteString("\n")
	}

	if plan.Timeline != "" {
		sb.WriteString(fmt.Sprintf("Timeline: %s\n\n", plan.Timeline))
	}

	if len(plan.Actions) > 0 {
		sb.WriteString(fmt.Sprintf("Actions (%d):\n", len(plan.Actions)))
		for i, a := range plan.Actions {
			priority := a.Priority
			if priority == "" {
				priority = "medium"
			}
			owner := a.OwnerName
			if owner == "" {
				owner = "unassigned"
			}
			sb.WriteString(fmt.Sprintf("\n%d. [%s] %s\n", i+1, strings.ToUpper(priority), a.Title))
			sb.WriteString(fmt.Sprintf("   Owner: %s\n", owner))
			if a.Description != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", a.Description))
			}
		}
	}

	return sb.String()
}

// safePercent returns done/total*100, handling the zero-total edge case.
func safePercent(done, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(done) / float64(total) * 100
}
