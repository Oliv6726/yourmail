package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"yourmail/config"
	"yourmail/internal/auth"
	"yourmail/internal/database"
	"yourmail/internal/federation"

	"github.com/gorilla/mux"
)

// SSEClient represents a Server-Sent Events client
type SSEClient struct {
	userID   int
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan bool
	lastPing time.Time
}

// Server represents the HTTP API server
type Server struct {
	config      *config.Config
	db          *database.DB
	userRepo    *database.UserRepository
	messageRepo *database.MessageRepository
	jwtService  *auth.JWTService
	relay       *federation.Relay
	
	// SSE client management
	sseClients    map[int][]*SSEClient // userID -> clients
	sseMutex      sync.RWMutex
	sseCloseChan  chan *SSEClient
}

// NewServer creates a new HTTP API server
func NewServer(cfg *config.Config, db *database.DB, relay *federation.Relay) *Server {
	server := &Server{
		config:       cfg,
		db:           db,
		userRepo:     database.NewUserRepository(db),
		messageRepo:  database.NewMessageRepository(db),
		jwtService:   auth.NewJWTService(cfg.JWTSecret, "yourmail"),
		relay:        relay,
		sseClients:   make(map[int][]*SSEClient),
		sseCloseChan: make(chan *SSEClient, 100),
	}
	
	// Start SSE client cleanup goroutine
	go server.cleanupSSEClients()
	
	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := mux.NewRouter()

	// CORS middleware
	router.Use(s.corsMiddleware)

	// Public routes (no auth required)
	router.HandleFunc("/api/register", s.handleRegister).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/login", s.handleLogin).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Protected routes (JWT auth required)
	router.HandleFunc("/api/messages", s.jwtService.AuthMiddleware(s.handleGetMessages)).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/messages/sent", s.jwtService.AuthMiddleware(s.handleGetSentMessages)).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/messages/unread-count", s.jwtService.AuthMiddleware(s.handleGetUnreadCount)).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/messages/{id}/read", s.jwtService.AuthMiddleware(s.handleMarkAsRead)).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/send", s.jwtService.AuthMiddleware(s.handleSendMessage)).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/profile", s.jwtService.AuthMiddleware(s.handleGetProfile)).Methods("GET", "OPTIONS")

	// Server-Sent Events for real-time updates
	router.HandleFunc("/api/sse/inbox", s.handleSSEInbox).Methods("GET")

	// Federation routes (for server-to-server communication)
	router.HandleFunc("/federation/relay", s.handleFederationRelay).Methods("POST")

	log.Printf("🚀 HTTP API server starting on :%s", s.config.HTTPPort)
	return http.ListenAndServe(":"+s.config.HTTPPort, router)
}

