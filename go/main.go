package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/database"
	"github.com/pliu/chatty/internal/handlers"
	"github.com/pliu/chatty/internal/ws"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize Database
	database.InitDB("./chatty.db")

	// Initialize WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	r := mux.NewRouter()

	// API Endpoints
	r.HandleFunc("/signup", handlers.Signup).Methods("POST")
	r.HandleFunc("/login", handlers.Login).Methods("POST")
	r.HandleFunc("/users/search", handlers.SearchUsers).Methods("GET")
	r.HandleFunc("/chats", handlers.CreateChat).Methods("POST")
	r.HandleFunc("/chats", handlers.GetChats).Methods("GET")
	r.HandleFunc("/chats/{id}/invite", handlers.InviteUser(hub)).Methods("POST")
	r.HandleFunc("/chats/{id}/messages", handlers.GetChatMessages).Methods("GET")

	// WebSocket Endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from cookie
		cookie, err := r.Cookie("user_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse user ID
		var userID int
		// We need to use strconv.Atoi
		// Since I forgot to add strconv to imports in the previous step, I will add it now.
		// But for now let's just assume I'll fix imports in next step.
		// Actually I can't use strconv if not imported.
		// I'll use a helper or just do it in next step.
		// Let's just write the code assuming strconv is there, and then add the import.
		userID, _ = strconv.Atoi(cookie.Value)

		ws.ServeWs(hub, w, r, userID)
	})

	// Serve static files (frontend)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("Starting server on", *addr)
	log.Fatal(http.ListenAndServe(*addr, r))
}
