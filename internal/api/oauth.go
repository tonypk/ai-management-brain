package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/tonypk/ai-management-brain/internal/auth"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// OAuthConfig holds Google OAuth configuration.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// OAuthHandler handles Google OAuth endpoints.
type OAuthHandler struct {
	queries   *sqlc.Queries
	jwtSecret []byte
	oauth     OAuthConfig
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(queries *sqlc.Queries, jwtSecret []byte, oauth OAuthConfig) *OAuthHandler {
	return &OAuthHandler{
		queries:   queries,
		jwtSecret: jwtSecret,
		oauth:     oauth,
	}
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

type googleUserInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// HandleGoogleCallback exchanges an auth code for user info and logs in or creates an account.
func (h *OAuthHandler) HandleGoogleCallback(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization code is required"})
		return
	}

	// Exchange code for tokens
	tokenData := url.Values{
		"code":          {req.Code},
		"client_id":     {h.oauth.ClientID},
		"client_secret": {h.oauth.ClientSecret},
		"redirect_uri":  {h.oauth.RedirectURI},
		"grant_type":    {"authorization_code"},
	}

	tokenResp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(tokenData.Encode()),
	)
	if err != nil {
		slog.Error("google oauth: token exchange", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange authorization code"})
		return
	}
	defer tokenResp.Body.Close()

	body, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		slog.Error("google oauth: read token response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read token response"})
		return
	}

	if tokenResp.StatusCode != http.StatusOK {
		slog.Error("google oauth: token exchange failed", "status", tokenResp.StatusCode, "body", string(body))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization code"})
		return
	}

	var tokenRes googleTokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		slog.Error("google oauth: parse token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse token response"})
		return
	}

	// Get user info
	userInfoReq, _ := http.NewRequestWithContext(c.Request.Context(), "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	userInfoReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenRes.AccessToken))

	userInfoResp, err := http.DefaultClient.Do(userInfoReq)
	if err != nil {
		slog.Error("google oauth: get user info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user information"})
		return
	}
	defer userInfoResp.Body.Close()

	var userInfo googleUserInfo
	if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
		slog.Error("google oauth: parse user info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user information"})
		return
	}

	if userInfo.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google account has no email"})
		return
	}

	// Check if user exists
	user, err := h.queries.GetUserByEmail(c.Request.Context(), userInfo.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("google oauth: check user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Create new user + tenant
		tenantName := userInfo.Name
		if tenantName == "" {
			tenantName = strings.Split(userInfo.Email, "@")[0]
		}

		tenant, err := h.queries.CreateTenant(c.Request.Context(), sqlc.CreateTenantParams{
			Name:       tenantName + "'s Team",
			Timezone:   "Asia/Singapore",
			MentorID:   "inamori",
			BossChatID: 0,
			Config:     []byte("{}"),
		})
		if err != nil {
			slog.Error("google oauth: create tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// OAuth users get a random password hash (they won't use password login)
		user, err = h.queries.CreateUser(c.Request.Context(), sqlc.CreateUserParams{
			TenantID:     tenant.ID,
			Email:        userInfo.Email,
			PasswordHash: "oauth:google",
			Role:         "boss",
		})
		if err != nil {
			slog.Error("google oauth: create user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	}

	userID := formatUUID(user.ID)
	tenantID := formatUUID(user.TenantID)

	token, err := auth.GenerateToken(userID, tenantID, user.Role, h.jwtSecret)
	if err != nil {
		slog.Error("google oauth: generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, authResponse{Token: token})
}

// HandleGoogleClientID returns the Google OAuth client ID for frontend use.
func (h *OAuthHandler) HandleGoogleClientID(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"client_id": h.oauth.ClientID})
}
