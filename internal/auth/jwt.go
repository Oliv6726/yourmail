package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTService handles JWT operations
type JWTService struct {
	secretKey []byte
	issuer    string
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey, issuer string) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// GenerateToken generates a JWT token for a user
func (j *JWTService) GenerateToken(userID int, username, email string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   strconv.Itoa(userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func (j *JWTService) ExtractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}

	// Check for Bearer token format
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("authorization header format must be 'Bearer {token}'")
	}

	return parts[1], nil
}

// AuthMiddleware creates a middleware for JWT authentication
func (j *JWTService) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from header
		tokenString, err := j.ExtractTokenFromHeader(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := j.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		ctx := r.Context()
		ctx = SetUserInContext(ctx, &AuthUser{
			ID:       claims.UserID,
			Username: claims.Username,
			Email:    claims.Email,
		})
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	}
}

// OptionalAuthMiddleware creates a middleware that extracts user info if present but doesn't require auth
func (j *JWTService) OptionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to extract token from header
		tokenString, err := j.ExtractTokenFromHeader(r)
		if err == nil {
			// Validate token if present
			claims, err := j.ValidateToken(tokenString)
			if err == nil {
				// Add user info to request context
				ctx := r.Context()
				ctx = SetUserInContext(ctx, &AuthUser{
					ID:       claims.UserID,
					Username: claims.Username,
					Email:    claims.Email,
				})
				r = r.WithContext(ctx)
			}
		}

		// Call next handler regardless of auth success
		next.ServeHTTP(w, r)
	}
} 