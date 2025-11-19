package sqlstore

import (
	"testing"
)

func TestCreateChat(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	id, err := testStore.CreateChat("General")
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

	testStore.CreateUser("user1", "pass")
	chatID, _ := testStore.CreateChat("Chat 1")
	user, _ := testStore.GetUserByUsername("user1")

	err := testStore.AddParticipant(int(chatID), user.ID)
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

	testStore.CreateUser("user1", "pass")
	chatID, _ := testStore.CreateChat("Chat 1")
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
