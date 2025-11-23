package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pliu/chatty/internal/auth"
	"github.com/pliu/chatty/internal/models"
	"github.com/pliu/chatty/internal/store/sqlstore"
	"golang.org/x/crypto/bcrypt"
)

func TestSignup(t *testing.T) {
	// Initialize DB for testing
	store, err := sqlstore.New("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	handler := &AuthHandler{Store: store}

	creds := map[string]string{
		"username":              "testuser",
		"password":              "password123",
		"public_key":            "mock_public_key",
		"encrypted_private_key": "mock_private_key",
	}
	body, _ := json.Marshal(creds)

	req, err := http.NewRequest("POST", "/signup", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.Signup).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	// Verify user was created with keys
	user, _ := store.GetUserByUsername("testuser")
	if user.PublicKey != "mock_public_key" {
		t.Errorf("Expected public key 'mock_public_key', got '%s'", user.PublicKey)
	}
	if user.EncryptedPrivateKey != "mock_private_key" {
		t.Errorf("Expected encrypted private key 'mock_private_key', got '%s'", user.EncryptedPrivateKey)
	}

	// Test duplicate user
	req, _ = http.NewRequest("POST", "/signup", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	http.HandlerFunc(handler.Signup).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code for duplicate user: got %v want %v",
			status, http.StatusConflict)
	}
}

func TestLogin(t *testing.T) {
	store, _ := sqlstore.New("sqlite3", ":memory:")
	handler := &AuthHandler{Store: store}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	store.CreateUser(&models.User{
		Username:            "testuser",
		Password:            string(hashedPassword),
		PublicKey:           "mock_public_key",
		EncryptedPrivateKey: "mock_private_key",
	})

	creds := Credentials{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(creds)

	req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.Login).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check cookies
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Expected cookies to be set")
	}

	var userIDCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "user_id" {
			userIDCookie = c
			break
		}
	}

	if userIDCookie == nil {
		t.Error("Expected user_id cookie to be set")
	} else {
		// Verify signature
		_, err := auth.VerifyCookie(userIDCookie.Value)
		if err != nil {
			t.Errorf("Cookie verification failed: %v", err)
		}
	}

	// Verify response body contains keys
	var user models.User
	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}

	if user.PublicKey != "mock_public_key" {
		t.Errorf("Expected public key 'mock_public_key', got '%s'", user.PublicKey)
	}
	if user.EncryptedPrivateKey != "mock_private_key" {
		t.Errorf("Expected encrypted private key 'mock_private_key', got '%s'", user.EncryptedPrivateKey)
	}
}
