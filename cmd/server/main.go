package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"yourmail/config"
	"yourmail/internal/database"
	"yourmail/internal/federation"
	"yourmail/internal/httpapi"
	"yourmail/internal/protocol"
)

func main() {
	log.Println("🚀 Starting YourMail Server v2.0.0")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewDatabase(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed test users in development
	if cfg.Environment == "development" {
		if err := db.SeedTestUsers(); err != nil {
			log.Printf("Failed to seed test users: %v", err)
		}
	}

	// Initialize federation relay
	relay := federation.NewRelay(cfg.ServerHost, cfg.HTTPPort)

	// Initialize HTTP API server
	httpServer := httpapi.NewServer(cfg, db, relay)

	// Initialize TCP protocol server
	tcpServer := protocol.NewServer(cfg, db)

	// Start servers in goroutines
	go func() {
		log.Printf("Starting TCP server on :%s", cfg.TCPPort)
		if err := tcpServer.Start(); err != nil {
			log.Fatalf("TCP server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting HTTP server on :%s", cfg.HTTPPort)
		if err := httpServer.Start(); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("✅ YourMail Server is running")
	log.Println("📧 TCP Protocol: localhost:" + cfg.TCPPort)
	log.Println("🌐 HTTP API: http://localhost:" + cfg.HTTPPort)
	log.Println("💾 Database: " + cfg.DatabasePath)
	log.Println("Press Ctrl+C to stop")

	// Block until signal received
	<-c
	log.Println("🛑 Shutting down YourMail Server...")
} 