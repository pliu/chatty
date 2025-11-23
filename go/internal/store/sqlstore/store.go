package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"           // Postgres driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/pliu/chatty/internal/models"
)

type SQLStore struct {
	db         *sql.DB
	driverName string
}

func New(driverName, dataSourceName string) (*SQLStore, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	s := &SQLStore{db: db, driverName: driverName}
	s.createTables()
	return s, nil
}

func (s *SQLStore) createTables() {
	// Simplified for brevity, ideally use migrations
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		public_key TEXT,
		encrypted_private_key TEXT
	);

	CREATE TABLE IF NOT EXISTS chats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		owner_id INTEGER REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS participants (
		chat_id INTEGER,
		user_id INTEGER,
		encrypted_chat_key TEXT,
		PRIMARY KEY (chat_id, user_id),
		FOREIGN KEY (chat_id) REFERENCES chats(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER,
		user_id INTEGER,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (chat_id) REFERENCES chats(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`

	if s.driverName == "postgres" {
		// Adjust for Postgres syntax
		query = strings.ReplaceAll(query, "INTEGER PRIMARY KEY AUTOINCREMENT", "SERIAL PRIMARY KEY")
		query = strings.ReplaceAll(query, "DATETIME", "TIMESTAMP")
	}

	_, err := s.db.Exec(query)
	if err != nil {
		panic(err)
	}
}

// Helper to handle placeholders
func (s *SQLStore) rebind(query string) string {
	if s.driverName == "postgres" {
		// Replace ? with $1, $2, etc.
		n := strings.Count(query, "?")
		for i := 1; i <= n; i++ {
			query = strings.Replace(query, "?", fmt.Sprintf("$%d", i), 1)
		}
	}
	return query
}

func (s *SQLStore) CreateUser(user *models.User) error {
	query := s.rebind("INSERT INTO users (username, password, public_key, encrypted_private_key) VALUES (?, ?, ?, ?)")
	_, err := s.db.Exec(query, user.Username, user.Password, user.PublicKey, user.EncryptedPrivateKey)
	return err
}

func (s *SQLStore) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := s.rebind("SELECT id, username, password, COALESCE(public_key, ''), COALESCE(encrypted_private_key, '') FROM users WHERE username = ?")

	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.PublicKey, &user.EncryptedPrivateKey)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *SQLStore) GetUserByID(id int) (*models.User, error) {
	var user models.User
	query := s.rebind("SELECT id, username, password, COALESCE(public_key, ''), COALESCE(encrypted_private_key, '') FROM users WHERE id = ?")
	err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Password, &user.PublicKey, &user.EncryptedPrivateKey)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *SQLStore) SearchUsers(queryStr string) ([]models.User, error) {
	query := s.rebind("SELECT id, username, COALESCE(public_key, '') FROM users WHERE username LIKE ? LIMIT 10")
	rows, err := s.db.Query(query, "%"+queryStr+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.PublicKey); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *SQLStore) CreateChat(name string, ownerID int) (int64, error) {
	var id int64
	query := s.rebind("INSERT INTO chats (name, owner_id) VALUES (?, ?) RETURNING id")
	err := s.db.QueryRow(query, name, ownerID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *SQLStore) AddParticipant(chatID, userID int, encryptedKey string) error {
	query := s.rebind("INSERT INTO participants (chat_id, user_id, encrypted_chat_key) VALUES (?, ?, ?)")
	_, err := s.db.Exec(query, chatID, userID, encryptedKey)
	return err
}

func (s *SQLStore) IsParticipant(chatID, userID int) (bool, error) {
	var exists bool
	query := s.rebind("SELECT EXISTS(SELECT 1 FROM participants WHERE chat_id = ? AND user_id = ?)")
	err := s.db.QueryRow(query, chatID, userID).Scan(&exists)
	return exists, err
}

func (s *SQLStore) GetUserChats(userID int) ([]models.Chat, error) {
	query := s.rebind(`
		SELECT c.id, c.name, c.owner_id, p.encrypted_chat_key
		FROM chats c
		JOIN participants p ON c.id = p.chat_id
		WHERE p.user_id = ?
	`)
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		if err := rows.Scan(&chat.ID, &chat.Name, &chat.OwnerID, &chat.EncryptedKey); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

func (s *SQLStore) GetChatParticipants(chatID int) ([]models.User, error) {
	query := s.rebind(`
		SELECT u.id, u.username, u.public_key, u.encrypted_private_key
		FROM users u
		JOIN participants p ON u.id = p.user_id
		WHERE p.chat_id = ?
	`)

	rows, err := s.db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PublicKey, &u.EncryptedPrivateKey); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *SQLStore) GetChatOwner(chatID int) (int, error) {
	var ownerID int
	query := s.rebind("SELECT owner_id FROM chats WHERE id = ?")
	err := s.db.QueryRow(query, chatID).Scan(&ownerID)
	return ownerID, err
}

func (s *SQLStore) DeleteChat(chatID int) error {
	// Delete messages first (foreign key constraint)
	query := s.rebind("DELETE FROM messages WHERE chat_id = ?")
	if _, err := s.db.Exec(query, chatID); err != nil {
		return err
	}

	// Delete participants
	query = s.rebind("DELETE FROM participants WHERE chat_id = ?")
	if _, err := s.db.Exec(query, chatID); err != nil {
		return err
	}

	// Delete chat
	query = s.rebind("DELETE FROM chats WHERE id = ?")
	_, err := s.db.Exec(query, chatID)
	return err
}

func (s *SQLStore) SaveMessage(chatID, userID int, content string) error {
	query := s.rebind("INSERT INTO messages (chat_id, user_id, content) VALUES (?, ?, ?)")
	_, err := s.db.Exec(query, chatID, userID, content)
	return err
}

func (s *SQLStore) GetChatMessages(chatID int) ([]models.Message, error) {
	query := s.rebind(`
		SELECT m.id, m.chat_id, m.user_id, u.username, m.content, m.created_at
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.chat_id = ?
		ORDER BY m.created_at ASC
	`)
	rows, err := s.db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.UserID, &m.Username, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
