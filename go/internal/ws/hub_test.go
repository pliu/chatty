package ws

import (
	"testing"
	"time"

	"github.com/pliu/chatty/internal/models"
	"github.com/pliu/chatty/internal/store/sqlstore"
)

func TestHubAuthorization(t *testing.T) {
	// Setup in-memory DB
	store, _ := sqlstore.New("sqlite3", ":memory:")
	store.CreateUser(&models.User{Username: "user1", Password: "pass"})
	store.CreateUser(&models.User{Username: "attacker", Password: "pass"})

	user1, _ := store.GetUserByUsername("user1")
	attacker, _ := store.GetUserByUsername("attacker")

	chatID, _ := store.CreateChat("Secret Chat", user1.ID)
	store.AddParticipant(int(chatID), user1.ID, "key")

	hub := NewHub(store)
	go hub.Run()

	// Attacker tries to send message
	msg := Message{
		ChatID:  int(chatID),
		UserID:  attacker.ID,
		Content: "Malicious Message",
	}

	// Send message to broadcast channel
	hub.broadcast <- msg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify message was NOT saved
	messages, _ := store.GetChatMessages(int(chatID))
	if len(messages) != 0 {
		t.Error("Expected 0 messages, got", len(messages))
	}

	// Now add attacker to chat
	store.AddParticipant(int(chatID), attacker.ID, "key")

	// Send again
	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)

	// Verify message WAS saved
	messages, _ = store.GetChatMessages(int(chatID))
	if len(messages) != 1 {
		t.Error("Expected 1 message, got", len(messages))
	}
}
