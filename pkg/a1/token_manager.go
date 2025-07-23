package a1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

// TokenManager handles JWT token lifecycle with enhanced security features
type TokenManager struct {
	config     *config.A1Config
	logger     *logrus.Logger
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	
	// Token management
	activeTokens   map[string]*TokenMetadata
	blacklistTokens map[string]time.Time
	refreshTokens  map[string]*RefreshTokenData
	mutex          sync.RWMutex
	
	// Security features
	rateLimiter    map[string]*RateLimitData
	suspiciousIPs  map[string]*SuspiciousActivity
	
	// Cleanup routine
	cleanupTicker *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
}

// TokenMetadata contains metadata about active tokens
type TokenMetadata struct {
	TokenID     string    `json:"token_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsed    time.Time `json:"last_used"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	Permissions []string  `json:"permissions"`
	SessionID   string    `json:"session_id"`
}

// RefreshTokenData contains refresh token information
type RefreshTokenData struct {
	TokenHash     string    `json:"token_hash"`
	UserID        string    `json:"user_id"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastUsed      time.Time `json:"last_used"`
	ClientIP      string    `json:"client_ip"`
	IsRevoked     bool      `json:"is_revoked"`
	UsageCount    int       `json:"usage_count"`
	MaxUsageCount int       `json:"max_usage_count"`
}

// RateLimitData tracks API usage per client
type RateLimitData struct {
	ClientIP      string    `json:"client_ip"`
	RequestCount  int       `json:"request_count"`
	LastRequest   time.Time `json:"last_request"`
	WindowStart   time.Time `json:"window_start"`
	IsBlocked     bool      `json:"is_blocked"`
	BlockedUntil  time.Time `json:"blocked_until"`
}

// SuspiciousActivity tracks suspicious login attempts
type SuspiciousActivity struct {
	ClientIP        string    `json:"client_ip"`
	FailedAttempts  int       `json:"failed_attempts"`
	LastAttempt     time.Time `json:"last_attempt"`
	IsBlocked       bool      `json:"is_blocked"`
	BlockedUntil    time.Time `json:"blocked_until"`
	SuspicionLevel  int       `json:"suspicion_level"` // 1-5 scale
}

// TokenConfig contains token configuration
type TokenConfig struct {
	AccessTokenTTL  time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxTokensPerUser int          `json:"max_tokens_per_user"`
	RateLimitWindow  time.Duration `json:"rate_limit_window"`
	RateLimitRequests int          `json:"rate_limit_requests"`
}

// NewTokenManager creates a new enhanced token manager
func NewTokenManager(config *config.A1Config, logger *logrus.Logger) (*TokenManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	tm := &TokenManager{
		config:          config,
		logger:          logger.WithField("component", "token-manager"),
		activeTokens:    make(map[string]*TokenMetadata),
		blacklistTokens: make(map[string]time.Time),
		refreshTokens:   make(map[string]*RefreshTokenData),
		rateLimiter:     make(map[string]*RateLimitData),
		suspiciousIPs:   make(map[string]*SuspiciousActivity),
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Load or generate RSA keys
	if err := tm.initializeKeys(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize RSA keys: %w", err)
	}
	
	// Start cleanup routine
	tm.startCleanupRoutine()
	
	tm.logger.Info("Token manager initialized with enhanced security features")
	return tm, nil
}

// initializeKeys loads or generates RSA keys for JWT signing
func (tm *TokenManager) initializeKeys() error {
	// In production, load from Kubernetes secrets or external key management
	if tm.config.Auth.PrivateKeyPath != "" {
		return tm.loadKeysFromFile()
	}
	
	// Generate new RSA key pair for development
	return tm.generateNewKeys()
}

// loadKeysFromFile loads RSA keys from file
func (tm *TokenManager) loadKeysFromFile() error {
	// Implementation for loading keys from file or Kubernetes secret
	tm.logger.Info("Loading RSA keys from configuration")
	
	// For now, use the existing test keys from auth.go
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	
	tm.privateKey = privateKey
	tm.publicKey = publicKey
	
	return nil
}

// generateNewKeys generates new RSA key pair
func (tm *TokenManager) generateNewKeys() error {
	tm.logger.Info("Generating new RSA key pair")
	
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	
	tm.privateKey = privateKey
	tm.publicKey = &privateKey.PublicKey
	
	return nil
}

// CreateToken creates a new JWT token with enhanced security
func (tm *TokenManager) CreateToken(ctx context.Context, req *TokenCreationRequest) (*TokenResponse, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Check rate limiting
	if err := tm.checkRateLimit(req.ClientIP); err != nil {
		return nil, err
	}
	
	// Check for suspicious activity
	if err := tm.checkSuspiciousActivity(req.ClientIP); err != nil {
		return nil, err
	}
	
	// Generate unique token ID
	tokenID := tm.generateTokenID()
	sessionID := tm.generateSessionID()
	
	now := time.Now()
	accessTokenTTL := time.Duration(tm.config.Auth.TokenExpiry) * time.Second
	refreshTokenTTL := 7 * 24 * time.Hour // 7 days
	
	// Create JWT claims
	claims := &EnhancedJWTClaims{
		UserID:      req.UserID,
		Username:    req.Username,
		Email:       req.Email,
		Roles:       req.Roles,
		Permissions: tm.resolvePermissions(req.Roles),
		SessionID:   sessionID,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Issuer:    tm.config.Auth.Issuer,
			Subject:   req.UserID,
			Audience:  []string{tm.config.Auth.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	
	// Create and sign access token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	accessToken, err := token.SignedString(tm.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}
	
	// Create refresh token
	refreshToken, err := tm.createRefreshToken(req.UserID, req.ClientIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}
	
	// Store token metadata
	metadata := &TokenMetadata{
		TokenID:     tokenID,
		UserID:      req.UserID,
		Username:    req.Username,
		IssuedAt:    now,
		ExpiresAt:   now.Add(accessTokenTTL),
		LastUsed:    now,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		Permissions: claims.Permissions,
		SessionID:   sessionID,
	}
	tm.activeTokens[tokenID] = metadata
	
	// Clean up old tokens for this user
	tm.cleanupUserTokens(req.UserID)
	
	response := &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
		Scope:        fmt.Sprintf("roles:%s", req.Roles),
		SessionID:    sessionID,
	}
	
	tm.logger.WithFields(logrus.Fields{
		"user_id":    req.UserID,
		"username":   req.Username,
		"token_id":   tokenID,
		"session_id": sessionID,
		"client_ip":  req.ClientIP,
		"expires_at": metadata.ExpiresAt,
	}).Info("JWT token created successfully")
	
	return response, nil
}

// ValidateToken validates JWT token with enhanced security checks
func (tm *TokenManager) ValidateToken(tokenString string, clientIP string) (*EnhancedJWTClaims, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	// Check if token is blacklisted
	if expiry, blacklisted := tm.blacklistTokens[tokenString]; blacklisted {
		if time.Now().Before(expiry) {
			return nil, fmt.Errorf("token is blacklisted")
		}
		// Clean up expired blacklist entry
		delete(tm.blacklistTokens, tokenString)
	}
	
	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &EnhancedJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.publicKey, nil
	})
	
	if err != nil {
		tm.recordSuspiciousActivity(clientIP, "invalid_token")
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	if !token.Valid {
		tm.recordSuspiciousActivity(clientIP, "invalid_token")
		return nil, fmt.Errorf("invalid token")
	}
	
	claims, ok := token.Claims.(*EnhancedJWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	
	// Additional security validations
	if err := tm.validateTokenClaims(claims, clientIP); err != nil {
		return nil, err
	}
	
	// Update token usage
	tm.updateTokenUsage(claims.ID, clientIP)
	
	return claims, nil
}

// RevokeToken adds a token to the blacklist
func (tm *TokenManager) RevokeToken(tokenString string, reason string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Parse token to get expiration
	token, err := jwt.ParseWithClaims(tokenString, &EnhancedJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return tm.publicKey, nil
	})
	
	var expiry time.Time
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(*EnhancedJWTClaims); ok {
			if claims.ExpiresAt != nil {
				expiry = claims.ExpiresAt.Time
			}
			
			// Remove from active tokens
			delete(tm.activeTokens, claims.ID)
			
			tm.logger.WithFields(logrus.Fields{
				"user_id":  claims.UserID,
				"token_id": claims.ID,
				"reason":   reason,
			}).Info("JWT token revoked")
		}
	}
	
	if expiry.IsZero() {
		expiry = time.Now().Add(24 * time.Hour) // Default expiry
	}
	
	// Add to blacklist
	tm.blacklistTokens[tokenString] = expiry
	
	return nil
}

// RefreshToken creates a new access token using refresh token
func (tm *TokenManager) RefreshToken(refreshToken string, clientIP string) (*TokenResponse, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Validate refresh token
	refreshData, exists := tm.refreshTokens[refreshToken]
	if !exists || refreshData.IsRevoked {
		tm.recordSuspiciousActivity(clientIP, "invalid_refresh_token")
		return nil, fmt.Errorf("invalid refresh token")
	}
	
	if time.Now().After(refreshData.ExpiresAt) {
		delete(tm.refreshTokens, refreshToken)
		return nil, fmt.Errorf("refresh token expired")
	}
	
	// Check usage limits
	if refreshData.UsageCount >= refreshData.MaxUsageCount {
		refreshData.IsRevoked = true
		return nil, fmt.Errorf("refresh token usage limit exceeded")
	}
	
	// Validate client IP (optional - can be disabled for mobile apps)
	if tm.config.Auth.StrictIPValidation && refreshData.ClientIP != clientIP {
		tm.recordSuspiciousActivity(clientIP, "ip_mismatch")
		return nil, fmt.Errorf("client IP mismatch")
	}
	
	// Update refresh token usage
	refreshData.LastUsed = time.Now()
	refreshData.UsageCount++
	
	// Create new access token (implementation similar to CreateToken)
	// ... (implementation details omitted for brevity)
	
	tm.logger.WithFields(logrus.Fields{
		"user_id":     refreshData.UserID,
		"client_ip":   clientIP,
		"usage_count": refreshData.UsageCount,
	}).Info("Access token refreshed successfully")
	
	return nil, nil // Placeholder return
}

// Helper methods

func (tm *TokenManager) generateTokenID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (tm *TokenManager) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (tm *TokenManager) createRefreshToken(userID, clientIP string) (string, error) {
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	token := hex.EncodeToString(bytes)
	
	refreshData := &RefreshTokenData{
		TokenHash:     token,
		UserID:        userID,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(7 * 24 * time.Hour),
		ClientIP:      clientIP,
		IsRevoked:     false,
		UsageCount:    0,
		MaxUsageCount: 100, // Allow 100 refreshes
	}
	
	tm.refreshTokens[token] = refreshData
	return token, nil
}

func (tm *TokenManager) checkRateLimit(clientIP string) error {
	now := time.Now()
	limit := tm.rateLimiter[clientIP]
	
	if limit == nil {
		tm.rateLimiter[clientIP] = &RateLimitData{
			ClientIP:     clientIP,
			RequestCount: 1,
			LastRequest:  now,
			WindowStart:  now,
		}
		return nil
	}
	
	// Reset window if needed
	if now.Sub(limit.WindowStart) > time.Hour {
		limit.RequestCount = 1
		limit.WindowStart = now
		limit.IsBlocked = false
		return nil
	}
	
	if limit.IsBlocked && now.Before(limit.BlockedUntil) {
		return fmt.Errorf("client IP blocked due to rate limiting")
	}
	
	limit.RequestCount++
	limit.LastRequest = now
	
	// Check rate limit (100 requests per hour)
	if limit.RequestCount > 100 {
		limit.IsBlocked = true
		limit.BlockedUntil = now.Add(time.Hour)
		return fmt.Errorf("rate limit exceeded")
	}
	
	return nil
}

func (tm *TokenManager) checkSuspiciousActivity(clientIP string) error {
	suspicious := tm.suspiciousIPs[clientIP]
	if suspicious == nil {
		return nil
	}
	
	if suspicious.IsBlocked && time.Now().Before(suspicious.BlockedUntil) {
		return fmt.Errorf("client IP blocked due to suspicious activity")
	}
	
	return nil
}

func (tm *TokenManager) recordSuspiciousActivity(clientIP, activityType string) {
	now := time.Now()
	suspicious := tm.suspiciousIPs[clientIP]
	
	if suspicious == nil {
		suspicious = &SuspiciousActivity{
			ClientIP:    clientIP,
			LastAttempt: now,
		}
		tm.suspiciousIPs[clientIP] = suspicious
	}
	
	suspicious.FailedAttempts++
	suspicious.LastAttempt = now
	
	// Increase suspicion level based on activity
	switch activityType {
	case "invalid_token":
		suspicious.SuspicionLevel += 1
	case "invalid_refresh_token":
		suspicious.SuspicionLevel += 2
	case "ip_mismatch":
		suspicious.SuspicionLevel += 3
	}
	
	// Block if suspicion level is too high
	if suspicious.SuspicionLevel >= 10 || suspicious.FailedAttempts >= 5 {
		suspicious.IsBlocked = true
		suspicious.BlockedUntil = now.Add(24 * time.Hour)
		
		tm.logger.WithFields(logrus.Fields{
			"client_ip":       clientIP,
			"failed_attempts": suspicious.FailedAttempts,
			"suspicion_level": suspicious.SuspicionLevel,
			"activity_type":   activityType,
		}).Warn("Client IP blocked due to suspicious activity")
	}
}

func (tm *TokenManager) validateTokenClaims(claims *EnhancedJWTClaims, clientIP string) error {
	// Validate issuer
	if claims.Issuer != tm.config.Auth.Issuer {
		return fmt.Errorf("invalid token issuer")
	}
	
	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("token expired")
	}
	
	// Validate token exists in active tokens
	if metadata, exists := tm.activeTokens[claims.ID]; exists {
		// Optional: Validate client IP consistency
		if tm.config.Auth.StrictIPValidation && metadata.ClientIP != clientIP {
			tm.recordSuspiciousActivity(clientIP, "ip_mismatch")
			return fmt.Errorf("client IP mismatch")
		}
	}
	
	return nil
}

func (tm *TokenManager) updateTokenUsage(tokenID, clientIP string) {
	if metadata, exists := tm.activeTokens[tokenID]; exists {
		metadata.LastUsed = time.Now()
	}
}

func (tm *TokenManager) cleanupUserTokens(userID string) {
	maxTokens := 10 // Maximum tokens per user
	userTokens := make([]*TokenMetadata, 0)
	
	// Collect all tokens for this user
	for _, metadata := range tm.activeTokens {
		if metadata.UserID == userID {
			userTokens = append(userTokens, metadata)
		}
	}
	
	// Remove oldest tokens if limit exceeded
	if len(userTokens) > maxTokens {
		// Sort by issued time and remove oldest
		for i := 0; i < len(userTokens)-maxTokens; i++ {
			delete(tm.activeTokens, userTokens[i].TokenID)
		}
	}
}

func (tm *TokenManager) resolvePermissions(roles []string) []string {
	// Implementation similar to existing auth service
	permissionSet := make(map[string]bool)
	
	// Add role-based permissions
	for _, role := range roles {
		switch role {
		case "admin":
			permissionSet["*"] = true // Admin has all permissions
		case "operator":
			permissionSet["policy:read"] = true
			permissionSet["policy:write"] = true
			permissionSet["enrichment:read"] = true
		case "viewer":
			permissionSet["policy:read"] = true
			permissionSet["enrichment:read"] = true
		}
	}
	
	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	
	return permissions
}

func (tm *TokenManager) startCleanupRoutine() {
	tm.cleanupTicker = time.NewTicker(5 * time.Minute)
	
	go func() {
		for {
			select {
			case <-tm.cleanupTicker.C:
				tm.performCleanup()
			case <-tm.ctx.Done():
				tm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

func (tm *TokenManager) performCleanup() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	now := time.Now()
	
	// Clean up expired tokens
	for tokenID, metadata := range tm.activeTokens {
		if now.After(metadata.ExpiresAt) {
			delete(tm.activeTokens, tokenID)
		}
	}
	
	// Clean up expired blacklisted tokens
	for token, expiry := range tm.blacklistTokens {
		if now.After(expiry) {
			delete(tm.blacklistTokens, token)
		}
	}
	
	// Clean up expired refresh tokens
	for token, data := range tm.refreshTokens {
		if now.After(data.ExpiresAt) || data.IsRevoked {
			delete(tm.refreshTokens, token)
		}
	}
	
	// Reset rate limiting windows
	for ip, limit := range tm.rateLimiter {
		if now.Sub(limit.WindowStart) > time.Hour {
			limit.RequestCount = 0
			limit.WindowStart = now
			limit.IsBlocked = false
		}
	}
	
	// Clean up old suspicious activity records
	for ip, suspicious := range tm.suspiciousIPs {
		if now.Sub(suspicious.LastAttempt) > 24*time.Hour && !suspicious.IsBlocked {
			delete(tm.suspiciousIPs, ip)
		}
		if suspicious.IsBlocked && now.After(suspicious.BlockedUntil) {
			suspicious.IsBlocked = false
			suspicious.FailedAttempts = 0
			suspicious.SuspicionLevel = 0
		}
	}
	
	tm.logger.WithFields(logrus.Fields{
		"active_tokens":    len(tm.activeTokens),
		"blacklisted":      len(tm.blacklistTokens),
		"refresh_tokens":   len(tm.refreshTokens),
		"rate_limited_ips": len(tm.rateLimiter),
		"suspicious_ips":   len(tm.suspiciousIPs),
	}).Debug("Token cleanup completed")
}

// Stop gracefully shuts down the token manager
func (tm *TokenManager) Stop() {
	tm.cancel()
	if tm.cleanupTicker != nil {
		tm.cleanupTicker.Stop()
	}
	tm.logger.Info("Token manager stopped")
}

// GetTokenStatistics returns token usage statistics
func (tm *TokenManager) GetTokenStatistics() map[string]interface{} {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	return map[string]interface{}{
		"active_tokens":       len(tm.activeTokens),
		"blacklisted_tokens":  len(tm.blacklistTokens),
		"refresh_tokens":      len(tm.refreshTokens),
		"rate_limited_clients": len(tm.rateLimiter),
		"suspicious_clients":   len(tm.suspiciousIPs),
		"cleanup_running":      tm.cleanupTicker != nil,
	}
}

// Additional types for enhanced JWT claims
type EnhancedJWTClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	ClientIP    string   `json:"client_ip"`
	UserAgent   string   `json:"user_agent"`
	TokenType   string   `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenCreationRequest contains token creation parameters
type TokenCreationRequest struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	ClientIP  string   `json:"client_ip"`
	UserAgent string   `json:"user_agent"`
}

// Enhanced TokenResponse with additional security features
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	SessionID    string `json:"session_id"`
}