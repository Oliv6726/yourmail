package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
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
	config         *config.Config
	db             *database.DB
	userRepo       *database.UserRepository
	messageRepo    *database.MessageRepository
	attachmentRepo *database.AttachmentRepository
	jwtService     *auth.JWTService
	relay          *federation.Relay
	
	// SSE client management
	sseClients    map[int][]*SSEClient // userID -> clients
	sseMutex      sync.RWMutex
	sseCloseChan  chan *SSEClient
}

// NewServer creates a new HTTP API server
func NewServer(cfg *config.Config, db *database.DB, relay *federation.Relay) *Server {
	server := &Server{
		config:         cfg,
		db:             db,
		userRepo:       database.NewUserRepository(db),
		messageRepo:    database.NewMessageRepository(db),
		attachmentRepo: database.NewAttachmentRepository(db),
		jwtService:     auth.NewJWTService(cfg.JWTSecret, "yourmail"),
		relay:          relay,
		sseClients:     make(map[int][]*SSEClient),
		sseCloseChan:   make(chan *SSEClient, 100),
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
	
	// Threading routes
	router.HandleFunc("/api/threads/{threadId}", s.jwtService.AuthMiddleware(s.handleGetThread)).Methods("GET", "OPTIONS")
	
	// Attachment routes
	router.HandleFunc("/api/attachments/{id}", s.jwtService.AuthMiddleware(s.handleGetAttachment)).Methods("GET", "OPTIONS")

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
	w.Header().Set("Content-Type", "application/json")
	
	var req database.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid_json",
			"message": fmt.Sprintf("Failed to parse JSON request: %v", err),
		})
		return
	}

	// Basic validation
	if len(req.Username) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "username_too_short",
			"message": "Username must be at least 3 characters",
		})
		return
	}
	if len(req.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "password_too_short",
			"message": "Password must be at least 6 characters",
		})
		return
	}
	if !strings.Contains(req.Email, "@") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid_email",
			"message": "Invalid email format",
		})
		return
	}

	// Check if user already exists
	existing, _ := s.userRepo.GetByUsername(req.Username)
	if existing != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "username_exists",
			"message": "Username already exists",
		})
		return
	}

	existing, _ = s.userRepo.GetByEmail(req.Email)
	if existing != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "email_exists",
			"message": "Email already exists",
		})
		return
	}

	// Create user
	user, err := s.userRepo.Create(req.Username, req.Email, req.Password)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "user_creation_failed",
			"message": fmt.Sprintf("Failed to create user: %v", err),
		})
		return
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "token_generation_failed",
			"message": fmt.Sprintf("Failed to generate token: %v", err),
		})
		return
	}

	response := database.LoginResponse{
		Success: true,
		Message: "User created successfully",
		Token:   token,
		User:    user,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req database.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid_json",
			"message": fmt.Sprintf("Failed to parse JSON request: %v", err),
		})
		return
	}

	// Authenticate user
	user, err := s.userRepo.Authenticate(req.Username, req.Password)
	if err != nil {
		log.Printf("Authentication error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "authentication_error",
			"message": fmt.Sprintf("Authentication failed: %v", err),
		})
		return
	}

	if user == nil {
		response := database.LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "token_generation_failed",
			"message": fmt.Sprintf("Failed to generate token: %v", err),
		})
		return
	}

	response := database.LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    user,
	}

	w.WriteHeader(http.StatusOK)
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
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	IsHTML   bool   `json:"is_html"`
	ThreadID string `json:"thread_id"`
	ParentID int    `json:"parent_id"`
}

// isValidEmail checks if an email address is valid, allowing localhost domains
func isValidEmail(email string) bool {
	// Must contain exactly one @
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	localPart := parts[0]
	domain := parts[1]
	
	// Local part must not be empty
	if localPart == "" {
		return false
	}
	
	// Domain must not be empty
	if domain == "" {
		return false
	}
	
	// Allow localhost and localhost-style domains (no dots required)
	// Also allow regular domains with dots
	return true // Basic validation passed
}

