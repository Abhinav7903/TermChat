package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"termchat/db/redis"
	"termchat/factory"
	"termchat/utils"
	"time"

	"github.com/lib/pq"
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
func (p *Postgres) SendPersonalMessage(senderUsername, receiverUsername, message, sessionID string) error {
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

	// Step 3: Load encryption key
	key, err := getEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Step 4: Encrypt the message before storing in DB
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

	// Step 6: Publish to Redis so live chat works
	// Payload format: <sessionID>|<senderUsername>|<timestamp>|<message>
	payload := fmt.Sprintf("%s|%s|%s|%s",
		sessionID,
		senderUsername,
		time.Now().Format("2006-01-02 15:04:05"),
		message, // plaintext so receiver can read immediately
	)

	ctx := context.Background()
	channel := fmt.Sprintf("chat:%d", chatID)

	if err := redis.NewRedis(nil).Client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish message to Redis: %w", err)
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

	p.fetchReactionsForMessages(messages)
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

// CreateGroupChat creates a new group chat
func (p *Postgres) CreateGroupChat(name, description string, ownerID int) (int, error) {
	var groupID int
	query := `
		INSERT INTO group_chats (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := p.DbConn.QueryRow(query, name, description, ownerID).Scan(&groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to create group chat: %w", err)
	}

	// Owner joins as an admin
	err = p.JoinGroupChat(ownerID, groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to join group chat as owner: %w", err)
	}

	return groupID, nil
}

// JoinGroupChat adds a user to a group chat
func (p *Postgres) JoinGroupChat(userID, groupID int) error {
	query := `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (group_id, user_id) DO NOTHING
	`
	// If it was already there, no problem. If owner, we might want to set role to owner.
	// But the migration says role defaults to 'member'.
	_, err := p.DbConn.Exec(query, groupID, userID)
	return err
}

// LeaveGroupChat removes a user from a group chat
func (p *Postgres) LeaveGroupChat(userID, groupID int) error {
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	_, err := p.DbConn.Exec(query, groupID, userID)
	return err
}

// GetGroupChatMessages retrieves decrypted messages for a group
func (p *Postgres) GetGroupChatMessages(groupID int) ([]factory.Message, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT m.id, m.sender_id, u.username, m.content, m.sent_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.chat_type = 'group' AND m.chat_id = $1
		ORDER BY m.sent_at ASC
	`
	rows, err := p.DbConn.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []factory.Message
	for rows.Next() {
		var msg factory.Message
		var encrypted string
		var sentAt time.Time
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.SenderName, &encrypted, &sentAt); err != nil {
			return nil, err
		}

		decrypted, _ := utils.DecryptAES256(encrypted, key)
		msg.Content = decrypted
		msg.ChatID = groupID
		msg.SentAt = sentAt.Format("2006-01-02 15:04:05")
		msg.ChatType = "group"
		messages = append(messages, msg)
	}

	p.fetchReactionsForMessages(messages)
	return messages, nil
}

