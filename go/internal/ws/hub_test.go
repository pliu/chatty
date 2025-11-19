package ws

import (
	"testing"
	"time"

	"github.com/pliu/chatty/internal/store/sqlstore"
)

func TestHubRun(t *testing.T) {
	store, _ := sqlstore.New("sqlite3", ":memory:")
	store.CreateUser("user1", "pass")
	user, _ := store.GetUserByUsername("user1")
	chatID, _ := store.CreateChat("Test Chat")
	store.AddParticipant(int(chatID), user.ID)

	hub := NewHub(store)
	go hub.Run()

	// Simulate a message broadcast
	msg := Message{
		ChatID:  int(chatID),
		UserID:  user.ID,
		Content: "Hello World",
	}

	hub.broadcast <- msg

	// Give some time for the hub to process
	time.Sleep(100 * time.Millisecond)

	// Verify message was saved to store
	messages, err := store.GetChatMessages(int(chatID))
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got '%s'", messages[0].Content)
	}
}
