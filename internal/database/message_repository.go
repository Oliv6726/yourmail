package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// MessageRepository handles message database operations
type MessageRepository struct {
	db *DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// CreateWithThreading creates a new message with threading support
func (r *MessageRepository) CreateWithThreading(fromUserID, toUserID *int, fromAddress, toAddress, subject, body string, isHTML bool, threadID *string, parentID *int) (*Message, error) {
	// Generate thread ID if not provided and this is not a reply
	if threadID == nil && parentID == nil {
		id, err := generateThreadID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate thread ID: %w", err)
		}
		threadID = &id
	}

	query := `
		INSERT INTO messages (from_user_id, to_user_id, from_address, to_address, subject, body, is_html, thread_id, parent_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.Exec(query, fromUserID, toUserID, fromAddress, toAddress, subject, body, isHTML, threadID, parentID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Get the created message ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get message ID: %w", err)
	}

	return r.GetByID(int(id))
}

// Create creates a new message (backward compatibility)
func (r *MessageRepository) Create(fromUserID, toUserID *int, fromAddress, toAddress, subject, body string) (*Message, error) {
	return r.CreateWithThreading(fromUserID, toUserID, fromAddress, toAddress, subject, body, false, nil, nil)
}

// generateThreadID generates a unique thread ID
func generateThreadID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetByID retrieves a message by ID with threading support
func (r *MessageRepository) GetByID(id int) (*Message, error) {
	message := &Message{}
	var fromUserID, toUserID, parentID sql.NullInt64
	var threadID sql.NullString
	query := `
		SELECT id, from_user_id, to_user_id, from_address, to_address, 
		       subject, body, is_html, thread_id, parent_id, read_status, created_at
		FROM messages WHERE id = ?
	`
	err := r.db.QueryRow(query, id).Scan(
		&message.ID, &fromUserID, &toUserID,
		&message.FromAddress, &message.ToAddress, &message.Subject,
		&message.Body, &message.IsHTML, &threadID, &parentID, 
		&message.ReadStatus, &message.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	
	// Convert nullable IDs
	if fromUserID.Valid {
		id := int(fromUserID.Int64)
		message.FromUserID = &id
	}
	if toUserID.Valid {
		id := int(toUserID.Int64)
		message.ToUserID = &id
	}
	if parentID.Valid {
		id := int(parentID.Int64)
		message.ParentID = &id
	}
	if threadID.Valid {
		message.ThreadID = &threadID.String
	}
	
	return message, nil
}

// GetThreadByID retrieves all messages in a thread
func (r *MessageRepository) GetThreadByID(threadID string) ([]*Message, error) {
	query := `
		SELECT m.id, m.from_user_id, m.to_user_id, m.from_address, m.to_address,
		       m.subject, m.body, m.is_html, m.thread_id, m.parent_id, m.read_status, m.created_at,
		       fu.id, fu.username, fu.email
		FROM messages m
		LEFT JOIN users fu ON m.from_user_id = fu.id
		WHERE m.thread_id = ?
		ORDER BY m.created_at ASC
	`
	rows, err := r.db.Query(query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		var fromUser User
		var msgFromUserID, msgToUserID, msgParentID sql.NullInt64
		var msgThreadID sql.NullString
		var fromUserIdDB sql.NullInt64
		var fromUsername, fromEmail sql.NullString

		err := rows.Scan(
			&message.ID, &msgFromUserID, &msgToUserID,
			&message.FromAddress, &message.ToAddress, &message.Subject,
			&message.Body, &message.IsHTML, &msgThreadID, &msgParentID,
			&message.ReadStatus, &message.CreatedAt,
			&fromUserIdDB, &fromUsername, &fromEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Convert nullable message IDs
		if msgFromUserID.Valid {
			id := int(msgFromUserID.Int64)
			message.FromUserID = &id
		}
		if msgToUserID.Valid {
			id := int(msgToUserID.Int64)
			message.ToUserID = &id
		}
		if msgParentID.Valid {
			id := int(msgParentID.Int64)
			message.ParentID = &id
		}
		if msgThreadID.Valid {
			message.ThreadID = &msgThreadID.String
		}

		// Set FromUser if exists
		if fromUserIdDB.Valid {
			fromUser.ID = int(fromUserIdDB.Int64)
			fromUser.Username = fromUsername.String
			fromUser.Email = fromEmail.String
			message.FromUser = &fromUser
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// GetInboxForUser retrieves all messages for a user's inbox (threaded)
func (r *MessageRepository) GetInboxForUser(userID int, limit, offset int) ([]*Message, error) {
	// Get thread roots first (messages with no parent)
	query := `
		SELECT DISTINCT m.thread_id, m.id, m.from_user_id, m.to_user_id, m.from_address, m.to_address,
		       m.subject, m.body, m.is_html, m.parent_id, m.read_status, m.created_at,
		       fu.id, fu.username, fu.email,
		       (SELECT COUNT(*) FROM messages WHERE thread_id = m.thread_id AND to_user_id = ?) as reply_count,
		       DATETIME((SELECT MAX(created_at) FROM messages WHERE thread_id = m.thread_id AND to_user_id = ?)) as last_message_time
		FROM messages m
		LEFT JOIN users fu ON m.from_user_id = fu.id
		WHERE m.to_user_id = ? AND (m.parent_id IS NULL OR m.id = (
			SELECT MIN(id) FROM messages WHERE thread_id = m.thread_id AND to_user_id = ?
		))
		GROUP BY m.thread_id
		ORDER BY last_message_time DESC
		LIMIT ? OFFSET ?
	`
	
	log.Printf("DEBUG: Executing GetInboxForUser query for userID: %d", userID)
	rows, err := r.db.Query(query, userID, userID, userID, userID, limit, offset)
	if err != nil {
		log.Printf("DEBUG: Query failed: %v", err)
		return nil, fmt.Errorf("failed to get inbox: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		var fromUser User
		var msgFromUserID, msgToUserID, msgParentID sql.NullInt64
		var msgThreadID sql.NullString
		var fromUserIdDB sql.NullInt64
		var fromUsername, fromEmail sql.NullString
		var replyCount int
		var lastMessageTimeStr sql.NullString

		log.Printf("DEBUG: About to scan row...")
		err := rows.Scan(
			&msgThreadID, &message.ID, &msgFromUserID, &msgToUserID,
			&message.FromAddress, &message.ToAddress, &message.Subject,
			&message.Body, &message.IsHTML, &msgParentID,
			&message.ReadStatus, &message.CreatedAt,
			&fromUserIdDB, &fromUsername, &fromEmail,
			&replyCount, &lastMessageTimeStr,
		)
		if err != nil {
			log.Printf("DEBUG: Scan failed with error: %v", err)
			log.Printf("DEBUG: lastMessageTimeStr value: %+v, valid: %t", lastMessageTimeStr.String, lastMessageTimeStr.Valid)
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		log.Printf("DEBUG: Successfully scanned row. lastMessageTimeStr: %+v, valid: %t", lastMessageTimeStr.String, lastMessageTimeStr.Valid)

		// Convert nullable message IDs
		if msgFromUserID.Valid {
			id := int(msgFromUserID.Int64)
			message.FromUserID = &id
		}
		if msgToUserID.Valid {
			id := int(msgToUserID.Int64)
			message.ToUserID = &id
		}
		if msgParentID.Valid {
			id := int(msgParentID.Int64)
			message.ParentID = &id
		}
		if msgThreadID.Valid {
			message.ThreadID = &msgThreadID.String
		}

		// Set FromUser if exists
		if fromUserIdDB.Valid {
			fromUser.ID = int(fromUserIdDB.Int64)
			fromUser.Username = fromUsername.String
			fromUser.Email = fromEmail.String
			message.FromUser = &fromUser
		}

		// Load replies if this is a thread
		if msgThreadID.Valid && replyCount > 1 {
			replies, err := r.GetThreadByID(msgThreadID.String)
			if err == nil && len(replies) > 1 {
				// Remove the first message (original) and set the rest as replies
				message.Replies = replies[1:]
			}
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// GetInboxForAddress retrieves messages for a specific address (for external messages)
func (r *MessageRepository) GetInboxForAddress(address string, limit, offset int) ([]*Message, error) {
	query := `
		SELECT m.id, m.from_user_id, m.to_user_id, m.from_address, m.to_address,
		       m.subject, m.body, m.read_status, m.created_at
		FROM messages m
		WHERE m.to_address = ?
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, address, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbox for address: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		var msgFromUserID, msgToUserID sql.NullInt64
		err := rows.Scan(
			&message.ID, &msgFromUserID, &msgToUserID,
			&message.FromAddress, &message.ToAddress, &message.Subject,
			&message.Body, &message.ReadStatus, &message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		// Convert nullable message IDs
		if msgFromUserID.Valid {
			id := int(msgFromUserID.Int64)
			message.FromUserID = &id
		}
		if msgToUserID.Valid {
			id := int(msgToUserID.Int64)
			message.ToUserID = &id
		}
		
		messages = append(messages, message)
	}

	return messages, nil
}

