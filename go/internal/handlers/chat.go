package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/store"
	"github.com/pliu/chatty/internal/ws"
)

type ChatHandler struct {
	Store store.Store
	Hub   *ws.Hub
}

type CreateChatRequest struct {
	Name string `json:"name"`
}

type InviteUserRequest struct {
	Username string `json:"username"`
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)

	var req struct {
		Name         string `json:"name"`
		EncryptedKey string `json:"encrypted_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chatID, err := h.Store.CreateChat(req.Name, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.Store.AddParticipant(int(chatID), userID, req.EncryptedKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *ChatHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, _ := strconv.Atoi(vars["id"])
	// userID, _ := strconv.Atoi(getCookie(r, "user_id")) // Inviter's ID

	var req struct {
		Username     string `json:"username"`
		EncryptedKey string `json:"encrypted_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if user is already a participant
	isParticipant, err := h.Store.IsParticipant(chatID, user.ID)
	if err != nil {
		http.Error(w, "Failed to check participant status", http.StatusInternalServerError)
		return
	}
	if isParticipant {
		http.Error(w, "User is already a participant in this chat", http.StatusConflict)
		return
	}

	if err := h.Store.AddParticipant(chatID, user.ID, req.EncryptedKey); err != nil {
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}

	// Notify all participants in the chat to refresh their participants list
	participants, err := h.Store.GetChatParticipants(chatID)
	if err == nil {
		for _, participant := range participants {
			h.Hub.SendNotification(participant.ID, map[string]interface{}{
				"type": "new_chat",
			})
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)

	chats, err := h.Store.GetUserChats(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chats)
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, _ := strconv.Atoi(vars["id"])

	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isParticipant, err := h.Store.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	messages, err := h.Store.GetChatMessages(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func (h *ChatHandler) GetChatParticipants(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, _ := strconv.Atoi(vars["id"])

	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isParticipant, err := h.Store.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	participants, err := h.Store.GetChatParticipants(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(participants)
}

func getUserIDFromCookie(r *http.Request) int {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return 0
	}
	// In a real app, this would be a session ID lookup, not direct ID
	// For this demo, we're cheating a bit by storing the ID directly as a rune string?
	// Wait, in Login I did string(rune(user.ID)). That's probably not right for an int ID.
	// Let's fix Login to use strconv.Itoa and here strconv.Atoi

	// Actually, let's fix the helper function to assume it's a string of the int
	id, _ := strconv.Atoi(cookie.Value)
	return id
}
