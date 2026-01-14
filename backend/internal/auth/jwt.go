package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
)

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	IssuedAt int64    `json:"iat"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        "maicivy",
	}
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(userID, email string, roles []string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessExpiry)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a new refresh token
func (m *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshExpiry)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken creates a new access token from a valid refresh token
func (m *JWTManager) RefreshAccessToken(refreshToken string, email string, roles []string) (string, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Generate new access token
	return m.GenerateAccessToken(claims.UserID, email, roles)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
// Expected format: "Bearer <token>"
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is empty")
	}

	// Check for Bearer prefix
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", errors.New("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is empty")
	}

	return token, nil
}

// GetClaims extracts claims from a valid token without full validation
// Use ValidateToken for security-critical operations
func (m *JWTManager) GetClaims(tokenString string) (*Claims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// IsExpired checks if a token is expired without full validation
func IsExpired(tokenString string) bool {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return true
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return true
	}

	return claims.ExpiresAt.Before(time.Now())
}

// HasRole checks if claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if claims have admin role
func (c *Claims) IsAdmin() bool {
	return c.HasRole("admin")
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"` // seconds
	TokenType    string    `json:"token_type"`
	IssuedAt     time.Time `json:"issued_at"`
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(userID, email string, roles []string) (*TokenPair, error) {
	accessToken, err := m.GenerateAccessToken(userID, email, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.accessExpiry.Seconds()),
		TokenType:    "Bearer",
		IssuedAt:     time.Now(),
	}, nil
}

// RevokeToken adds a token to blacklist (requires Redis)
// This should be implemented with Redis in a production environment
func (m *JWTManager) RevokeToken(tokenString string) error {
	// TODO: Implement with Redis
	// 1. Extract claims to get expiry
	// 2. Store token hash in Redis with TTL = remaining time until expiry
	// 3. Check blacklist before validating tokens
	return nil
}

// IsRevoked checks if a token has been revoked (requires Redis)
func (m *JWTManager) IsRevoked(tokenString string) bool {
	// TODO: Implement with Redis
	// Check if token hash exists in blacklist
	return false
}

// DefaultJWTManager creates a JWT manager with default settings
func DefaultJWTManager(secretKey string) *JWTManager {
	return NewJWTManager(
		secretKey,
		time.Hour,        // Access token: 1 hour
		24*7*time.Hour,   // Refresh token: 7 days
	)
}
