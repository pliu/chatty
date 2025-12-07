package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pliu/chatty/internal/auth"
	"github.com/pliu/chatty/internal/models"
	"github.com/pliu/chatty/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthHandler struct {
	Store store.Store
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	type SignupRequest struct {
		Username            string `json:"username"`
		Email               string `json:"email"`
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

	// Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	verificationToken := hex.EncodeToString(tokenBytes)

	user := &models.User{
		Username:            req.Username,
		Email:               req.Email,
		Password:            string(hashedPassword),
		PublicKey:           req.PublicKey,
		EncryptedPrivateKey: req.EncryptedPrivateKey,
		IsVerified:          false,
		VerificationToken:   verificationToken,
	}

	if err := h.Store.CreateUser(user); err != nil {
		// Could be username or email conflict
		http.Error(w, "Username or Email already exists", http.StatusConflict)
		return
	}

	// Log the verification link
	fmt.Printf("VERIFICATION LINK: http://%s/verify?token=%s\n", r.Host, verificationToken)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created. Please check your email (server logs) to verify account.",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.GetUserByEmail(creds.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.IsVerified {
		http.Error(w, "Account not verified. Please check your email.", http.StatusForbidden)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie (signed)
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: auth.SignCookie(strconv.Itoa(user.ID)),
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

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	if err := h.Store.VerifyUser(token); err != nil {
		http.Error(w, "Invalid or expired token", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, "static/verify_success.html")
}
