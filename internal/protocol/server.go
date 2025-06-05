package protocol

import (
	"log"
	"net"

	"yourmail/config"
	"yourmail/internal/database"
)

// Server represents the TCP protocol server
type Server struct {
	config       *config.Config
	db           *database.DB
	userRepo     *database.UserRepository
	messageRepo  *database.MessageRepository
	listener     net.Listener
	shutdownChan chan struct{}
}

// NewServer creates a new TCP protocol server
func NewServer(cfg *config.Config, db *database.DB) *Server {
	attachmentRepo := database.NewAttachmentRepository(db)
	return &Server{
		config:       cfg,
		db:           db,
		userRepo:     database.NewUserRepository(db),
		messageRepo:  database.NewMessageRepository(db, attachmentRepo),
		listener:     nil,
		shutdownChan: make(chan struct{}),
	}
}

// Start starts the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":"+s.config.TCPPort)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("🔌 TCP server listening on :%s", s.config.TCPPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each client connection in a separate goroutine
		go func() {
			session := NewSession(conn, s.userRepo, s.messageRepo, s.config.ServerHost)
			session.Handle()
		}()
	}
} 