package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/handlers"
	"github.com/pliu/chatty/internal/store/sqlstore"
	"github.com/pliu/chatty/internal/ws"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize Database
	// Connect to Postgres (running via docker-compose)
	connStr := "user=user password=password dbname=chatty sslmode=disable host=localhost port=5432"
	store, err := sqlstore.New("postgres", connStr)
	// store, err := sqlstore.New("sqlite3", "chatty.db")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize WebSocket Hub
	hub := ws.NewHub(store)
	go hub.Run()

	// Initialize Handlers
	authHandler := &handlers.AuthHandler{Store: store}
	chatHandler := &handlers.ChatHandler{Store: store, Hub: hub}

	r := mux.NewRouter()
	r.Use(loggingMiddleware)

	// API Endpoints
	r.HandleFunc("/signup", authHandler.Signup).Methods("POST")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/users/search", authHandler.SearchUsers).Methods("GET")
	r.HandleFunc("/chats", chatHandler.CreateChat).Methods("POST")
	r.HandleFunc("/chats", chatHandler.GetChats).Methods("GET")
	r.HandleFunc("/chats/{id}/invite", chatHandler.InviteUser).Methods("POST")
	r.HandleFunc("/chats/{id}/messages", chatHandler.GetChatMessages).Methods("GET")

	// WebSocket Endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from cookie
		cookie, err := r.Cookie("user_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse user ID
		userID, _ := strconv.Atoi(cookie.Value)

		ws.ServeWs(hub, w, r, userID)
	})

	// Serve static files (frontend)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("Starting server on", *addr)
	log.Fatal(http.ListenAndServe(*addr, r))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
