package models

import "time"

type User struct {
	ID                  int    `json:"id"`
	Username            string `json:"username"`
	Password            string `json:"-"`
	PublicKey           string `json:"public_key"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
}

type Chat struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	EncryptedKey string `json:"encrypted_key"`
}

type Message struct {
	ID        int       `json:"id"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
