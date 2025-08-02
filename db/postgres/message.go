package postgres

import (
	"database/sql"
	"encoding/base64"
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

// utility function to get the key and handle errors
func getEncryptionKey() ([]byte, error) {
	// Step 1: Get the encoded key string from Viper
	encodedKey := viper.GetString("ENCRYPTION_KEY")
	if encodedKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is not set in configuration")
	}

	// Step 2: Decode the Base64 string into a byte slice
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding ENCRYPTION_KEY from Base64: %w", err)
	}

	// Step 3: Check the length of the decoded byte slice
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes after decoding, got %d bytes", len(key))
	}

	return key, nil
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

	// Step 3: Load encryption key from Viper
	key, err := getEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Step 4: Encrypt the message
	encrypted, err := utils.EncryptAES256(message, key)

	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
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
	key, err := getEncryptionKey()

	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Check if the key is valid
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d bytes", len(key))
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

// GetChatPartners returns usernames of users the given user has chatted with
func (p *Postgres) GetChatPartners(userID int) ([]string, error) {
	query := `
		SELECT DISTINCT u.username
		FROM personal_chats pc
		JOIN users u ON 
			(u.id = pc.user1_id AND pc.user2_id = $1)
			OR (u.id = pc.user2_id AND pc.user1_id = $1)
	`

	rows, err := p.DbConn.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat partners: %w", err)
	}
	defer rows.Close()

	var partners []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan username: %w", err)
		}
		partners = append(partners, name)
	}

	return partners, nil
}

// GetMessagesAfter returns new decrypted messages after a given time
func (p *Postgres) GetMessagesAfter(user1, user2 string, since time.Time) ([]*factory.Message, error) {
	var user1ID, user2ID int

	// Get user IDs
	if err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", user1).Scan(&user1ID); err != nil {
		return nil, fmt.Errorf("failed to get user1 ID: %w", err)
	}
	if err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", user2).Scan(&user2ID); err != nil {
		return nil, fmt.Errorf("failed to get user2 ID: %w", err)
	}

	// Get chat ID
	chatID, err := p.CreatePersonalChat(user1ID, user2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat ID: %w", err)
	}

	// Load encryption key
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	// Query new messages
	query := `
		SELECT m.id, m.sender_id, u.username, m.content, m.sent_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.chat_type = 'personal' AND m.chat_id = $1 AND m.sent_at > $2
		ORDER BY m.sent_at ASC
	`
	rows, err := p.DbConn.Query(query, chatID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query new messages: %w", err)
	}
	defer rows.Close()

	var messages []*factory.Message
	for rows.Next() {
		var msg factory.Message
		var encrypted string
		var sentAt time.Time
		err := rows.Scan(&msg.ID, &msg.SenderID, &msg.SenderName, &encrypted, &sentAt)
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
		messages = append(messages, &msg)
	}
	return messages, nil
}

func (p *Postgres) GetChatID(username1, username2 string) (int, error) {
	var user1ID, user2ID int

	// Get user IDs
	err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", username1).Scan(&user1ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user1 ID: %w", err)
	}
	err = p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", username2).Scan(&user2ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user2 ID: %w", err)
	}

	// Ensure order
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// Check if chat exists
	var chatID int
	err = p.DbConn.QueryRow(`
		SELECT id FROM personal_chats WHERE user1_id = $1 AND user2_id = $2
	`, user1ID, user2ID).Scan(&chatID)
	if err == sql.ErrNoRows {
		// Create if not exists
		err = p.DbConn.QueryRow(`
			INSERT INTO personal_chats (user1_id, user2_id) VALUES ($1, $2) RETURNING id
		`, user1ID, user2ID).Scan(&chatID)
		if err != nil {
			return 0, fmt.Errorf("failed to create chat: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to get chat ID: %w", err)
	}

	return chatID, nil
}

func (p *Postgres) GetLastMessagesBetweenUsers(user1, user2 string, limit int) ([]*factory.Message, error) {
	var user1ID, user2ID int

	// Get user IDs
	err := p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", user1).Scan(&user1ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user1 ID: %w", err)
	}
	err = p.DbConn.QueryRow("SELECT id FROM users WHERE username = $1", user2).Scan(&user2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user2 ID: %w", err)
	}

	chatID, err := p.CreatePersonalChat(user1ID, user2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat ID: %w", err)
	}

	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, sender_id, content, sent_at
		FROM messages
		WHERE chat_type = 'personal' AND chat_id = $1
		ORDER BY sent_at DESC
		LIMIT $2
	`
	rows, err := p.DbConn.Query(query, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get last messages: %w", err)
	}
	defer rows.Close()

	var messages []*factory.Message
	for rows.Next() {
		var msg factory.Message
		var encrypted string
		var sentAt time.Time

		err := rows.Scan(&msg.ID, &msg.SenderID, &encrypted, &sentAt)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		msg.ChatID = chatID
		msg.SentAt = sentAt.Format("2006-01-02 15:04:05")
		msg.ChatType = "personal"
		msg.Content, _ = utils.DecryptAES256(encrypted, key)

		messages = append([]*factory.Message{&msg}, messages...) // reverse order
	}

	return messages, nil
}
