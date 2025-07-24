package a1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

// AuthService handles JWT authentication for A1 interface
type AuthService struct {
	config            *config.A1Config
	logger            *logrus.Entry
	privateKey        *rsa.PrivateKey
	publicKey         *rsa.PublicKey
	refreshTokenStore map[string]string // Maps refresh token to user ID
	tokenBlacklist    map[string]time.Time
	roles             map[string]*Role
	permissions       map[string]*Permission
}

// Role represents a user role with associated permissions
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission represents a specific permission
type Permission struct {
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserClaims represents JWT claims for A1 users
type UserClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// AuthenticatedUser represents an authenticated user context
type AuthenticatedUser struct {
	UserID      string
	Username    string
	Email       string
	Roles       []string
	Permissions []string
	Token       string
	ExpiresAt   time.Time
}

// TokenRequest represents a token request
type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ClientID string `json:"client_id,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// NewAuthService creates a new authentication service
func NewAuthService(config *config.A1Config, logger *logrus.Logger) (*AuthService, error) {
	auth := &AuthService{
		config:            config,
		logger:            logger.WithField("component", "a1-auth"),
		refreshTokenStore: make(map[string]string),
		tokenBlacklist:    make(map[string]time.Time),
		roles:             make(map[string]*Role),
		permissions:       make(map[string]*Permission),
	}

	// Initialize RSA keys for JWT signing
	if err := auth.loadRSAKeys(); err != nil {
		return nil, fmt.Errorf("failed to load RSA keys: %w", err)
	}

	// Initialize default roles and permissions
	auth.initializeDefaultRBAC()

	auth.logger.Info("A1 authentication service initialized")
	return auth, nil
}

// loadRSAKeys loads RSA keys from environment variables or generates them for testing.
func (a *AuthService) loadRSAKeys() error {
	privateKey, err := a.loadPrivateKey()
	if err != nil {
		a.logger.WithError(err).Warn("Could not load private key from environment")
		// Fallback for testing or development environments
		a.logger.Info("Generating in-memory RSA key pair for development")
		privateKey, publicKey, genErr := generateTestKeys()
		if genErr != nil {
			return fmt.Errorf("failed to generate test keys: %w", genErr)
		}
		a.privateKey = privateKey
		a.publicKey = publicKey
		return nil
	}

	publicKey, err := a.loadPublicKey()
	if err != nil {
		return fmt.Errorf("could not load public key: %w", err)
	}

	a.privateKey = privateKey
	a.publicKey = publicKey
	a.logger.Info("Successfully loaded RSA keys from environment variables")
	return nil
}

// loadPrivateKey loads the RSA private key from an environment variable.
func (a *AuthService) loadPrivateKey() (*rsa.PrivateKey, error) {
	keyData := os.Getenv("JWT_PRIVATE_KEY")
	if keyData == "" {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY not configured")
	}
	return jwt.ParseRSAPrivateKeyFromPEM([]byte(keyData))
}

// loadPublicKey loads the RSA public key from an environment variable.
func (a *AuthService) loadPublicKey() (*rsa.PublicKey, error) {
	keyData := os.Getenv("JWT_PUBLIC_KEY")
	if keyData == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY not configured")
	}
	return jwt.ParseRSAPublicKeyFromPEM([]byte(keyData))
}

// initializeDefaultRBAC sets up default roles and permissions
func (a *AuthService) initializeDefaultRBAC() {
	// Define permissions
	permissions := []*Permission{
		{Name: "policy:read", Resource: "policy", Action: "read", Description: "Read policies"},
		{Name: "policy:write", Resource: "policy", Action: "write", Description: "Create/update policies"},
		{Name: "policy:delete", Resource: "policy", Action: "delete", Description: "Delete policies"},
		{Name: "policytype:read", Resource: "policytype", Action: "read", Description: "Read policy types"},
		{Name: "policytype:write", Resource: "policytype", Action: "write", Description: "Create/update policy types"},
		{Name: "policytype:delete", Resource: "policytype", Action: "delete", Description: "Delete policy types"},
		{Name: "enrichment:read", Resource: "enrichment", Action: "read", Description: "Read enrichment info"},
		{Name: "enrichment:write", Resource: "enrichment", Action: "write", Description: "Create/update enrichment info"},
		{Name: "model:read", Resource: "model", Action: "read", Description: "Read ML models"},
		{Name: "model:write", Resource: "model", Action: "write", Description: "Deploy ML models"},
		{Name: "admin:read", Resource: "admin", Action: "read", Description: "Read admin resources"},
		{Name: "admin:write", Resource: "admin", Action: "write", Description: "Manage admin resources"},
	}

	for _, perm := range permissions {
		perm.CreatedAt = time.Now()
		a.permissions[perm.Name] = perm
	}

	// Define roles
	roles := []*Role{
		{
			Name:        "viewer",
			Description: "Read-only access to policies and types",
			Permissions: []string{"policy:read", "policytype:read", "enrichment:read", "model:read"},
		},
		{
			Name:        "operator",
			Description: "Can manage policies but not types",
			Permissions: []string{"policy:read", "policy:write", "policy:delete", "policytype:read", "enrichment:read", "enrichment:write", "model:read"},
		},
		{
			Name:        "admin",
			Description: "Full access to all resources",
			Permissions: []string{
				"policy:read", "policy:write", "policy:delete",
				"policytype:read", "policytype:write", "policytype:delete",
				"enrichment:read", "enrichment:write",
				"model:read", "model:write",
				"admin:read", "admin:write",
			},
		},
	}

	for _, role := range roles {
		role.CreatedAt = time.Now()
		a.roles[role.Name] = role
	}

	a.logger.WithFields(logrus.Fields{
		"permissions": len(a.permissions),
		"roles":       len(a.roles),
	}).Info("Default RBAC configuration initialized")
}

// GenerateToken generates a JWT token for an authenticated user
func (a *AuthService) GenerateToken(userID, username, email string, roles []string) (*TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(a.config.Auth.TokenExpiry) * time.Second)

	// Resolve permissions from roles
	permissions := a.resolvePermissions(roles)

	// Create claims
	claims := UserClaims{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.config.Auth.Issuer,
			Subject:   userID,
			Audience:  []string{a.config.Auth.Audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("%s-%d", userID, now.Unix()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign token
	tokenString, err := token.SignedString(a.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate and store refresh token
	refreshToken := fmt.Sprintf("refresh-%s-%d", userID, now.UnixNano())
	a.refreshTokenStore[refreshToken] = userID

	response := &TokenResponse{
		AccessToken:  tokenString,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(a.config.Auth.TokenExpiry),
		Scope:        strings.Join(permissions, " "),
	}

	a.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"username":    username,
		"roles":       roles,
		"permissions": len(permissions),
		"expires_at":  expiresAt,
	}).Info("JWT token generated")

	return response, nil
}

// RefreshToken validates a refresh token and issues a new token pair
func (a *AuthService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	userID, ok := a.refreshTokenStore[refreshToken]
	if !ok {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// In a real implementation, you would look up the user by userID
	// For this example, we'll just re-use the userID
	// In a real implementation, you would also check if the user is still valid/active
	// and has the same roles/permissions.

	// Invalidate the old refresh token
	delete(a.refreshTokenStore, refreshToken)

	// Generate a new token pair
	// In a real implementation, you would fetch the user's details from a database
	// For this example, we'll use placeholder values for username and email.
	return a.GenerateToken(userID, "refreshed_user", "refreshed_user@example.com", []string{"operator"})
}

// ValidateToken validates a JWT token and returns user claims
func (a *AuthService) ValidateToken(tokenString string) (*UserClaims, error) {
	// Check if token is blacklisted
	if _, blacklisted := a.tokenBlacklist[tokenString]; blacklisted {
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
		return a.publicKey, nil
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
	if claims.Issuer != a.config.Auth.Issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token is expired")
	}

	return claims, nil
}

// RevokeToken adds a token to the blacklist
func (a *AuthService) RevokeToken(tokenString string) error {
	// Parse token to get expiration
	claims, err := a.ValidateToken(tokenString)
	if err != nil {
		// Token might already be invalid, but still blacklist it
		a.tokenBlacklist[tokenString] = time.Now().Add(24 * time.Hour)
		return nil
	}

	// Add to blacklist until expiration
	if claims.ExpiresAt != nil {
		a.tokenBlacklist[tokenString] = claims.ExpiresAt.Time
	} else {
		a.tokenBlacklist[tokenString] = time.Now().Add(24 * time.Hour)
	}

	a.logger.WithFields(logrus.Fields{
		"user_id":  claims.UserID,
		"username": claims.Username,
	}).Info("JWT token revoked")

	return nil
}

// resolvePermissions resolves permissions from a list of roles
func (a *AuthService) resolvePermissions(roles []string) []string {
	permissionSet := make(map[string]bool)

	for _, roleName := range roles {
		if role, exists := a.roles[roleName]; exists {
			for _, permission := range role.Permissions {
				permissionSet[permission] = true
			}
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}

// HasPermission checks if a user has a specific permission
func (a *AuthService) HasPermission(userPermissions []string, requiredPermission string) bool {
	for _, permission := range userPermissions {
		if permission == requiredPermission {
			return true
		}
	}
	return false
}

// AuthMiddleware returns an HTTP middleware for JWT authentication
func (a *AuthService) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				a.writeAuthError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			// Check Bearer prefix
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				a.writeAuthError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := authHeader[len(bearerPrefix):]

			// Validate token
			claims, err := a.ValidateToken(tokenString)
			if err != nil {
				a.logger.WithError(err).Warn("Token validation failed")
				a.writeAuthError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Create authenticated user context
			user := &AuthenticatedUser{
				UserID:      claims.UserID,
				Username:    claims.Username,
				Email:       claims.Email,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
				Token:       tokenString,
				ExpiresAt:   claims.ExpiresAt.Time,
			}

			// Add user to request context
			ctx := context.WithValue(r.Context(), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns middleware that checks for a specific permission
func (a *AuthService) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value("user").(*AuthenticatedUser)
			if !ok {
				a.writeAuthError(w, http.StatusUnauthorized, "User not authenticated")
				return
			}

			if !a.HasPermission(user.Permissions, permission) {
				a.logger.WithFields(logrus.Fields{
					"user_id":            user.UserID,
					"required_permission": permission,
					"user_permissions":   user.Permissions,
				}).Warn("Access denied - insufficient permissions")

				a.writeAuthError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeAuthError writes an authentication error response
func (a *AuthService) writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := A1ErrorResponse{
		Type:   "https://tools.ietf.org/html/rfc7235#section-3.1",
		Title:  "Authentication Error",
		Status: statusCode,
		Detail: message,
	}

	// In a real implementation, you'd use a JSON encoder
	fmt.Fprintf(w, `{"type":"%s","title":"%s","status":%d,"detail":"%s"}`,
		errorResp.Type, errorResp.Title, errorResp.Status, errorResp.Detail)
}

// GetUserFromContext extracts the authenticated user from request context
func GetUserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value("user").(*AuthenticatedUser)
	return user, ok
}

// CleanupExpiredTokens removes expired tokens from blacklist
func (a *AuthService) CleanupExpiredTokens() {
	now := time.Now()
	count := 0

	for token, expiry := range a.tokenBlacklist {
		if now.After(expiry) {
			delete(a.tokenBlacklist, token)
			count++
		}
	}

	if count > 0 {
		a.logger.WithField("cleaned_tokens", count).Debug("Cleaned up expired blacklisted tokens")
	}
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
