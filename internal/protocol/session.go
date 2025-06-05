package protocol

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

	"yourmail/internal/database"
)

// Session represents a TCP client session
type Session struct {
	conn         net.Conn
	scanner      *bufio.Scanner
	userRepo     *database.UserRepository
	msgRepo      *database.MessageRepository
	serverHost   string
	authenticated bool
	currentUser   *database.User
	currentMessage struct {
		to      string
		subject string
		body    string
	}
}

// NewSession creates a new session
func NewSession(conn net.Conn, userRepo *database.UserRepository, msgRepo *database.MessageRepository, serverHost string) *Session {
	return &Session{
		conn:       conn,
		scanner:    bufio.NewScanner(conn),
		userRepo:   userRepo,
		msgRepo:    msgRepo,
		serverHost: serverHost,
	}
}

// Handle processes the client session
func (s *Session) Handle() {
	defer s.conn.Close()
	
	clientAddr := s.conn.RemoteAddr().String()
	log.Printf("New TCP connection from %s", clientAddr)
	
	s.sendResponse("220 YourMail Server ready")
	
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" {
			continue
		}
		
		log.Printf("[%s] Command: %s", clientAddr, line)
		
		parts := strings.SplitN(line, " ", 2)
		command := strings.ToUpper(parts[0])
		var args string
		if len(parts) > 1 {
			args = parts[1]
		}
		
		switch command {
		case "CONNECT":
			s.handleConnect(args)
		case "SEND":
			s.handleSend(args)
		case "SUBJECT":
			s.handleSubject(args)
		case "BODY":
			s.handleBody(args)
		case "QUIT":
			s.handleQuit()
			return
		case "HELP":
			s.handleHelp()
		case "LIST":
			s.handleList()
		case "READ":
			s.handleRead(args)
		default:
			s.sendResponse("500 Unknown command: " + command)
		}
	}
	
	if err := s.scanner.Err(); err != nil {
		log.Printf("Scanner error for %s: %v", clientAddr, err)
	}
	
	log.Printf("Connection from %s closed", clientAddr)
}

// handleConnect authenticates the user
func (s *Session) handleConnect(args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) != 2 {
		s.sendResponse("501 Usage: CONNECT <username> <password>")
		return
	}
	
	username, password := parts[0], parts[1]
	
	// Authenticate user
	user, err := s.userRepo.Authenticate(username, password)
	if err != nil {
		log.Printf("Authentication error: %v", err)
		s.sendResponse("500 Authentication failed")
		return
	}
	
	if user == nil {
		s.sendResponse("535 Authentication failed")
		return
	}
	
	s.authenticated = true
	s.currentUser = user
	s.sendResponse(fmt.Sprintf("250 Hello %s, authenticated successfully", username))
	log.Printf("User %s authenticated successfully", username)
}

// handleSend sets the recipient for the message
func (s *Session) handleSend(args string) {
	if !s.authenticated {
		s.sendResponse("530 Not authenticated")
		return
	}
	
	if args == "" {
		s.sendResponse("501 Usage: SEND <recipient@host>")
		return
	}
	
	s.currentMessage.to = args
	s.sendResponse("250 Recipient set to " + args)
}

// handleSubject sets the subject for the message
func (s *Session) handleSubject(args string) {
	if !s.authenticated {
		s.sendResponse("530 Not authenticated")
		return
	}
	
	if s.currentMessage.to == "" {
		s.sendResponse("503 Use SEND command first")
		return
	}
	
	s.currentMessage.subject = args
	s.sendResponse("250 Subject set")
}

