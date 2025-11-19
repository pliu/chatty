package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

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
