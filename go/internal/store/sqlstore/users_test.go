package sqlstore

import (
	"testing"

	"github.com/pliu/chatty/internal/models"
)

func TestCreateUser(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	err := testStore.CreateUser(&models.User{Username: "testuser", Password: "password123"})
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}

	// Test duplicate user
	err = testStore.CreateUser(&models.User{Username: "testuser", Password: "password123"})
	if err == nil {
		t.Error("Expected error when creating duplicate user, got nil")
	}
}

func TestCreateUserWithKeys(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	// Simulate realistic keys
	// Public key: 32 bytes -> base64
	publicKey := "MCowBQYDK2VuAyEA6g9..." // truncated for brevity, just needs to be a string
	// Encrypted private key: Salt(16) + Nonce(24) + Ciphertext(32+16 tag) = ~88 bytes -> base64
	encryptedPrivateKey := "..."

	user := &models.User{
		Username:            "keyuser",
		Password:            "password123",
		PublicKey:           publicKey,
		EncryptedPrivateKey: encryptedPrivateKey,
	}

	err := testStore.CreateUser(user)
	if err != nil {
		t.Errorf("Failed to create user with keys: %v", err)
	}

	storedUser, err := testStore.GetUserByUsername("keyuser")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if storedUser.PublicKey != publicKey {
		t.Errorf("Expected public key %s, got %s", publicKey, storedUser.PublicKey)
	}
	if storedUser.EncryptedPrivateKey != encryptedPrivateKey {
		t.Errorf("Expected encrypted private key %s, got %s", encryptedPrivateKey, storedUser.EncryptedPrivateKey)
	}
}

func TestGetUserByUsername(t *testing.T) {
	SetupTestDB(t)
	defer TeardownTestDB()

	testStore.CreateUser(&models.User{Username: "testuser", Password: "password123"})

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

	testStore.CreateUser(&models.User{Username: "alice", Password: "pass"})
	testStore.CreateUser(&models.User{Username: "bob", Password: "pass"})
	testStore.CreateUser(&models.User{Username: "alex", Password: "pass"})

	users, err := testStore.SearchUsers("al")
	if err != nil {
		t.Errorf("SearchUsers failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}
