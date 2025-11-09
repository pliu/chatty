package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pliu/chatty/internal/database"
)

// Message represents a message received from a client.
// For broadcasting, a more complete message (like database.Message) is constructed.
type Message struct {
	ChatID  int    `json:"chat_id"`
	UserID  int    `json:"user_id"`
	Content string `json:"content"`
}

type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			// Save message to DB
			err := database.SaveMessage(message.ChatID, message.UserID, message.Content)
			if err != nil {
				log.Printf("Error saving message: %v", err)
				continue
			}

			// Broadcast to clients in the same chat
			for client := range h.clients {
				// Check if client is participant of the chat
				isParticipant, err := database.IsParticipant(message.ChatID, client.userID)
				if err != nil {
					log.Printf("Error checking participant: %v", err)
					continue
				}
				if isParticipant {
					// Fetch full message details including username and timestamp
					// For simplicity, we just send what we have + username if possible,
					// but better to re-fetch or construct the full object.
					// Let's construct a response object.
					user, _ := database.GetUserByID(message.UserID)
					response := database.Message{
						ChatID:    message.ChatID,
						UserID:    message.UserID,
						Username:  user.Username,
						Content:   message.Content,
						CreatedAt: time.Now(),
					}

					msgBytes, _ := json.Marshal(response)

					select {
					case client.send <- msgBytes:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) SendNotification(userID int, message interface{}) {
	msgBytes, _ := json.Marshal(message)
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- msgBytes:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}