// SendGroupMessage encrypts and stores a message for a group
func (p *Postgres) SendGroupMessage(senderID, groupID int, message, sessionID string) error {
	key, err := getEncryptionKey()
	if err != nil {
		return err
	}

	encrypted, err := utils.EncryptAES256(message, key)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO messages (sender_id, chat_type, chat_id, content, sent_at)
		VALUES ($1, 'group', $2, $3, NOW())
	`
	_, err = p.DbConn.Exec(query, senderID, groupID, encrypted)
	if err != nil {
		return err
	}

	// Get sender name for Redis
	var senderName string
	p.DbConn.QueryRow("SELECT username FROM users WHERE id = $1", senderID).Scan(&senderName)

	// Publish to Redis
	// Format: <sessionID>|<senderName>|<timestamp>|<message>
	payload := fmt.Sprintf("%s|%s|%s|%s", sessionID, senderName, time.Now().Format("2006-01-02 15:04:05"), message)
	channel := fmt.Sprintf("group:%d", groupID)
	err = redis.NewRedis(nil).Client.Publish(context.Background(), channel, payload).Err()
	if err != nil {
		return err
	}

	// Notify other members
	var groupName string
	p.DbConn.QueryRow("SELECT name FROM group_chats WHERE id = $1", groupID).Scan(&groupName)

	rows, err := p.DbConn.Query(`
		SELECT u.username FROM users u
		JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND u.id != $2
	`, groupID, senderID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var memberName string
			if err := rows.Scan(&memberName); err == nil {
				notif := fmt.Sprintf("GROUP_MSG %s|%s", senderName, groupName)
				// Use a different channel prefix for notifications
				notifChan := "notify:" + strings.ToLower(memberName)
				redis.NewRedis(nil).Client.Publish(context.Background(), notifChan, notif)
			}
		}
	}

	return nil
}

// GetGroupChatID returns the ID of a group chat by name
func (p *Postgres) GetGroupChatID(name string) (int, error) {
	var id int
	err := p.DbConn.QueryRow("SELECT id FROM group_chats WHERE LOWER(name) = LOWER($1)", name).Scan(&id)
	return id, err
}

// GetUserGroupChats returns all groups a user belongs to
func (p *Postgres) GetUserGroupChats(userID int) ([]factory.GroupChat, error) {
	query := `
		SELECT g.id, g.name, g.description, g.owner_id, g.is_global
		FROM group_chats g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
	`
	rows, err := p.DbConn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []factory.GroupChat
	for rows.Next() {
		var g factory.GroupChat
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID, &g.IsGlobal); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// GetGlobalChatID returns the ID of the global chat, creating it if necessary
func (p *Postgres) GetGlobalChatID() (int, error) {
	var id int
	err := p.DbConn.QueryRow("SELECT id FROM group_chats WHERE is_global = TRUE").Scan(&id)
	if err == sql.ErrNoRows {
		query := `
			INSERT INTO group_chats (name, description, is_global)
			VALUES ('Global', 'The global chat room for everyone', TRUE)
			RETURNING id
		`
		err = p.DbConn.QueryRow(query).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	return id, err
}

func (p *Postgres) IsGroupOwner(userID, groupID int) (bool, error) {
	var ownerID int
	err := p.DbConn.QueryRow("SELECT owner_id FROM group_chats WHERE id = $1", groupID).Scan(&ownerID)
	if err != nil {
		return false, err
	}
	return ownerID == userID, nil
}

func (p *Postgres) AddGroupMember(userID, groupID int) error {
	return p.JoinGroupChat(userID, groupID)
}

func (p *Postgres) RemoveGroupMember(userID, groupID int) error {
	return p.LeaveGroupChat(userID, groupID)
}

func (p *Postgres) AddReaction(messageID, userID int, emoji string) error {
	query := `
		INSERT INTO reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`
	_, err := p.DbConn.Exec(query, messageID, userID, emoji)
	return err
}

func (p *Postgres) GetLastMessageID(chatType string, chatID int) (int, error) {
	var id int
	query := `SELECT id FROM messages WHERE chat_type = $1 AND chat_id = $2 ORDER BY sent_at DESC LIMIT 1`
	err := p.DbConn.QueryRow(query, chatType, chatID).Scan(&id)
	return id, err
}

func (p *Postgres) fetchReactionsForMessages(messages []factory.Message) {
	if len(messages) == 0 {
		return
	}

	ids := make([]int, len(messages))
	msgMap := make(map[int]*factory.Message)
	for i := range messages {
		ids[i] = messages[i].ID
		messages[i].Reactions = make(map[string]string)
		msgMap[messages[i].ID] = &messages[i]
	}

	// Simple way to handle IN clause with lib/pq
	query := `
		SELECT r.message_id, u.username, r.emoji
		FROM reactions r
		JOIN users u ON r.user_id = u.id
		WHERE r.message_id = ANY($1)
	`
	rows, err := p.DbConn.Query(query, pq.Array(ids))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msgID int
		var username, emoji string
		if err := rows.Scan(&msgID, &username, &emoji); err == nil {
			if msg, ok := msgMap[msgID]; ok {
				msg.Reactions[username] = emoji
			}
		}
	}
}
