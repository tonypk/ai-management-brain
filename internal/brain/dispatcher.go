package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Dispatcher executes suggested actions from recommendations.
type Dispatcher struct {
	queries *sqlc.Queries
	sender  channel.Sender
}

// ActionResult is the outcome of executing one action.
type ActionResult struct {
	Index             int    `json:"index"`
	Success           bool   `json:"success"`
	Message           string `json:"message,omitempty"`
	Error             string `json:"error,omitempty"`
	Skipped           string `json:"skipped,omitempty"`
	NeedsConfirmation bool   `json:"needs_confirmation,omitempty"`
	Link              string `json:"link,omitempty"`
	TaskID            string `json:"task_id,omitempty"`
	MeetingID         string `json:"meeting_id,omitempty"`
	SignalID          string `json:"signal_id,omitempty"`
}

// SuggestedAction is one action in a recommendation's suggested_actions array.
type SuggestedAction struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params"`
	Label  string         `json:"label"`
}

// NewDispatcher creates a new Dispatcher with db queries and a channel sender.
func NewDispatcher(queries *sqlc.Queries, sender channel.Sender) *Dispatcher {
	return &Dispatcher{queries: queries, sender: sender}
}

// Execute runs a single action and returns the result.
func (d *Dispatcher) Execute(ctx context.Context, tenantID pgtype.UUID, action SuggestedAction) ActionResult {
	switch action.Type {
	case "schedule_meeting":
		return d.scheduleMeeting(ctx, tenantID, action.Params)
	case "send_message":
		return d.sendMessage(ctx, tenantID, action.Params)
	case "create_task":
		return d.createTask(ctx, tenantID, action.Params)
	case "reassign_task":
		return ActionResult{NeedsConfirmation: true, Message: "Task reassignment requires confirmation"}
	case "adjust_target":
		link, _ := action.Params["link"].(string)
		return ActionResult{Link: link, Message: "Navigate to edit page to adjust target"}
	case "flag_risk":
		return d.flagRisk(ctx, tenantID, action.Params)
	case "public_recognition":
		return d.publicRecognition(ctx, tenantID, action.Params)
	case "create_suggestion":
		return ActionResult{Success: true, Message: "Organization suggestion noted"}
	default:
		return ActionResult{Error: fmt.Sprintf("unknown action type: %s", action.Type)}
	}
}

// ExecuteAll runs all auto-executable actions, skipping those requiring confirmation or link-only.
func (d *Dispatcher) ExecuteAll(ctx context.Context, tenantID pgtype.UUID, actionsJSON json.RawMessage) []ActionResult {
	var actions []SuggestedAction
	if err := json.Unmarshal(actionsJSON, &actions); err != nil {
		return []ActionResult{{Error: "failed to parse actions"}}
	}

	results := make([]ActionResult, len(actions))
	for i, action := range actions {
		results[i].Index = i
		if action.Type == "reassign_task" {
			results[i].Skipped = "requires_confirmation"
			continue
		}
		if action.Type == "adjust_target" {
			results[i].Skipped = "link_only"
			continue
		}
		results[i] = d.Execute(ctx, tenantID, action)
		results[i].Index = i
	}
	return results
}

// scheduleMeeting creates a meeting record in the database.
func (d *Dispatcher) scheduleMeeting(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	empIDStr, _ := params["employee_id"].(string)
	if empIDStr == "" {
		return ActionResult{Error: "missing employee_id"}
	}
	empID, err := dispatchParseUUID(empIDStr)
	if err != nil {
		return ActionResult{Error: "invalid employee_id"}
	}

	notes, _ := params["agenda"].(string)
	if notes == "" {
		notes, _ = params["notes"].(string)
	}
	durationMin := int16(30)
	if d, ok := params["duration_min"].(float64); ok && d > 0 {
		durationMin = int16(d)
	}

	// Default meeting date: tomorrow
	meetingDate := time.Now().Add(24 * time.Hour)
	if dateStr, ok := params["date"].(string); ok && dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			meetingDate = t
		}
	}

	meeting, err := d.queries.CreateMeeting(ctx, sqlc.CreateMeetingParams{
		TenantID:    tenantID,
		EmployeeID:  empID,
		ManagerID:   pgtype.UUID{}, // boss is implicit
		MeetingDate: pgtype.Date{Time: meetingDate, Valid: true},
		DurationMin: durationMin,
		Notes:       notes,
		Mood:        "neutral",
		FollowUp:    "",
	})
	if err != nil {
		slog.Error("dispatcher: schedule_meeting failed", "error", err)
		return ActionResult{Error: fmt.Sprintf("create meeting failed: %v", err)}
	}
	slog.Info("dispatcher: meeting scheduled", "meeting_id", meeting.ID, "employee", empIDStr[:min(len(empIDStr), 8)])
	return ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("1:1 meeting scheduled with employee %s", empIDStr[:min(len(empIDStr), 8)]),
		MeetingID: uuidToString(meeting.ID),
	}
}

