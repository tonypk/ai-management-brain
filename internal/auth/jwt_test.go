package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/tonypk/ai-management-brain/internal/auth"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func TestGenerateAndValidate(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	tenantID := "660e8400-e29b-41d4-a716-446655440000"
	role := "boss"

	token, err := auth.GenerateToken(userID, tenantID, role, testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := auth.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("TenantID = %q, want %q", claims.TenantID, tenantID)
	}
	if claims.Role != role {
		t.Errorf("Role = %q, want %q", claims.Role, role)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	// Verify expiry is approximately 24h from now
	expiry := claims.ExpiresAt.Time
	expected := time.Now().Add(24 * time.Hour)
	diff := expiry.Sub(expected)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expiry diff = %v, want within 1 minute of 24h", diff)
	}
}

func TestExpiredToken(t *testing.T) {
	// Create a token that is already expired by crafting claims manually
	now := time.Now()
	claims := auth.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Role:     "boss",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-48 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, err = auth.ValidateToken(signed, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestInvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"garbage", "not.a.jwt"},
		{"wrong secret", ""},
	}

	// Generate a valid token with a different secret for "wrong secret" case
	wrongSecretToken, err := auth.GenerateToken("user-1", "tenant-1", "boss", []byte("different-secret-key-32-bytes!!!"))
	if err != nil {
		t.Fatalf("GenerateToken with wrong secret: %v", err)
	}
	tests[2].token = wrongSecretToken

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateToken(tt.token, testSecret)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
