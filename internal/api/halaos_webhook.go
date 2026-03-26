package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// halaosMapper is a local interface for the HalaOS event mapper.
// The concrete implementation (brain.HalaOSMapper) is wired in Task 14.
type halaosMapper interface {
	MapRiskUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
	MapBurnoutUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
	MapAttendanceUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
	MapLeaveUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
	MapPayrollUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
	MapEmployeeUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error
}

// halaosWebhookEnvelope is the top-level structure of every HalaOS webhook payload.
type halaosWebhookEnvelope struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// halaosEventMeta holds the common fields present on all HalaOS event data objects.
type halaosEventMeta struct {
	HRCompanyID int64 `json:"hr_company_id"`
}

// HalaOSWebhookHandler handles incoming webhooks from HalaOS.
type HalaOSWebhookHandler struct {
	queries *sqlc.Queries
	mapper  halaosMapper // nil until Task 14 wires the concrete implementation
}

// NewHalaOSWebhookHandler creates a new HalaOSWebhookHandler.
// mapper may be nil; in that case events are logged but not dispatched.
func NewHalaOSWebhookHandler(q *sqlc.Queries, mapper halaosMapper) *HalaOSWebhookHandler {
	return &HalaOSWebhookHandler{
		queries: q,
		mapper:  mapper,
	}
}

// HandleWebhook processes a POST /webhooks/halaos request.
//
// Flow:
//  1. Read raw body
//  2. Parse envelope to get event_type and data
//  3. Extract hr_company_id from data
//  4. Lookup halaos_links by company_id to get tenant_id + webhook_secret
//  5. Verify HMAC-SHA256 using X-Signature-256 header
//  6. Check idempotency (skip if already processed)
//  7. Dispatch to mapper by event_type (if mapper is wired)
//  8. Persist event log
//  9. Return 200 OK
func (h *HalaOSWebhookHandler) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Read raw body — required for HMAC verification before JSON parsing.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("halaos webhook: read body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// 2. Parse envelope.
	var envelope halaosWebhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		slog.Error("halaos webhook: parse envelope", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON envelope"})
		return
	}
	if envelope.ID == "" || envelope.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "envelope missing id or event_type"})
		return
	}

	// 3. Extract hr_company_id from data.
	var meta halaosEventMeta
	if err := json.Unmarshal(envelope.Data, &meta); err != nil {
		slog.Error("halaos webhook: parse event meta", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event data: missing hr_company_id"})
		return
	}
	if meta.HRCompanyID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event data missing hr_company_id"})
		return
	}

	// 4. Lookup halaos_links by company_id.
	link, err := h.queries.GetHalaOSLinkByCompanyID(ctx, meta.HRCompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("halaos webhook: unknown company", "hr_company_id", meta.HRCompanyID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unknown company"})
			return
		}
		slog.Error("halaos webhook: lookup link", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 5. Verify HMAC-SHA256.
	signature := c.GetHeader("X-Signature-256")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-Signature-256 header"})
		return
	}
	// Strip optional "sha256=" prefix so both forms are accepted.
	rawSig := strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(link.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(rawSig)) {
		slog.Warn("halaos webhook: invalid signature", "event_id", envelope.ID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// 6. Check idempotency — skip if this event was already processed.
	_, err = h.queries.GetHalaOSEventByKey(ctx, sqlc.GetHalaOSEventByKeyParams{
		TenantID:       link.TenantID,
		IdempotencyKey: envelope.ID,
	})
	if err == nil {
		// Event already recorded — acknowledge without reprocessing.
		c.JSON(http.StatusOK, gin.H{"status": "already_processed"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("halaos webhook: check idempotency", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 7. Dispatch to mapper by event_type (only if the mapper is wired in).
	if h.mapper != nil {
		var dispatchErr error
		switch envelope.EventType {
		case "hr.risk.updated":
			dispatchErr = h.mapper.MapRiskUpdated(ctx, link.TenantID, envelope.Data)
		case "hr.burnout.updated":
			dispatchErr = h.mapper.MapBurnoutUpdated(ctx, link.TenantID, envelope.Data)
		case "hr.attendance.updated":
			dispatchErr = h.mapper.MapAttendanceUpdated(ctx, link.TenantID, envelope.Data)
		case "hr.leave.updated":
			dispatchErr = h.mapper.MapLeaveUpdated(ctx, link.TenantID, envelope.Data)
		case "hr.payroll.updated":
			dispatchErr = h.mapper.MapPayrollUpdated(ctx, link.TenantID, envelope.Data)
		case "hr.employee.updated":
			dispatchErr = h.mapper.MapEmployeeUpdated(ctx, link.TenantID, envelope.Data)
		default:
			slog.Info("halaos webhook: unhandled event type", "event_type", envelope.EventType)
		}
		if dispatchErr != nil {
			slog.Error("halaos webhook: dispatch error",
				"event_type", envelope.EventType,
				"event_id", envelope.ID,
				"error", dispatchErr,
			)
			// Continue to log the event; do not return an error to prevent HalaOS retries
			// for dispatch failures that are internal to this system.
		}
	}

	// 8. Persist event log for idempotency and audit trail.
	if _, err := h.queries.CreateHalaOSEvent(ctx, sqlc.CreateHalaOSEventParams{
		TenantID:       link.TenantID,
		EventType:      envelope.EventType,
		IdempotencyKey: envelope.ID,
		Payload:        envelope.Data,
	}); err != nil {
		slog.Error("halaos webhook: persist event", "error", err)
		// Not fatal for the caller — event was already dispatched.
	}

	// 9. Return 200 OK.
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
