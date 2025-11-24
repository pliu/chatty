# Chatty - End-to-End Encrypted Chat Application

A real-time chat application with end-to-end encryption, built with Go and vanilla JavaScript.

## Features

### 🔐 Security
- **End-to-End Encryption**: Messages encrypted with AES-GCM using chat-specific symmetric keys
- **Asymmetric Key Exchange**: ECDH (P-256) for secure chat key distribution
- **Password-Based Encryption**: User private keys encrypted with Argon2id-derived keys
- **Signed Cookies**: HMAC-SHA256 signed authentication cookies
- **HTTPS Support**: TLS encryption for all communications

### 💬 Chat Features
- **Real-time Messaging**: WebSocket-based instant messaging
- **Multiple Chats**: Create and participate in multiple chat rooms
- **User Invitations**: Invite users to existing chats with encrypted key sharing
- **Participant Management**:
  - Owners can remove participants
  - Non-owners can leave chats
  - Real-time participant list updates
- **Chat Ownership**: 
  - Owners can delete entire chats
  - Participants can only leave
- **Message Persistence**: Messages remain even after users leave

### 👥 User Management
- User registration and authentication
- Bcrypt password hashing
- User search functionality
- Session management

## Architecture

### Backend (Go)
- **Framework**: Gorilla Mux for routing, Gorilla WebSocket for real-time communication
- **Database**: PostgreSQL with prepared statements
- **Authentication**: Cookie-based sessions with HMAC signing
- **Middleware**: Logging, authentication, and authorization
- **Clean Architecture**: Separated handlers, store, models, and middleware layers

### Frontend (Vanilla JavaScript)
- **No Framework**: Pure JavaScript, HTML, and CSS
- **Web Crypto API**: Client-side encryption/decryption
- **WebSocket**: Real-time updates for messages and notifications
- **Responsive Design**: Modern, clean UI with dark mode aesthetics

