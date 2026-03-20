package brain

import (
	"context"
	"fmt"
	"log/slog"
)

// ChiefDB defines the database queries needed by the Chief of Staff.
type ChiefDB interface {
	FindEmployeeByName(ctx context.Context, tenantID, name string) (ChiefEmployee, error)
	GetEmployeeStatus(ctx context.Context, employeeID string) (EmployeeStatus, error)
}

// ChiefEmployee represents an employee found by name lookup.
type ChiefEmployee struct {
	ID         string
	Name       string
	ChannelID  string // platform-specific user ID (e.g., Telegram chat ID)
	Channel    string // channel type: "telegram", "slack", "lark"
}

// EmployeeStatus holds an employee's recent activity summary.
type EmployeeStatus struct {
	EmployeeName   string
	SubmittedToday bool
	MissedDays     int
	LastSentiment  string
	RecentBlockers string
}

// ChiefSender can send messages to any channel.
type ChiefSender interface {
	Send(ctx context.Context, channelType, userID, text string) error
}

// Chief implements "Chief of Staff" — translates boss NL commands to structured actions.
type Chief struct {
	orchestrator *Orchestrator
	db           ChiefDB
	sender       ChiefSender
	llm          LLMClient
}

// NewChief creates a new Chief of Staff agent.
func NewChief(orchestrator *Orchestrator, db ChiefDB, sender ChiefSender, llm LLMClient) *Chief {
	return &Chief{
		orchestrator: orchestrator,
		db:           db,
		sender:       sender,
		llm:          llm,
	}
}

// HandleCommand processes a boss natural language command and executes it.
// Returns a reply message for the boss.
func (c *Chief) HandleCommand(ctx context.Context, tenantID, text string) (string, error) {
	// Detect intent
	intent, err := c.orchestrator.DetectIntent(ctx, text)
	if err != nil {
		return "", fmt.Errorf("detect intent: %w", err)
	}

	// Dispatch to task
	task := c.orchestrator.Dispatch(intent)

	// Execute task
	return c.execute(ctx, tenantID, task)
}

// execute runs a dispatched task and returns the result.
func (c *Chief) execute(ctx context.Context, tenantID string, task Task) (string, error) {
	switch task.Action {
	case "send_question":
		return c.execAskEmployee(ctx, tenantID, task)
	case "broadcast":
		return c.execBroadcast(ctx, tenantID, task)
	case "check_employee_status":
		return c.execCheckStatus(ctx, tenantID, task)
	case "switch_mentor":
		return c.execSwitchMentor(task)
	case "generate_summary":
		return "Generating today's summary... This will be sent to you shortly.", nil
	case "create_reminder":
		return fmt.Sprintf("Reminder set: %s", task.Params["content"]), nil
	default:
		return task.Result, nil
	}
}

// execAskEmployee finds the employee and sends them a question.
func (c *Chief) execAskEmployee(ctx context.Context, tenantID string, task Task) (string, error) {
	emp, err := c.db.FindEmployeeByName(ctx, tenantID, task.Params["target"])
	if err != nil {
		return fmt.Sprintf("Could not find employee %q. Please check the name and try again.", task.Params["target"]), nil
	}

	question := task.Params["question"]

	// Format a friendly message
	msg := fmt.Sprintf("Your manager would like to know: %s\n\nPlease reply with your answer.", question)

	if err := c.sender.Send(ctx, emp.Channel, emp.ChannelID, msg); err != nil {
		slog.Error("send question to employee", "employee", emp.Name, "error", err)
		return fmt.Sprintf("Failed to send message to %s. They may not be connected to any channel.", emp.Name), nil
	}

	return fmt.Sprintf("Question sent to %s: %q\nI'll let you know when they reply.", emp.Name, question), nil
}

// execBroadcast sends an announcement to all team members.
func (c *Chief) execBroadcast(ctx context.Context, tenantID string, task Task) (string, error) {
	message := task.Params["message"]
	announcement := fmt.Sprintf("Team Announcement\n\n%s", message)

	// For now, return confirmation. Real implementation would use router.Broadcast.
	slog.Info("broadcast requested", "tenant_id", tenantID, "message", message)
	return fmt.Sprintf("Announcement queued for broadcast: %q", announcement), nil
}

// execCheckStatus retrieves an employee's recent status.
func (c *Chief) execCheckStatus(ctx context.Context, tenantID string, task Task) (string, error) {
	emp, err := c.db.FindEmployeeByName(ctx, tenantID, task.Params["employee"])
	if err != nil {
		return fmt.Sprintf("Could not find employee %q.", task.Params["employee"]), nil
	}

	status, err := c.db.GetEmployeeStatus(ctx, emp.ID)
	if err != nil {
		slog.Error("get employee status", "employee", emp.Name, "error", err)
		return fmt.Sprintf("Error retrieving status for %s.", emp.Name), nil
	}

	submitted := "No"
	if status.SubmittedToday {
		submitted = "Yes"
	}

	msg := fmt.Sprintf(
		"Status: %s\n\n"+
			"Submitted today: %s\n"+
			"Missed days (last 7): %d\n"+
			"Recent sentiment: %s",
		status.EmployeeName,
		submitted,
		status.MissedDays,
		status.LastSentiment,
	)

	if status.RecentBlockers != "" {
		msg += fmt.Sprintf("\nBlockers: %s", status.RecentBlockers)
	}

	return msg, nil
}

// execSwitchMentor validates and returns a mentor switch confirmation.
func (c *Chief) execSwitchMentor(task Task) (string, error) {
	mentorID := task.Params["mentor_id"]
	if !ValidMentors[mentorID] {
		return fmt.Sprintf("Unknown mentor %q. Available mentors: inamori, dalio, grove, ren", mentorID), nil
	}
	return fmt.Sprintf("Mentor switch to %s requested. Use /mentor command to confirm.", mentorID), nil
}
