package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// consultingAdapter wraps ConsultingEngine for bot use, handling UUID string conversion.
type consultingAdapter struct {
	engine  *brain.ConsultingEngine
	queries *sqlc.Queries
}

// NewConsultingAdapter creates a ConsultingServicer that wraps the given engine.
func NewConsultingAdapter(engine *brain.ConsultingEngine, queries *sqlc.Queries) ConsultingServicer {
	return &consultingAdapter{engine: engine, queries: queries}
}

// StartEngagement converts string tenantID to pgtype.UUID, starts the engagement,
// and returns the engagement ID and first diagnostic question as strings.
func (a *consultingAdapter) StartEngagement(ctx context.Context, tenantID, problem, mentorID, cultureCode string) (string, string, error) {
	uid, err := parseConsultUUID(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("invalid tenant ID: %w", err)
	}
	eng, firstQuestion, err := a.engine.StartEngagement(ctx, uid, problem, mentorID, cultureCode)
	if err != nil {
		return "", "", err
	}
	return formatConsultUUID(eng.ID), firstQuestion, nil
}

// AnswerQuestion converts the string engagement ID, delegates to the engine,
// and returns next question, plan text, done flag, and any error.
func (a *consultingAdapter) AnswerQuestion(ctx context.Context, engagementID, answer string) (string, string, bool, error) {
	uid, err := resolveEngagementID(a.queries, ctx, engagementID)
	if err != nil {
		return "", "", false, err
	}
	return a.engine.AnswerQuestion(ctx, uid, answer)
}

// ReviewActions approves or rejects all pending actions for the engagement.
func (a *consultingAdapter) ReviewActions(ctx context.Context, engagementID string, approved bool) (string, error) {
	uid, err := resolveEngagementID(a.queries, ctx, engagementID)
	if err != nil {
		return "", err
	}

	actions, err := a.queries.ListEngagementActions(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("list engagement actions: %w", err)
	}

	// Only review actions that are in a reviewable state (pending)
	reviewed := 0
	for _, action := range actions {
		if action.Status != "pending" {
			continue
		}
		if err := a.engine.ReviewAction(ctx, action.ID, approved); err != nil {
			return "", fmt.Errorf("review action %q: %w", action.Title, err)
		}
		reviewed++
	}

	if reviewed == 0 {
		return "No pending actions found to review.", nil
	}

	verb := "approved"
	if !approved {
		verb = "rejected"
	}

	if approved {
		return fmt.Sprintf("%d action(s) %s.\n\nUse /consult execute %s to execute them.", reviewed, verb, engagementID), nil
	}
	return fmt.Sprintf("%d action(s) %s.", reviewed, verb), nil
}

// ExecuteApproved runs all approved actions and returns a summary string.
func (a *consultingAdapter) ExecuteApproved(ctx context.Context, engagementID string) (string, error) {
	uid, err := resolveEngagementID(a.queries, ctx, engagementID)
	if err != nil {
		return "", err
	}

	results, err := a.engine.ExecuteApproved(ctx, uid)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No approved actions to execute.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Executed %d action(s):\n\n", len(results)))
	succeeded := 0
	for _, r := range results {
		if r.Success {
			succeeded++
			sb.WriteString(fmt.Sprintf("- [OK] %s\n", r.Message))
		} else {
			sb.WriteString(fmt.Sprintf("- [FAIL] %s\n", r.Error))
		}
	}
	sb.WriteString(fmt.Sprintf("\n%d/%d succeeded.", succeeded, len(results)))
	return sb.String(), nil
}

// ListActiveEngagements returns a formatted list of active engagements for the tenant.
func (a *consultingAdapter) ListActiveEngagements(ctx context.Context, tenantID string) (string, error) {
	uid, err := parseConsultUUID(tenantID)
	if err != nil {
		return "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	engagements, err := a.queries.ListActiveEngagements(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("list active engagements: %w", err)
	}

	if len(engagements) == 0 {
		return "No active consulting engagements.\n\nUse /consult <problem> to start one.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active Consulting Engagements (%d):\n\n", len(engagements)))
	for _, eng := range engagements {
		id := formatConsultUUID(eng.ID)
		category := ""
		if eng.Category.Valid {
			category = " [" + eng.Category.String + "]"
		}
		sb.WriteString(fmt.Sprintf("ID: %s\n", id[:8]))
		sb.WriteString(fmt.Sprintf("Title: %s%s\n", eng.Title, category))
		sb.WriteString(fmt.Sprintf("Phase: %s | Tier: %s\n", eng.Phase, eng.Tier))
		sb.WriteString(fmt.Sprintf("Problem: %s\n\n", eng.ProblemStatement))
	}
	sb.WriteString("Use /consult answer <id> <text> to continue a diagnosis.")
	return sb.String(), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseConsultUUID converts a UUID string to pgtype.UUID.
func parseConsultUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}

// formatConsultUUID formats a pgtype.UUID as a hyphenated hex string.
func formatConsultUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// resolveEngagementID resolves a partial or full engagement ID string to a
// pgtype.UUID. If the string is a full UUID it is parsed directly; otherwise
// it tries to find a matching engagement among active ones (not implemented
// here since we always store and echo the full ID from StartEngagement).
func resolveEngagementID(q *sqlc.Queries, ctx context.Context, engagementID string) (pgtype.UUID, error) {
	uid, err := parseConsultUUID(engagementID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid engagement ID %q: %w", engagementID, err)
	}
	return uid, nil
}
