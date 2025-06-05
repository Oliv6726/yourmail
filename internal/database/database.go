package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=1&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{sqlDB}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("✅ Database connected and migrated: %s", dbPath)
	return db, nil
}

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Messages table with threading support
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_user_id INTEGER,
			to_user_id INTEGER,
			from_address TEXT NOT NULL,
			to_address TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			is_html BOOLEAN DEFAULT FALSE,
			thread_id TEXT,
			parent_id INTEGER,
			read_status BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE SET NULL,
			FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE SET NULL,
			FOREIGN KEY (parent_id) REFERENCES messages(id) ON DELETE SET NULL
		)`,

		// Attachments table
		`CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			file_path TEXT,
			file_data BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,

		// Add new columns to existing messages table (for backward compatibility)
		`ALTER TABLE messages ADD COLUMN is_html BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE messages ADD COLUMN thread_id TEXT`,
		`ALTER TABLE messages ADD COLUMN parent_id INTEGER REFERENCES messages(id) ON DELETE SET NULL`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_from_user ON messages(from_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to_user ON messages(to_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to_address ON messages(to_address)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON messages(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id)`,

		// Trigger to update updated_at timestamp
		`CREATE TRIGGER IF NOT EXISTS update_users_updated_at 
		 AFTER UPDATE ON users 
		 BEGIN 
			UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		 END`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore column already exists errors for ALTER TABLE statements
			if i >= 3 && i <= 5 { // ALTER TABLE statements
				continue
			}
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// SeedTestUsers creates test users for development
func (db *DB) SeedTestUsers() error {
	testUsers := []struct {
		username string
		email    string
		password string
	}{
		{"alice", "alice@yourmail.local", "password123"},
		{"bob", "bob@yourmail.local", "password456"},
		{"charlie", "charlie@yourmail.local", "password789"},
	}

	userRepo := NewUserRepository(db)

	for _, user := range testUsers {
		// Check if user already exists
		existing, _ := userRepo.GetByUsername(user.username)
		if existing != nil {
			continue // Skip if user already exists
		}

		// Create user
		_, err := userRepo.Create(user.username, user.email, user.password)
		if err != nil {
			log.Printf("Failed to create test user %s: %v", user.username, err)
		} else {
			log.Printf("✅ Created test user: %s", user.username)
		}
	}

	return nil
} 