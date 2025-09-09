package auth

import (
	"log"
)

// AuthbossLogger implements authboss.Logger interface
type AuthbossLogger struct{}

// NewAuthbossLogger creates a new Authboss logger
func NewAuthbossLogger() *AuthbossLogger {
	return &AuthbossLogger{}
}

// Info logs an info message
func (l *AuthbossLogger) Info(msg string) {
	log.Printf("[AUTHBOSS INFO] %s", msg)
}

// Error logs an error message
func (l *AuthbossLogger) Error(msg string) {
	log.Printf("[AUTHBOSS ERROR] %s", msg)
}