// GetSentForUser retrieves all sent messages for a user
func (r *MessageRepository) GetSentForUser(userID int, limit, offset int) ([]*Message, error) {
	query := `
		SELECT m.id, m.from_user_id, m.to_user_id, m.from_address, m.to_address,
		       m.subject, m.body, m.read_status, m.created_at,
		       tu.id, tu.username, tu.email
		FROM messages m
		LEFT JOIN users tu ON m.to_user_id = tu.id
		WHERE m.from_user_id = ?
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		var toUser User
		var msgFromUserID, msgToUserID sql.NullInt64
		var toUserIdDB sql.NullInt64
		var toUsername, toEmail sql.NullString

		err := rows.Scan(
			&message.ID, &msgFromUserID, &msgToUserID,
			&message.FromAddress, &message.ToAddress, &message.Subject,
			&message.Body, &message.ReadStatus, &message.CreatedAt,
			&toUserIdDB, &toUsername, &toEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Convert nullable message IDs
		if msgFromUserID.Valid {
			id := int(msgFromUserID.Int64)
			message.FromUserID = &id
		}
		if msgToUserID.Valid {
			id := int(msgToUserID.Int64)
			message.ToUserID = &id
		}

		// Set ToUser if exists
		if toUserIdDB.Valid {
			toUser.ID = int(toUserIdDB.Int64)
			toUser.Username = toUsername.String
			toUser.Email = toEmail.String
			message.ToUser = &toUser
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// MarkAsRead marks a message as read
func (r *MessageRepository) MarkAsRead(messageID int) error {
	query := `UPDATE messages SET read_status = TRUE WHERE id = ?`
	_, err := r.db.Exec(query, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}
	return nil
}

// Delete deletes a message
func (r *MessageRepository) Delete(messageID int) error {
	query := `DELETE FROM messages WHERE id = ?`
	_, err := r.db.Exec(query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// GetUnreadCount returns the count of unread messages for a user
func (r *MessageRepository) GetUnreadCount(userID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM messages WHERE to_user_id = ? AND read_status = FALSE`
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
} 