// handleSendMessage handles sending messages with threading and attachment support
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== SEND MESSAGE REQUEST START ===")
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))
	log.Printf("Content-Length: %s", r.Header.Get("Content-Length"))
	log.Printf("User-Agent: %s", r.Header.Get("User-Agent"))
	log.Printf("Authorization present: %t", r.Header.Get("Authorization") != "")
	
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: User not found in context")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "authentication_error",
			"message": "User not found in context",
		})
		log.Printf("=== SEND MESSAGE REQUEST END (AUTH ERROR) ===")
		return
	}
	
	log.Printf("Authenticated user: %s (ID: %d)", user.Username, user.ID)

	// Check if this is a multipart form (for file uploads) or JSON
	contentType := r.Header.Get("Content-Type")
	log.Printf("Detected content type: %s", contentType)
	
	if strings.Contains(contentType, "multipart/form-data") {
		log.Printf("Routing to handleSendMessageWithFiles")
		s.handleSendMessageWithFiles(w, r, user)
		return
	}

	log.Printf("Processing as JSON request")
	// Set JSON response header
	w.Header().Set("Content-Type", "application/json")

	// Handle JSON request (backward compatibility)
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "invalid_json",
			"message": fmt.Sprintf("Failed to parse JSON request: %v", err),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE REQUEST END (JSON ERROR) ===")
		return
	}
	
	log.Printf("Decoded JSON request: To=%s, Subject=%s, Body length=%d, IsHTML=%t", 
		req.To, req.Subject, len(req.Body), req.IsHTML)

	// Validation with detailed error messages
	if req.To == "" {
		log.Printf("ERROR: Missing recipient")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "missing_recipient",
			"message": "Recipient email address is required",
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE REQUEST END (MISSING RECIPIENT) ===")
		return
	}

	if req.Subject == "" {
		log.Printf("ERROR: Missing subject")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "missing_subject",
			"message": "Email subject is required",
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE REQUEST END (MISSING SUBJECT) ===")
		return
	}

	// Validate email format
	if !isValidEmail(req.To) {
		log.Printf("ERROR: Invalid email format: %s", req.To)
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "invalid_email",
			"message": fmt.Sprintf("Invalid email format: %s", req.To),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE REQUEST END (INVALID EMAIL) ===")
		return
	}

	// Create from address
	fromAddress := fmt.Sprintf("%s@%s", user.Username, s.config.ServerHost)
	log.Printf("From address: %s", fromAddress)

	// Check if recipient is local or external
	var toUserID *int
	if strings.Contains(req.To, "@") {
		parts := strings.Split(req.To, "@")
		if len(parts) == 2 && parts[1] == s.config.ServerHost {
			// Local user
			localUser, err := s.userRepo.GetByUsername(parts[0])
			if err != nil {
				log.Printf("ERROR: Failed to lookup local user: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				response := map[string]interface{}{
					"success": false,
					"error":   "user_lookup_failed",
					"message": fmt.Sprintf("Failed to lookup recipient user: %v", err),
				}
				log.Printf("Sending error response: %+v", response)
				json.NewEncoder(w).Encode(response)
				log.Printf("=== SEND MESSAGE REQUEST END (USER LOOKUP FAILED) ===")
				return
			} else if localUser != nil {
				toUserID = &localUser.ID
				log.Printf("Found local recipient: %s (ID: %d)", parts[0], localUser.ID)
			} else {
				log.Printf("Local user %s not found, treating as external", parts[0])
			}
		} else {
			log.Printf("External recipient: %s", req.To)
		}
	}

	// Prepare threading parameters
	var threadIDPtr *string
	var parentIDPtr *int
	if req.ThreadID != "" {
		threadIDPtr = &req.ThreadID
		log.Printf("Thread ID: %s", req.ThreadID)
	}
	if req.ParentID > 0 {
		parentIDPtr = &req.ParentID
		log.Printf("Parent ID: %d", req.ParentID)
	}

	// Store message in database with threading support
	log.Printf("Creating message in database...")
	message, err := s.messageRepo.CreateWithThreading(&user.ID, toUserID, fromAddress, req.To, req.Subject, req.Body, req.IsHTML, threadIDPtr, parentIDPtr)
	if err != nil {
		log.Printf("ERROR: Failed to store message: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]interface{}{
			"success": false,
			"error":   "message_creation_failed",
			"message": fmt.Sprintf("Failed to create message in database: %v", err),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE REQUEST END (DB ERROR) ===")
		return
	}
	
	log.Printf("Message created successfully with ID: %d", message.ID)

	// Notify SSE clients if it's a local message
	if toUserID != nil {
		log.Printf("Notifying SSE clients for local message")
		go s.notifyNewMessage(message)
	}

	// If external recipient, try federation
	federationError := ""
	if toUserID == nil && strings.Contains(req.To, "@") {
		parts := strings.Split(req.To, "@")
		if len(parts) == 2 {
			log.Printf("Attempting federation to %s", parts[1])
			err := s.relay.SendMessage(fromAddress, req.To, req.Subject, req.Body, parts[1])
			if err != nil {
				federationError = fmt.Sprintf("Federation to %s failed: %v", parts[1], err)
				log.Printf("WARNING: %s", federationError)
				// Don't fail the whole request - message is stored locally
			} else {
				log.Printf("Federation successful to %s", parts[1])
			}
		}
	}

	// Prepare response
	response := map[string]interface{}{
		"success": true,
		"message": "Message sent successfully",
		"id":      message.ID,
	}

	// Include federation warning if there was an issue
	if federationError != "" {
		response["warnings"] = []string{federationError}
	}

	log.Printf("Sending success response: %+v", response)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	log.Printf("=== SEND MESSAGE REQUEST END (SUCCESS) ===")
}

