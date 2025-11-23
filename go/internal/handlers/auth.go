package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pliu/chatty/internal/models"
	"github.com/pliu/chatty/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthHandler struct {
	Store store.Store
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	type SignupRequest struct {
		Username            string `json:"username"`
		Password            string `json:"password"`
		PublicKey           string `json:"public_key"`
		EncryptedPrivateKey string `json:"encrypted_private_key"`
	}

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user := &models.User{
		Username:            req.Username,
		Password:            string(hashedPassword),
		PublicKey:           req.PublicKey,
		EncryptedPrivateKey: req.EncryptedPrivateKey,
	}

	if err := h.Store.CreateUser(user); err != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.GetUserByUsername(creds.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie (simplified for demo)
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: strconv.Itoa(user.ID), // Insecure: exposing ID directly. In prod use sessions.
		// Expires: time.Now().Add(24 * time.Hour), // Session cookie
		Path: "/",
	})

	// Also setting a username cookie for frontend convenience
	http.SetCookie(w, &http.Cookie{
		Name:  "username",
		Value: user.Username,
		// Expires: time.Now().Add(24 * time.Hour), // Session cookie
		Path: "/",
	})

	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	users, err := h.Store.SearchUsers(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}
