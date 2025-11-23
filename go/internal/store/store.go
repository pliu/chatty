package store

import "github.com/pliu/chatty/internal/models"

type Store interface {
	// User operations
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	SearchUsers(query string) ([]models.User, error)

	// Chat operations
	CreateChat(name string, ownerID int) (int64, error)
	AddParticipant(chatID, userID int, encryptedKey string) error
	IsParticipant(chatID, userID int) (bool, error)
	GetUserChats(userID int) ([]models.Chat, error)
	GetChatParticipants(chatID int) ([]models.User, error)
	SaveMessage(chatID, userID int, content string) error
	GetChatMessages(chatID int) ([]models.Message, error)
}