// handleSendMessageWithFiles handles sending messages with file attachments
func (s *Server) handleSendMessageWithFiles(w http.ResponseWriter, r *http.Request, user *auth.AuthUser) {
	log.Printf("=== SEND MESSAGE WITH FILES REQUEST START ===")
	log.Printf("Handling multipart form upload for user %s (ID: %d)", user.Username, user.ID)
	log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))
	log.Printf("Content-Length: %s", r.Header.Get("Content-Length"))
	
	// Ensure all responses are JSON
	w.Header().Set("Content-Type", "application/json")
	
	// Parse multipart form for file uploads
	log.Printf("Parsing multipart form (max 50MB)...")
	err := r.ParseMultipartForm(50 << 20) // 50MB max memory
	if err != nil {
		log.Printf("ERROR: Failed to parse multipart form: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "failed_to_parse_form",
			"message": fmt.Sprintf("Failed to parse multipart form: %v", err),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (FORM PARSE ERROR) ===")
		return
	}
	
	log.Printf("Multipart form parsed successfully")

	// Extract form fields
	to := r.FormValue("to")
	subject := r.FormValue("subject")
	body := r.FormValue("body")
	isHTMLStr := r.FormValue("is_html")
	isHTML := isHTMLStr == "true"
	threadID := r.FormValue("thread_id")
	parentIDStr := r.FormValue("parent_id")

	log.Printf("Form values extracted:")
	log.Printf("  to: '%s'", to)
	log.Printf("  subject: '%s'", subject)
	log.Printf("  body length: %d", len(body))
	log.Printf("  body preview: '%.100s%s'", body, func() string { if len(body) > 100 { return "..." } else { return "" } }())
	log.Printf("  is_html (raw): '%s'", isHTMLStr)
	log.Printf("  is_html (parsed): %t", isHTML)
	log.Printf("  thread_id: '%s'", threadID)
	log.Printf("  parent_id: '%s'", parentIDStr)

	// Count available files
	fileCount := 0
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if files, exists := r.MultipartForm.File["attachments"]; exists {
			fileCount = len(files)
		}
	}
	log.Printf("  attachments count: %d", fileCount)

	// Validation with detailed error messages
	if to == "" {
		log.Printf("ERROR: Missing required field: to")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "missing_recipient",
			"message": "Recipient email address is required",
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (MISSING TO) ===")
		return
	}

	if subject == "" {
		log.Printf("ERROR: Missing required field: subject")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "missing_subject",
			"message": "Email subject is required",
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (MISSING SUBJECT) ===")
		return
	}

	// Validate email format
	if !isValidEmail(to) {
		log.Printf("ERROR: Invalid email format: %s", to)
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"success": false,
			"error":   "invalid_email",
			"message": fmt.Sprintf("Invalid email format: %s", to),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (INVALID EMAIL) ===")
		return
	}

	// Parse parent ID if provided
	var parentID *int
	if parentIDStr != "" {
		if pid, err := strconv.Atoi(parentIDStr); err == nil {
			parentID = &pid
			log.Printf("Parsed parent ID: %d", pid)
		} else {
			log.Printf("ERROR: Invalid parent ID format: %s", parentIDStr)
			w.WriteHeader(http.StatusBadRequest)
			response := map[string]interface{}{
				"success": false,
				"error":   "invalid_parent_id",
				"message": fmt.Sprintf("Invalid parent ID format: %s", parentIDStr),
			}
			log.Printf("Sending error response: %+v", response)
			json.NewEncoder(w).Encode(response)
			log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (INVALID PARENT ID) ===")
			return
		}
	}

	// Parse thread ID if provided
	var threadIDPtr *string
	if threadID != "" {
		threadIDPtr = &threadID
		log.Printf("Thread ID: %s", threadID)
	}

	// Create from address
	fromAddress := fmt.Sprintf("%s@%s", user.Username, s.config.ServerHost)
	log.Printf("From address: %s -> To address: %s", fromAddress, to)

	// Check if recipient is local or external
	var toUserID *int
	if strings.Contains(to, "@") {
		parts := strings.Split(to, "@")
		if len(parts) == 2 && parts[1] == s.config.ServerHost {
			// Local user
			log.Printf("Looking up local user: %s", parts[0])
			localUser, err := s.userRepo.GetByUsername(parts[0])
			if err != nil {
				log.Printf("ERROR: Failed to lookup local user: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				response := map[string]interface{}{
					"success": false,
					"error":   "user_lookup_failed",
					"message": fmt.Sprintf("Failed to lookup recipient user: %v", err),
				}
				log.Printf("Sending error response: %+v", response)
				json.NewEncoder(w).Encode(response)
				log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (USER LOOKUP FAILED) ===")
				return
			} else if localUser != nil {
				toUserID = &localUser.ID
				log.Printf("Found local user %s with ID %d", parts[0], localUser.ID)
			} else {
				log.Printf("Local user %s not found, treating as external", parts[0])
			}
		} else {
			log.Printf("External recipient: %s (host: %s)", to, parts[1])
		}
	}

	// Store message in database with threading support
	log.Printf("Creating message with threading support...")
	message, err := s.messageRepo.CreateWithThreading(&user.ID, toUserID, fromAddress, to, subject, body, isHTML, threadIDPtr, parentID)
	if err != nil {
		log.Printf("ERROR: Failed to store message: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]interface{}{
			"success": false,
			"error":   "message_creation_failed",
			"message": fmt.Sprintf("Failed to create message in database: %v", err),
		}
		log.Printf("Sending error response: %+v", response)
		json.NewEncoder(w).Encode(response)
		log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (DB ERROR) ===")
		return
	}
	log.Printf("Message created successfully with ID: %d", message.ID)

	// Handle file attachments
	attachmentCount := 0
	attachmentErrors := []string{}
	if files := r.MultipartForm.File["attachments"]; len(files) > 0 {
		log.Printf("Processing %d file attachments", len(files))
		for i, fileHeader := range files {
			log.Printf("Processing attachment %d: %s (%d bytes)", i+1, fileHeader.Filename, fileHeader.Size)
			
			// Check file size limit (50MB)
			if fileHeader.Size > 50*1024*1024 {
				errorMsg := fmt.Sprintf("File %s is too large (%d bytes, max 50MB)", fileHeader.Filename, fileHeader.Size)
				log.Printf("WARNING: %s", errorMsg)
				attachmentErrors = append(attachmentErrors, errorMsg)
				continue
			}
			
			// Open uploaded file
			file, err := fileHeader.Open()
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err)
				log.Printf("WARNING: %s", errorMsg)
				attachmentErrors = append(attachmentErrors, errorMsg)
				continue
			}
			defer file.Close()

			// Read file content
			fileData, err := io.ReadAll(file)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to read file %s: %v", fileHeader.Filename, err)
				log.Printf("WARNING: %s", errorMsg)
				attachmentErrors = append(attachmentErrors, errorMsg)
				continue
			}

			// Generate unique filename
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), fileHeader.Filename)
			contentType := fileHeader.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			
			log.Printf("Storing attachment: %s (original: %s, type: %s, size: %d)", 
				filename, fileHeader.Filename, contentType, len(fileData))
			
			// Store attachment in database
			attachment, err := s.attachmentRepo.Create(
				message.ID,
				filename,
				fileHeader.Filename,
				contentType,
				int64(len(fileData)),
				nil, // file_path (we store in DB for now)
				fileData,
			)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to store attachment %s: %v", fileHeader.Filename, err)
				log.Printf("WARNING: %s", errorMsg)
				attachmentErrors = append(attachmentErrors, errorMsg)
			} else {
				log.Printf("Attachment stored successfully with ID: %d", attachment.ID)
				attachmentCount++
			}
		}
	}
	log.Printf("Successfully processed %d attachments (errors: %d)", attachmentCount, len(attachmentErrors))

	// Notify SSE clients if it's a local message
	if toUserID != nil {
		log.Printf("Notifying SSE clients for local message")
		go s.notifyNewMessage(message)
	}

	// If external recipient, try federation
	federationError := ""
	if toUserID == nil && strings.Contains(to, "@") {
		parts := strings.Split(to, "@")
		if len(parts) == 2 {
			log.Printf("Attempting federation to %s", parts[1])
			err := s.relay.SendMessage(fromAddress, to, subject, body, parts[1])
			if err != nil {
				federationError = fmt.Sprintf("Federation to %s failed: %v", parts[1], err)
				log.Printf("WARNING: %s", federationError)
				// Don't fail the whole request - message is stored locally
			} else {
				log.Printf("Federation successful to %s", parts[1])
			}
		}
	}

	log.Printf("Message sent successfully - ID: %d, attachments: %d", message.ID, attachmentCount)
	
	// Prepare response with detailed information
	response := map[string]interface{}{
		"success":    true,
		"message":    "Message sent successfully",
		"id":         message.ID,
		"attachments": map[string]interface{}{
			"processed": attachmentCount,
			"total":     len(r.MultipartForm.File["attachments"]),
		},
	}

	// Include warnings if there were attachment errors
	if len(attachmentErrors) > 0 {
		response["warnings"] = attachmentErrors
		log.Printf("Including %d attachment warnings", len(attachmentErrors))
	}

	// Include federation warning if there was an issue
	if federationError != "" {
		if response["warnings"] == nil {
			response["warnings"] = []string{}
		}
		response["warnings"] = append(response["warnings"].([]string), federationError)
		log.Printf("Including federation warning")
	}

	log.Printf("Sending success response: %+v", response)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	log.Printf("=== SEND MESSAGE WITH FILES REQUEST END (SUCCESS) ===")
}

