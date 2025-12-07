package sqlstore

import (
	"testing"

	"github.com/pliu/chatty/internal/models"
)

func TestCreateChat(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	id, err := testStore.CreateChat("General", 1)
	if err != nil {
		t.Errorf("Failed to create chat: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero chat ID")
	}
}

func TestAddParticipant(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser(&models.User{Username: "user1", Email: "user1@example.com", Password: "pass"})
	chatID, _ := testStore.CreateChat("Chat 1", 1)
	user, _ := testStore.GetUserByUsername("user1")

	err := testStore.AddParticipant(int(chatID), user.ID, "encrypted_key_mock")
	if err != nil {
		t.Errorf("Failed to add participant: %v", err)
	}

	isParticipant, err := testStore.IsParticipant(int(chatID), user.ID)
	if err != nil {
		t.Errorf("IsParticipant failed: %v", err)
	}

	if !isParticipant {
		t.Error("Expected user to be participant")
	}
}

func TestSaveMessage(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser(&models.User{Username: "user1", Email: "user1@example.com", Password: "pass"})
	chatID, _ := testStore.CreateChat("Chat 1", 1)
	user, _ := testStore.GetUserByUsername("user1")

	err := testStore.SaveMessage(int(chatID), user.ID, "Hello")
	if err != nil {
		t.Errorf("Failed to save message: %v", err)
	}

	messages, err := testStore.GetChatMessages(int(chatID))
	if err != nil {
		t.Errorf("Failed to get messages: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello" {
		t.Errorf("Expected message content 'Hello', got '%s'", messages[0].Content)
	}
}

func TestDeleteChat(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser(&models.User{Username: "owner", Email: "owner@example.com", Password: "pass"})
	owner, _ := testStore.GetUserByUsername("owner")
	chatID, _ := testStore.CreateChat("Chat to Delete", owner.ID)

	// Add participant and message
	testStore.AddParticipant(int(chatID), owner.ID, "key")
	testStore.SaveMessage(int(chatID), owner.ID, "Message")

	// Delete chat
	err := testStore.DeleteChat(int(chatID))
	if err != nil {
		t.Errorf("Failed to delete chat: %v", err)
	}

	// Verify chat is gone
	_, err = testStore.GetChatParticipants(int(chatID))
	// GetChatParticipants might return empty list, let's check IsParticipant
	isParticipant, _ := testStore.IsParticipant(int(chatID), owner.ID)
	if isParticipant {
		t.Error("Expected user to not be participant after deletion")
	}

	// Verify messages are gone
	messages, _ := testStore.GetChatMessages(int(chatID))
	if len(messages) != 0 {
		t.Error("Expected messages to be deleted")
	}
}
