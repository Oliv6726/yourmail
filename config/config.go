package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	TCPPort    string
	HTTPPort   string
	Host       string
	ServerHost string

	// Database settings
	DatabasePath string

	// JWT settings
	JWTSecret     string
	JWTExpiration time.Duration

	// Environment
	Environment string
	
	// CORS settings
	AllowedOrigins []string
}

// Load loads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		TCPPort:    getEnv("TCP_PORT", "7777"),
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		Host:       getEnv("HOST", "localhost"),
		ServerHost: getEnv("SERVER_HOST", "localhost"),

		// Database
		DatabasePath: getEnv("DATABASE_PATH", "./data/yourmail.db"),

		// JWT
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		JWTExpiration: getEnvDuration("JWT_EXPIRATION", "24h"),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),

		// CORS
		AllowedOrigins: []string{
			getEnv("FRONTEND_URL", "http://localhost:3000"),
			"http://localhost:3001", // Alternative frontend port
		},
	}

	log.Printf("✅ Configuration loaded:")
	log.Printf("   TCP Port: %s", config.TCPPort)
	log.Printf("   HTTP Port: %s", config.HTTPPort)
	log.Printf("   Database: %s", config.DatabasePath)
	log.Printf("   Environment: %s", config.Environment)
	log.Printf("   JWT Expiration: %s", config.JWTExpiration)

	return config
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as duration or returns default
func getEnvDuration(key, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("Invalid duration format for %s: %s, using default: %s", key, value, defaultValue)
		duration, _ = time.ParseDuration(defaultValue)
	}
	return duration
}

// getEnvInt gets an environment variable as int or returns default
func getEnvInt(key string, defaultValue int) int {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Invalid integer format for %s: %s, using default: %d", key, value, defaultValue)
		return defaultValue
	}
	
	return intValue
} 