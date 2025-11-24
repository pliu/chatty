package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pliu/chatty/internal/auth"
	"github.com/pliu/chatty/internal/handlers"
	"github.com/pliu/chatty/internal/middleware"
	"github.com/pliu/chatty/internal/store/sqlstore"
	"github.com/pliu/chatty/internal/ws"
)

var addr = flag.String("addr", ":8080", "http service address")
var httpsAddr = flag.String("https-addr", ":8443", "https service address")

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
	r.Use(middleware.LoggingMiddleware)

	// API Endpoints
	r.HandleFunc("/signup", authHandler.Signup).Methods("POST")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/users/search", authHandler.SearchUsers).Methods("GET")

	// Chat routes (protected)
	chatRouter := r.PathPrefix("/chats").Subrouter()
	chatRouter.Use(middleware.AuthMiddleware)
	chatRouter.HandleFunc("", chatHandler.CreateChat).Methods("POST")
	chatRouter.HandleFunc("", chatHandler.GetChats).Methods("GET")
	chatRouter.HandleFunc("/{id}/invite", chatHandler.InviteUser).Methods("POST")
	chatRouter.HandleFunc("/{id}/messages", chatHandler.GetChatMessages).Methods("GET")
	chatRouter.HandleFunc("/{id}/participants", chatHandler.GetChatParticipants).Methods("GET")
	chatRouter.HandleFunc("/{id}/leave", chatHandler.LeaveChat).Methods("DELETE")
	chatRouter.HandleFunc("/{id}", chatHandler.DeleteChat).Methods("DELETE")

	// WebSocket Endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from cookie
		cookie, err := r.Cookie("user_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse user ID
		userIDStr, err := auth.VerifyCookie(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID, _ := strconv.Atoi(userIDStr)

		ws.ServeWs(hub, w, r, userID)
	})

	// Serve index.html with cache-busting timestamp
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	// Serve static files with cache-busting headers for development
	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Disable caching for CSS and JS files in development
		if strings.HasSuffix(r.URL.Path, ".css") || strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}
		http.FileServer(http.Dir("static")).ServeHTTP(w, r)
	}))

	// Check if certs exist
	_, errCert := os.Stat("cert.pem")
	_, errKey := os.Stat("key.pem")
	if errCert == nil && errKey == nil {
		// HTTPS Mode
		// Start HTTP Redirect Server in goroutine
		go func() {
			log.Printf("Starting HTTP Redirect Server on %s -> %s", *addr, *httpsAddr)
			err := http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					host = r.Host
				}
				// Construct target URL
				// Assuming httpsAddr is like ":8443", we need to extract the port
				_, port, _ := net.SplitHostPort(*httpsAddr)
				target := "https://" + host + ":" + port + r.URL.Path
				if len(r.URL.RawQuery) > 0 {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			}))
			if err != nil {
				log.Printf("HTTP Redirect Server failed: %v", err)
			}
		}()

		log.Printf("Starting HTTPS Server on %s", *httpsAddr)
		log.Fatal(http.ListenAndServeTLS(*httpsAddr, "cert.pem", "key.pem", r))
	} else {
		// HTTP Mode
		log.Println("Starting server on", *addr)
		log.Fatal(http.ListenAndServe(*addr, r))
	}
}
