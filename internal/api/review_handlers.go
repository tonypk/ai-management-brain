package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Request types ---

type createReviewCycleRequest struct {
	Title     string `json:"title" binding:"required"`
	Period    string `json:"period" binding:"required"`
	Status    string `json:"status"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type updateReviewCycleRequest struct {
	Title     string `json:"title" binding:"required"`
	Status    string `json:"status" binding:"required,oneof=draft active completed"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type createReviewRequest struct {
	EmployeeID string  `json:"employee_id" binding:"required"`
	ReviewerID *string `json:"reviewer_id"`
}

type updateReviewRequest struct {
	Status         string `json:"status" binding:"required,oneof=pending in_progress submitted acknowledged"`
	SelfRating     *int16 `json:"self_rating"`
	ManagerRating  *int16 `json:"manager_rating"`
	SelfSummary    string `json:"self_summary"`
	ManagerSummary string `json:"manager_summary"`
	Strengths      string `json:"strengths"`
	Improvements   string `json:"improvements"`
}

// --- Handlers ---

func handleListReviewCycles(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		cycles, err := q.ListReviewCycles(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list review cycles", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list review cycles"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": cycles})
	}
}

func handleCreateReviewCycle(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createReviewCycleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		startDate, err := parseDate(req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date"})
			return
		}
		endDate, err := parseDate(req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
			return
		}

		status := req.Status
		if status == "" {
			status = "draft"
		}

		cycle, err := q.CreateReviewCycle(c.Request.Context(), sqlc.CreateReviewCycleParams{
			TenantID:  tenantID,
			Title:     req.Title,
			Period:    req.Period,
			Status:    status,
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			slog.Error("create review cycle", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review cycle"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": cycle})
	}
}

func handleUpdateReviewCycle(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req updateReviewCycleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		startDate, err := parseDate(req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date"})
			return
		}
		endDate, err := parseDate(req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
			return
		}

		if err := q.UpdateReviewCycle(c.Request.Context(), sqlc.UpdateReviewCycleParams{
			ID:        id,
			TenantID:  tenantID,
			Title:     req.Title,
			Status:    req.Status,
			StartDate: startDate,
			EndDate:   endDate,
		}); err != nil {
			slog.Error("update review cycle", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review cycle"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleDeleteReviewCycle(q *sqlc.Queries) gin.HandlerFunc {
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
		if err := q.DeleteReviewCycle(c.Request.Context(), sqlc.DeleteReviewCycleParams{ID: id, TenantID: tenantID}); err != nil {
			slog.Error("delete review cycle", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete review cycle"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleListReviewsByCycle(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		cycleID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		// Verify cycle belongs to tenant
		if _, err := q.GetReviewCycle(c.Request.Context(), sqlc.GetReviewCycleParams{ID: cycleID, TenantID: tenantID}); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "review cycle not found"})
			return
		}

		reviews, err := q.ListReviewsByCycle(c.Request.Context(), cycleID)
		if err != nil {
			slog.Error("list reviews", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reviews"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": reviews})
	}
}

func handleCreateReview(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createReviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cycleID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		// Verify cycle belongs to tenant
		if _, err := q.GetReviewCycle(c.Request.Context(), sqlc.GetReviewCycleParams{ID: cycleID, TenantID: tenantID}); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "review cycle not found"})
			return
		}

		empID, err := parseUUID(req.EmployeeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
			return
		}

		params := sqlc.CreateReviewParams{
			CycleID:    cycleID,
			EmployeeID: empID,
		}
		if req.ReviewerID != nil {
			revID, err := parseUUID(*req.ReviewerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reviewer_id"})
				return
			}
			params.ReviewerID = pgtype.UUID{Bytes: revID.Bytes, Valid: true}
		}

		review, err := q.CreateReview(c.Request.Context(), params)
		if err != nil {
			slog.Error("create review", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": review})
	}
}

func handleUpdateReview(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req updateReviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		reviewID, err := parseUUID(c.Param("review_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review_id"})
			return
		}

		// Verify review belongs to tenant via cycle
		review, err := q.GetReview(c.Request.Context(), reviewID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
			return
		}
		cycleTenant, err := q.VerifyReviewCycleTenant(c.Request.Context(), review.CycleID)
		if err != nil || cycleTenant != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
			return
		}

		params := sqlc.UpdateReviewParams{
			ID:             reviewID,
			Status:         req.Status,
			SelfSummary:    req.SelfSummary,
			ManagerSummary: req.ManagerSummary,
			Strengths:      req.Strengths,
			Improvements:   req.Improvements,
		}
		if req.SelfRating != nil {
			params.SelfRating = pgtype.Int2{Int16: *req.SelfRating, Valid: true}
		}
		if req.ManagerRating != nil {
			params.ManagerRating = pgtype.Int2{Int16: *req.ManagerRating, Valid: true}
		}

		if err := q.UpdateReview(c.Request.Context(), params); err != nil {
			slog.Error("update review", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}
