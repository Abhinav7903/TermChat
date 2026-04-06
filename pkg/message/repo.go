package message

import (
	"termchat/factory"
	"time"
)

type Repository interface {
	CreatePersonalChat(user1ID, user2ID int) (int, error)
	SendPersonalMessage(senderUsername, receiverUsername, message, sessionID string) error
	GetMessagesBetweenUsers(username1, username2 string) ([]factory.Message, error)
	GetChatPartners(userID int) ([]string, error)
	GetMessagesAfter(user1, user2 string, since time.Time) ([]*factory.Message, error)
	GetChatID(user1, user2 string) (int, error)
	GetLastMessagesBetweenUsers(user1, user2 string, limit int) ([]*factory.Message, error)

	// Group Chat Methods
	CreateGroupChat(name, description string, ownerID int) (int, error)
	JoinGroupChat(userID, groupID int) error
	LeaveGroupChat(userID, groupID int) error
	GetGroupChatMessages(groupID int) ([]factory.Message, error)
	SendGroupMessage(senderID, groupID int, message, sessionID string) error
	GetGroupChatID(name string) (int, error)
	GetUserGroupChats(userID int) ([]factory.GroupChat, error)
	GetGlobalChatID() (int, error)
	IsGroupOwner(userID, groupID int) (bool, error)
	AddGroupMember(userID, groupID int) error
	RemoveGroupMember(userID, groupID int) error
	AddReaction(messageID, userID int, emoji string) error
	GetLastMessageID(chatType string, chatID int) (int, error)
}
