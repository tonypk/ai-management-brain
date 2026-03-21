package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Request/Response types ---

type createAPIKeyRequest struct {
	Name string `json:"name" binding:"required,min=1"`
}

type apiKeyResponse struct {
	ID        string  `json:"id"`
	Prefix    string  `json:"prefix"`
	Name      string  `json:"name"`
	Key       string  `json:"key,omitempty"` // only on create
	CreatedAt string  `json:"created_at"`
	LastUsed  *string `json:"last_used_at"`
}

// --- Key generation helpers ---

// generateAPIKey creates a new API key with "mb_" prefix and 40 random hex chars.
func generateAPIKey() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return "mb_" + hex.EncodeToString(b)
}

// apiKeyPrefix returns the first 10 characters of the key for display purposes.
func apiKeyPrefix(key string) string {
	if len(key) <= 10 {
		return key
	}
	return key[:10]
}

// hashAPIKey returns the SHA-256 hex digest of the key.
func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// --- Middleware ---

// APIKeyMiddleware checks for "Bearer mb_..." tokens and resolves to user/tenant context.
// If the token is not an API key (no "mb_" prefix or empty), it falls through to
// the next middleware (e.g. JWT). If it IS an API key, it looks it up and sets context.
func APIKeyMiddleware(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		token := parts[1]
		if !strings.HasPrefix(token, "mb_") {
			// Not an API key — let JWT middleware handle it
			c.Next()
			return
		}

		// It IS an API key — we must resolve it
		if queries == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key authentication unavailable"})
			return
		}

		hash := hashAPIKey(token)
		row, err := queries.GetAPIKeyByHash(c.Request.Context(), hash)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		// Set context values matching what AuthMiddleware sets
		c.Set("user_id", formatUUID(row.UserID))
		c.Set("tenant_id", formatUUID(row.TenantID))
		c.Set("role", row.Role)
		c.Set("auth_method", "api_key")
		c.Set("api_key_id", formatUUID(row.ID))

		// Touch last_used_at in the background (best-effort)
		go func() {
			_ = queries.TouchAPIKeyLastUsed(c.Request.Context(), row.ID)
		}()

		c.Next()
	}
}

// --- CRUD Handlers ---

// handleCreateAPIKey creates a new API key and returns the full key only once.
func handleCreateAPIKey(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := parseUUID(userIDStr.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		rawKey := generateAPIKey()
		prefix := apiKeyPrefix(rawKey)
		hash := hashAPIKey(rawKey)

		apiKey, err := queries.CreateAPIKey(c.Request.Context(), sqlc.CreateAPIKeyParams{
			UserID:   userID,
			TenantID: tenantID,
			Prefix:   prefix,
			KeyHash:  hash,
			Name:     req.Name,
			Scopes:   []string{},
		})
		if err != nil {
			slog.Error("create api key", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": apiKeyResponse{
				ID:        formatUUID(apiKey.ID),
				Prefix:    apiKey.Prefix,
				Name:      apiKey.Name,
				Key:       rawKey,
				CreatedAt: apiKey.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
				LastUsed:  nil,
			},
		})
	}
}

// handleListAPIKeys lists the authenticated user's active API keys.
func handleListAPIKeys(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := parseUUID(userIDStr.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
			return
		}

		keys, err := queries.ListAPIKeysByUser(c.Request.Context(), userID)
		if err != nil {
			slog.Error("list api keys", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		result := make([]apiKeyResponse, 0, len(keys))
		for _, k := range keys {
			resp := apiKeyResponse{
				ID:        formatUUID(k.ID),
				Prefix:    k.Prefix,
				Name:      k.Name,
				CreatedAt: k.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			}
			if k.LastUsedAt.Valid {
				t := k.LastUsedAt.Time.Format("2006-01-02T15:04:05Z")
				resp.LastUsed = &t
			}
			result = append(result, resp)
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// handleRevokeAPIKey deactivates an API key by ID.
func handleRevokeAPIKey(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := parseUUID(userIDStr.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
			return
		}

		if err := queries.RevokeAPIKey(c.Request.Context(), sqlc.RevokeAPIKeyParams{
			ID:     keyID,
			UserID: userID,
		}); err != nil {
			slog.Error("revoke api key", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"revoked": true}})
	}
}

// toAPIKeyResponse converts an sqlc.ApiKey to the response type (helper for DRY).
func toAPIKeyResponse(k sqlc.ApiKey) apiKeyResponse {
	resp := apiKeyResponse{
		ID:        formatUUID(k.ID),
		Prefix:    k.Prefix,
		Name:      k.Name,
		CreatedAt: k.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
	if k.LastUsedAt.Valid {
		t := k.LastUsedAt.Time.Format("2006-01-02T15:04:05Z")
		resp.LastUsed = &t
	}
	return resp
}