### Database Schema
```sql
users (
  id SERIAL PRIMARY KEY,
  username TEXT UNIQUE,
  password TEXT,
  public_key TEXT,
  encrypted_private_key TEXT
)

chats (
  id SERIAL PRIMARY KEY,
  name TEXT,
  owner_id INTEGER REFERENCES users(id)
)

participants (
  chat_id INTEGER REFERENCES chats(id),
  user_id INTEGER REFERENCES users(id),
  encrypted_chat_key TEXT,
  PRIMARY KEY (chat_id, user_id)
)

messages (
  id SERIAL PRIMARY KEY,
  chat_id INTEGER REFERENCES chats(id),
  user_id INTEGER REFERENCES users(id),
  encrypted_content TEXT,
  timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for convenience commands)

## Getting Started

### 1. Clone the Repository
```bash
git clone <repository-url>
cd chatty
```

### 2. Start the Database
```bash
make docker-up
```

This starts a PostgreSQL container with the following credentials:
- **Host**: localhost:5432
- **Database**: chatty
- **User**: user
- **Password**: password

### 3. Run the Application
```bash
make run
```

The server will start on:
- **HTTPS**: https://localhost:8443
- **HTTP**: http://localhost:8080 (redirects to HTTPS)

### 4. Access the Application
Open your browser and navigate to:
```
https://localhost:8443
```

**Note**: You'll see a certificate warning because the app uses self-signed certificates. Click "Advanced" and proceed to the site.

## Development

### Available Make Commands
```bash
make run          # Run the application
make test         # Run all tests
make build        # Build the binary
make clean        # Clean build artifacts
make docker-up    # Start PostgreSQL container
make docker-down  # Stop PostgreSQL container
```

### Project Structure
```
chatty/
├── go/
│   ├── main.go                    # Application entry point
│   ├── internal/
│   │   ├── auth/                  # Cookie signing/verification
│   │   ├── handlers/              # HTTP handlers
│   │   │   ├── auth.go           # Login/signup
│   │   │   └── chat.go           # Chat operations
│   │   ├── middleware/            # HTTP middleware
│   │   │   ├── auth.go           # Authentication
│   │   │   └── logging.go        # Request logging
│   │   ├── models/                # Data models
│   │   ├── store/                 # Data access layer
│   │   │   └── sqlstore/         # PostgreSQL implementation
│   │   └── ws/                    # WebSocket hub and clients
│   └── static/
│       ├── index.html             # Main HTML
│       ├── app.js                 # Frontend logic
│       └── style.css              # Styling
├── docker-compose.yml             # PostgreSQL setup
└── Makefile                       # Build commands
```

### Running Tests
```bash
make test
```

Tests include:
- Handler tests (auth, chat operations)
- Middleware tests (auth, logging)
- WebSocket hub tests
- Store tests (database operations)

## Security Considerations

### Encryption Flow
1. **User Signup**:
   - Generate ECDH key pair (P-256)
   - Encrypt private key with password-derived key (Argon2id)
   - Store public key and encrypted private key in database

2. **User Login**:
   - Decrypt private key with password
   - Store private key in memory for session

3. **Chat Creation**:
   - Generate random 256-bit symmetric key
   - Encrypt with creator's public key
   - Store encrypted key in participants table

4. **User Invitation**:
   - Retrieve invitee's public key
   - Encrypt chat symmetric key with invitee's public key
   - Store in participants table

5. **Message Encryption**:
   - Encrypt message with chat's symmetric key (AES-GCM)
   - Send encrypted content to server
   - Server stores encrypted content (cannot decrypt)

6. **Message Decryption**:
   - Retrieve encrypted message from server
   - Decrypt with chat's symmetric key
   - Display plaintext to user

### Known Limitations
- Private keys stored in browser memory (lost on page refresh)
- Self-signed certificates for development
- No key rotation mechanism
- No forward secrecy for messages
- Database stores encrypted data but server has access to metadata

## Production Deployment

### Environment Variables
Before deploying to production, configure:

```bash
export SECRET_KEY="your-secure-random-key-here"
export DB_HOST="your-db-host"
export DB_PORT="5432"
export DB_USER="your-db-user"
export DB_PASSWORD="your-db-password"
export DB_NAME="chatty"
```

### TLS Certificates
Replace self-signed certificates with proper TLS certificates:
- Place `server.crt` and `server.key` in the `go/` directory
- Or configure Let's Encrypt with a reverse proxy

### Database
- Use a managed PostgreSQL instance
- Enable SSL/TLS connections
- Configure backups and replication
- Set up connection pooling

### Security Hardening
- [ ] Use environment variables for secrets
- [ ] Implement rate limiting
- [ ] Add CSRF protection
- [ ] Configure CORS properly
- [ ] Enable security headers (CSP, HSTS, etc.)
- [ ] Implement key rotation
- [ ] Add audit logging
- [ ] Set up monitoring and alerts

## API Endpoints

### Authentication
- `POST /signup` - Register new user
- `POST /login` - Authenticate user
- `GET /users/search?q=<query>` - Search users

### Chats
- `GET /chats` - List user's chats
- `POST /chats` - Create new chat
- `DELETE /chats/{id}` - Delete chat (owner only)
- `DELETE /chats/{id}/leave` - Leave chat (non-owners)
- `POST /chats/{id}/invite` - Invite user to chat
- `GET /chats/{id}/messages` - Get chat messages
- `GET /chats/{id}/participants` - Get chat participants
- `DELETE /chats/{id}/participants/{userID}` - Remove participant (owner only)

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

## WebSocket Events

### Client → Server
- Chat messages (encrypted)

### Server → Client
- `new_chat` - New chat created or user invited
- `chat_deleted` - Chat was deleted
- `participant_left` - User left or was removed
- `removed_from_chat` - Current user was removed
- Message broadcasts (encrypted)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[Your License Here]

## Acknowledgments

- Built with Go and vanilla JavaScript
- Uses Gorilla WebSocket and Gorilla Mux
- PostgreSQL for data persistence
- Web Crypto API for client-side encryption
