package message

import "termchat/factory"

type Repository interface {
	CreatePersonalChat(user1ID, user2ID int) (int, error)
	SendPersonalMessage(senderUsername, receiverUsername, message string) error
	GetMessagesBetweenUsers(username1, username2 string) ([]factory.Message, error)
}
