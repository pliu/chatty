package models

import "time"

type User struct {
	ID                  int    `json:"id"`
	Username            string `json:"username"`
	Email               string `json:"email"`
	Password            string `json:"-"`
	PublicKey           string `json:"public_key"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
	IsVerified          bool   `json:"is_verified"`
	VerificationToken   string `json:"-"`
}

type Chat struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	OwnerID      int    `json:"owner_id"`
	EncryptedKey string `json:"encrypted_key,omitempty"` // Per-user encrypted chat key
}

type Message struct {
	ID        int       `json:"id"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
