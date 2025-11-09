package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/database"
	"github.com/pliu/chatty/internal/ws"
)

func CreateChat(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chatID, err := database.CreateChat(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := database.AddParticipant(int(chatID), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": chatID})
}

func InviteUser(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		chatID, _ := strconv.Atoi(vars["id"])

		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := database.GetUserByUsername(req.Username)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if err := database.AddParticipant(chatID, user.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Notify the user
		hub.SendNotification(user.ID, map[string]string{
			"type": "new_chat",
		})

		w.WriteHeader(http.StatusOK)
	}
}

func GetChats(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chats, err := database.GetUserChats(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chats)
}

func GetChatMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, _ := strconv.Atoi(vars["id"])

	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isParticipant, err := database.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	messages, err := database.GetChatMessages(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
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
