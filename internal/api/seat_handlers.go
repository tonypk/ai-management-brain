package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/seats"
)

// handleListSeats returns all seats for the authenticated tenant.
func handleListSeats(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		seatsList, err := q.ListSeatsByTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list seats", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		result := make([]gin.H, len(seatsList))
		for i, s := range seatsList {
			result[i] = gin.H{
				"id":         formatUUID(s.ID),
				"seat_type":  s.SeatType,
				"title":      s.Title,
				"persona_id": s.PersonaID,
				"scope":      s.Scope,
				"is_active":  s.IsActive.Bool,
				"created_at": s.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
				"updated_at": s.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// createSeatRequest holds the request body for creating a seat.
type createSeatRequest struct {
	SeatType  string `json:"seat_type" binding:"required,min=1"`
	PersonaID string `json:"persona_id" binding:"required,min=1"`
	Title     string `json:"title"`
	Scope     string `json:"scope"`
}

// handleCreateSeat creates a new C-suite seat for the tenant.
func handleCreateSeat(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createSeatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate persona exists
		if !brain.ValidMentors[req.PersonaID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown persona_id"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Default title from seat type if not provided
		title := req.Title
		if title == "" {
			title = defaultTitleForSeatType(req.SeatType)
		}

		seat, err := q.CreateSeat(c.Request.Context(), sqlc.CreateSeatParams{
			TenantID:  tenantID,
			SeatType:  req.SeatType,
			Title:     title,
			PersonaID: req.PersonaID,
			Scope:     req.Scope,
		})
		if err != nil {
			if isUniqueViolation(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "seat type already assigned for this tenant"})
				return
			}
			slog.Error("create seat", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{
				"id":         formatUUID(seat.ID),
				"seat_type":  seat.SeatType,
				"title":      seat.Title,
				"persona_id": seat.PersonaID,
				"scope":      seat.Scope,
				"is_active":  seat.IsActive.Bool,
			},
		})
	}
}

// updateSeatRequest holds the request body for updating a seat.
type updateSeatRequest struct {
	Title     string `json:"title" binding:"required,min=1"`
	PersonaID string `json:"persona_id" binding:"required,min=1"`
	Scope     string `json:"scope"`
}

// handleUpdateSeat updates a seat's title, persona, and scope.
func handleUpdateSeat(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateSeatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !brain.ValidMentors[req.PersonaID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown persona_id"})
			return
		}

		seatID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seat ID"})
			return
		}

		// Verify tenant ownership
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		existing, err := q.GetSeatByID(c.Request.Context(), seatID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "seat not found"})
				return
			}
			slog.Error("get seat", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if formatUUID(existing.TenantID) != formatUUID(tenantID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		updated, err := q.UpdateSeat(c.Request.Context(), sqlc.UpdateSeatParams{
			ID:        seatID,
			Title:     req.Title,
			PersonaID: req.PersonaID,
			Scope:     req.Scope,
		})
		if err != nil {
			slog.Error("update seat", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":         formatUUID(updated.ID),
				"seat_type":  updated.SeatType,
				"title":      updated.Title,
				"persona_id": updated.PersonaID,
				"scope":      updated.Scope,
				"is_active":  updated.IsActive.Bool,
			},
		})
	}
}

// handleDeleteSeat removes a seat by ID after verifying tenant ownership.
func handleDeleteSeat(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		seatID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seat ID"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		existing, err := q.GetSeatByID(c.Request.Context(), seatID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "seat not found"})
				return
			}
			slog.Error("get seat for delete", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if formatUUID(existing.TenantID) != formatUUID(tenantID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		if err := q.DeleteSeat(c.Request.Context(), seatID); err != nil {
			slog.Error("delete seat", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
	}
}

// boardDiscussRequest holds the request body for a board discussion.
type boardDiscussRequest struct {
	Topic string `json:"topic" binding:"required,min=1"`
}

// handleBoardDiscuss triggers a multi-seat board discussion on a topic.
func handleBoardDiscuss(seatSvc *seats.SeatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req boardDiscussRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenantID := TenantFromContext(c)
		cultureCode := "default"

		responses, synthesis, err := seatSvc.BoardDiscuss(c.Request.Context(), tenantID, cultureCode, req.Topic)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "limited") {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": errMsg})
				return
			}
			if strings.Contains(errMsg, "no active seats") || strings.Contains(errMsg, "invalid tenant") {
				c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
				return
			}
			slog.Error("board discuss", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"topic":     req.Topic,
				"responses": responses,
				"synthesis": synthesis,
			},
		})
	}
}

// handleSeatChat lets the MCP server chat with a specific C-Suite seat.
func handleSeatChat(seatSvc *seats.SeatService, q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SeatType string `json:"seat_type" binding:"required"`
			Message  string `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "seat_type and message are required"})
			return
		}

		tenantID := TenantFromContext(c)
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Look up seat for metadata (title, persona_id)
		seat, err := q.GetSeatByType(c.Request.Context(), sqlc.GetSeatByTypeParams{
			TenantID: tenantUUID,
			SeatType: req.SeatType,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown seat type"})
			return
		}

		if !seat.IsActive.Bool {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{
				"message": "The " + seat.Title + " seat is currently inactive.",
			}})
			return
		}

		response, err := seatSvc.Chat(c.Request.Context(), tenantID, req.SeatType, "default", req.Message)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "limited") {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": errMsg})
				return
			}
			slog.Error("seat chat error", "error", errMsg, "seat_type", req.SeatType)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"seat_type":  req.SeatType,
			"title":      seat.Title,
			"persona_id": seat.PersonaID,
			"response":   response,
		}})
	}
}

// handleListMentorsWithDomain returns all mentors with domain and recommendation metadata.
func handleListMentorsWithDomain() gin.HandlerFunc {
	return func(c *gin.Context) {
		mentors := make([]gin.H, 0, len(brain.ValidMentors))
		for id := range brain.ValidMentors {
			cfg, err := brain.LoadMentor(id)
			if err != nil {
				continue
			}
			mentors = append(mentors, gin.H{
				"id":                cfg.ID,
				"name":              cfg.Name,
				"name_en":           cfg.NameEn,
				"company":           cfg.Company,
				"philosophy":        cfg.Philosophy,
				"domain":            cfg.Domain,
				"tags":              cfg.Tags,
				"recommended_seats": cfg.RecommendedSeats,
			})
		}
		c.JSON(http.StatusOK, gin.H{"data": mentors})
	}
}

// defaultTitleForSeatType returns a human-readable title for well-known seat types.
func defaultTitleForSeatType(seatType string) string {
	defaults := map[string]string{
		"ceo":  "Chief Executive Officer",
		"cfo":  "Chief Financial Officer",
		"cmo":  "Chief Marketing Officer",
		"cto":  "Chief Technology Officer",
		"chro": "Chief Human Resources Officer",
		"coo":  "Chief Operations Officer",
	}
	if t, ok := defaults[seatType]; ok {
		return t
	}
	return seatType
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "unique") ||
		strings.Contains(err.Error(), "duplicate") ||
		strings.Contains(err.Error(), "23505"))
}
