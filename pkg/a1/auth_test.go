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

	t.Run("ValidToken", func(t *testing.T) {
		claims := &Claims{
			Roles: []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, err := token.SignedString(privateKey)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rr := httptest.NewRecorder()

		authMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		authMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("NoAuthHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		authMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthorization(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("AdminAccess", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), "roles", []string{"admin"})
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		Authorize(testHandler, AdminRole).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("OperatorAccessDenied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), "roles", []string{"operator"})
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		Authorize(testHandler, AdminRole).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