// handleFederationRelay handles incoming federation messages
func (s *Server) handleFederationRelay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var msg federation.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid_json",
			"message": fmt.Sprintf("Failed to parse JSON request: %v", err),
		})
		return
	}

	// Find recipient user
	if !strings.Contains(msg.To, "@") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid_recipient_format",
			"message": "Invalid recipient format",
		})
		return
	}

	parts := strings.Split(msg.To, "@")
	if len(parts) != 2 || parts[1] != s.config.ServerHost {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "recipient_not_on_server",
			"message": "Recipient not on this server",
		})
		return
	}

	user, err := s.userRepo.GetByUsername(parts[0])
	if err != nil {
		log.Printf("Failed to lookup user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "user_lookup_failed",
			"message": fmt.Sprintf("Failed to lookup user: %v", err),
		})
		return
	}

	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "user_not_found",
			"message": "User not found",
		})
		return
	}

	// Store message
	_, err = s.messageRepo.Create(nil, &user.ID, msg.From, msg.To, msg.Subject, msg.Body)
	if err != nil {
		log.Printf("Failed to store federated message: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "message_storage_failed",
			"message": fmt.Sprintf("Failed to store message: %v", err),
		})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  "delivered",
	})
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

// handleGetThread retrieves all messages in a thread
func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	threadID := vars["threadId"]

	if threadID == "" {
		http.Error(w, "Thread ID is required", http.StatusBadRequest)
		return
	}

	// Get all messages in the thread
	messages, err := s.messageRepo.GetThreadByID(threadID)
	if err != nil {
		log.Printf("Failed to get thread: %v", err)
		http.Error(w, "Failed to get thread", http.StatusInternalServerError)
		return
	}

	// Filter messages to only show those the user can access
	var filteredMessages []*database.Message
	for _, msg := range messages {
		if (msg.ToUserID != nil && *msg.ToUserID == user.ID) || 
		   (msg.FromUserID != nil && *msg.FromUserID == user.ID) {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredMessages)
}

// handleGetAttachment serves attachment files
func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	attachmentIDStr := vars["id"]

	attachmentID, err := strconv.Atoi(attachmentIDStr)
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}

	// Get attachment info
	attachment, err := s.attachmentRepo.GetByID(attachmentID)
	if err != nil {
		log.Printf("Failed to get attachment: %v", err)
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	if attachment == nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Get the message to verify user access
	message, err := s.messageRepo.GetByID(attachment.MessageID)
	if err != nil || message == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Verify user has access to this message
	if (message.ToUserID == nil || *message.ToUserID != user.ID) &&
	   (message.FromUserID == nil || *message.FromUserID != user.ID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get file data
	fileData, err := s.attachmentRepo.GetFileData(attachmentID)
	if err != nil {
		log.Printf("Failed to get file data: %v", err)
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.OriginalName))
	w.Header().Set("Content-Length", strconv.FormatInt(attachment.FileSize, 10))

	// Serve file
	w.Write(fileData)
} 