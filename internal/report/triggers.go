package report

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
)

// TriggerDB defines the database operations needed by the trigger checker.
type TriggerDB interface {
	ListActiveEmployees(ctx context.Context, tenantID string) ([]EmployeeInfo, error)
	GetMissedDaysLast7(ctx context.Context, employeeID string) (int, error)
	GetSubmittedDaysLast7(ctx context.Context, employeeID string) (int, error)
}

// TriggerResult holds the result of a trigger check for logging.
type TriggerResult struct {
	EmployeeID   string
	EmployeeName string
	Event        string
	Action       string
	Message      string
}

// TriggerChecker detects events and executes trigger rules from mentor config.
type TriggerChecker struct {
	db          TriggerDB
	sender      channel.Sender
	factory     *brain.EngineFactory
	recommender *brain.Recommender
}

// NewTriggerChecker creates a new trigger checker.
func NewTriggerChecker(db TriggerDB, sender channel.Sender, factory *brain.EngineFactory) *TriggerChecker {
	return &TriggerChecker{db: db, sender: sender, factory: factory}
}

// SetRecommender sets the recommender for realtime recommendation generation on trigger matches.
func (t *TriggerChecker) SetRecommender(r *brain.Recommender) {
	t.recommender = r
}

// CheckAll runs all trigger rules for the tenant's employees.
// Returns the list of triggered actions for logging.
func (t *TriggerChecker) CheckAll(ctx context.Context, tenantID, mentorID string, bossInfo EmployeeInfo) ([]TriggerResult, error) {
	engine, err := t.factory.ForTenant(mentorID, "default")
	if err != nil {
		return nil, fmt.Errorf("load engine for triggers: %w", err)
	}

	triggers := engine.GetTriggerRules()
	if len(triggers) == 0 {
		return nil, nil
	}

	// Resolve boss channel once for all trigger actions
	bossChType, bossChID := resolveEmployeeChannel(bossInfo)
	if bossChType == "" {
		return nil, fmt.Errorf("boss has no channel configured")
	}

	employees, err := t.db.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	var results []TriggerResult

	for _, emp := range employees {
		for _, trigger := range triggers {
			matched, err := t.eventMatches(ctx, emp, trigger.Event)
			if err != nil {
				slog.Error("trigger event check", "employee", emp.Name, "event", trigger.Event, "error", err)
				continue
			}
			if !matched {
				continue
			}

			msg := strings.ReplaceAll(trigger.Message, "{name}", emp.Name)
			result := TriggerResult{
				EmployeeID:   emp.ID,
				EmployeeName: emp.Name,
				Event:        trigger.Event,
				Action:       trigger.Action,
				Message:      msg,
			}

			if err := t.executeAction(ctx, trigger.Action, msg, emp, bossChType, bossChID); err != nil {
				slog.Error("trigger action", "employee", emp.Name, "action", trigger.Action, "error", err)
			} else {
				slog.Info("trigger fired", "employee", emp.Name, "event", trigger.Event, "action", trigger.Action)
			}

			results = append(results, result)

			// Generate AI recommendation for the triggered event
			if t.recommender != nil {
				var tenantUUID pgtype.UUID
				if err := tenantUUID.Scan(tenantID); err == nil {
					var empUUID pgtype.UUID
					_ = empUUID.Scan(emp.ID)
					data := map[string]any{"trigger_event": trigger.Event}
					if err := t.recommender.RealtimeEvaluate(ctx, tenantUUID, trigger.Event, emp.Name, empUUID, data); err != nil {
						slog.Error("trigger recommendation", "employee", emp.Name, "event", trigger.Event, "error", err)
					}
				}
			}
		}
	}

	return results, nil
}

// eventMatches checks if an event condition is met for the given employee.
func (t *TriggerChecker) eventMatches(ctx context.Context, emp EmployeeInfo, event string) (bool, error) {
	switch event {
	case "consecutive_miss_3days", "output_decline_3days", "consecutive_low_output":
		// Employee missed 3+ days in the last 7 days
		missed, err := t.db.GetMissedDaysLast7(ctx, emp.ID)
		if err != nil {
			return false, err
		}
		return missed >= 3, nil

	case "exceptional_performance", "exceptional_transparency":
		// Employee submitted 6+ days out of last 7
		submitted, err := t.db.GetSubmittedDaysLast7(ctx, emp.ID)
		if err != nil {
			return false, err
		}
		return submitted >= 6, nil

	case "sentiment_drop", "blocker_unresolved", "repeat_mistake":
		// These require AI analysis -- not yet implemented
		return false, nil

	default:
		slog.Debug("unknown trigger event", "event", event)
		return false, nil
	}
}

// executeAction performs the triggered action.
// bossChType/bossChID are pre-resolved boss channel coordinates.
func (t *TriggerChecker) executeAction(ctx context.Context, action, msg string, emp EmployeeInfo, bossChType channel.Type, bossChID string) error {
	switch action {
	case "manager_notify", "manager_private_chat", "suggest_one_on_one", "performance_warning":
		// Notify the boss
		return t.sender.Send(ctx, bossChType, bossChID, fmt.Sprintf("⚠️ Trigger Alert\n\n%s", msg))

	case "private_checkin", "private_message":
		// Send to the employee directly
		chType, chID := resolveEmployeeChannel(emp)
		if chType == "" {
			return fmt.Errorf("employee %s has no channel", emp.Name)
		}
		return t.sender.Send(ctx, chType, chID, msg)

	case "public_recognition":
		// Notify the boss about the recognition (they can share)
		return t.sender.Send(ctx, bossChType, bossChID, fmt.Sprintf("🌟 Recognition\n\n%s", msg))

	case "create_principle":
		// Notify boss to create a principle
		return t.sender.Send(ctx, bossChType, bossChID, fmt.Sprintf("📋 Principle Suggestion\n\n%s", msg))

	default:
		slog.Debug("unknown trigger action", "action", action)
		return nil
	}
}
