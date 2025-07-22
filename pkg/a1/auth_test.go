package a1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestConfig() *config.A1Config {
	return &config.A1Config{
		Auth: config.AuthConfig{
			Enabled:    true,
			Issuer:     "test-ric-issuer",
			Audience:   "test-ric-audience",
			TokenExpiry: 3600, // 1 hour
		},
	}
}

func TestNewAuthService(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	
	authService, err := NewAuthService(config, logger)
	
	require.NoError(t, err)
	assert.NotNil(t, authService)
	assert.NotNil(t, authService.config)
	assert.NotNil(t, authService.logger)
	assert.NotNil(t, authService.privateKey)
	assert.NotNil(t, authService.publicKey)
	assert.NotEmpty(t, authService.roles)
	assert.NotEmpty(t, authService.permissions)
}

func TestGenerateToken(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Test token generation
	userID := "test-user-123"
	username := "testuser"
	email := "test@example.com"
	roles := []string{"admin"}
	
	tokenResp, err := authService.GenerateToken(userID, username, email, roles)
	
	require.NoError(t, err)
	assert.NotNil(t, tokenResp)
	assert.NotEmpty(t, tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Equal(t, int64(3600), tokenResp.ExpiresIn)
	assert.NotEmpty(t, tokenResp.Scope)
}

func TestValidateToken_Valid(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Generate a valid token
	userID := "test-user-123"
	username := "testuser"
	email := "test@example.com"
	roles := []string{"admin"}
	
	tokenResp, err := authService.GenerateToken(userID, username, email, roles)
	require.NoError(t, err)
	
	// Validate the token
	claims, err := authService.ValidateToken(tokenResp.AccessToken)
	
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, roles, claims.Roles)
	assert.NotEmpty(t, claims.Permissions)
}