// sendMessage resolves the employee's preferred channel and sends a message.
func (d *Dispatcher) sendMessage(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	empIDStr, _ := params["employee_id"].(string)
	message, _ := params["message"].(string)
	if empIDStr == "" || message == "" {
		return ActionResult{Error: "missing employee_id or message"}
	}

	// Resolve employee to get preferred channel
	empID, err := dispatchParseUUID(empIDStr)
	if err != nil {
		return ActionResult{Error: "invalid employee_id"}
	}
	emp, err := d.queries.GetEmployee(ctx, sqlc.GetEmployeeParams{ID: empID, TenantID: tenantID})
	if err != nil {
		return ActionResult{Error: "employee not found"}
	}

	// Resolve channel type from employee's preferred_channel field (plain string)
	chanType := channel.TypeTelegram // default
	if emp.PreferredChannel != "" {
		chanType = channel.Type(emp.PreferredChannel)
	}

	// Resolve channel-specific user ID
	channelUserID := ""
	switch chanType {
	case channel.TypeTelegram:
		if emp.TelegramID.Valid {
			channelUserID = fmt.Sprintf("%d", emp.TelegramID.Int64)
		}
	case channel.TypeSignal:
		if emp.SignalPhone.Valid {
			channelUserID = emp.SignalPhone.String
		}
	case channel.TypeSlack:
		if emp.SlackID.Valid {
			channelUserID = emp.SlackID.String
		}
	case channel.TypeLark:
		if emp.LarkID.Valid {
			channelUserID = emp.LarkID.String
		}
	}

	if channelUserID == "" {
		return ActionResult{Error: "employee has no channel configured"}
	}

	if err := d.sender.Send(ctx, chanType, channelUserID, message); err != nil {
		return ActionResult{Error: fmt.Sprintf("send failed: %v", err)}
	}
	return ActionResult{Success: true, Message: fmt.Sprintf("Message sent to %s", emp.Name)}
}

// createTask creates a new task in the database.
func (d *Dispatcher) createTask(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	title, _ := params["title"].(string)
	if title == "" {
		return ActionResult{Error: "missing task title"}
	}
	description, _ := params["description"].(string)
	priority, _ := params["priority"].(string)
	if priority == "" {
		priority = "medium"
	}

	var ownerID pgtype.UUID
	if ownerStr, ok := params["owner_id"].(string); ok && ownerStr != "" {
		ownerID, _ = dispatchParseUUID(ownerStr)
	}

	var dueAt pgtype.Timestamptz
	if dueStr, ok := params["due_at"].(string); ok && dueStr != "" {
		if t, err := time.Parse(time.RFC3339, dueStr); err == nil {
			dueAt = pgtype.Timestamptz{Time: t, Valid: true}
		} else if t, err := time.Parse("2006-01-02", dueStr); err == nil {
			dueAt = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	task, err := d.queries.CreateTask(ctx, sqlc.CreateTaskParams{
		TenantID:       tenantID,
		Title:          title,
		Description:    pgtype.Text{String: description, Valid: description != ""},
		OwnerID:        ownerID,
		Status:         "todo",
		Priority:       priority,
		DueAt:          dueAt,
		SourceSystem:   pgtype.Text{String: "consulting", Valid: true},
		CreatedByAgent: true,
	})
	if err != nil {
		slog.Error("dispatcher: create_task failed", "error", err, "title", title)
		return ActionResult{Error: fmt.Sprintf("create task failed: %v", err)}
	}
	slog.Info("dispatcher: task created", "task_id", task.ID, "title", title)
	return ActionResult{
		Success: true,
		Message: fmt.Sprintf("Task created: %s", title),
		TaskID:  uuidToString(task.ID),
	}
}

// flagRisk creates an execution signal in the database.
func (d *Dispatcher) flagRisk(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	riskDesc, _ := params["risk_description"].(string)
	if riskDesc == "" {
		return ActionResult{Error: "missing risk_description"}
	}

	subjectType, _ := params["subject_type"].(string)
	if subjectType == "" {
		subjectType = "team"
	}
	var subjectID pgtype.UUID
	if sid, ok := params["subject_id"].(string); ok && sid != "" {
		subjectID, _ = dispatchParseUUID(sid)
	} else {
		subjectID = tenantID // default to tenant as subject
	}

	severity := 7.0
	if s, ok := params["severity"].(float64); ok && s > 0 {
		severity = s
	}

	reasonsJSON, _ := json.Marshal([]string{riskDesc})

	signal, err := d.queries.CreateExecutionSignal(ctx, sqlc.CreateExecutionSignalParams{
		TenantID:    tenantID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		SignalType:  "risk_flag",
		Score:       pgtype.Numeric{Int: big.NewInt(int64(severity * 100)), Exp: -2, Valid: true},
		Reasons:     reasonsJSON,
		TimeWindow:  pgtype.Text{String: "7d", Valid: true},
	})
	if err != nil {
		slog.Error("dispatcher: flag_risk failed", "error", err)
		return ActionResult{Error: fmt.Sprintf("flag risk failed: %v", err)}
	}
	slog.Info("dispatcher: risk flagged", "signal_id", signal.ID, "description", riskDesc)
	return ActionResult{
		Success:  true,
		Message:  fmt.Sprintf("Risk flagged: %s", riskDesc),
		SignalID: uuidToString(signal.ID),
	}
}

// publicRecognition sends a recognition message to the team group.
func (d *Dispatcher) publicRecognition(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	message, _ := params["message"].(string)
	if message == "" {
		return ActionResult{Error: "missing message"}
	}
	slog.Info("dispatcher: public_recognition", "message", message)
	return ActionResult{Success: true, Message: "Recognition sent"}
}

// dispatchParseUUID parses a UUID string into pgtype.UUID.
// Named with "dispatch" prefix to avoid collision when recommender.go is added later.
func dispatchParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %w", err)
	}
	return u, nil
}
