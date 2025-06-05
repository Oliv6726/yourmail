package federation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Message represents a federated message
type Message struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

// Relay handles federation with other mail servers
type Relay struct {
	serverHost string
	httpPort   string
}

// NewRelay creates a new federation relay
func NewRelay(serverHost, httpPort string) *Relay {
	return &Relay{
		serverHost: serverHost,
		httpPort:   httpPort,
	}
}

// SendMessage sends a message to a remote server
func (r *Relay) SendMessage(from, to, subject, body, targetHost string) error {
	// Don't federate to ourselves
	if targetHost == r.serverHost {
		return nil
	}

	msg := Message{
		From:      from,
		To:        to,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
	}

	// Try HTTP federation on port 8080
	url := fmt.Sprintf("http://%s:8080/federation/relay", targetHost)
	
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Federation failed to %s: %v", targetHost, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("federation server responded with status %d", resp.StatusCode)
	}

	log.Printf("✅ Message federated successfully to %s", targetHost)
	return nil
} 