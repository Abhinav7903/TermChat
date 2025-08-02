package message

import (
	"termchat/factory"
	"time"
)

type Repository interface {
	CreatePersonalChat(user1ID, user2ID int) (int, error)
	SendPersonalMessage(senderUsername, receiverUsername, message string) error
	GetMessagesBetweenUsers(username1, username2 string) ([]factory.Message, error)
	GetChatPartners(userID int) ([]string, error)
	GetMessagesAfter(user1, user2 string, since time.Time) ([]*factory.Message, error)
	GetChatID(user1, user2 string) (int, error)
	GetLastMessagesBetweenUsers(user1, user2 string, limit int) ([]*factory.Message, error)
}
