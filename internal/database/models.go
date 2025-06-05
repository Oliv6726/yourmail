package database

import (
	"time"
)

// User represents a user in the database
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never include in JSON
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Message represents a message in the database
type Message struct {
	ID          int       `json:"id" db:"id"`
	FromUserID  *int      `json:"from_user_id" db:"from_user_id"`
	ToUserID    *int      `json:"to_user_id" db:"to_user_id"`
	FromAddress string    `json:"from" db:"from_address"`
	ToAddress   string    `json:"to" db:"to_address"`
	Subject     string    `json:"subject" db:"subject"`
	Body        string    `json:"body" db:"body"`
	ReadStatus  bool      `json:"read" db:"read_status"`
	CreatedAt   time.Time `json:"timestamp" db:"created_at"`
	
	// Virtual fields populated by joins
	FromUser *User `json:"from_user,omitempty"`
	ToUser   *User `json:"to_user,omitempty"`
}

// CreateUserRequest represents a user registration request
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
} 