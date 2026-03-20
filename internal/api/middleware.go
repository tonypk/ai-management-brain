package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/auth"
)

// AuthMiddleware validates the Bearer token and sets user claims in the gin context.
func AuthMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// TenantFromContext extracts the tenant ID from the gin context.
func TenantFromContext(c *gin.Context) string {
	v, _ := c.Get("tenant_id")
	s, _ := v.(string)
	return s
}

// RoleLevel maps roles to numeric levels for hierarchical comparison.
// Higher level = more permissions.
var RoleLevel = map[string]int{
	"member": 1,
	"admin":  2,
	"boss":   3,
	"owner":  3, // alias for boss
}

// RequireRole returns middleware that checks if the user's role is in the allowed list.
// Also supports hierarchical check: a boss/owner can do anything an admin can do.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	minLevel := 100
	for _, r := range roles {
		allowed[r] = true
		if lvl, ok := RoleLevel[r]; ok && lvl < minLevel {
			minLevel = lvl
		}
	}
	return func(c *gin.Context) {
		v, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		role, ok := v.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		// Direct match or hierarchical match
		if allowed[role] {
			c.Next()
			return
		}
		userLevel := RoleLevel[role]
		if userLevel >= minLevel {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
