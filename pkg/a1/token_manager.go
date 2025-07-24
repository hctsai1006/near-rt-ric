package a1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenManager handles JWT token generation, validation, and revocation
type TokenManager struct {
	config            *config.A1Config
	logger            *logrus.Entry
	privateKey        *rsa.PrivateKey
	publicKey         *rsa.PublicKey
	refreshTokenStore map[string]string // Maps refresh token to user ID
	tokenBlacklist    map[string]time.Time
	mutex             sync.RWMutex
}

// NewTokenManager creates a new TokenManager
func NewTokenManager(config *config.A1Config, logger *logrus.Logger) (*TokenManager, error) {
	tm := &TokenManager{
		config:         config,
		logger:         logger.WithField("component", "token-manager"),
		tokenBlacklist: make(map[string]time.Time),
	}

	// Initialize RSA keys for JWT signing
	if err := tm.loadRSAKeys(); err != nil {
		return nil, fmt.Errorf("failed to load RSA keys: %w", err)
	}

	tm.logger.Info("Token manager initialized")
	return tm, nil
}

// loadRSAKeys loads RSA keys from environment variables or generates them for testing.
func (tm *TokenManager) loadRSAKeys() error {
	privateKey, err := tm.loadPrivateKey()
	if err != nil {
		tm.logger.WithError(err).Warn("Could not load private key from environment")
		// Fallback for testing or development environments
		tm.logger.Info("Generating in-memory RSA key pair for development")
		privateKey, publicKey, genErr := generateTestKeys()
		if genErr != nil {
			return fmt.Errorf("failed to generate test keys: %w", genErr)
		}
		tm.privateKey = privateKey
		tm.publicKey = publicKey
		return nil
	}

	publicKey, err := tm.loadPublicKey()
	if err != nil {
		return fmt.Errorf("could not load public key: %w", err)
	}

	tm.privateKey = privateKey
	tm.publicKey = publicKey
	tm.logger.Info("Successfully loaded RSA keys from environment variables")
	return nil
}

// loadPrivateKey loads the RSA private key from an environment variable.
func (tm *TokenManager) loadPrivateKey() (*rsa.PrivateKey, error) {
	keyData := os.Getenv("JWT_PRIVATE_KEY")
	if keyData == "" {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY not configured")
	}
	return jwt.ParseRSAPrivateKeyFromPEM([]byte(keyData))
}

// loadPublicKey loads the RSA public key from an environment variable.
func (tm *TokenManager) loadPublicKey() (*rsa.PublicKey, error) {
	keyData := os.Getenv("JWT_PUBLIC_KEY")
	if keyData == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY not configured")
	}
	return jwt.ParseRSAPublicKeyFromPEM([]byte(keyData))
}

// generateTestKeys generates a new RSA key pair for testing purposes.
// This should not be used in production.
func generateTestKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// GenerateToken generates a JWT token for an authenticated user
func (tm *TokenManager) GenerateToken(userID, username, email string, roles []string) (*TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(tm.config.Auth.TokenExpiry) * time.Second)

	// Create claims
	claims := UserClaims{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.config.Auth.Issuer,
			Subject:   userID,
			Audience:  []string{tm.config.Auth.Audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("%s-%d", userID, now.Unix()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign token
	tokenString, err := token.SignedString(tm.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate and store refresh token
	refreshToken := fmt.Sprintf("refresh-%s-%d", userID, now.UnixNano())
	tm.refreshTokenStore[refreshToken] = userID

	response := &TokenResponse{
		AccessToken:  tokenString,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tm.config.Auth.TokenExpiry),
		Scope:        strings.Join(roles, " "), // Simplified scope for now
	}

	tm.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"username":    username,
		"roles":       roles,
		"expires_at":  expiresAt,
	}).Info("JWT token generated")

	return response, nil
}

// RefreshToken validates a refresh token and issues a new token pair
func (tm *TokenManager) RefreshToken(refreshToken string) (*TokenResponse, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	userID, ok := tm.refreshTokenStore[refreshToken]
	if !ok {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Invalidate the old refresh token
	delete(tm.refreshTokenStore, refreshToken)

	// Generate a new token pair
	// In a real implementation, you would fetch the user's details from a database
	// For this example, we'll use placeholder values for username and email.
	return tm.GenerateToken(userID, "refreshed_user", "refreshed_user@example.com", []string{"operator"})
}

// ValidateToken validates a JWT token and returns user claims
func (tm *TokenManager) ValidateToken(tokenString string) (*UserClaims, error) {
	// Check if token is blacklisted
	if _, blacklisted := tm.tokenBlacklist[tokenString]; blacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			// This check is critical to prevent the "alg:none" vulnerability
			if token.Header["alg"] == "none" {
				return nil, fmt.Errorf("unsupported signing method: none")
			}
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.publicKey, nil
	}, jwt.WithStrictDecoding()) // Ensure strict decoding

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Additional validation
	if claims.Issuer != tm.config.Auth.Issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token is expired")
	}

	return claims, nil
}

// RevokeToken adds a token to the blacklist
func (tm *TokenManager) RevokeToken(tokenString string) error {
	// Parse token to get expiration
	claims, err := tm.ValidateToken(tokenString)
	if err != nil {
		// Token might already be invalid, but still blacklist it
		tm.tokenBlacklist[tokenString] = time.Now().Add(24 * time.Hour)
		return nil
	}

	// Add to blacklist until expiration
	if claims.ExpiresAt != nil {
		tm.tokenBlacklist[tokenString] = claims.ExpiresAt.Time
	} else {
		tm.tokenBlacklist[tokenString] = time.Now().Add(24 * time.Hour)
	}

	tm.logger.WithFields(logrus.Fields{
		"user_id":  claims.UserID,
		"username": claims.Username,
	}).Info("JWT token revoked")

	return nil
}

// CleanupExpiredTokens removes expired tokens from blacklist
func (tm *TokenManager) CleanupExpiredTokens() {
	now := time.Now()
	count := 0

	for token, expiry := range tm.tokenBlacklist {
		if now.After(expiry) {
			delete(tm.tokenBlacklist, token)
			count++
		}
	}

	if count > 0 {
		tm.logger.WithField("cleaned_tokens", count).Debug("Cleaned up expired blacklisted tokens")
	}
}
