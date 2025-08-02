package postgres

import (
	"database/sql"
	"fmt"
	"termchat/factory"
	"termchat/utils"
	"time"

	"github.com/spf13/viper"
)

// CreatePersonalChat creates or retrieves an existing chat between two users
func (p *Postgres) CreatePersonalChat(user1ID, user2ID int) (int, error) {
	// Ensure consistent ordering
	var u1, u2 int
	if user1ID < user2ID {
		u1, u2 = user1ID, user2ID
	} else {
		u1, u2 = user2ID, user1ID
	}

	// Check if chat already exists
	var chatID int
	query := `
		SELECT id FROM personal_chats
		WHERE user1_id = $1 AND user2_id = $2
	`
	err := p.DbConn.QueryRow(query, u1, u2).Scan(&chatID)
	if err == nil {
		return chatID, nil // Found existing chat
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to check existing chat: %w", err)
	}

	// Create new chat
	insert := `
		INSERT INTO personal_chats (user1_id, user2_id)
		VALUES ($1, $2)
		RETURNING id
	`
	err = p.DbConn.QueryRow(insert, u1, u2).Scan(&chatID)
	if err != nil {
		return 0, fmt.Errorf("failed to create chat: %w", err)
	}

	return chatID, nil
}

// SendPersonalMessage encrypts the message using AES-256 and stores it
func (p *Postgres) SendPersonalMessage(senderUsername, receiverUsername, message string) error {
	var senderID, receiverID int

	// Step 1: Get user IDs
	err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", senderUsername).Scan(&senderID)
	if err != nil {
		return fmt.Errorf("sender '%s' not found: %w", senderUsername, err)
	}
	err = p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", receiverUsername).Scan(&receiverID)
	if err != nil {
		return fmt.Errorf("receiver '%s' not found: %w", receiverUsername, err)
	}

	// Step 2: Get or create personal chat
	chatID, err := p.CreatePersonalChat(senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to get/create chat: %w", err)
	}

	// Step 3: Get encryption key from Viper
	key := []byte(viper.GetString("ENCRYPTION_KEY"))
	if len(key) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes")
	}

	// Step 4: Encrypt message
	encrypted, err := utils.EncryptAES256(message, key)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Step 5: Insert into DB
	query := `
		INSERT INTO messages (sender_id, chat_type, chat_id, content, sent_at)
		VALUES ($1, 'personal', $2, $3, NOW())
	`
	_, err = p.DbConn.Exec(query, senderID, chatID, encrypted)
	if err != nil {
		return fmt.Errorf("failed to insert encrypted message: %w", err)
	}

	return nil
}

// GetMessagesBetweenUsers retrieves and decrypts messages between two users
func (p *Postgres) GetMessagesBetweenUsers(username1, username2 string) ([]factory.Message, error) {
	var user1ID, user2ID int

	// Step 1: Get user IDs
	err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", username1).Scan(&user1ID)
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found: %w", username1, err)
	}
	err = p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", username2).Scan(&user2ID)
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found: %w", username2, err)
	}

	// Step 2: Get or create personal chat
	chatID, err := p.CreatePersonalChat(user1ID, user2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create chat: %w", err)
	}

	// Step 3: Load encryption key from Viper
	key := []byte(viper.GetString("ENCRYPTION_KEY"))
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 32 bytes")
	}

	// Step 4: Query messages
	query := `
		SELECT id, sender_id, content, sent_at
		FROM messages
		WHERE chat_type = 'personal' AND chat_id = $1
		ORDER BY sent_at ASC
	`

	rows, err := p.DbConn.Query(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []factory.Message
	for rows.Next() {
		var msg factory.Message
		var encrypted string
		var sentAt time.Time

		err := rows.Scan(&msg.ID, &msg.SenderID, &encrypted, &sentAt)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		decrypted, err := utils.DecryptAES256(encrypted, key)
		if err != nil {
			msg.Content = "[decryption failed]"
		} else {
			msg.Content = decrypted
		}

		msg.ChatID = chatID
		msg.SentAt = sentAt.Format("2006-01-02 15:04:05")
		msg.ChatType = "personal"
		messages = append(messages, msg)
	}

	return messages, nil
}
