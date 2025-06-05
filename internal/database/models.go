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
	IsHTML      bool      `json:"is_html" db:"is_html"`
	ThreadID    *string   `json:"thread_id" db:"thread_id"`
	ParentID    *int      `json:"parent_id" db:"parent_id"`
	ReadStatus  bool      `json:"read" db:"read_status"`
	CreatedAt   time.Time `json:"timestamp" db:"created_at"`
	
	// Virtual fields populated by joins
	FromUser *User `json:"from_user,omitempty"`
	ToUser   *User `json:"to_user,omitempty"`
	
	// Thread-related fields
	Replies []*Message `json:"replies,omitempty"`
	AttachmentCount int `json:"attachment_count,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID          int       `json:"id" db:"id"`
	MessageID   int       `json:"message_id" db:"message_id"`
	FileName    string    `json:"filename" db:"filename"`
	OriginalName string   `json:"original_name" db:"original_name"`
	ContentType string    `json:"content_type" db:"content_type"`
	FileSize    int64     `json:"file_size" db:"file_size"`
	FilePath    *string   `json:"file_path" db:"file_path"` // For file system storage
	FileData    []byte    `json:"-" db:"file_data"` // For database storage (small files)
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
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