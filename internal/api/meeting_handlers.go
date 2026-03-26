package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Request types ---

type createMeetingRequest struct {
	EmployeeID  string `json:"employee_id" binding:"required"`
	ManagerID   string `json:"manager_id"`
	MeetingDate string `json:"meeting_date" binding:"required"`
	DurationMin int16  `json:"duration_min"`
	Notes       string `json:"notes"`
	Mood        string `json:"mood"`
	FollowUp    string `json:"follow_up"`
}

type updateMeetingRequest struct {
	Notes       string `json:"notes"`
	Mood        string `json:"mood"`
	FollowUp    string `json:"follow_up"`
	DurationMin int16  `json:"duration_min"`
}

type createActionItemRequest struct {
	Title      string  `json:"title" binding:"required"`
	AssigneeID *string `json:"assignee_id"`
	DueDate    *string `json:"due_date"`
}

type updateActionItemRequest struct {
	Title      string  `json:"title" binding:"required"`
	Status     string  `json:"status" binding:"required,oneof=open in_progress done"`
	AssigneeID *string `json:"assignee_id"`
	DueDate    *string `json:"due_date"`
}

// --- Handlers ---

func handleListMeetings(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32)
		offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)

		meetings, err := q.ListMeetings(c.Request.Context(), sqlc.ListMeetingsParams{
			TenantID: tenantID,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
		if err != nil {
			slog.Error("list meetings", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list meetings"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": meetings})
	}
}

func handleGetMeeting(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		id, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		meeting, err := q.GetMeeting(c.Request.Context(), sqlc.GetMeetingParams{ID: id, TenantID: tenantID})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "meeting not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": meeting})
	}
}

func handleCreateMeeting(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createMeetingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		empID, err := parseUUID(req.EmployeeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
			return
		}
		meetingDate, err := parseDate(req.MeetingDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meeting_date"})
			return
		}

		dur := req.DurationMin
		if dur == 0 {
			dur = 30
		}

		params := sqlc.CreateMeetingParams{
			TenantID:    tenantID,
			EmployeeID:  empID,
			MeetingDate: meetingDate,
			DurationMin: dur,
			Notes:       req.Notes,
			Mood:        req.Mood,
			FollowUp:    req.FollowUp,
		}
		if req.ManagerID != "" {
			mgrID, err := parseUUID(req.ManagerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manager_id"})
				return
			}
			params.ManagerID = pgtype.UUID{Bytes: mgrID.Bytes, Valid: true}
		}

		meeting, err := q.CreateMeeting(c.Request.Context(), params)
		if err != nil {
			slog.Error("create meeting", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create meeting"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": meeting})
	}
}

func handleUpdateMeeting(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		id, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req updateMeetingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := q.UpdateMeeting(c.Request.Context(), sqlc.UpdateMeetingParams{
			ID:          id,
			TenantID:    tenantID,
			Notes:       req.Notes,
			Mood:        req.Mood,
			FollowUp:    req.FollowUp,
			DurationMin: req.DurationMin,
		}); err != nil {
			slog.Error("update meeting", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update meeting"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleDeleteMeeting(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		id, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := q.DeleteMeeting(c.Request.Context(), sqlc.DeleteMeetingParams{ID: id, TenantID: tenantID}); err != nil {
			slog.Error("delete meeting", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete meeting"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleListActionItems(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		meetingID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		// Verify meeting belongs to tenant
		if _, err := q.GetMeeting(c.Request.Context(), sqlc.GetMeetingParams{ID: meetingID, TenantID: tenantID}); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "meeting not found"})
			return
		}

		items, err := q.ListActionItems(c.Request.Context(), meetingID)
		if err != nil {
			slog.Error("list action items", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list action items"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func handleCreateActionItem(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		meetingID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req createActionItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify meeting belongs to tenant
		if _, err := q.GetMeeting(c.Request.Context(), sqlc.GetMeetingParams{ID: meetingID, TenantID: tenantID}); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "meeting not found"})
			return
		}

		params := sqlc.CreateActionItemParams{
			MeetingID: meetingID,
			Title:     req.Title,
		}
		if req.AssigneeID != nil {
			aID, err := parseUUID(*req.AssigneeID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignee_id"})
				return
			}
			params.AssigneeID = pgtype.UUID{Bytes: aID.Bytes, Valid: true}
		}
		if req.DueDate != nil {
			d, err := parseDate(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date"})
				return
			}
			params.DueDate = d
		}

		item, err := q.CreateActionItem(c.Request.Context(), params)
		if err != nil {
			slog.Error("create action item", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create action item"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": item})
	}
}

func handleUpdateActionItem(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		aiID, err := parseUUID(c.Param("ai_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action item id"})
			return
		}
		var req updateActionItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		params := sqlc.UpdateActionItemParams{
			ID:     aiID,
			Title:  req.Title,
			Status: req.Status,
		}
		if req.AssigneeID != nil {
			aID, err := parseUUID(*req.AssigneeID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignee_id"})
				return
			}
			params.AssigneeID = pgtype.UUID{Bytes: aID.Bytes, Valid: true}
		}
		if req.DueDate != nil {
			d, err := parseDate(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date"})
				return
			}
			params.DueDate = d
		}

		if err := q.UpdateActionItem(c.Request.Context(), params); err != nil {
			slog.Error("update action item", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update action item"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleDeleteActionItem(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		aiID, err := parseUUID(c.Param("ai_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action item id"})
			return
		}
		if err := q.DeleteActionItem(c.Request.Context(), aiID); err != nil {
			slog.Error("delete action item", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete action item"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleListOpenActionItems(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		items, err := q.ListOpenActionItemsByTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list open action items", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list action items"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}
