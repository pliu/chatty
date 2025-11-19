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
		password TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS chats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS participants (
		chat_id INTEGER,
		user_id INTEGER,
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

func (s *SQLStore) CreateUser(username, password string) error {
	query := s.rebind("INSERT INTO users (username, password) VALUES (?, ?)")
	_, err := s.db.Exec(query, username, password)
	return err
}

func (s *SQLStore) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := s.rebind("SELECT id, username, password FROM users WHERE username = ?")
	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *SQLStore) GetUserByID(id int) (*models.User, error) {
	var user models.User
	query := s.rebind("SELECT id, username, password FROM users WHERE id = ?")
	err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *SQLStore) SearchUsers(q string) ([]models.User, error) {
	query := s.rebind("SELECT id, username FROM users WHERE username LIKE ?")
	rows, err := s.db.Query(query, q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *SQLStore) CreateChat(name string) (int64, error) {
	query := s.rebind("INSERT INTO chats (name) VALUES (?)")
	if s.driverName == "postgres" {
		var id int64
		err := s.db.QueryRow(query+" RETURNING id", name).Scan(&id)
		return id, err
	}
	result, err := s.db.Exec(query, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLStore) AddParticipant(chatID, userID int) error {
	query := s.rebind("INSERT INTO participants (chat_id, user_id) VALUES (?, ?)")
	_, err := s.db.Exec(query, chatID, userID)
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
		SELECT c.id, c.name 
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
		var c models.Chat
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
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
