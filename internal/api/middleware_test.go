package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/api"
	"github.com/tonypk/ai-management-brain/internal/auth"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(api.AuthMiddleware(testSecret))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"user_id":   c.GetString("user_id"),
			"tenant_id": c.GetString("tenant_id"),
			"role":      c.GetString("role"),
		})
	})
	return r
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	r := setupAuthRouter()
	token, _ := auth.GenerateToken("user-1", "tenant-1", "boss", testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["user_id"] != "user-1" {
		t.Errorf("user_id = %q, want user-1", resp["user_id"])
	}
	if resp["tenant_id"] != "tenant-1" {
		t.Errorf("tenant_id = %q, want tenant-1", resp["tenant_id"])
	}
	if resp["role"] != "boss" {
		t.Errorf("role = %q, want boss", resp["role"])
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	r := setupAuthRouter()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	r := setupAuthRouter()

	tests := []struct {
		name   string
		header string
	}{
		{"no Bearer prefix", "token-only"},
		{"Basic auth", "Basic dXNlcjpwYXNz"},
		{"empty Bearer", "Bearer "},
		{"just Bearer", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %q, got %d", tt.header, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	r := setupAuthRouter()

	// Token signed with wrong secret
	token, _ := auth.GenerateToken("user-1", "tenant-1", "boss", []byte("wrong-secret-wrong-secret-32byte"))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong-secret token, got %d", w.Code)
	}
}

func TestRequireRole_BossAllowed(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "boss")
		c.Next()
	})
	r.Use(api.RequireRole("boss"))
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for boss, got %d", w.Code)
	}
}

func TestRequireRole_MemberDenied(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "member")
		c.Next()
	})
	r.Use(api.RequireRole("boss"))
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for member, got %d", w.Code)
	}
}

func TestRequireRole_HierarchicalAccess(t *testing.T) {
	// Boss should be able to access admin-level endpoints
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "boss")
		c.Next()
	})
	r.Use(api.RequireRole("admin")) // requires admin
	r.GET("/admin-page", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin-page", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for boss accessing admin endpoint, got %d", w.Code)
	}
}

func TestRequireRole_OwnerAlias(t *testing.T) {
	// Owner should have same permissions as boss
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "owner")
		c.Next()
	})
	r.Use(api.RequireRole("boss"))
	r.GET("/boss-page", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/boss-page", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for owner (same level as boss), got %d", w.Code)
	}
}

func TestRequireRole_NoRoleSet(t *testing.T) {
	r := gin.New()
	// Don't set role in context
	r.Use(api.RequireRole("boss"))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no role set, got %d", w.Code)
	}
}

func TestRequireRole_InvalidRoleType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", 123) // wrong type
		c.Next()
	})
	r.Use(api.RequireRole("boss"))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-string role, got %d", w.Code)
	}
}

func TestTenantFromContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant_id", "tenant-abc")

	got := api.TenantFromContext(c)
	if got != "tenant-abc" {
		t.Errorf("TenantFromContext = %q, want tenant-abc", got)
	}
}

func TestTenantFromContext_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	got := api.TenantFromContext(c)
	if got != "" {
		t.Errorf("TenantFromContext = %q, want empty", got)
	}
}

func TestRoleLevel_Values(t *testing.T) {
	if api.RoleLevel["member"] != 1 {
		t.Errorf("member level = %d, want 1", api.RoleLevel["member"])
	}
	if api.RoleLevel["admin"] != 2 {
		t.Errorf("admin level = %d, want 2", api.RoleLevel["admin"])
	}
	if api.RoleLevel["boss"] != 3 {
		t.Errorf("boss level = %d, want 3", api.RoleLevel["boss"])
	}
	if api.RoleLevel["owner"] != 3 {
		t.Errorf("owner level = %d, want 3", api.RoleLevel["owner"])
	}
}
