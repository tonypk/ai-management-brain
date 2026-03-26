package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Training Programs ---

type createTrainingProgramRequest struct {
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	DurationHours int32  `json:"duration_hours"`
	Provider      string `json:"provider"`
	URL           string `json:"url"`
	IsMandatory   bool   `json:"is_mandatory"`
}

type updateTrainingProgramRequest struct {
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	DurationHours int32  `json:"duration_hours"`
	Provider      string `json:"provider"`
	URL           string `json:"url"`
	IsMandatory   bool   `json:"is_mandatory"`
	Status        string `json:"status"`
}

func handleListTrainingPrograms(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		programs, err := q.ListTrainingPrograms(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list programs"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": programs})
	}
}

func handleCreateTrainingProgram(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createTrainingProgramRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		program, err := q.CreateTrainingProgram(c.Request.Context(), sqlc.CreateTrainingProgramParams{
			TenantID:      tenantID,
			Title:         req.Title,
			Description:   req.Description,
			Category:      req.Category,
			DurationHours: req.DurationHours,
			Provider:      req.Provider,
			Url:           req.URL,
			IsMandatory:   req.IsMandatory,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create program"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": program})
	}
}

func handleUpdateTrainingProgram(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		programID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program id"})
			return
		}
		owner, err := q.VerifyTrainingProgramTenant(c.Request.Context(), programID)
		if err != nil || owner != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
			return
		}
		var req updateTrainingProgramRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		status := req.Status
		if status == "" {
			status = "active"
		}
		program, err := q.UpdateTrainingProgram(c.Request.Context(), sqlc.UpdateTrainingProgramParams{
			ID:            programID,
			Title:         req.Title,
			Description:   req.Description,
			Category:      req.Category,
			DurationHours: req.DurationHours,
			Provider:      req.Provider,
			Url:           req.URL,
			IsMandatory:   req.IsMandatory,
			Status:        status,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update program"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": program})
	}
}

func handleDeleteTrainingProgram(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		programID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program id"})
			return
		}
		owner, err := q.VerifyTrainingProgramTenant(c.Request.Context(), programID)
		if err != nil || owner != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
			return
		}
		if err := q.DeleteTrainingProgram(c.Request.Context(), programID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete program"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

// --- Enrollments ---

type createEnrollmentRequest struct {
	EmployeeID string `json:"employee_id" binding:"required"`
}

type updateEnrollmentRequest struct {
	Status string `json:"status" binding:"required"`
	Score  *int32 `json:"score"`
	Notes  string `json:"notes"`
}

func handleListEnrollments(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		programID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program id"})
			return
		}
		enrollments, err := q.ListEnrollments(c.Request.Context(), programID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list enrollments"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": enrollments})
	}
}

func handleCreateEnrollment(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		programID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program id"})
			return
		}
		var req createEnrollmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		empID, err := parseUUID(req.EmployeeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
			return
		}
		enrollment, err := q.CreateEnrollment(c.Request.Context(), sqlc.CreateEnrollmentParams{
			ProgramID:  programID,
			EmployeeID: empID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create enrollment"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": enrollment})
	}
}

func handleUpdateEnrollment(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		enrollmentID, err := parseUUID(c.Param("eid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment id"})
			return
		}
		var req updateEnrollmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var score pgtype.Int4
		if req.Score != nil {
			score = pgtype.Int4{Int32: *req.Score, Valid: true}
		}
		enrollment, err := q.UpdateEnrollment(c.Request.Context(), sqlc.UpdateEnrollmentParams{
			ID:     enrollmentID,
			Status: req.Status,
			Score:  score,
			Notes:  req.Notes,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update enrollment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": enrollment})
	}
}

func handleDeleteEnrollment(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		enrollmentID, err := parseUUID(c.Param("eid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment id"})
			return
		}
		if err := q.DeleteEnrollment(c.Request.Context(), enrollmentID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete enrollment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