// CORS middleware
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.AllowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req database.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if len(req.Username) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}
	if !strings.Contains(req.Email, "@") {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existing, _ := s.userRepo.GetByUsername(req.Username)
	if existing != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	existing, _ = s.userRepo.GetByEmail(req.Email)
	if existing != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Create user
	user, err := s.userRepo.Create(req.Username, req.Email, req.Password)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := database.LoginResponse{
		Success: true,
		Message: "User created successfully",
		Token:   token,
		User:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req database.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := s.userRepo.Authenticate(req.Username, req.Password)
	if err != nil {
		log.Printf("Authentication error: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	if user == nil {
		response := database.LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := database.LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetProfile returns the current user's profile
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Get full user details from database
	fullUser, err := s.userRepo.GetByID(user.ID)
	if err != nil {
		log.Printf("Failed to get user profile: %v", err)
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullUser)
}

// handleGetMessages returns messages for the authenticated user
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := s.messageRepo.GetInboxForUser(user.ID, limit, offset)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, never null
	if messages == nil {
		messages = []*database.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleGetSentMessages returns sent messages for the authenticated user
func (s *Server) handleGetSentMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := s.messageRepo.GetSentForUser(user.ID, limit, offset)
	if err != nil {
		log.Printf("Failed to get sent messages: %v", err)
		http.Error(w, "Failed to get sent messages", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, never null
	if messages == nil {
		messages = []*database.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleGetUnreadCount returns the count of unread messages
func (s *Server) handleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	count, err := s.messageRepo.GetUnreadCount(user.ID)
	if err != nil {
		log.Printf("Failed to get unread count: %v", err)
		http.Error(w, "Failed to get unread count", http.StatusInternalServerError)
		return
	}

	response := map[string]int{"unread_count": count}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMarkAsRead marks a message as read
func (s *Server) handleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	messageID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// Get message to verify ownership
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		log.Printf("Failed to get message: %v", err)
		http.Error(w, "Failed to get message", http.StatusInternalServerError)
		return
	}

	if message == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Verify user owns this message (is the recipient)
	if message.ToUserID == nil || *message.ToUserID != user.ID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	err = s.messageRepo.MarkAsRead(messageID)
	if err != nil {
		log.Printf("Failed to mark message as read: %v", err)
		http.Error(w, "Failed to mark message as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// handleSendMessage handles sending messages
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.To == "" || req.Subject == "" {
		http.Error(w, "To and Subject are required", http.StatusBadRequest)
		return
	}

	// Create from address
	fromAddress := fmt.Sprintf("%s@%s", user.Username, s.config.ServerHost)

	// Check if recipient is local or external
	var toUserID *int
	if strings.Contains(req.To, "@") {
		parts := strings.Split(req.To, "@")
		if len(parts) == 2 && parts[1] == s.config.ServerHost {
			// Local user
			localUser, err := s.userRepo.GetByUsername(parts[0])
			if err != nil {
				log.Printf("Failed to lookup local user: %v", err)
			} else if localUser != nil {
				toUserID = &localUser.ID
			}
		}
	}

	// Store message in database
	message, err := s.messageRepo.Create(&user.ID, toUserID, fromAddress, req.To, req.Subject, req.Body)
	if err != nil {
		log.Printf("Failed to store message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Notify SSE clients if it's a local message
	if toUserID != nil {
		go s.notifyNewMessage(message)
	}

	// If external recipient, try federation
	if toUserID == nil && strings.Contains(req.To, "@") {
		parts := strings.Split(req.To, "@")
		if len(parts) == 2 {
			err := s.relay.SendMessage(fromAddress, req.To, req.Subject, req.Body, parts[1])
			if err != nil {
				log.Printf("Federation send failed: %v", err)
				// Don't fail the whole request - message is stored locally
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message sent successfully",
		"id":      message.ID,
	})
}

// handleFederationRelay handles incoming federation messages
func (s *Server) handleFederationRelay(w http.ResponseWriter, r *http.Request) {
	var msg federation.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find recipient user
	if !strings.Contains(msg.To, "@") {
		http.Error(w, "Invalid recipient format", http.StatusBadRequest)
		return
	}

	parts := strings.Split(msg.To, "@")
	if len(parts) != 2 || parts[1] != s.config.ServerHost {
		http.Error(w, "Recipient not on this server", http.StatusBadRequest)
		return
	}

	user, err := s.userRepo.GetByUsername(parts[0])
	if err != nil {
		log.Printf("Failed to lookup user: %v", err)
		http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Store message
	_, err = s.messageRepo.Create(nil, &user.ID, msg.From, msg.To, msg.Subject, msg.Body)
	if err != nil {
		log.Printf("Failed to store federated message: %v", err)
		http.Error(w, "Failed to store message", http.StatusInternalServerError)
		return
	}

	// Notify SSE clients about the new federated message
	go func() {
		// Create a message object for notification
		newMsg := &database.Message{
			FromUserID:  nil,
			ToUserID:    &user.ID,
			FromAddress: msg.From,
			ToAddress:   msg.To,
			Subject:     msg.Subject,
			Body:        msg.Body,
			ReadStatus:  false,
			CreatedAt:   time.Now(),
		}
		s.notifyNewMessage(newMsg)
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "delivered"})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "2.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// cleanupSSEClients manages SSE client connections and removes dead ones
func (s *Server) cleanupSSEClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.pingSSEClients()
		case client := <-s.sseCloseChan:
			s.removeSSEClient(client)
		}
	}
}

// pingSSEClients sends ping messages to keep connections alive
func (s *Server) pingSSEClients() {
	s.sseMutex.Lock()
	defer s.sseMutex.Unlock()

	for userID, clients := range s.sseClients {
		var activeClients []*SSEClient
		for _, client := range clients {
			// Send ping
			if s.sendSSEPing(client) {
				activeClients = append(activeClients, client)
			}
		}
		s.sseClients[userID] = activeClients
		if len(activeClients) == 0 {
			delete(s.sseClients, userID)
		}
	}
}

// sendSSEPing sends a ping message to an SSE client
func (s *Server) sendSSEPing(client *SSEClient) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SSE ping failed for user %d: %v", client.userID, r)
		}
	}()

	_, err := fmt.Fprintf(client.writer, ": ping\n\n")
	if err != nil {
		return false
	}
	client.flusher.Flush()
	client.lastPing = time.Now()
	return true
}

// removeSSEClient removes a client from the SSE client list
func (s *Server) removeSSEClient(client *SSEClient) {
	s.sseMutex.Lock()
	defer s.sseMutex.Unlock()

	clients := s.sseClients[client.userID]
	for i, c := range clients {
		if c == client {
			// Remove client from slice
			s.sseClients[client.userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(s.sseClients[client.userID]) == 0 {
		delete(s.sseClients, client.userID)
	}

	// Close client's done channel
	close(client.done)
}

// handleSSEInbox handles Server-Sent Events for inbox updates
func (s *Server) handleSSEInbox(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter (since EventSource can't send custom headers)
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers with proper CORS
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Create SSE client
	client := &SSEClient{
		userID:   claims.UserID,
		writer:   w,
		flusher:  flusher,
		done:     make(chan bool),
		lastPing: time.Now(),
	}

	// Add client to the list
	s.sseMutex.Lock()
	s.sseClients[client.userID] = append(s.sseClients[client.userID], client)
	s.sseMutex.Unlock()

	// Send initial unread count
	go func() {
		count, err := s.messageRepo.GetUnreadCount(client.userID)
		if err == nil {
			s.sendSSEEvent(client, "unread-count", map[string]int{"count": count})
		}
	}()

	// Send welcome message
	s.sendSSEEvent(client, "connected", map[string]string{"message": "Connected to inbox updates"})

	// Wait for client disconnect or server shutdown
	select {
	case <-client.done:
		log.Printf("SSE client disconnected for user %d", client.userID)
	case <-r.Context().Done():
		log.Printf("SSE client context cancelled for user %d", client.userID)
		s.sseCloseChan <- client
	}
}

// sendSSEEvent sends an event to an SSE client
func (s *Server) sendSSEEvent(client *SSEClient, eventType string, data interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SSE send failed for user %d: %v", client.userID, r)
		}
	}()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal SSE data: %v", err)
		return
	}

	_, err = fmt.Fprintf(client.writer, "event: %s\ndata: %s\n\n", eventType, jsonData)
	if err != nil {
		log.Printf("Failed to write SSE event: %v", err)
		return
	}

	client.flusher.Flush()
}

// notifyNewMessage notifies all SSE clients about a new message
func (s *Server) notifyNewMessage(message *database.Message) {
	if message.ToUserID == nil {
		return // External message, no local recipient to notify
	}

	s.sseMutex.RLock()
	clients := s.sseClients[*message.ToUserID]
	s.sseMutex.RUnlock()

	for _, client := range clients {
		go s.sendSSEEvent(client, "new-message", message)
		
		// Also send updated unread count
		go func(c *SSEClient) {
			count, err := s.messageRepo.GetUnreadCount(c.userID)
			if err == nil {
				s.sendSSEEvent(c, "unread-count", map[string]int{"count": count})
			}
		}(client)
	}
} 