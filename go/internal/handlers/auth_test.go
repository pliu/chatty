package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

	creds := Credentials{
		Username: "testuser",
		Password: "password123",
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
	store.CreateUser("testuser", string(hashedPassword))

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
}
