package api

import (
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Numeric helpers ---

// numericFromFloat converts a float64 to pgtype.Numeric.
func numericFromFloat(f float64) pgtype.Numeric {
	// Use big.Float for exact conversion
	bf := new(big.Float).SetFloat64(f)
	// Scale to 2 decimal places
	scaled := new(big.Float).Mul(bf, big.NewFloat(100))
	intVal, _ := scaled.Int(nil)
	return pgtype.Numeric{
		Int:   intVal,
		Exp:   -2,
		Valid: true,
	}
}

// numericToFloat converts pgtype.Numeric to float64.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(n.Int).Float64()
	for i := int32(0); i < -n.Exp; i++ {
		f /= 10
	}
	for i := int32(0); i < n.Exp; i++ {
		f *= 10
	}
	return f
}

// --- Request types ---

type createGoalRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Cycle       string  `json:"cycle" binding:"required"`
	OwnerID     *string `json:"owner_id"`
	Status      string  `json:"status"`
}

type updateGoalRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Cycle       string  `json:"cycle" binding:"required"`
	OwnerID     *string `json:"owner_id"`
	Status      string  `json:"status" binding:"required,oneof=draft active completed cancelled"`
}

type createKeyResultRequest struct {
	Title        string  `json:"title" binding:"required"`
	Target       float64 `json:"target" binding:"gt=0"`
	CurrentValue float64 `json:"current_value"`
	Unit         string  `json:"unit"`
	DueDate      *string `json:"due_date"`
}

type updateKeyResultRequest struct {
	Title        string  `json:"title" binding:"required"`
	Target       float64 `json:"target" binding:"gt=0"`
	CurrentValue float64 `json:"current_value"`
	Unit         string  `json:"unit"`
	DueDate      *string `json:"due_date"`
}

// --- Handlers ---

func handleListGoals(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		cycle := c.Query("cycle")
		if cycle == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cycle query parameter is required"})
			return
		}

		rows, err := queries.ListGoalsByCycle(c.Request.Context(), sqlc.ListGoalsByCycleParams{
			TenantID: tenantID,
			Cycle:    cycle,
		})
		if err != nil {
			slog.Error("list goals", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": rows})
	}
}

func handleCreateGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req createGoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status := req.Status
		if status == "" {
			status = "draft"
		}

		var ownerID pgtype.UUID
		if req.OwnerID != nil && *req.OwnerID != "" {
			parsed, err := parseUUID(*req.OwnerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_id"})
				return
			}
			ownerID = parsed
		}

		goal, err := queries.CreateGoal(c.Request.Context(), sqlc.CreateGoalParams{
			TenantID:    tenantID,
			OwnerID:     ownerID,
			Title:       req.Title,
			Description: req.Description,
			Status:      status,
			Cycle:       req.Cycle,
		})
		if err != nil {
			slog.Error("create goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": goal})
	}
}

func handleUpdateGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		goalID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
			return
		}

		var req updateGoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != nil && *req.OwnerID != "" {
			parsed, err := parseUUID(*req.OwnerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_id"})
				return
			}
			ownerID = parsed
		}

		goal, err := queries.UpdateGoal(c.Request.Context(), sqlc.UpdateGoalParams{
			ID:          goalID,
			TenantID:    tenantID,
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Cycle:       req.Cycle,
			OwnerID:     ownerID,
		})
		if err != nil {
			slog.Error("update goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": goal})
	}
}

func handleDeleteGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		goalID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
			return
		}

		if err := queries.DeleteGoal(c.Request.Context(), sqlc.DeleteGoalParams{
			ID:       goalID,
			TenantID: tenantID,
		}); err != nil {
			slog.Error("delete goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "deleted"})
	}
}

// verifyGoalTenant checks that a goal belongs to the requesting tenant.
func verifyGoalTenant(c *gin.Context, queries *sqlc.Queries) (pgtype.UUID, pgtype.UUID, bool) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	goalID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	if _, err = queries.GetGoal(c.Request.Context(), sqlc.GetGoalParams{
		ID:       goalID,
		TenantID: tenantID,
	}); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	return goalID, tenantID, true
}

func handleCreateKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		var req createKeyResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		unit := req.Unit
		if unit == "" {
			unit = "%"
		}

		var dueDate pgtype.Date
		if req.DueDate != nil && *req.DueDate != "" {
			parsed, err := parseDate(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid due_date: %v", err)})
				return
			}
			dueDate = parsed
		}

		kr, err := queries.CreateKeyResult(c.Request.Context(), sqlc.CreateKeyResultParams{
			GoalID:       goalID,
			Title:        req.Title,
			Target:       numericFromFloat(req.Target),
			CurrentValue: numericFromFloat(req.CurrentValue),
			Unit:         unit,
			DueDate:      dueDate,
		})
		if err != nil {
			slog.Error("create key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": kr})
	}
}

func handleUpdateKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		krID, err := parseUUID(c.Param("kr_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key result id"})
			return
		}

		var req updateKeyResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		unit := req.Unit
		if unit == "" {
			unit = "%"
		}

		var dueDate pgtype.Date
		if req.DueDate != nil && *req.DueDate != "" {
			parsed, err := parseDate(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid due_date: %v", err)})
				return
			}
			dueDate = parsed
		}

		if err := queries.UpdateKeyResult(c.Request.Context(), sqlc.UpdateKeyResultParams{
			ID:           krID,
			Title:        req.Title,
			Target:       numericFromFloat(req.Target),
			CurrentValue: numericFromFloat(req.CurrentValue),
			Unit:         unit,
			DueDate:      dueDate,
		}); err != nil {
			slog.Error("update key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "updated"})
	}
}

func handleDeleteKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		krID, err := parseUUID(c.Param("kr_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key result id"})
			return
		}

		if err := queries.DeleteKeyResult(c.Request.Context(), krID); err != nil {
			slog.Error("delete key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "deleted"})
	}
}

func handleListSnapshots(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		snapshots, err := queries.ListGoalSnapshots(c.Request.Context(), goalID)
		if err != nil {
			slog.Error("list snapshots", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": snapshots})
	}
}

// --- Exported helper for cron job ---

// CalculateGoalProgress computes overall progress for a goal from its key results.
func CalculateGoalProgress(krs []sqlc.KeyResult) float64 {
	if len(krs) == 0 {
		return 0
	}
	var sum float64
	for _, kr := range krs {
		target := numericToFloat(kr.Target)
		current := numericToFloat(kr.CurrentValue)
		if target > 0 {
			sum += math.Min(current/target*100, 100)
		}
	}
	return math.Round(sum/float64(len(krs))*100) / 100
}
