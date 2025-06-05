package database

import (
	"database/sql"
	"fmt"
)

// AttachmentRepository handles attachment database operations
type AttachmentRepository struct {
	db *DB
}

// NewAttachmentRepository creates a new attachment repository
func NewAttachmentRepository(db *DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

// Create creates a new attachment
func (r *AttachmentRepository) Create(messageID int, filename, originalName, contentType string, fileSize int64, filePath *string, fileData []byte) (*Attachment, error) {
	query := `
		INSERT INTO attachments (message_id, filename, original_name, content_type, file_size, file_path, file_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := r.db.Exec(query, messageID, filename, originalName, contentType, fileSize, filePath, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment ID: %w", err)
	}

	return r.GetByID(int(id))
}

// GetByID retrieves an attachment by ID
func (r *AttachmentRepository) GetByID(id int) (*Attachment, error) {
	query := `
		SELECT id, message_id, filename, original_name, content_type, file_size, file_path, file_data, created_at
		FROM attachments 
		WHERE id = ?
	`
	
	attachment := &Attachment{}
	err := r.db.QueryRow(query, id).Scan(
		&attachment.ID,
		&attachment.MessageID,
		&attachment.FileName,
		&attachment.OriginalName,
		&attachment.ContentType,
		&attachment.FileSize,
		&attachment.FilePath,
		&attachment.FileData,
		&attachment.CreatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	return attachment, nil
}

// GetByMessageID retrieves all attachments for a message
func (r *AttachmentRepository) GetByMessageID(messageID int) ([]*Attachment, error) {
	query := `
		SELECT id, message_id, filename, original_name, content_type, file_size, file_path, created_at
		FROM attachments 
		WHERE message_id = ?
		ORDER BY created_at ASC
	`
	
	rows, err := r.db.Query(query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*Attachment
	for rows.Next() {
		attachment := &Attachment{}
		err := rows.Scan(
			&attachment.ID,
			&attachment.MessageID,
			&attachment.FileName,
			&attachment.OriginalName,
			&attachment.ContentType,
			&attachment.FileSize,
			&attachment.FilePath,
			&attachment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// GetFileData retrieves the file data for an attachment (for database storage)
func (r *AttachmentRepository) GetFileData(id int) ([]byte, error) {
	query := `SELECT file_data FROM attachments WHERE id = ?`
	
	var fileData []byte
	err := r.db.QueryRow(query, id).Scan(&fileData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file data: %w", err)
	}

	return fileData, nil
}

// Delete deletes an attachment
func (r *AttachmentRepository) Delete(id int) error {
	query := `DELETE FROM attachments WHERE id = ?`
	
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

// GetAttachmentCountByMessageID gets the count of attachments for a message
func (r *AttachmentRepository) GetAttachmentCountByMessageID(messageID int) (int, error) {
	query := `SELECT COUNT(*) FROM attachments WHERE message_id = ?`
	
	var count int
	err := r.db.QueryRow(query, messageID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get attachment count: %w", err)
	}

	return count, nil
} 