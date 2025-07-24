package a1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Create a temporary public key file
	tmpfile, err := os.CreateTemp("", "test_public_key.pem")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(publicKeyPEM)
	require.NoError(t, err)
	tmpfile.Close()

	authConfig := &config.AuthConfig{
		PublicKeyPath: tmpfile.Name(),
	}

	authMiddleware, err := NewAuthMiddleware(authConfig)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name: "ValidToken",
			token: func() string {
				claims := &Claims{
					Roles: []string{"admin"},
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "InvalidToken",
			token: "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "NoAuthHeader",
			token: "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "ExpiredToken",
			token: func() string {
				claims := &Claims{
					Roles: []string{"admin"},
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			}(),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rr := httptest.NewRecorder()

			authMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestAuthorization(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		roles          []string
		requiredRole   Role
		expectedStatus int
	}{
		{
			name: "AdminAccess",
			roles: []string{"admin"},
			requiredRole: AdminRole,
			expectedStatus: http.StatusOK,
		},
		{
			name: "OperatorAccessDenied",
			roles: []string{"operator"},
			requiredRole: AdminRole,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "NoRoles",
			roles: []string{},
			requiredRole: AdminRole,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ViewerAccessToViewer",
			roles: []string{"viewer"},
			requiredRole: ViewerRole,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			ctx := context.WithValue(req.Context(), "roles", tt.roles)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			Authorize(testHandler, tt.requiredRole).ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}