package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/service"
)

// OpenClawActionHandler handles write operations exposed via OpenClaw MCP.
type OpenClawActionHandler struct {
	actionSvc *service.ActionService
}

// NewOpenClawActionHandler creates a new handler for OpenClaw action endpoints.
func NewOpenClawActionHandler(svc *service.ActionService) *OpenClawActionHandler {
	return &OpenClawActionHandler{actionSvc: svc}
}

type checkinRequest struct {
	EmployeeName string `json:"employee_name"`
}

// HandleCheckin triggers check-in questions for all or a specific employee.
func (h *OpenClawActionHandler) HandleCheckin(c *gin.Context) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return
	}

	var req checkinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK — means "all employees"
		req = checkinRequest{}
	}

	result, err := h.actionSvc.TriggerCheckin(c.Request.Context(), tenantID, req.EmployeeName)
	if err != nil {
		slog.Error("openclaw checkin", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sent_to": result.SentTo,
		"skipped": result.Skipped,
	})
}

type chaseRequest struct {
	EmployeeName string `json:"employee_name"`
}

// HandleChase triggers chase reminders for non-submitters.
func (h *OpenClawActionHandler) HandleChase(c *gin.Context) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return
	}

	var req chaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = chaseRequest{}
	}

	result, err := h.actionSvc.TriggerChase(c.Request.Context(), tenantID, req.EmployeeName)
	if err != nil {
		slog.Error("openclaw chase", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chased":  result.Chased,
		"skipped": result.Skipped,
	})
}

// HandleSummary triggers generation and delivery of the daily summary.
func (h *OpenClawActionHandler) HandleSummary(c *gin.Context) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return
	}

	result, err := h.actionSvc.TriggerSummary(c.Request.Context(), tenantID)
	if err != nil {
		slog.Error("openclaw summary", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":         result.Summary,
		"submission_rate": result.SubmissionRate,
		"sent_to":         result.SentTo,
	})
}

type messageRequest struct {
	EmployeeName string `json:"employee_name" binding:"required"`
	Message      string `json:"message" binding:"required"`
}

// HandleMessage sends an arbitrary message to an employee.
func (h *OpenClawActionHandler) HandleMessage(c *gin.Context) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return
	}

	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee_name and message are required"})
		return
	}

	result, err := h.actionSvc.SendMessage(c.Request.Context(), tenantID, req.EmployeeName, req.Message)
	if err != nil {
		slog.Error("openclaw message", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sent_to": result.SentTo,
		"channel": result.Channel,
	})
}