// handleBody sets the body and sends the message
func (s *Session) handleBody(args string) {
	if !s.authenticated {
		s.sendResponse("530 Not authenticated")
		return
	}
	
	if s.currentMessage.to == "" || s.currentMessage.subject == "" {
		s.sendResponse("503 Use SEND and SUBJECT commands first")
		return
	}
	
	s.currentMessage.body = args
	
	// Create from address
	fromAddress := fmt.Sprintf("%s@%s", s.currentUser.Username, s.serverHost)
	
	// Check if recipient is local
	var toUserID *int
	if strings.Contains(s.currentMessage.to, "@") {
		parts := strings.Split(s.currentMessage.to, "@")
		if len(parts) == 2 && parts[1] == s.serverHost {
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
	message, err := s.msgRepo.Create(&s.currentUser.ID, toUserID, fromAddress, s.currentMessage.to, s.currentMessage.subject, s.currentMessage.body)
	if err != nil {
		log.Printf("Failed to store message: %v", err)
		s.sendResponse("550 Failed to send message")
		return
	}
	
	// Clear current message
	s.currentMessage = struct {
		to      string
		subject string
		body    string
	}{}
	
	s.sendResponse(fmt.Sprintf("250 Message sent successfully (ID: %d)", message.ID))
	log.Printf("Message sent from %s to %s", fromAddress, s.currentMessage.to)
}

// handleList shows the user's inbox
func (s *Session) handleList() {
	if !s.authenticated {
		s.sendResponse("530 Not authenticated")
		return
	}
	
	messages, err := s.msgRepo.GetInboxForUser(s.currentUser.ID, 20, 0)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		s.sendResponse("550 Failed to retrieve messages")
		return
	}
	
	if len(messages) == 0 {
		s.sendResponse("250 No messages in inbox")
		return
	}
	
	s.sendResponse(fmt.Sprintf("250 %d messages:", len(messages)))
	for i, msg := range messages {
		readStatus := "unread"
		if msg.ReadStatus {
			readStatus = "read"
		}
		s.sendResponse(fmt.Sprintf("  %d. From: %s | Subject: %s | %s | %s", 
			i+1, msg.FromAddress, msg.Subject, readStatus, msg.CreatedAt.Format("2006-01-02 15:04")))
	}
}

// handleRead shows a specific message
func (s *Session) handleRead(args string) {
	if !s.authenticated {
		s.sendResponse("530 Not authenticated")
		return
	}
	
	if args == "" {
		s.sendResponse("501 Usage: READ <message_number>")
		return
	}
	
	// For simplicity, let's get recent messages and use the number as index
	messages, err := s.msgRepo.GetInboxForUser(s.currentUser.ID, 20, 0)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		s.sendResponse("550 Failed to retrieve messages")
		return
	}
	
	// Parse message number (1-based)
	msgNum := 0
	if _, err := fmt.Sscanf(args, "%d", &msgNum); err != nil || msgNum < 1 || msgNum > len(messages) {
		s.sendResponse("501 Invalid message number")
		return
	}
	
	msg := messages[msgNum-1]
	
	// Mark as read
	s.msgRepo.MarkAsRead(msg.ID)
	
	s.sendResponse("250 Message content:")
	s.sendResponse(fmt.Sprintf("From: %s", msg.FromAddress))
	s.sendResponse(fmt.Sprintf("To: %s", msg.ToAddress))
	s.sendResponse(fmt.Sprintf("Subject: %s", msg.Subject))
	s.sendResponse(fmt.Sprintf("Date: %s", msg.CreatedAt.Format("2006-01-02 15:04:05")))
	s.sendResponse("")
	s.sendResponse(msg.Body)
	s.sendResponse(".")
}

// handleHelp shows available commands
func (s *Session) handleHelp() {
	s.sendResponse("214 Available commands:")
	s.sendResponse("  CONNECT <username> <password> - Authenticate")
	s.sendResponse("  SEND <recipient@host> - Set recipient")
	s.sendResponse("  SUBJECT <subject> - Set message subject")
	s.sendResponse("  BODY <body> - Set message body and send")
	s.sendResponse("  LIST - Show inbox")
	s.sendResponse("  READ <number> - Read specific message")
	s.sendResponse("  HELP - Show this help")
	s.sendResponse("  QUIT - Close connection")
}

// handleQuit closes the connection
func (s *Session) handleQuit() {
	s.sendResponse("221 Goodbye")
	s.conn.Close()
}

// sendResponse sends a response to the client
func (s *Session) sendResponse(message string) {
	response := message + "\r\n"
	s.conn.Write([]byte(response))
} 