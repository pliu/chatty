package database

import (
	"time"
)

type Chat struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	ID        int       `json:"id"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateChat(name string) (int64, error) {
	result, err := DB.Exec("INSERT INTO chats (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func AddParticipant(chatID, userID int) error {
	_, err := DB.Exec("INSERT INTO participants (chat_id, user_id) VALUES (?, ?)", chatID, userID)
	return err
}

func IsParticipant(chatID, userID int) (bool, error) {
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM participants WHERE chat_id = ? AND user_id = ?)", chatID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func GetUserChats(userID int) ([]Chat, error) {
	rows, err := DB.Query(`
		SELECT c.id, c.name 
		FROM chats c
		JOIN participants p ON c.id = p.chat_id
		WHERE p.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}

func SaveMessage(chatID, userID int, content string) error {
	_, err := DB.Exec("INSERT INTO messages (chat_id, user_id, content) VALUES (?, ?, ?)", chatID, userID, content)
	return err
}

func GetChatMessages(chatID int) ([]Message, error) {
	rows, err := DB.Query(`
		SELECT m.id, m.chat_id, m.user_id, u.username, m.content, m.created_at
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.chat_id = ?
		ORDER BY m.created_at ASC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.UserID, &m.Username, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
