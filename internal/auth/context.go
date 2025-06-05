package auth

import (
	"context"
)

// AuthUser represents an authenticated user
type AuthUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userContextKey contextKey = "user"

// SetUserInContext stores a user in the request context
func SetUserInContext(ctx context.Context, user *AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext retrieves a user from the request context
func GetUserFromContext(ctx context.Context) (*AuthUser, bool) {
	user, ok := ctx.Value(userContextKey).(*AuthUser)
	return user, ok
}

// MustGetUserFromContext retrieves a user from context or panics
func MustGetUserFromContext(ctx context.Context) *AuthUser {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		panic("user not found in context")
	}
	return user
} 