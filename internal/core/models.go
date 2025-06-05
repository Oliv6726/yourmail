package core

import (
	"sync"
	"time"
)

// Message represents an email message in the system
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

// User represents a user account on this server
type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // In production, this should be hashed
	Inbox    []Message `json:"inbox"`
	mu       sync.RWMutex
}

// AddMessage adds a message to the user's inbox
func (u *User) AddMessage(msg Message) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Inbox = append(u.Inbox, msg)
}

// GetInbox returns a copy of the user's inbox
func (u *User) GetInbox() []Message {
	u.mu.RLock()
	defer u.mu.RUnlock()
	inbox := make([]Message, len(u.Inbox))
	copy(inbox, u.Inbox)
	return inbox
}

// UserRegistry manages all users on this server
type UserRegistry struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewUserRegistry creates a new user registry
func NewUserRegistry() *UserRegistry {
	return &UserRegistry{
		users: make(map[string]*User),
	}
}

// AddUser adds a new user to the registry
func (r *UserRegistry) AddUser(username, password string) *User {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	user := &User{
		Username: username,
		Password: password,
		Inbox:    make([]Message, 0),
	}
	r.users[username] = user
	return user
}

// GetUser retrieves a user by username
func (r *UserRegistry) GetUser(username string) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, exists := r.users[username]
	return user, exists
}

// Authenticate checks if the username/password combination is valid
func (r *UserRegistry) Authenticate(username, password string) bool {
	user, exists := r.GetUser(username)
	if !exists {
		return false
	}
	return user.Password == password // In production, use proper password hashing
}

// IsLocalUser checks if a user exists on this server
func (r *UserRegistry) IsLocalUser(username string) bool {
	_, exists := r.GetUser(username)
	return exists
} 