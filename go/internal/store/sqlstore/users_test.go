package sqlstore

import (
	"testing"
)

func TestCreateUser(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	err := testStore.CreateUser("testuser", "password123")
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}

	// Test duplicate user
	err = testStore.CreateUser("testuser", "password123")
	if err == nil {
		t.Error("Expected error when creating duplicate user, got nil")
	}
}

func TestGetUserByUsername(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser("testuser", "password123")

	user, err := testStore.GetUserByUsername("testuser")
	if err != nil {
		t.Errorf("Failed to get user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	_, err = testStore.GetUserByUsername("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent user, got nil")
	}
}

func TestSearchUsers(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser("alice", "pass")
	testStore.CreateUser("bob", "pass")
	testStore.CreateUser("alex", "pass")

	users, err := testStore.SearchUsers("al")
	if err != nil {
		t.Errorf("SearchUsers failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}