func TestValidateToken_Invalid(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	tests := []struct {
		name  string
		token string
	}{
		{"Empty token", ""},
		{"Invalid format", "invalid-token"},
		{"Wrong signature", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.invalid"},
		{"Malformed JWT", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.malformed.signature"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateToken(tt.token)
			
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateToken_Blacklisted(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Generate and then revoke a token
	tokenResp, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"viewer"})
	require.NoError(t, err)
	
	// First validation should succeed
	claims, err := authService.ValidateToken(tokenResp.AccessToken)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	
	// Revoke the token
	err = authService.RevokeToken(tokenResp.AccessToken)
	require.NoError(t, err)
	
	// Second validation should fail
	claims, err = authService.ValidateToken(tokenResp.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "blacklisted")
}

func TestHasPermission(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	userPermissions := []string{"policy:read", "policy:write", "model:read"}
	
	tests := []struct {
		permission string
		expected   bool
	}{
		{"policy:read", true},
		{"policy:write", true},
		{"policy:delete", false},
		{"model:read", true},
		{"model:write", false},
		{"admin:read", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.permission, func(t *testing.T) {
			hasPermission := authService.HasPermission(userPermissions, tt.permission)
			assert.Equal(t, tt.expected, hasPermission)
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Generate a valid token
	tokenResp, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"admin"})
	require.NoError(t, err)
	
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if ok {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user": user.Username})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	
	middleware := authService.AuthMiddleware()
	handler := middleware(testHandler)
	
	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", response["user"])
	})
	
	t.Run("Missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("Invalid token format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("Invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRequirePermission(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Generate tokens with different roles
	adminToken, err := authService.GenerateToken("admin", "adminuser", "admin@example.com", []string{"admin"})
	require.NoError(t, err)
	
	viewerToken, err := authService.GenerateToken("viewer", "vieweruser", "viewer@example.com", []string{"viewer"})
	require.NoError(t, err)
	
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	authMiddleware := authService.AuthMiddleware()
	permissionMiddleware := authService.RequirePermission("policy:write")
	handler := authMiddleware(permissionMiddleware(testHandler))
	
	t.Run("Admin with policy:write permission", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("Viewer without policy:write permission", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken.AccessToken)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRolePermissionResolution(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		roles       []string
		expectedPerms int
	}{
		{"Admin role", []string{"admin"}, 10}, // Admin has all permissions
		{"Operator role", []string{"operator"}, 7}, // Operator has subset
		{"Viewer role", []string{"viewer"}, 4}, // Viewer has read-only
		{"Multiple roles", []string{"viewer", "operator"}, 7}, // Union of permissions
		{"No roles", []string{}, 0},
		{"Unknown role", []string{"unknown"}, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := authService.resolvePermissions(tt.roles)
			assert.Len(t, permissions, tt.expectedPerms)
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	config := createTestConfig()
	config.Auth.TokenExpiry = 1 // 1 second for testing
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Generate token
	tokenResp, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"viewer"})
	require.NoError(t, err)
	
	// Token should be valid immediately
	claims, err := authService.ValidateToken(tokenResp.AccessToken)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	
	// Wait for expiration
	time.Sleep(2 * time.Second)
	
	// Token should now be expired
	claims, err = authService.ValidateToken(tokenResp.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "expired")
}

func TestGetUserFromContext(t *testing.T) {
	// Test with valid user in context
	user := &AuthenticatedUser{
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin"},
	}
	
	ctx := context.WithValue(context.Background(), "user", user)
	
	retrievedUser, ok := GetUserFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, user, retrievedUser)
	
	// Test with no user in context
	emptyCtx := context.Background()
	retrievedUser, ok = GetUserFromContext(emptyCtx)
	assert.False(t, ok)
	assert.Nil(t, retrievedUser)
	
	// Test with wrong type in context
	wrongCtx := context.WithValue(context.Background(), "user", "not-a-user")
	retrievedUser, ok = GetUserFromContext(wrongCtx)
	assert.False(t, ok)
	assert.Nil(t, retrievedUser)
}

func TestCleanupExpiredTokens(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Add some test tokens to blacklist with different expiration times
	authService.tokenBlacklist["expired1"] = time.Now().Add(-1 * time.Hour)
	authService.tokenBlacklist["expired2"] = time.Now().Add(-2 * time.Hour)
	authService.tokenBlacklist["valid"] = time.Now().Add(1 * time.Hour)
	
	assert.Len(t, authService.tokenBlacklist, 3)
	
	// Run cleanup
	authService.CleanupExpiredTokens()
	
	// Only the valid token should remain
	assert.Len(t, authService.tokenBlacklist, 1)
	assert.Contains(t, authService.tokenBlacklist, "valid")
}

// Benchmark tests for performance validation
func BenchmarkGenerateToken(b *testing.B) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(b, err)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"admin"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateToken(b *testing.B) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(b, err)
	
	// Generate token for benchmarking
	tokenResp, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"admin"})
	require.NoError(b, err)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := authService.ValidateToken(tokenResp.AccessToken)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test A1 interface authentication latency requirements
func TestA1AuthLatencyCompliance(t *testing.T) {
	config := createTestConfig()
	logger := logrus.New()
	authService, err := NewAuthService(config, logger)
	require.NoError(t, err)
	
	// Test token generation latency
	start := time.Now()
	tokenResp, err := authService.GenerateToken("user123", "testuser", "test@example.com", []string{"admin"})
	genDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Less(t, genDuration, 100*time.Millisecond, "Token generation should be < 100ms, got %v", genDuration)
	
	// Test token validation latency
	start = time.Now()
	_, err = authService.ValidateToken(tokenResp.AccessToken)
	valDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Less(t, valDuration, 10*time.Millisecond, "Token validation should be < 10ms, got %v", valDuration)
	
	// Test permission check latency
	permissions := []string{"policy:read", "policy:write", "model:read"}
	start = time.Now()
	hasPermission := authService.HasPermission(permissions, "policy:write")
	permDuration := time.Since(start)
	
	assert.True(t, hasPermission)
	assert.Less(t, permDuration, 1*time.Millisecond, "Permission check should be < 1ms, got %v", permDuration)
	
	t.Logf("A1 Auth Performance - Generation: %v, Validation: %v, Permission: %v", 
		genDuration, valDuration, permDuration)
}