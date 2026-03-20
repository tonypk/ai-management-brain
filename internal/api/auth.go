package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/tonypk/ai-management-brain/internal/auth"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

const bcryptCost = 12

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	queries   *sqlc.Queries
	jwtSecret []byte
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(queries *sqlc.Queries, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		queries:   queries,
		jwtSecret: jwtSecret,
	}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type registerRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	TenantName string `json:"tenant_name" binding:"required,min=1"`
}

type authResponse struct {
	Token string `json:"token"`
}

// HandleLogin authenticates a user and returns a JWT token.
func (h *AuthHandler) HandleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: email and password (min 8 chars) required"})
		return
	}

	user, err := h.queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		slog.Error("login: get user by email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	userID := formatUUID(user.ID)
	tenantID := formatUUID(user.TenantID)

	token, err := auth.GenerateToken(userID, tenantID, user.Role, h.jwtSecret)
	if err != nil {
		slog.Error("login: generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, authResponse{Token: token})
}

// HandleRegister creates a new tenant and boss user, returning a JWT token.
func (h *AuthHandler) HandleRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: email, password (min 8 chars), and tenant_name required"})
		return
	}

	// Check if email already exists
	_, err := h.queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("register: check existing email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		slog.Error("register: hash password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create tenant (with placeholder boss_chat_id=0 for web-only users)
	tenant, err := h.queries.CreateTenant(c.Request.Context(), sqlc.CreateTenantParams{
		Name:       req.TenantName,
		Timezone:   "Asia/Singapore",
		MentorID:   "inamori",
		BossChatID: 0,
		Config:     []byte("{}"),
	})
	if err != nil {
		slog.Error("register: create tenant", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create user with boss role
	user, err := h.queries.CreateUser(c.Request.Context(), sqlc.CreateUserParams{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "boss",
	})
	if err != nil {
		slog.Error("register: create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	userID := formatUUID(user.ID)
	tenantID := formatUUID(tenant.ID)

	token, err := auth.GenerateToken(userID, tenantID, user.Role, h.jwtSecret)
	if err != nil {
		slog.Error("register: generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, authResponse{Token: token})
}

// formatUUID converts pgtype.UUID to string representation.
func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
