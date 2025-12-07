package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/auth"
	"github.com/pliu/chatty/internal/middleware"
	"github.com/pliu/chatty/internal/models"
	"github.com/pliu/chatty/internal/store/sqlstore"
	"github.com/pliu/chatty/internal/ws"
)

func TestCreateChat(t *testing.T) {
	store, _ := sqlstore.New("sqlite3", ":memory:")
	store.CreateUser(&models.User{Username: "user1", Email: "user1@example.com", Password: "pass"})
	user, _ := store.GetUserByUsername("user1")

	handler := &ChatHandler{Store: store}

	reqBody := map[string]string{"name": "Test Chat", "encrypted_key": "mock_key"}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/chats", bytes.NewBuffer(body))
	// Simulate logged-in user
	req.AddCookie(&http.Cookie{Name: "user_id", Value: auth.SignCookie(strconv.Itoa(user.ID))})

	rr := httptest.NewRecorder()
	middleware.AuthMiddleware(http.HandlerFunc(handler.CreateChat)).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	// Verify chat was created
	chats, _ := store.GetUserChats(user.ID)
	if len(chats) != 1 {
		t.Errorf("Expected 1 chat, got %d", len(chats))
	}
	if chats[0].Name != "Test Chat" {
		t.Errorf("Expected chat name 'Test Chat', got '%s'", chats[0].Name)
	}
}

func TestInviteUser(t *testing.T) {
	store, _ := sqlstore.New("sqlite3", ":memory:")
	store.CreateUser(&models.User{Username: "owner", Email: "owner@example.com", Password: "pass"})
	store.CreateUser(&models.User{Username: "invitee", Email: "invitee@example.com", Password: "pass"})

	chatID, _ := store.CreateChat("Test Chat", 1)
	owner, _ := store.GetUserByUsername("owner")
	store.AddParticipant(int(chatID), owner.ID, "key")

	// Mock Hub (or use real one, it's safe for tests if we don't attach clients)
	hub := ws.NewHub(store)
	go hub.Run()

	handler := &ChatHandler{Store: store, Hub: hub}

	reqBody := map[string]string{"username": "invitee", "encrypted_key": "mock_key_invitee"}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/chats/"+strconv.Itoa(int(chatID))+"/invite", bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(chatID))})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: auth.SignCookie(strconv.Itoa(owner.ID))})

	rr := httptest.NewRecorder()
	middleware.AuthMiddleware(http.HandlerFunc(handler.InviteUser)).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify invitee is now a participant
	invitee, _ := store.GetUserByUsername("invitee")
	isParticipant, _ := store.IsParticipant(int(chatID), invitee.ID)
	if !isParticipant {
		t.Error("Expected invitee to be a participant")
	}
}

func TestGetChats(t *testing.T) {
	store, _ := sqlstore.New("sqlite3", ":memory:")
	store.CreateUser(&models.User{Username: "user1", Email: "user1@example.com", Password: "pass"})
	user, _ := store.GetUserByUsername("user1")

	_, _ = store.CreateChat("Chat 1", 1)
	_, _ = store.CreateChat("Chat 2", 1)
	// Add user to Chat 1 only
	store.GetUserChats(user.ID) // Should be 0 initially

	chatID, _ := store.CreateChat("My Chat", 1)
	store.AddParticipant(int(chatID), user.ID, "key")

	handler := &ChatHandler{Store: store}

	req, _ := http.NewRequest("GET", "/chats", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: auth.SignCookie(strconv.Itoa(user.ID))})

	rr := httptest.NewRecorder()
	middleware.AuthMiddleware(http.HandlerFunc(handler.GetChats)).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var responseChats []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&responseChats)

	if len(responseChats) != 1 {
		t.Errorf("Expected 1 chat, got %d", len(responseChats))
	}
}
