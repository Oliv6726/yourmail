package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the mail server
type Config struct {
	TCPPort    int    `json:"tcp_port"`
	HTTPPort   int    `json:"http_port"`
	ServerHost string `json:"server_host"`
	ServerID   string `json:"server_id"`
	LogLevel   string `json:"log_level"`
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Try to load .env file (optional)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		TCPPort:    getEnvAsInt("TCP_PORT", 7777),
		HTTPPort:   getEnvAsInt("HTTP_PORT", 8080),
		ServerHost: getEnv("SERVER_HOST", "localhost"),
		ServerID:   getEnv("SERVER_ID", "localhost"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}

	return config
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as integer with a fallback value
func getEnvAsInt(name string, fallback int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

// GetTCPAddr returns the TCP server address
func (c *Config) GetTCPAddr() string {
	return ":" + strconv.Itoa(c.TCPPort)
}

// GetHTTPAddr returns the HTTP server address
func (c *Config) GetHTTPAddr() string {
	return ":" + strconv.Itoa(c.HTTPPort)
} 